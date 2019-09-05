package cmd

import (
	cli "github.com/spf13/cobra"
	"go.uber.org/zap"

	"git.coinninja.net/backend/thunderdome/conf"
	"git.coinninja.net/backend/thunderdome/tdrpc"
)

func init() {
	rootCmd.AddCommand(apiCmd)

	apiCmd.PersistentFlags().BoolVarP(&apiCmdTxMonitor, "txmonitor", "t", false, "Start the TXMonitr also")
}

var (
	apiCmdTxMonitor bool

	apiCmd = &cli.Command{
		Use:   "api",
		Short: "Start API",
		Long:  `Start API`,
		Run: func(cmd *cli.Command, args []string) { // Initialize the databse

			tdrpcServer, err := NewTDRPCServer()
			if err != nil {
				logger.Fatalw("Could not create tdrpcserver",
					"error", err,
				)
			}

			adminServer, err := NewAdminRPCServer()
			if err != nil {
				logger.Fatalw("Could not create adminrpcserver",
					"error", err,
				)
			}

			server, err := NewServer()
			if err != nil {
				logger.Fatalw("Could not create server",
					"error", err,
				)
			}

			// Register the RPC server and it's GRPC Gateway for when it starts
			tdrpc.RegisterThunderdomeRPCServer(server.GRPCServer(), tdrpcServer)
			server.GWReg(tdrpc.RegisterThunderdomeRPCHandlerFromEndpoint)

			// Register the admin server
			tdrpc.RegisterAdminRPCServer(server.GRPCServer(), adminServer)
			server.GWReg(tdrpc.RegisterAdminRPCHandlerFromEndpoint)

			// Start it up
			err = server.ListenAndServe()
			if err != nil {
				logger.Fatalw("Could not start server",
					"error", err,
				)
			}

			if apiCmdTxMonitor {
				startTxMonitor()
			}

			<-conf.Stop.Chan() // Wait until StopChan
			conf.Stop.Wait()   // Wait until everyone cleans up
			_ = zap.L().Sync() // Flush the logger

		},
	}
)
