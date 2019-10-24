package monitor

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
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
			m.logger.Infow("Processing existing transaction", "monitor", "btc", "hash", tx.TxHash, "confirmations", tx.NumConfirmations, "value", tx.Amount, "fees", tx.TotalFees)
			rawTx, err := hex.DecodeString(tx.RawTxHex)
			if err != nil {
				m.logger.Errorw("Could not decode transaction", "monitor", "btc", "hash", tx.TxHash)
				continue
			}
			m.parseBTCTranaction(ctx, rawTx, tx.NumConfirmations, false)
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

			m.logger.Infow("Processing transaction", "monitor", "btc", "hash", tx.TxHash, "confirmations", tx.NumConfirmations, "value", tx.Amount, "fees", tx.TotalFees)
			rawTx, err := hex.DecodeString(tx.RawTxHex)
			if err != nil {
				m.logger.Errorw("Could not decode transaction", "monitor", "btc", "hash", tx.TxHash)
				continue
			}
			m.parseBTCTranaction(ctx, rawTx, tx.NumConfirmations, true)
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
func (m *Monitor) parseBTCTranaction(ctx context.Context, rawTx []byte, confirmations int32, shouldAlert bool) {

	// Decode the transaction
	tx, err := btcutil.NewTxFromBytes(rawTx)
	if err != nil {
		m.logger.Errorw("Could not decode transaction", "monitor", "btc", "error", err)
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

	// This is the amount of possible credit we can get for a fee free topup (if enabled). It will be adjusted as it's used
	var txFee int64
	var feeFreeTopupCredit int64

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

		// Find the associated account
		account, err := m.store.GetAccountByAddress(ctx, addresses[0].String())
		if err == store.ErrNotFound {
			continue
		} else if err != nil {
			m.logger.Fatalw("GetAccountByAddress Error", "monitor", "btc", "error", err)
		}

		// The ledgerRecordId is the txHash:height
		ledgerRecordId := fmt.Sprintf("%s:%d", txHash, height)

		// Check to see if there is a previous lr
		prevLr, err := m.store.GetLedgerRecord(ctx, ledgerRecordId, tdrpc.IN)
		if err == store.ErrNotFound {
			prevLr = nil // No prevLr
		} else if err != nil {
			m.logger.Fatalw("GetLedgerRecord Error", "monitor", "btc", "error", err)
		}

		// If the record is already completed in the database and the request isn't still instant_pending, there is nothing to process
		if prevLr != nil && prevLr.Status == tdrpc.COMPLETED && prevLr.Request != tdrpc.RequestInstantPending {
			foundTx = true
			continue
		}

		// We do not yet have the fee for this transaction, fetch it from blocc if we can
		if txFee == 0 && m.bclient != nil {
			var hasBech32Inputs bool
			// Fetch the fee by looking up the inputs and subtracting this txn outputs
			txFee, hasBech32Inputs = m.GetTxnFeeAndBech32Inputs(ctx, txHash, wTx)
			// We looked up the fee, we can also credit the feeFreeTopupCredit
			// But we will only credit fee free when the inputs are bech32 (from drop bit)
			if hasBech32Inputs {
				feeFreeTopupCredit = txFee
			}
		}

		m.logger.Debugw("Processing TxOut", "tx", ledgerRecordId, "value", vout.Value, "address", addresses[0].String(), "account_id", account.Id, "fee", txFee)
		memo := "TopUp"

		// Handle fee free topup
		if feeFreeTopupCredit > 0 && config.GetBool("tdome.topup_fee_free") {
			if feeLimit := config.GetInt64("tdome.topup_fee_free_limit"); feeFreeTopupCredit > feeLimit {
				feeFreeTopupCredit = feeLimit
			}
			vout.Value += feeFreeTopupCredit
			feeFreeTopupCredit = 0 // Make sure if this tx somehow pays multiple people we don't double up the fee
			memo += " FeeFree"
		}

		// Convert it to a LedgerRecord
		lr := &tdrpc.LedgerRecord{
			Id:         ledgerRecordId,
			AccountId:  account.Id,
			Status:     tdrpc.PENDING,
			Type:       tdrpc.BTC,
			Direction:  tdrpc.IN,
			Value:      vout.Value,
			NetworkFee: txFee, // For inbound transactions this is for documentation only and can be
			Memo:       memo,
		}

		// No confirmations yet,
		if confirmations == 0 {

			// If there are no confirmations, we can check to see if this transaction is eligible for instant TopUp.
			// To be eligible all inputs must be:
			// - Sequence >= wire.MaxTxInSequenceNum-1 (not replace by fee)
			// - Fee must be at least 1 sat/vbyte
			// - The user must have less than tdome.topup_instant_user_count_limit
			// - Combined with this transaction it must be less than tdome.topup_instant_user_value_limit
			// - Combined all system pending transactions must be less than tdome.topup_instant_system_value_limit
			var validForInstantTopUp = true

			// Check none of the inputs are replacable by fee
			for _, vin := range wTx.TxIn {
				// If any the of the inputs have a sequence less than MaxTxInSequenceNum - 1, they could be replaced and are not valid
				if vin.Sequence < wire.MaxTxInSequenceNum-1 {
					m.logger.Infow("Invalid sequence for instant top-up", "tx", ledgerRecordId, "value", lr.Value, "address", addresses[0].String(), "account_id", account.Id, "sequence", vin.Sequence)
					validForInstantTopUp = false
				}
			}

			// Calculate and ensure that we have at least a mininum fee attached (if we could not determine the fee, this will fail)
			feePerVByte := float64(txFee) / float64(wTx.SerializeSizeStripped())
			if feePerVByte < 1.0 {
				m.logger.Infow("Insufficient or unknown fee for instant top-up", "tx", ledgerRecordId, "value", lr.Value, "address", addresses[0].String(), "account_id", account.Id, "fee", txFee)
				validForInstantTopUp = false
			}

			// If we're still valid, look up user stats to ensure they are not exceeding limits
			if validForInstantTopUp {
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
					if m.ddclient != nil {
						m.ddclient.Event(&statsd.Event{
							Title:     "TopUp User Limit Exceeded",
							Text:      fmt.Sprintf(`TopUp TX:%s Value:%d Address:%s AccountId:%s`, ledgerRecordId, lr.Value, addresses[0].String(), account.Id),
							Priority:  statsd.Normal,
							AlertType: statsd.Warning,
						})
					}
					m.logger.Warnw("TopUp User Count Exceeded", "tx", ledgerRecordId, "value", lr.Value, "address", addresses[0].String(), "account_id", account.Id)
					validForInstantTopUp = false
				}

				// Their pending transactions cannot exceed the user limit
				if lrStats.Value+vout.Value > config.GetInt64("tdome.topup_instant_user_value_limit") {
					if m.ddclient != nil {
						m.ddclient.Event(&statsd.Event{
							Title:     "TopUp User Value Exceeded",
							Text:      fmt.Sprintf(`TopUp TX:%s Value:%d Address:%s AccountId:%s`, ledgerRecordId, lr.Value, addresses[0].String(), account.Id),
							Priority:  statsd.Normal,
							AlertType: statsd.Warning,
						})
					}
					m.logger.Warnw("TopUp User Value Exceeded", "tx", ledgerRecordId, "value", lr.Value, "address", addresses[0].String(), "account_id", account.Id)
					validForInstantTopUp = false
				}
			}

			// If we're still valid, look up the system wide stats to ensure no limits are exceeded
			if validForInstantTopUp {

				// Check the system stats
				lrStats, err := m.store.GetLedgerRecordStats(ctx, map[string]string{
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
					if m.ddclient != nil {
						m.ddclient.Event(&statsd.Event{
							Title:     "TopUp System Value Exceeded",
							Text:      fmt.Sprintf(`TopUp TX:%s Value:%d Address:%s AccountId:%s`, ledgerRecordId, lr.Value, addresses[0].String(), account.Id),
							Priority:  statsd.Normal,
							AlertType: statsd.Warning,
						})
					}
					m.logger.Warnw("TopUp System Value Exceeded", "tx", ledgerRecordId, "value", lr.Value, "address", addresses[0].String(), "account_id", account.Id)
					validForInstantTopUp = false
				}
			}

			// If we're still validForInstantTopUp after limit checks
			if validForInstantTopUp {
				lr.Status = tdrpc.COMPLETED
				lr.Memo += " Instant"
				lr.Request = tdrpc.RequestInstantPending
				m.logger.Infow("Transaction marked for instant top-up", "hash", ledgerRecordId)
			}

		} else if confirmations > 0 {

			// If a previous record exists, use it rather than creating a new one (keeping memo and request)
			if prevLr != nil {
				lr = prevLr
			}

			// Ensure status = completed
			lr.Status = tdrpc.COMPLETED

			// If it was an instant topup transaction request, mark it as completed
			if lr.Request == tdrpc.RequestInstantPending {
				lr.Request = tdrpc.RequestInstantCompleted
			}

		}

		// If it's a large transaction, send an alert
		if lr.Value > config.GetInt64("tdome.topup_alert_large") && shouldAlert {
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
	if confirmations == 0 && m.bclient != nil && shouldAlert {
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

// Returns the fee for this transaction (or zero) by fetching the inputs from this transaction and calculating fees
func (m *Monitor) GetTxnFeeAndBech32Inputs(ctx context.Context, txHash string, wTx *wire.MsgTx) (int64, bool) {

	idsMap := make(map[string]*blocc.Tx)
	idsSlice := make([]string, 0)
	// Build the list of txids
	for _, vin := range wTx.TxIn {
		hash := vin.PreviousOutPoint.Hash.String()
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
		m.logger.Errorw("GetTxnFee FindTransactions Error", "error", err)
		return 0, false
	}

	// Parse them out into the map for lookup
	for _, bloccTx := range txns.Transactions {
		idsMap[bloccTx.TxId] = bloccTx
	}

	var fee int64
	var foundInputs bool
	var hasBech32Inputs = true

	// Sum the input values
	for _, vin := range wTx.TxIn {
		hash := vin.PreviousOutPoint.Hash.String()
		bloccTx := idsMap[hash]

		// Transaction was missing from blocc
		if bloccTx == nil {
			m.logger.Infow("GetTxnFee missing input", "hash", txHash, "input_hash", hash)
			return 0, false
		}

		// Look through the addresses, if any are not bech32, make note
		for _, bloccVOut := range bloccTx.Out {
			foundInputs = true // Just a safety to ensure we found some inputs
			if len(bloccVOut.Addresses) != 1 || !strings.HasPrefix(bloccVOut.Addresses[0], "bc1") {
				hasBech32Inputs = false
			}
		}

		if int(vin.PreviousOutPoint.Index) < len(bloccTx.Out) {
			fee += bloccTx.Out[int(vin.PreviousOutPoint.Index)].Value
		} else {
			m.logger.Infow("GetTxnFee missing input index", "hash", txHash, "input_hash", hash, "input_index", vin.PreviousOutPoint.Index)
			return 0, false
		}
	}

	// Subtract the output values
	for _, vout := range wTx.TxOut {
		fee -= vout.Value
	}

	return fee, (foundInputs && hasBech32Inputs)

}
