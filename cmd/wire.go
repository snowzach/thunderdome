// This file uses wire to build all the depdendancies required

// +build wireinject

package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"git.coinninja.net/backend/blocc/blocc"
	"git.coinninja.net/backend/cnauth"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/google/wire"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	config "github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	macaroon "gopkg.in/macaroon.v2"

	"git.coinninja.net/backend/thunderdome/monitor"
	"git.coinninja.net/backend/thunderdome/server"
	"git.coinninja.net/backend/thunderdome/store"
	"git.coinninja.net/backend/thunderdome/store/postgres"
	"git.coinninja.net/backend/thunderdome/store/redis"
	"git.coinninja.net/backend/thunderdome/tdrpc"
	"git.coinninja.net/backend/thunderdome/tdrpc/adminrpcserver"
	"git.coinninja.net/backend/thunderdome/tdrpc/tdrpcserver"
)

// NewServer will create a new webserver
func NewServer() (*server.Server, error) {
	wire.Build(server.New)
	return &server.Server{}, nil
}

// NewTDRPCServer will create a new grpc/rest server on the webserver
func NewTDRPCServer() (tdrpc.ThunderdomeRPCServer, error) {
	wire.Build(tdrpcserver.NewTDRPCServer, NewStore, NewLightningClient, NewDistCache)
	return nil, nil
}

// NewTDRPCServer will create a new grpc/rest server on the webserver
func NewAdminRPCServer() (tdrpc.AdminRPCServer, error) {
	wire.Build(adminrpcserver.NewAdminRPCServer, NewStore, NewCNAuthClient)
	return nil, nil
}

// NewTXMonitor will create a new BTC and LN transaction monitor
func NewMonitor() (*monitor.Monitor, error) {
	wire.Build(monitor.NewMonitor, NewStore, NewLightningClient, NewBloccClient, NewDogStatsDClient)
	return nil, nil
}

// NewStore is the store for the application
func NewStore() tdrpc.Store {
	var store tdrpc.Store
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

func NewLightningClient() lnrpc.LightningClient {

	conn := NewLndGrpcClientConn()

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

	lclient := lnrpc.NewLightningClient(conn)

	// Continuously monitor the connection to LND, exit if it goes bad
	go func() {
		for {
			_, err := lclient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
			if err != nil {
				logger.Fatalw("LND Connection Invalid.", "error", err)
			}
			logger.Debug("LND Healthcheck OK")
			time.Sleep(config.GetDuration("lnd.health_check_interval"))
		}
	}()

	return lclient
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

func NewCNAuthClient() (*cnauth.Client, error) {

	if config.GetBool("tdome.disable_auth") {
		return nil, nil
	}

	client, err := cnauth.New(context.Background(), config.GetString("tdome.firebase_credentials_file"))
	if err != nil {
		logger.Fatalw("Could not create firebase auth client", "error", err, "credentials_file", config.GetString("tdome.firebase_credentials_file"))
	}

	return client, nil

}

func NewBloccClient() blocc.BloccRPCClient {

	// We only need the blocc client when topup_instant is enabled so we can verify input transactions are confirmed
	if !config.GetBool("tdome.topup_instant_enabled") {
		return nil
	}

	// Dial options for use with the connection
	dialOptions := []grpc.DialOption{
		grpc.WithTimeout(10 * time.Second),
		grpc.WithBlock(),
	}

	if config.GetBool("blocc.tls") {
		if config.GetBool("blocc.tls_insecure") {
			creds := credentials.NewTLS(&tls.Config{
				InsecureSkipVerify: true,
			})
			dialOptions = append(dialOptions, grpc.WithTransportCredentials(creds))
		} else {
			// Create the connection to lightning
			creds, err := credentials.NewClientTLSFromFile(config.GetString("blocc.tls_cert"), config.GetString("blocc.tls_host"))
			if err != nil {
				logger.Fatalw("Could not load credentials", "error", err)
			}
			dialOptions = append(dialOptions, grpc.WithTransportCredentials(creds))
		}
	} else {
		dialOptions = append(dialOptions, grpc.WithInsecure())
	}

	// Create the connection
	conn, err := grpc.Dial(net.JoinHostPort(config.GetString("blocc.host"), config.GetString("blocc.port")), dialOptions...)
	if err != nil {
		logger.Fatalw("Could not connect to blocc", "error", err, "host:port", net.JoinHostPort(config.GetString("blocc.host"), config.GetString("blocc.port")))
	}

	return blocc.NewBloccRPCClient(conn)

}

func NewDistCache() store.DistCache {

	r, err := redis.New(config.GetStringSlice("redis.prefixes")...)
	if err != nil {
		logger.Fatalw("Could not connect to redis", "error", err)
	}
	return r

}

// NewDogStatsDClient creates a new statsd client
func NewDogStatsDClient() *statsd.Client {

	if !config.GetBool("dogstatsd.enabled") {
		return nil
	}

	client, err := statsd.New(net.JoinHostPort(config.GetString("dogstatsd.host"), config.GetString("dogstatsd.port")))
	if err != nil {
		logger.Fatalw("Could not connect to datadog", "error", err, "host:port", net.JoinHostPort(config.GetString("dogstatsd.host"), config.GetString("dogstatsd.port")))
	}

	client.Namespace = config.GetString("dogstatsd.namespace")
	client.Tags = append(client.Tags, config.GetStringSlice("dogstatsd.tags")...)

	client.Event(&statsd.Event{
		Title:     "Thunderdome Started",
		Text:      fmt.Sprintf(`Thunderdome Started`),
		Priority:  statsd.Normal,
		AlertType: statsd.Info,
	})

	return client
}
