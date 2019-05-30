package cmd

import (
	cli "github.com/spf13/cobra"
	"go.uber.org/zap"

	"git.coinninja.net/backend/thunderdome/conf"
	"git.coinninja.net/backend/thunderdome/tdrpc"
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

			rpcServer, err := NewRPCServer()
			if err != nil {
				logger.Fatalw("Could not create rpcserver",
					"error", err,
				)
			}

			// go rpcServer.BTCMonitor()
			// go rpcServer.LightningMonitor()

			server, err := NewServer()
			if err != nil {
				logger.Fatalw("Could not create server",
					"error", err,
				)
			}

			// Register the RPC server and it's GRPC Gateway for when it starts
			tdrpc.RegisterThunderdomeRPCServer(server.GRPCServer(), rpcServer)
			server.GWReg(tdrpc.RegisterThunderdomeRPCHandlerFromEndpoint)

			// Start it up
			err = server.ListenAndServe()
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
