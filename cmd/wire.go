// This file uses wire to build all the depdendancies required

// +build wireinject

package cmd

import (
	"io/ioutil"
	"net"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/google/wire"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	config "github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	macaroon "gopkg.in/macaroon.v2"

	"git.coinninja.net/backend/thunderdome/rpcserver"
	"git.coinninja.net/backend/thunderdome/server"
	"git.coinninja.net/backend/thunderdome/store/postgres"
	"git.coinninja.net/backend/thunderdome/thunderdome"
	"git.coinninja.net/backend/thunderdome/txmonitor"
)

// NewServer will create a new webserver
func NewServer() (*server.Server, error) {
	wire.Build(server.New)
	return &server.Server{}, nil
}

// NewRPCServer will create a new grpc/rest server on the webserver
func NewRPCServer() (*rpcserver.RPCServer, error) {
	wire.Build(rpcserver.NewRPCServer, NewStore, NewLndGrpcClientConn, NewLightningClient)
	return &rpcserver.RPCServer{}, nil
}

// NewTXMonitor will create a new BTC and LN transaction monitor
func NewTXMonitor() (*txmonitor.TXMonitor, error) {
	wire.Build(txmonitor.NewTXMonitor, NewStore, NewLndGrpcClientConn, NewChainParams, NewBTCRPCClient)
	return &txmonitor.TXMonitor{}, nil
}

// NewStore is the store for the application
func NewStore() thunderdome.Store {
	var store thunderdome.Store
	var err error
	switch config.GetString("storage.type") {
	case "postgres":
		store, err = postgres.New()
	}
	if err != nil {
		logger.Fatalw("Database Error", "error", err)
	}
	return store
}

func NewBTCRPCClient() *rpcclient.Client {

	// create new client instance
	rpcc, err := rpcclient.New(&rpcclient.ConnConfig{
		HTTPPostMode: config.GetBool("btc.post_mode"),
		DisableTLS:   config.GetBool("btc.disable_tls"),
		Host:         net.JoinHostPort(config.GetString("btc.host"), config.GetString("btc.port")),
		User:         config.GetString("btc.username"),
		Pass:         config.GetString("btc.password"),
	}, nil)
	if err != nil {
		logger.Fatalf("error creating new btc client: %v", err)
	}

	// Run a GetInfo request to validate connectivity
	_, err = rpcc.GetBlockChainInfo()
	if err != nil {
		logger.Fatalf("Could not GetBlockChainInfo from BTC RPC Server %v", err)
	}

	return rpcc

}

func NewChainParams(rpcc *rpcclient.Client) *chaincfg.Params {

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
		if config.GetString("btc.chain") == cp.Name {
			chain = cp
			break
		}
	}
	if chain == nil {
		logger.Fatalf("Could not find chain %s", config.GetString("btc.chain"))
	}

	// Run a GetBlockChainInfo request to validate connectivity
	blockChainInfo, err := rpcc.GetBlockChainInfo()
	if err != nil {
		logger.Fatalf("Could not GetBlockChainInfo from BTC RPC Server %v", err)
	}

	// Make sure the chain our rpc is on is the same chain we are working with
	if blockChainInfo.Chain != chain.Name {
		logger.Fatalf("Chain mismatch rpc:%s config:%s", blockChainInfo.Chain, chain.Name)
	}

	return chain

}

func NewLightningClient(conn *grpc.ClientConn) lnrpc.LightningClient {
	return lnrpc.NewLightningClient(conn)
}

// NewLndGrpcClientConn creates a new GRPC connection to LND
func NewLndGrpcClientConn() *grpc.ClientConn {

	// Create the connection to lightning
	creds, err := credentials.NewClientTLSFromFile(config.GetString("lnd.tls_cert"), config.GetString("lnd.host"))
	if err != nil {
		logger.Fatalw("Could not load credentials", "error", err)
	}

	// Dial options for use with the connection
	dialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithTimeout(10 * time.Second),
		grpc.WithBlock(),
	}

	macBytes, err := ioutil.ReadFile(config.GetString("lnd.macaroon"))
	if err != nil {
		logger.Fatalw("Unable to read macaroon", "error", err)
	}

	mac := &macaroon.Macaroon{}
	if err = mac.UnmarshalBinary(macBytes); err != nil {
		logger.Fatalw("Unable to decode macaroon", "error", err)
	}

	// Now we append the macaroon credentials to the dial options.
	cred := macaroons.NewMacaroonCredential(mac)
	dialOptions = append(dialOptions, grpc.WithPerRPCCredentials(cred))

	// Create the connection
	conn, err := grpc.Dial(net.JoinHostPort(config.GetString("lnd.host"), config.GetString("lnd.port")), dialOptions...)
	if err != nil {
		logger.Fatalw("Could not connect to lnd", "error", err)
	}

	return conn

}
