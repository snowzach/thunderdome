package monitor

import (
	"context"
	"encoding/hex"
	"io"

	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"git.coinninja.net/backend/thunderdome/conf"
	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func (m *Monitor) MonitorLN() {

	// Handle shutting down
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-conf.Stop.Chan()
		cancel()
	}()

	// Connect to the transaction stream
	conf.Stop.Add(1)
	invclient, err := m.lclient.SubscribeInvoices(ctx, &lnrpc.InvoiceSubscription{
		// TODO: We could start with the highest completed, not expired index
		AddIndex:    1,
		SettleIndex: 1,
	})
	if err != nil {
		m.logger.Fatalw("Could not SubscribeInvoices", "monitor", "ln", "error", err)
	}

	m.logger.Info("Listening for lightning transactions...", "monitor", "ln")

	for !conf.Stop.Bool() {

		var handledTx bool

		// Get the next message
		invoice, err := invclient.Recv()
		if err == io.EOF {
			m.logger.Fatalw("LightningMonitor EOF", "monitor", "ln")
			continue
		} else if status.Code(err) == codes.Canceled {
			m.logger.Info("LightningMonitor Shutting Down")
			break
		} else if err != nil {
			m.logger.Fatalw("LightningMonitor Failure", "monitor", "ln", "error", err)
		}

		// We only need to process settled transactions
		if !invoice.Settled {
			continue
		}

		// Get the payment_hash
		paymentHash := hex.EncodeToString(invoice.RHash)

		// Find the existing ledger record outbound
		lr, err := m.store.GetLedgerRecord(ctx, paymentHash, tdrpc.OUT)
		if err == nil {
			// Update it with the value and status
			lr.Status = tdrpc.COMPLETED
			lr.Value = invoice.AmtPaidSat
			handledTx = true

			// Process the payment
			err = m.store.ProcessLedgerRecord(ctx, lr)
			if err != nil {
				m.logger.Errorw("ProcessLedgerRecord Out Error", "monitor", "ln", "error", err, "payment_hash", paymentHash)
				continue
			}

			m.logger.Infow("Processed Out Invoice", "monitor", "ln", "payment_hash", paymentHash, "value", invoice.AmtPaidSat)

		} else if err != store.ErrNotFound {
			m.logger.Fatalw("GetLedgerRecord Error", "monitor", "ln", "error", err)
		}

		// Find the existing ledger record inbound
		lr, err = m.store.GetLedgerRecord(ctx, paymentHash, tdrpc.IN)
		if err == nil {
			// Update it with the value and status
			lr.Status = tdrpc.COMPLETED
			lr.Value = invoice.AmtPaidSat
			handledTx = true

			// Process the payment
			err = m.store.ProcessLedgerRecord(ctx, lr)
			if err != nil {
				m.logger.Errorw("ProcessLedgerRecord In Error", "monitor", "ln", "error", err, "payment_hash", paymentHash)
				continue
			}

			m.logger.Infow("Processed In Invoice", "monitor", "ln", "payment_hash", paymentHash, "value", invoice.AmtPaidSat)

		} else if err != store.ErrNotFound {
			m.logger.Fatalw("GetLedgerRecord Error", "monitor", "ln", "error", err)
		}

		if !handledTx {
			m.logger.Infow("Did not find LedgerRecord for Invoice", "monitor", "ln", "payment_hash", paymentHash, "value", invoice.AmtPaidSat)
		}

	}

	_ = invclient.CloseSend()
	conf.Stop.Done()

}