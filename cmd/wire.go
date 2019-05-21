// This file uses wire to build all the depdendancies required

// +build wireinject

package cmd

import (
	"io/ioutil"
	"net"
	"time"

	"github.com/google/wire"
	"github.com/lightningnetwork/lnd/macaroons"
	config "github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	macaroon "gopkg.in/macaroon.v2"

	"git.coinninja.net/backend/thunderdome/rpcserver"
	"git.coinninja.net/backend/thunderdome/server"
	"git.coinninja.net/backend/thunderdome/store/postgres"
)

// NewServer will create a new webserver
func NewServer() (*server.Server, error) {
	wire.Build(server.New)
	return &server.Server{}, nil
}

// NewRPCServer will create a new grpc/rest server on the webserver
func NewRPCServer() (*rpcserver.RPCServer, error) {
	wire.Build(rpcserver.NewRPCServer, NewRPCStore, NewLndGrpcClientConn)
	return &rpcserver.RPCServer{}, nil
}

// NewRPCStore is the store for the RPCServer
func NewRPCStore() rpcserver.RPCStore {
	var rpcStore rpcserver.RPCStore
	var err error
	switch config.GetString("storage.type") {
	case "postgres":
		rpcStore, err = postgres.New()
	}
	if err != nil {
		logger.Fatalw("Database Error", "error", err)
	}
	return rpcStore
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
