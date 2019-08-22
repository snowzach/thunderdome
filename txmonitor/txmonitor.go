package txmonitor

import (
	"context"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
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

	chain *chaincfg.Params
}

func NewTXMonitor(store thunderdome.Store, conn *grpc.ClientConn) (*TXMonitor, error) {

	logger := zap.S().With("package", "txmonitor")

	// Fetch the node info to make sure we know our own identity for self-payments
	lclient := lnrpc.NewLightningClient(conn)
	info, err := lclient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, err
	}

	// Make sure we are connected to only one chain
	if len(info.Chains) != 1 {
		return nil, fmt.Errorf("Could not determine chain. len=%d", len(info.Chains))
	}

	lndChain := info.Chains[0]

	if lndChain.Chain != "bitcoin" {
		return nil, fmt.Errorf("LND chain = %s", lndChain.Chain)
	}

	// Create an array of chains such that we can pick the one we want
	var chain *chaincfg.Params
	chains := []*chaincfg.Params{
		&chaincfg.MainNetParams,
		&chaincfg.RegressionNetParams,
		&chaincfg.SimNetParams,
		&chaincfg.TestNet3Params,
	}
	// Find the selected chain
	for _, cp := range chains {
		if lndChain.Network == cp.Name {
			chain = cp
			break
		}
	}
	if chain == nil {
		return nil, fmt.Errorf("Could not find chain %s", lndChain.Network)
	}

	logger.Infof("TXMonitor auto-configured for chain %s", lndChain.Network)

	// Return the server
	return &TXMonitor{
		logger: logger,
		store:  store,

		conn:    conn,
		lclient: lclient,

		chain: chain,
	}, nil

}
