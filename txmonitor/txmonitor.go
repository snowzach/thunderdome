package txmonitor

import (
	"context"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/lightningnetwork/lnd/lnrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"git.coinninja.net/backend/blocc/blocc"

	"git.coinninja.net/backend/thunderdome/thunderdome"
)

type TXMonitor struct {
	logger *zap.SugaredLogger
	store  thunderdome.Store

	lclient lnrpc.LightningClient
	bclient blocc.BloccRPCClient

	chain *chaincfg.Params
}

func NewTXMonitor(store thunderdome.Store, lndConn *grpc.ClientConn, bloccConn *grpc.ClientConn) (*TXMonitor, error) {

	logger := zap.S().With("package", "txmonitor")

	// Fetch the node info to make sure we know our own identity for self-payments
	lclient := lnrpc.NewLightningClient(lndConn)
	info, err := lclient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, err
	}

	// Fetch a simple request from blocc to make sure it's functioning
	var bclient blocc.BloccRPCClient
	if bloccConn != nil {
		bclient = blocc.NewBloccRPCClient(bloccConn)
		_, err = bclient.GetBlock(context.Background(), &blocc.Get{Id: blocc.BlockIdTip})
		if err != nil {
			return nil, err
		}
	}

	// Make sure we are connected to only one chain
	if len(info.Chains) != 1 {
		return nil, fmt.Errorf("Could not determine chain. len=%d", len(info.Chains))
	}

	lndChain := info.Chains[0]

	// If we're not using bitcoin (ie litecoin) print an error
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

		lclient: lclient,
		bclient: bclient,

		chain: chain,
	}, nil

}
