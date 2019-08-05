package txmonitor

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/lightningnetwork/lnd/lnrpc"
	config "github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/conf"
	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// MonitorBTC will spin up, search for existing transactions, and listen for incoming transactions
func (txm *TXMonitor) MonitorBTC() {

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
		txclient, err = txm.lclient.SubscribeTransactions(ctx, &lnrpc.GetTransactionsRequest{})
		if err != nil {
			txm.logger.Fatalw("Could not SubscribeTransactions", "error", err)
		}
		txm.logger.Infow("Listening for transactions...", "monitor", "btc")

		// Catch up on existing transactions
		// TODO: Optimize what transactions we will fetch
		txsDetails, err := txm.lclient.GetTransactions(ctx, &lnrpc.GetTransactionsRequest{})
		if err != nil {
			txm.logger.Fatalw("Could not GetTransactions", "monitor", "btc", "error", err)
		}

		for _, tx := range txsDetails.Transactions {
			txm.logger.Infow("Processing existing transaction", "monitor", "btc", "hash", tx.TxHash, "confirmations", tx.NumConfirmations)
			rawTx, err := hex.DecodeString(tx.RawTxHex)
			if err != nil {
				txm.logger.Errorw("Could not decode transaction", "monitor", "btc", "hash", tx.TxHash)
				continue
			}
			txm.parseBTCTranaction(ctx, rawTx, tx.NumConfirmations, tx.TotalFees)
		}

		// Main loop
		for {
			tx, err := txclient.Recv()
			if err == io.EOF {
				txm.logger.Errorw("TXM Closed Connection", "monitor", "btc")
				break
			} else if status.Code(err) == codes.Canceled {
				txm.logger.Info("TXM Shutting Down")
				break
			} else if err != nil {
				txm.logger.Errorw("TXM Error", "monitor", "btc", "error", err)
				break
			}

			// Don't process a transaction until it has at least 1 confirmation
			txm.logger.Infow("Processing transaction", "monitor", "btc", "hash", tx.TxHash, "confirmations", tx.NumConfirmations)
			rawTx, err := hex.DecodeString(tx.RawTxHex)
			if err != nil {
				txm.logger.Errorw("Could not decode transaction", "monitor", "btc", "hash", tx.TxHash)
				continue
			}
			txm.parseBTCTranaction(ctx, rawTx, tx.NumConfirmations, tx.TotalFees)
		}

		// We were disconnected, reconnect and try again
		select {
		case <-time.After(10 * time.Second):
		case <-conf.Stop.Chan():
		}
	}

	if txclient != nil {
		txclient.CloseSend()
	}
	conf.Stop.Done()
}

// This will parse the transaction and add it to the ledger
func (txm *TXMonitor) parseBTCTranaction(ctx context.Context, rawTx []byte, confirmations int32, txFee int64) {

	// Decode the transaction
	tx, err := btcutil.NewTxFromBytes(rawTx)
	if err != nil {
		txm.logger.Errorw("Could not decode transaction", "monitor", "btc", "hash")
		return
	}
	txHash := tx.Hash().String() // Get txHash
	wTx := tx.MsgTx()            // Convert to wire format

	var foundTx bool

	// Check to see if this has an outbound transaction we know about already
	lrOut, err := txm.store.GetLedgerRecord(ctx, txHash, tdrpc.OUT)
	if err == nil {
		if confirmations > 0 && lrOut.Status != tdrpc.COMPLETED {
			lrOut.Status = tdrpc.COMPLETED
			err = txm.store.ProcessLedgerRecord(ctx, lrOut)
			if err != nil {
				txm.logger.Fatalw("ProcessLedgerRecord Out Error", "monitor", "btc", "error", err)
			}
		}
		// On the insane chance we somehow paid another address in this wallet, let it continue to process
		foundTx = true
	}

	// Parse all of the outputs
	for height, vout := range wTx.TxOut {

		// Attempt to parse simple addresses out of the script
		_, addresses, _, err := txscript.ExtractPkScriptAddrs(vout.PkScript, txm.chain)
		if err != nil { // Could not decode, it's not one of ours
			txm.logger.Errorw("Could not decode ouput script", "monitor", "btc", "hash", txHash, "height", height)
			continue
		} else if vout.Value == 0 {
			// No value, just skip it
			continue
		} else if len(addresses) != 1 {
			txm.logger.Warnw("Could not determine output addresses. Skipping.", "monitor", "btc", "hash", txHash, "height", height, "addresses", len(addresses))
			continue
		}

		// Find the associated account
		account, err := txm.store.AccountGetByAddress(ctx, addresses[0].String())
		if err == store.ErrNotFound {
			continue
		} else if err != nil {
			txm.logger.Fatalw("AccountGetByAddress Error", "monitor", "btc", "error", err)
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
		if confirmations > 0 {
			lr.Status = tdrpc.COMPLETED
		}

		err = txm.store.ProcessLedgerRecord(ctx, lr)
		if err != nil {
			txm.logger.Errorw("ProcessLedgerRecord Error", "monitor", "btc", "error", err)
		}

		foundTx = true
	}

	// We had this transaction but could not relate it to an account
	if !foundTx {
		txm.logger.Warnw("No account found for transaction", "monitor", "btc", "hash", txHash)
	}

}
