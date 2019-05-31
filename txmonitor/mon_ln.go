package txmonitor

import (
	"context"
	"encoding/hex"
	"io"

	"github.com/lightningnetwork/lnd/lnrpc"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func (txm *TXMonitor) MonitorLN() {

	ctx := context.Background()

	// Connect to the transaction stream
	invclient, err := txm.lclient.SubscribeInvoices(ctx, &lnrpc.InvoiceSubscription{
		// TODO: We could start with the highest completed, not expired index
		AddIndex:    1,
		SettleIndex: 1,
	})
	if err != nil {
		txm.logger.Fatalw("Could not SubscribeInvoices", "monitor", "ln", "error", err)
	}

	txm.logger.Info("Listening for lightning transactions...", "monitor", "ln")

	for {

		var handledTx bool

		// Get the next message
		invoice, err := invclient.Recv()
		if err == io.EOF {
			txm.logger.Fatalw("LightningMonitor EOF", "monitor", "ln")
		} else if err != nil {
			txm.logger.Fatalw("LightningMonitor Failure", "monitor", "ln", "error", err)
		}

		// We only need to process settled transactions
		if !invoice.Settled {
			continue
		}

		// Get the payment_hash
		paymentHash := hex.EncodeToString(invoice.RHash)

		// Find the existing ledger record outbound
		lr, err := txm.store.GetLedgerRecord(ctx, paymentHash, tdrpc.OUT)
		if err == nil {
			// Update it with the value and status
			lr.Status = tdrpc.COMPLETED
			lr.Value = invoice.AmtPaidSat
			handledTx = true

			// Process the payment
			err = txm.store.ProcessLedgerRecord(ctx, lr)
			if err != nil {
				txm.logger.Fatalw("ProcessLedgerRecord Out Error", "monitor", "ln", "error", err, "payment_hash", paymentHash)
				continue
			}

			txm.logger.Infow("Processed Out Invoice", "monitor", "ln", "payment_hash", paymentHash, "value", invoice.AmtPaidSat)

		} else if err != store.ErrNotFound {
			txm.logger.Fatalw("GetLedgerRecord Error", "monitor", "ln", "error", err)
		}

		// Find the existing ledger record inbound
		lr, err = txm.store.GetLedgerRecord(ctx, paymentHash, tdrpc.IN)
		if err == nil {
			// Update it with the value and status
			lr.Status = tdrpc.COMPLETED
			lr.Value = invoice.AmtPaidSat
			handledTx = true

			// Process the payment
			err = txm.store.ProcessLedgerRecord(ctx, lr)
			if err != nil {
				txm.logger.Fatalw("ProcessLedgerRecord In Error", "monitor", "ln", "error", err, "payment_hash", paymentHash)
				continue
			}

			txm.logger.Infow("Processed In Invoice", "monitor", "ln", "payment_hash", paymentHash, "value", invoice.AmtPaidSat)

		} else if err != store.ErrNotFound {
			txm.logger.Fatalw("GetLedgerRecord Error", "monitor", "ln", "error", err)
		}

		if !handledTx {
			txm.logger.Infow("Did not find LedgerRecord for Invoice", "monitor", "ln", "payment_hash", paymentHash, "value", invoice.AmtPaidSat)
		}

	}

}
