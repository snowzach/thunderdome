package rpcserver

import (
	"context"
	"encoding/hex"
	"io"

	"github.com/lightningnetwork/lnd/lnrpc"

	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func (s *RPCServer) LightningMonitor() {

	invmlogger := s.logger.With("package", "invmonitor")
	ctx := context.Background()

	// Connect to the transaction stream
	invclient, err := s.lclient.SubscribeInvoices(context.Background(), &lnrpc.InvoiceSubscription{
		// TODO: We could start with the highest completed, not expired index
		AddIndex:    1,
		SettleIndex: 1,
	})
	if err != nil {
		invmlogger.Fatalw("Could not SubscribeInvoices", "error", err)
	}

	invmlogger.Info("Listening for lightning transactions...")

	for {

		// Get the next message
		invoice, err := invclient.Recv()
		if err == io.EOF {
			invmlogger.Fatal("LightningMonitor EOF")
		} else if err != nil {
			invmlogger.Fatalw("LightningMonitor Failure", "error", err)
		}

		// We will only handle these when they are settled
		if !invoice.Settled {
			continue
		}

		// Get the payment_hash
		paymentHash := hex.EncodeToString(invoice.RHash)

		// Find the existing ledger record
		lr, err := s.rpcStore.GetLedgerRecord(ctx, paymentHash, tdrpc.IN)
		if err == store.ErrNotFound {
			invmlogger.Errorw("Could not find incoming invoice", "payment_hash", paymentHash)
			continue
		} else if err != nil {
			invmlogger.Fatalw("GetLedgerRecord Error", "error", err)
		}

		// Update it with the value and status
		lr.Status = tdrpc.COMPLETED
		lr.Value = invoice.AmtPaidSat

		// Process the payment
		err = s.rpcStore.ProcessLedgerRecord(ctx, lr)
		if err != nil {
			invmlogger.Errorw("ProcessLedgerRecord Error", "error", err, "payment_hash", paymentHash)
			continue
		}

		invmlogger.Infow("Processed Invoice", "payment_hash", paymentHash, "value", invoice.AmtPaidSat)

	}

}
