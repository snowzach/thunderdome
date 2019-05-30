package txmonitor

import (
	"context"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/lightningnetwork/lnd/lnrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"git.coinninja.net/backend/thunderdome/thunderdome"
)

type TXMonitor struct {
	logger *zap.SugaredLogger
	store  thunderdome.Store

	conn    *grpc.ClientConn
	lclient lnrpc.LightningClient

	rpcc  *rpcclient.Client
	chain *chaincfg.Params
}

func NewTXMonitor(store thunderdome.Store, conn *grpc.ClientConn, rpcc *rpcclient.Client, chain *chaincfg.Params) (*TXMonitor, error) {

	// Fetch the node info to make sure we know our own identity for self-payments
	lclient := lnrpc.NewLightningClient(conn)
	_, err := lclient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, err
	}

	// Return the server
	return &TXMonitor{
		logger: zap.S().With("package", "txmonitor"),
		store:  store,

		conn:    conn,
		lclient: lclient,

		rpcc:  rpcc,
		chain: chain,
	}, nil

}
