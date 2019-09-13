package monitor

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/lightningnetwork/lnd/lnrpc"
	config "github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/blocc/blocc"

	"git.coinninja.net/backend/thunderdome/conf"
	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// MonitorBTC will spin up, search for existing transactions, and listen for incoming transactions
func (m *Monitor) MonitorBTC() {

	// Handle shutting down
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-conf.Stop.Chan()
		cancel()
	}()

	conf.Stop.Add(1)
	var txclient lnrpc.Lightning_SubscribeTransactionsClient
	var err error

	// If we disconnect, loop and try again
	for !conf.Stop.Bool() {

		// Connect to the transaction stream, subscribe to transactions
		txclient, err = m.lclient.SubscribeTransactions(ctx, &lnrpc.GetTransactionsRequest{})
		if err != nil {
			m.logger.Fatalw("Could not SubscribeTransactions", "error", err)
		}
		m.logger.Infow("Listening for transactions...", "monitor", "btc")

		// Catch up on existing transactions
		// TODO: Optimize what transactions we will fetch by setting addr index
		txsDetails, err := m.lclient.GetTransactions(ctx, &lnrpc.GetTransactionsRequest{})
		if err != nil {
			m.logger.Fatalw("Could not GetTransactions", "monitor", "btc", "error", err)
		}

		for _, tx := range txsDetails.Transactions {
			m.logger.Infow("Processing existing transaction", "monitor", "btc", "hash", tx.TxHash, "confirmations", tx.NumConfirmations)
			rawTx, err := hex.DecodeString(tx.RawTxHex)
			if err != nil {
				m.logger.Errorw("Could not decode transaction", "monitor", "btc", "hash", tx.TxHash)
				continue
			}
			m.parseBTCTranaction(ctx, rawTx, tx.NumConfirmations, tx.TotalFees)
		}

		// Main loop
		for {
			tx, err := txclient.Recv()
			if err == io.EOF {
				m.logger.Errorw("TXM Closed Connection", "monitor", "btc")
				break
			} else if status.Code(err) == codes.Canceled {
				m.logger.Info("TXM Shutting Down")
				break
			} else if err != nil {
				m.logger.Errorw("TXM Error", "monitor", "btc", "error", err)
				break
			}

			// Don't process a transaction until it has at least 1 confirmation
			m.logger.Infow("Processing transaction", "monitor", "btc", "hash", tx.TxHash, "confirmations", tx.NumConfirmations)
			rawTx, err := hex.DecodeString(tx.RawTxHex)
			if err != nil {
				m.logger.Errorw("Could not decode transaction", "monitor", "btc", "hash", tx.TxHash)
				continue
			}
			m.parseBTCTranaction(ctx, rawTx, tx.NumConfirmations, tx.TotalFees)
		}

		// We were disconnected, reconnect and try again
		select {
		case <-time.After(10 * time.Second):
		case <-conf.Stop.Chan():
		}
	}

	if txclient != nil {
		_ = txclient.CloseSend()
	}
	conf.Stop.Done()
}

