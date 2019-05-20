package cmd

import (
	"context"
	"io/ioutil"
	"net"
	"time"

	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	cli "github.com/spf13/cobra"
	config "github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	macaroon "gopkg.in/macaroon.v2"

	"git.coinninja.net/backend/thunderdome/conf"
	"git.coinninja.net/backend/thunderdome/server"
	"git.coinninja.net/backend/thunderdome/store/postgres"
	"git.coinninja.net/backend/thunderdome/tdrpc"
	"git.coinninja.net/backend/thunderdome/thunderdome"
)

func init() {
	rootCmd.AddCommand(apiCmd)
}

var (
	apiCmd = &cli.Command{
		Use:   "api",
		Short: "Start API",
		Long:  `Start API`,
		Run: func(cmd *cli.Command, args []string) { // Initialize the databse

			// Create the database connection
			var store thunderdome.Store
			var err error
			switch config.GetString("storage.type") {
			case "postgres":
				store, err = postgres.New()
			}
			if err != nil {
				logger.Fatalw("Database Error", "error", err)
			}

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
			defer conn.Close()

			// Do a quick ping without health check to ensure the plugin is alive and functional
			lclient := lnrpc.NewLightningClient(conn)

			_, err = lclient.GetNetworkInfo(context.Background(), &lnrpc.NetworkInfoRequest{})
			if err != nil {
				logger.Fatalw("Could not GetNetworkInfo", "error", err)
			}

			// Create the thunderdome server
			tserver, err := thunderdome.NewServer(store, lclient)
			if err != nil {
				logger.Fatalw("Could not create thunderdome server",
					"error", err,
				)
			}

			// Create the grpc server
			wserver, err := server.New()
			if err != nil {
				logger.Fatalw("Could not create server",
					"error", err,
				)
			}

			// Register the RPC server and it's GRPC Gateway for when it starts
			tdrpc.RegisterTdomeRPCServer(wserver.GRPCServer(), tserver)
			wserver.GWReg(tdrpc.RegisterTdomeRPCHandlerFromEndpoint)

			// Start it up
			err = wserver.ListenAndServe()
			if err != nil {
				logger.Fatalw("Could not start server",
					"error", err,
				)
			}

			<-conf.Stop.Chan() // Wait until StopChan
			conf.Stop.Wait()   // Wait until everyone cleans up
			zap.L().Sync()     // Flush the logger

		},
	}
)
