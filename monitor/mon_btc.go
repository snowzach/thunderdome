package monitor

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/DataDog/datadog-go/statsd"
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
			m.parseBTCTranaction(ctx, rawTx, tx.NumConfirmations, tx.TotalFees, false)
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
			m.parseBTCTranaction(ctx, rawTx, tx.NumConfirmations, tx.TotalFees, true)
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
func (m *Monitor) parseBTCTranaction(ctx context.Context, rawTx []byte, confirmations int32, txFee int64, alert bool) {

	// Decode the transaction
	tx, err := btcutil.NewTxFromBytes(rawTx)
	if err != nil {
		m.logger.Errorw("Could not decode transaction", "monitor", "btc", "hash")
		return
	}
	txHash := tx.Hash().String() // Get txHash
	wTx := tx.MsgTx()            // Convert to wire format

	var foundTx bool // Check to see if we've processed this transaction at all

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
	// - NO LONGER REQUIRED: Inputs no longer need to be confirmed
	// - Sequence >= wire.MaxTxInSequenceNum-1 (not replace by fee)
	// - Fee must be at least 1 sat/vbyte
	// - The user must have less than tdome.topup_instant_user_count_limit
	// - Combined with this transaction it must be less than tdome.topup_instant_user_value_limit
	// - Combined all system pending transactions must be less than tdome.topup_instant_system_value_limit
	// We must also have a blocc client we can ask
	var validForInstantTopUp bool = false
topUp: // Use a for so we can break at any time on failure and drop out of the block
	for confirmations == 0 && config.GetBool("tdome.topup_instant_enabled") && m.bclient != nil {

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

		// Get the transactions from blocc
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

		var fee int64

		// Sum the input values
		for _, vin := range wTx.TxIn {
			hash := vin.PreviousOutPoint.Hash.String()
			bloccTx := idsMap[hash]

			// Transaction was missing from blocc
			if bloccTx == nil {
				m.logger.Infow("Missing input for instant top-up", "hash", txHash, "input_hash", hash)
				break topUp
			}

			if int(vin.PreviousOutPoint.Index) < len(bloccTx.Out) {
				fee += bloccTx.Out[int(vin.PreviousOutPoint.Index)].Value
			} else {
				m.logger.Infow("Missing input index for instant top-up", "hash", txHash, "input_hash", hash, "input_index", vin.PreviousOutPoint.Index)
				break topUp
			}
		}

		// Subtract the output values
		for _, vout := range wTx.TxOut {
			fee -= vout.Value
		}

		feePerVByte := float64(fee) / float64(wTx.SerializeSizeStripped())
		if feePerVByte < 1.0 {
			m.logger.Infow("Insufficient fee for instant top-up", "hash", txHash, "fee_per_vbyte", feePerVByte)
			break topUp
		}

		// Everything succeeded, set to true, break out of for loop
		validForInstantTopUp = true
		break
	}

	// Keep track of the original fee in all the ledger record
	// txFee will be set to zero if used for a fee free top up on an output (so it's not double credited)
	networkFee := txFee

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

		// The ledgerRecordId is the txHash:height
		ledgerRecordId := fmt.Sprintf("%s:%d", txHash, height)

		// Convert it to a LedgerRecord
		lr := &tdrpc.LedgerRecord{
			Id:         ledgerRecordId,
			AccountId:  account.Id,
			Status:     tdrpc.PENDING,
			Type:       tdrpc.BTC,
			Direction:  tdrpc.IN,
			Value:      vout.Value,
			NetworkFee: networkFee,
		}
		// No confirmations and is thus far still validForInstantTopUp
		if confirmations == 0 && validForInstantTopUp {

			// Check this users current top up activity
			lrStats, err := m.store.GetLedgerRecordStats(ctx, map[string]string{
				"type":       tdrpc.BTC.String(),
				"direction":  tdrpc.IN.String(),
				"status":     tdrpc.COMPLETED.String(),
				"request":    tdrpc.RequestInstantPending,
				"account_id": account.Id,
			}, time.Time{})
			if err != nil {
				m.logger.Errorw("Could not get user ledger record stats", "error", err, "account_id", account.Id)
				continue
			}

			// The user is allowed only so many pending transactions
			if lrStats.Count >= config.GetInt64("tdome.topup_instant_user_count_limit") {
				m.ddclient.Event(&statsd.Event{
					Title:     "TopUp User Limit Exceeded",
					Text:      fmt.Sprintf(`TopUp TX:%s Value:%d Address:%s AccountId:%s`, ledgerRecordId, lr.Value, addresses[0].String(), account.Id),
					Priority:  statsd.Normal,
					AlertType: statsd.Warning,
				})
				m.logger.Warnw("TopUp User Count Exceeded", "tx", ledgerRecordId, "value", lr.Value, "address", addresses[0].String(), "account_id", account.Id)
				validForInstantTopUp = false
			}

			// Their pending transactions cannot exceed the user limit
			if lrStats.Value+vout.Value > config.GetInt64("tdome.topup_instant_user_value_limit") {
				m.ddclient.Event(&statsd.Event{
					Title:     "TopUp User Value Exceeded",
					Text:      fmt.Sprintf(`TopUp TX:%s Value:%d Address:%s AccountId:%s`, ledgerRecordId, lr.Value, addresses[0].String(), account.Id),
					Priority:  statsd.Normal,
					AlertType: statsd.Warning,
				})
				m.logger.Warnw("TopUp User Value Exceeded", "tx", ledgerRecordId, "value", lr.Value, "address", addresses[0].String(), "account_id", account.Id)
				validForInstantTopUp = false
			}

			// If we're still valid, look up the system wide stats
			if validForInstantTopUp {

				// Check the system stats
				lrStats, err = m.store.GetLedgerRecordStats(ctx, map[string]string{
					"type":      tdrpc.BTC.String(),
					"direction": tdrpc.IN.String(),
					"status":    tdrpc.COMPLETED.String(),
					"request":   tdrpc.RequestInstantPending,
				}, time.Time{})
				if err != nil {
					m.logger.Errorw("Could not get system ledger record stats", "error", err, "account_id", account.Id)
					continue
				}

				// The system can only have a total value pending at any given time
				if lrStats.Value > config.GetInt64("tdome.topup_instant_system_value_limit") {
					m.ddclient.Event(&statsd.Event{
						Title:     "TopUp System Value Exceeded",
						Text:      fmt.Sprintf(`TopUp TX:%s Value:%d Address:%s AccountId:%s`, ledgerRecordId, lr.Value, addresses[0].String(), account.Id),
						Priority:  statsd.Normal,
						AlertType: statsd.Warning,
					})
					m.logger.Warnw("TopUp System Value Exceeded", "tx", ledgerRecordId, "value", lr.Value, "address", addresses[0].String(), "account_id", account.Id)
					validForInstantTopUp = false
				}
			}

			// If we're still validForInstantTopUp after limit checks
			if validForInstantTopUp {
				lr.Status = tdrpc.COMPLETED
				lr.Memo = "Instant Topup"
				lr.Request = tdrpc.RequestInstantPending
				m.logger.Infow("Transaction marked for instant top-up", "hash", ledgerRecordId)
			}

		} else if confirmations > 0 {

			// Check to see if there is a previous lr and update the status rather than replace it (keeping memo and request)
			prevLr, err := m.store.GetLedgerRecord(ctx, ledgerRecordId, tdrpc.IN)
			if err == nil {
				lr = prevLr
			}

			lr.Status = tdrpc.COMPLETED

			// If it was an instant topup transaction request, mark it as completed
			if lr.Request == tdrpc.RequestInstantPending {
				lr.Request = tdrpc.RequestInstantCompleted
			}

		}

		// If it's a large transaction, send an alert
		if lr.Value > config.GetInt64("tdome.topup_alert_large") && alert {
			if m.ddclient != nil {
				m.ddclient.Event(&statsd.Event{
					Title:     "Large TopUp Received",
					Text:      fmt.Sprintf(`Large TopUp TX:%s Value:%d Address:%s AccountId:%s`, txHash, lr.Value, addresses[0].String(), account.Id),
					Priority:  statsd.Normal,
					AlertType: statsd.Warning,
				})
			}
			m.logger.Warnw("Large TopUp Received", "tx", txHash, "value", lr.Value, "address", addresses[0].String(), "account_id", account.Id)
		}

		err = m.store.ProcessLedgerRecord(ctx, lr)
		if err != nil {
			m.logger.Errorw("ProcessLedgerRecord Error", "monitor", "btc", "error", err)
		}

		foundTx = true
	}

	// Check the received time and alert if it's signifigantly delayed
	if confirmations == 0 && m.bclient != nil && alert {
		if tx, err := m.bclient.GetTransaction(ctx, &blocc.Get{Id: txHash, Data: true}); err == nil {
			if receivedTimeString, ok := tx.Data["received_time"]; ok {
				if receivedTime, err := strconv.ParseInt(receivedTimeString, 10, 64); err == nil {
					if time.Now().UTC().Unix()-receivedTime > 300 {
						if m.ddclient != nil {
							m.ddclient.Event(&statsd.Event{
								Title:     "Delayed TopUp Transaction",
								Text:      fmt.Sprintf(`Delayed TopUp Transaction TX:%s Seconds:%d`, txHash, time.Now().UTC().Unix()-receivedTime),
								Priority:  statsd.Normal,
								AlertType: statsd.Error,
							})
						}
						m.logger.Warnw("Delayed TopUp Transaction", "tx", txHash, "seconds", time.Now().UTC().Unix()-receivedTime)
					}
				}
			}
		}
	}

	// We had this transaction but could not relate it to an account
	if !foundTx {
		m.logger.Warnw("No account found for transaction", "monitor", "btc", "hash", txHash)
	}

}
