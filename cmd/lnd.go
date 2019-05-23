package cmd

import (
	"context"

	"github.com/davecgh/go-spew/spew"
	"github.com/lightningnetwork/lnd/lnrpc"
	cli "github.com/spf13/cobra"
	// "go.uber.org/zap"
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

			// Create GRPC client and add LightningClient to it
			conn := NewLndGrpcClientConn()
			lclient := lnrpc.NewLightningClient(conn)

			// Do a quick ping without health check to ensure the plugin is alive and functional
			a, err := lclient.GetNetworkInfo(context.Background(), &lnrpc.NetworkInfoRequest{})
			if err != nil {
				logger.Fatalw("Could not GetNetworkInfo", "error", err)
			}
			spew.Dump(a)

			// Example wallet balance
			b, err := lclient.WalletBalance(context.Background(), &lnrpc.WalletBalanceRequest{})
			if err != nil {
				logger.Fatalw("Could not WalletBalance", "error", err)
			}
			spew.Dump(b)

			// d, err := lclient.EstimateFee(context.Background(), &lnrpc.EstimateFeeRequest{
			// 	AddrToAmount: map[string]int64{
			// 		"3PEdU1mVicejXLGgi2dyfr9H55z4ndXFba": 12586,
			// 	},
			// })
			// if err != nil {
			// 	logger.Fatalw("Could not EstimateFee", "error", err)
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

			// Do a quick ping without health check to ensure the plugin is alive and functional
			g, err := lclient.GetInfo(context.Background(), &lnrpc.GetInfoRequest{})
			if err != nil {
				logger.Fatalw("Could not GetNetworkInfo", "error", err)
			}
			spew.Dump(g)

		},
	}
)
