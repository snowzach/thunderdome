package txmonitor

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/lightningnetwork/lnd/lnrpc"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

// MonitorBTC will spin up, search for existing transactions, and listen for incoming transactions
func (txm *TXMonitor) MonitorBTC() {

	ctx := context.Background()

	// If we disconnect, loop and try again
	for {

		// Connect to the transaction stream, subscribe to transactions
		txclient, err := txm.lclient.SubscribeTransactions(ctx, &lnrpc.GetTransactionsRequest{})
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
			txm.parseBTCTranaction(ctx, tx.TxHash, tx.NumConfirmations)
		}

		for {
			tx, err := txclient.Recv()
			if err == io.EOF {
				txm.logger.Errorw("TXM Closed Connection", "monitor", "btc")
				break
			}
			// Don't process a transaction until it has at least 1 confirmation
			txm.logger.Infow("Processing transaction", "monitor", "btc", "hash", tx.TxHash, "confirmations", tx.NumConfirmations)
			txm.parseBTCTranaction(ctx, tx.TxHash, tx.NumConfirmations)
		}

		time.Sleep(10 * time.Second)
	}
}

// This will parse the transaction and add it to the ledger
func (txm *TXMonitor) parseBTCTranaction(ctx context.Context, txHash string, confirmations int32) {

	var foundTxOut bool

	// Check to see if this is an outbound transaction we know about already
	lrIn, err := txm.store.GetLedgerRecord(ctx, txHash, tdrpc.OUT)
	if err == nil {
		if confirmations > 0 && lrIn.Status != tdrpc.COMPLETED {
			lrIn.Status = tdrpc.COMPLETED
			err = txm.store.ProcessLedgerRecord(ctx, lrIn)
			if err != nil {
				txm.logger.Fatalw("ProcessLedgerRecord Out Error", "monitor", "btc", "error", err)
			}
		}
		// On the insane chance we somehow paid another address in this wallet, let it continue to process
	}

	chHash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		txm.logger.Errorw("Could not parse hash", "monitor", "btc", "hash", txHash, "error", err)
		return
	}

	// Get the raw transaction from the JSON RPC - this is generally mempool transactions or more recent
	rawTx, err := txm.rpcc.GetRawTransaction(chHash)
	if err != nil {

		// It may not have hit the database yet, wait 2 seconds and try again
		if confirmations == 0 {
			time.Sleep(2 * time.Second)
			rawTx, err = txm.rpcc.GetRawTransaction(chHash)
		}
	}
	// We found the raw transaction
	if err == nil {

		// Convert it into the wire format
		wTx := rawTx.MsgTx()

		// Parse all of the outputs
		for height, vout := range wTx.TxOut {

			// Attempt to parse simple addresses out of the script
			_, addresses, _, err := txscript.ExtractPkScriptAddrs(vout.PkScript, &chaincfg.RegressionNetParams)
			if err != nil { // Could not decode, it's not one of ours
				txm.logger.Errorw("Could not decode transaction script", "monitor", "btc", "hash", txHash, "height", height)
				continue
			} else if len(addresses) != 1 {
				txm.logger.Errorw("Multiple addresses found for transaction", "monitor", "btc", "hash", txHash, "height", height)
				continue
			}

			// Find the associated account
			account, err := txm.store.AccountGetByAddress(ctx, addresses[0].String())
			if err == store.ErrNotFound {
				continue
			} else if err != nil {
				txm.logger.Fatalw("AccountGetByAddress Error", "monitor", "btc", "error", err)
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
				txm.logger.Fatalw("ProcessLedgerRecord Error", "monitor", "btc", "error", err)
			}

			foundTxOut = true
		}

	} else {

		// This checks farther back in history but gives more sparse details
		tx, err := txm.rpcc.GetTransaction(chHash)
		if err != nil {
			txm.logger.Errorw("Could not find transaction", "monitor", "btc", "hash", txHash, "error", err)
			return
		}

		// Process the details
		for _, d := range tx.Details {
			// It's a payment to us, for now we will ignore outgoing payments
			if d.Amount > 0 {
				continue
			}

			// Find the associated account
			account, err := txm.store.AccountGetByAddress(ctx, d.Address)
			if err == store.ErrNotFound {
				continue
			} else if err != nil {
				txm.logger.Fatalw("AccountGetByAddress Error", "monitor", "btc", "error", err)
			}

			// Convert it to a LedgerRecord
			lr := &tdrpc.LedgerRecord{
				Id:        fmt.Sprintf("%s:%d", txHash, d.Vout),
				AccountId: account.Id,
				Status:    tdrpc.PENDING,
				Type:      tdrpc.BTC,
				Direction: tdrpc.IN,
				Value:     -int64(d.Amount * 100000000), // BTC -> Satoshi
			}
			if confirmations > 0 {
				lr.Status = tdrpc.COMPLETED
			}

			err = txm.store.ProcessLedgerRecord(ctx, lr)
			if err != nil {
				txm.logger.Fatalw("ProcessLedgerRecord Error", "monitor", "btc", "error", err)
			}

			foundTxOut = true

		}
	}

	// We had this transaction but could not relate it to an account
	if !foundTxOut {
		txm.logger.Warnw("No account found for transaction", "monitor", "btc", "hash", txHash)
	}

}
