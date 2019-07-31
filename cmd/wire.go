// This file uses wire to build all the depdendancies required

// +build wireinject

package cmd

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/google/wire"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	config "github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
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
	wire.Build(txmonitor.NewTXMonitor, NewStore, NewLndGrpcClientConn, NewChainParams)
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

func NewChainParams() *chaincfg.Params {

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

	return chain

}

func NewLightningClient(conn *grpc.ClientConn) lnrpc.LightningClient {

	_, err := lnrpc.NewWalletUnlockerClient(conn).UnlockWallet(context.Background(), &lnrpc.UnlockWalletRequest{
		WalletPassword: []byte(config.GetString("lnd.unlock_password")),
	})
	if err == nil {
		logger.Info("Wallet Unlocked")
		// Disconnect and reconnect
		conn.Close()
		conn = NewLndGrpcClientConn()
	} else if status.Code(err) == codes.Unimplemented {
		// Wallet is alreay unlocked
	} else {
		logger.Fatalf("Could not UnlockWallet: %v", err)
	}

	return lnrpc.NewLightningClient(conn)
}

// NewLndGrpcClientConn creates a new GRPC connection to LND
func NewLndGrpcClientConn() *grpc.ClientConn {

	// Dial options for use with the connection
	dialOptions := []grpc.DialOption{
		grpc.WithTimeout(10 * time.Second),
		grpc.WithBlock(),
	}

	if config.GetBool("lnd.tls_insecure") {
		creds := credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(creds))
	} else {
		// Create the connection to lightning
		creds, err := credentials.NewClientTLSFromFile(config.GetString("lnd.tls_cert"), config.GetString("lnd.tls_host"))
		if err != nil {
			logger.Fatalw("Could not load credentials", "error", err)
		}
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(creds))
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
		logger.Fatalw("Could not connect to lnd", "error", err, "host:port", net.JoinHostPort(config.GetString("lnd.host"), config.GetString("lnd.port")))
	}

	return conn

}
