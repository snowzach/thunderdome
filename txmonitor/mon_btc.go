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

	// If we disconnect, loop and try again
	for {

		// Connect to the transaction stream, subscribe to transactions
		txclient, err := txm.lclient.SubscribeTransactions(context.Background(), &lnrpc.GetTransactionsRequest{})
		if err != nil {
			txm.logger.Fatalw("Could not SubscribeTransactions", "error", err)
		}
		txm.logger.Info("Listening for transactions...")

		// Catch up on existing transactions
		// TODO: Optimize what transactions we will fetch
		txsDetails, err := txm.lclient.GetTransactions(context.Background(), &lnrpc.GetTransactionsRequest{})
		if err != nil {
			txm.logger.Fatalw("Could not GetTransactions", "error", err)
		}

		for _, tx := range txsDetails.Transactions {
			txm.logger.Infow("Processing existing transaction", "hash", tx.TxHash, "confirmations", tx.NumConfirmations)
			txm.parseBTCTranaction(tx.TxHash, tx.NumConfirmations)
		}

		for {
			tx, err := txclient.Recv()
			if err == io.EOF {
				txm.logger.Error("TXM Closed Connection")
				break
			}
			// Don't process a transaction until it has at least 1 confirmation
			txm.logger.Infow("Processing transaction", "hash", tx.TxHash, "confirmations", tx.NumConfirmations)
			txm.parseBTCTranaction(tx.TxHash, tx.NumConfirmations)
		}

		time.Sleep(10 * time.Second)
	}
}

// This will parse the transaction and add it to the ledger
func (txm *TXMonitor) parseBTCTranaction(txHash string, confirmations int32) {

	// Fetch the transaction via the JSON RPC
	chHash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		txm.logger.Errorf("Could not parse hash %s: %v", txHash, err)
		return
	}

	if confirmations > 5 {

		tx, err := txm.rpcc.GetTransaction(chHash)
		if err != nil {
			txm.logger.Errorw("Error getting transaction tx:%s %v", txHash, err)
			return
		}

		for _, d := range tx.Details {
			txm.logger.Infof("Details: account:%s, address:%s, amount:%f vout:%d", d.Account, d.Address, d.Amount, d.Vout)
		}

	} else {

		// Get the raw transaction from the JSON RPC
		rawTx, err := txm.rpcc.GetRawTransaction(chHash)
		if err != nil {
			txm.logger.Errorw("Error getting raw transaction tx:%s %v", txHash, err)
			test, err := txm.rpcc.GetTransaction(chHash)
			if err != nil {
				txm.logger.Errorf("Error getting transaction tx:%s %v", txHash, err)
			}
			txm.logger.Info("TX %v", test)
			return
		}
		// Convert it into the wire format
		wTx := rawTx.MsgTx()

		// Parse all of the outputs
		for height, vout := range wTx.TxOut {

			// Attempt to parse simple addresses out of the script
			_, addresses, _, err := txscript.ExtractPkScriptAddrs(vout.PkScript, &chaincfg.RegressionNetParams)
			if err != nil { // Could not decode
				continue
			} else if len(addresses) != 1 {
				continue
			}

			account, err := txm.store.AccountGetByAddress(context.Background(), addresses[0].String())
			if err == store.ErrNotFound {
				continue
			} else if err != nil {
				continue
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

			err = txm.store.ProcessLedgerRecord(context.Background(), lr)

		}
	}

}
