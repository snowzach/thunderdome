package rpcserver

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
)

// TransactionMonitor will spin up, search for existing transactions, and listen for incoming transactions
func (s *RPCServer) TransactionMonitor() {

	txmlogger := s.logger.With("package", "txmonitor")

	// If we disconnect, loop and try again
	for {

		// Catch up on existing transactions
		txsDetails, err := s.lclient.GetTransactions(context.Background(), &lnrpc.GetTransactionsRequest{})
		if err != nil {
			txmlogger.Fatalw("Could not GetTransactions", "error", err)
		}

		for _, tx := range txsDetails.Transactions {
			txmlogger.Infow("TXMonitor Tx", "tx", tx, "addresses", strings.Join(tx.DestAddresses, ","))
		}

		// Connect to the transaction stream
		txclient, err := s.lclient.SubscribeTransactions(context.Background(), &lnrpc.GetTransactionsRequest{})
		if err != nil {
			txmlogger.Fatalw("Could not SubscribeTransactions", "error", err)
		}

		txmlogger.Info("Listening for transactions...")

		for {
			tx, err := txclient.Recv()
			if err == io.EOF {
				txmlogger.Error("TXM Closed Connection")
				break
			}

			txmlogger.Infow("TXMonitor Message", "tx", tx, "addresses", strings.Join(tx.DestAddresses, ","))
		}

		time.Sleep(10 * time.Second)

	}

}

func (s *RPCServer) InvoiceMonitor() {

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
