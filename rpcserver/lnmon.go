package rpcserver

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
)

func (s *RPCServer) LightningMonitor() {

	invmlogger := s.logger.With("package", "invmonitor")

	// If we disconnect, loop and try again
	for {
		// Connect to the transaction stream
		invclient, err := s.lclient.SubscribeInvoices(context.Background(), &lnrpc.InvoiceSubscription{
			AddIndex:    1,
			SettleIndex: 1,
		})
		if err != nil {
			invmlogger.Fatalw("Could not SubscribeInvoices", "error", err)
		}

		invmlogger.Info("Listening for transactions...")

		for {
			inv, err := invclient.Recv()
			if err == io.EOF {
				invmlogger.Error("TXM Closed Connection")
				break
			}

			invmlogger.Infow("INVMonitor Message", "inv", inv)
		}

		time.Sleep(10 * time.Second)

	}

}