// This will parse the transaction and add it to the ledger
func (m *Monitor) parseBTCTranaction(ctx context.Context, rawTx []byte, confirmations int32, txFee int64) {

	// Decode the transaction
	tx, err := btcutil.NewTxFromBytes(rawTx)
	if err != nil {
		m.logger.Errorw("Could not decode transaction", "monitor", "btc", "hash")
		return
	}
	txHash := tx.Hash().String() // Get txHash
	wTx := tx.MsgTx()            // Convert to wire format

	var foundTx bool

	// Check to see if this has an outbound transaction we know about already
	// These are transactions being send from Thunderdome and will have the txHash as he id
	lrOut, err := m.store.GetLedgerRecord(ctx, txHash, tdrpc.OUT)
	if err == nil {
		if confirmations > 0 && lrOut.Status != tdrpc.COMPLETED {
			lrOut.Status = tdrpc.COMPLETED
			err = m.store.ProcessLedgerRecord(ctx, lrOut)
			if err != nil {
				m.logger.Fatalw("ProcessLedgerRecord Out Error", "monitor", "btc", "error", err)
			}
		}
		// On the insane chance we somehow paid another address in this wallet, let it continue to process
		foundTx = true
	}

	// If there are no confirmations, we can check to see if this transaction is eligible for instant TopUp.
	// To be eligible all inputs must be:
	// - Confirmed
	// - Sequence >= wire.MaxTxInSequenceNum-1
	// We must also have a blocc client we can ask
	var validForInstantTopUp bool = false
topUp: // Use a for so we can break at any time on failure and drop out of the block
	for confirmations == 0 && m.bclient != nil {

		// Build a slice and map of previous transaction ids, deduplicate at the same time
		idsMap := make(map[string]*blocc.Tx)
		idsSlice := make([]string, 0)
		for _, vin := range wTx.TxIn {

			hash := vin.PreviousOutPoint.Hash.String()

			// If any the of the inputs have a sequence less than MaxTxInSequenceNum - 1, they could be replaced and are not valid
			if vin.Sequence < wire.MaxTxInSequenceNum-1 {
				m.logger.Infow("Invalid sequence for instant top-up", "hash", txHash, "input_hash", hash, "sequence", vin.Sequence)
				break topUp
			}

			// Dedup
			if _, ok := idsMap[hash]; ok {
				continue
			}

			idsSlice = append(idsSlice, hash)
			idsMap[hash] = nil
		}

		// Get the transactions
		txns, err := m.bclient.FindTransactions(ctx, &blocc.Find{
			Symbol: blocc.SymbolBTC,
			Ids:    idsSlice,
			Count:  int64(len(idsSlice)),
		})
		if err != nil {
			m.logger.Errorw("Unable to fetch input transactions for instant top-up", "error", err)
			break
		}

		// Parse them out into the map for lookup
		for _, bloccTx := range txns.Transactions {
			idsMap[bloccTx.TxId] = bloccTx
		}

		// Check the input transactions
		for hash, bloccTx := range idsMap {

			// Transaction was missing from blocc
			if bloccTx == nil {
				m.logger.Infow("Missing input for instant top-up", "hash", txHash, "input_hash", hash)
				break topUp
			}

			// This transaction is still unconfirmed
			if bloccTx.BlockHeight == blocc.HeightUnknown {
				m.logger.Infow("Unconfirmed input for instant top-up", "hash", txHash, "input_hash", hash)
				break topUp
			}

		}

		// Everything succeeded, set to true, break out of for loop
		validForInstantTopUp = true
		m.logger.Infow("Transaction eligible for instant top-up", "hash", txHash)
		break
	}

	// Parse all of inbound to thunderdome transactions. These are transaction outputs destined for an address in thunderdome
	for height, vout := range wTx.TxOut {

		// Attempt to parse simple addresses out of the script
		_, addresses, _, err := txscript.ExtractPkScriptAddrs(vout.PkScript, m.chain)
		if err != nil { // Could not decode, it's not one of ours
			m.logger.Errorw("Could not decode ouput script", "monitor", "btc", "hash", txHash, "height", height)
			continue
		} else if vout.Value == 0 {
			// No value, just skip it
			continue
		} else if len(addresses) != 1 {
			m.logger.Warnw("Could not determine output addresses. Skipping.", "monitor", "btc", "hash", txHash, "height", height, "addresses", len(addresses))
			continue
		}

		m.logger.Debugw("Processing TxOut", "height", height, "addresses", addresses[0].String(), "value", vout.Value)

		// Find the associated account
		account, err := m.store.GetAccountByAddress(ctx, addresses[0].String())
		if err == store.ErrNotFound {
			continue
		} else if err != nil {
			m.logger.Fatalw("GetAccountByAddress Error", "monitor", "btc", "error", err)
		}

		// Handle fee free topup
		if config.GetBool("tdome.topup_fee_free") {
			if feeLimit := config.GetInt64("tdome.topup_fee_free_limit"); txFee > feeLimit {
				txFee = feeLimit
			}
			vout.Value += txFee
			txFee = 0 // Make sure if this tx somehow pays multiple people we don't double up the fee
		}

		// Convert it to a LedgerRecord
		lr := &tdrpc.LedgerRecord{
			Id:        fmt.Sprintf("%s:%d", txHash, height),
			AccountId: account.Id,
			Status:    tdrpc.PENDING,
			Type:      tdrpc.BTC,
			Direction: tdrpc.IN,
			Value:     vout.Value,
		}
		// No confirmations, is not replace by fee and instant topup is enabled
		if confirmations == 0 && validForInstantTopUp {
			lr.Status = tdrpc.COMPLETED
		} else if confirmations > 0 {
			lr.Status = tdrpc.COMPLETED
		}

		err = m.store.ProcessLedgerRecord(ctx, lr)
		if err != nil {
			m.logger.Errorw("ProcessLedgerRecord Error", "monitor", "btc", "error", err)
		}

		foundTx = true
	}

	// We had this transaction but could not relate it to an account
	if !foundTx {
		m.logger.Warnw("No account found for transaction", "monitor", "btc", "hash", txHash)
	}

}
