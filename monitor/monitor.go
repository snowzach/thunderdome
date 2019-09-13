package monitor

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/lightningnetwork/lnd/lnrpc"
	"go.uber.org/zap"

	"git.coinninja.net/backend/blocc/blocc"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

type Monitor struct {
	logger *zap.SugaredLogger
	store  tdrpc.Store

	lclient lnrpc.LightningClient
	bclient blocc.BloccRPCClient

	ddclient *statsd.Client

	chain *chaincfg.Params
}

func NewMonitor(store tdrpc.Store, lclient lnrpc.LightningClient, bclient blocc.BloccRPCClient, ddclient *statsd.Client) (*Monitor, error) {

	logger := zap.S().With("package", "txmonitor")

	// Fetch the node info to make sure we know our own identity for self-payments
	info, err := lclient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
	if err != nil {
		return nil, err
	}

	// Fetch a simple request from blocc to make sure it's functioning
	if bclient != nil {
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

	logger.Infof("Monitor auto-configured for chain %s", lndChain.Network)

	// Return the server
	m := &Monitor{
		logger: logger,
		store:  store,

		lclient: lclient,
		bclient: bclient,

		ddclient: ddclient,

		chain: chain,
	}

	go m.MonitorBTC()
	go m.MonitorLN()
	go m.MonitorExpired()
	go m.MonitorLND()

	return m, nil

}
