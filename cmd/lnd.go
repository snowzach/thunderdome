package cmd

import (
	"context"
	"io/ioutil"
	"net"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	cli "github.com/spf13/cobra"
	config "github.com/spf13/viper"
	macaroon "gopkg.in/macaroon.v2"
	// "go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func init() {
	rootCmd.AddCommand(verifyCmd)
}

var (
	verifyCmd = &cli.Command{
		Use:   "lnd",
		Short: "LND Test",
		Long:  `LND Test`,
		Run: func(cmd *cli.Command, args []string) { // Initialize the databse

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

			a, err := lclient.GetNetworkInfo(context.Background(), &lnrpc.NetworkInfoRequest{})
			if err != nil {
				logger.Fatalw("Could not GetNetworkInfo", "error", err)
			}
			spew.Dump(a)

			b, err := lclient.WalletBalance(context.Background(), &lnrpc.WalletBalanceRequest{})
			if err != nil {
				logger.Fatalw("Could not WalletBalance", "error", err)
			}
			spew.Dump(b)

			// d, err := client.EstimateFee(context.Background(), &lnrpc.EstimateFeeRequest{
			//      AddrToAmount: map[string]int64{
			//              "3PEdU1mVicejXLGgi2dyfr9H55z4ndXFba": 1000000,
			//      },
			// })
			// if err != nil {
			//      logger.Fatalw("Could not EstimateFee", "error", err)
			// }
			// spew.Dump(d)

			// e, err := lclient.NewAddress(context.Background(), &lnrpc.NewAddressRequest{
			// 	Type: lnrpc.AddressType_NESTED_PUBKEY_HASH,
			// })
			// if err != nil {
			// 	logger.Fatalw("Could not EstimateFee", "error", err)
			// }
			// spew.Dump(e)

			f, err := lclient.GetTransactions(context.Background(), &lnrpc.GetTransactionsRequest{})
			if err != nil {
				logger.Fatalw("Could not GetTransactions", "error", err)
			}
			spew.Dump(f)

		},
	}
)
