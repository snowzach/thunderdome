// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package cmd

import (
	"git.coinninja.net/backend/thunderdome/rpcserver"
	"git.coinninja.net/backend/thunderdome/server"
	"git.coinninja.net/backend/thunderdome/store/postgres"
	"github.com/lightningnetwork/lnd/macaroons"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/macaroon.v2"
	"io/ioutil"
	"net"
	"time"
)

import (
	_ "net/http/pprof"
)

// Injectors from wire.go:

func NewServer() (*server.Server, error) {
	serverServer, err := server.New()
	if err != nil {
		return nil, err
	}
	return serverServer, nil
}

func NewRPCServer() (*rpcserver.RPCServer, error) {
	rpcStore := NewRPCStore()
	clientConn := NewLndGrpcClientConn()
	rpcServer, err := rpcserver.NewRPCServer(rpcStore, clientConn)
	if err != nil {
		return nil, err
	}
	return rpcServer, nil
}

// wire.go:

// NewRPCStore is the store for the RPCServer
func NewRPCStore() rpcserver.RPCStore {
	var rpcStore rpcserver.RPCStore
	var err error
	switch viper.GetString("storage.type") {
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

	creds, err := credentials.NewClientTLSFromFile(viper.GetString("lnd.tls_cert"), viper.GetString("lnd.host"))
	if err != nil {
		logger.Fatalw("Could not load credentials", "error", err)
	}

	dialOptions := []grpc.DialOption{grpc.WithTransportCredentials(creds), grpc.WithTimeout(10 * time.Second), grpc.WithBlock()}

	macBytes, err := ioutil.ReadFile(viper.GetString("lnd.macaroon"))
	if err != nil {
		logger.Fatalw("Unable to read macaroon", "error", err)
	}

	mac := &macaroon.Macaroon{}
	if err = mac.UnmarshalBinary(macBytes); err != nil {
		logger.Fatalw("Unable to decode macaroon", "error", err)
	}

	cred := macaroons.NewMacaroonCredential(mac)
	dialOptions = append(dialOptions, grpc.WithPerRPCCredentials(cred))

	conn, err := grpc.Dial(net.JoinHostPort(viper.GetString("lnd.host"), viper.GetString("lnd.port")), dialOptions...)
	if err != nil {
		logger.Fatalw("Could not connect to lnd", "error", err)
	}

	return conn

}
