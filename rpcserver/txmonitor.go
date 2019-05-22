package rpcserver

import (
	"context"
	"io"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
)

// TransactionMonitor will spin up, search for existing transactions, and listen for incoming transactions
func (s *RPCServer) TransactionMonitor() {

	txmlogger := s.logger.With("package", "txmonitor")

	// If we disconnect, loop and try again
	for {

		// Catch up on existing transactions

		// Connect to the transaction stream
		txclient, err := s.lclient.SubscribeTransactions(context.Background(), &lnrpc.GetTransactionsRequest{})
		if err != nil {
			txmlogger.Fatalw("Could not SubscribeTransactions", "error", err)
		}

		for {
			m, err := txclient.Recv()
			if err == io.EOF {
				txmlogger.Error("TXM Closed Connection")
				break
			}

			txmlogger.Infow("TXMonitor Message", "message", m)
		}

		time.Sleep(10 * time.Second)

	}

}
