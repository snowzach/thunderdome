package cmd

import (
	cli "github.com/spf13/cobra"
	"go.uber.org/zap"

	"git.coinninja.net/backend/thunderdome/conf"
)

func init() {
	rootCmd.AddCommand(txMonitorCmd)
}

var (
	txMonitorCmd = &cli.Command{
		Use:   "txmonitor",
		Short: "TX Monitor",
		Long:  `TX Monitor`,
		Run: func(cmd *cli.Command, args []string) { // Initialize the databse

			txm, err := NewTXMonitor()
			if err != nil {
				logger.Fatalw("Could not create TXMonitor", "error", err)
			}

			go txm.MonitorBTC()

			<-conf.Stop.Chan() // Wait until StopChan
			conf.Stop.Wait()   // Wait until everyone cleans up
			zap.L().Sync()     // Flush the logger

		},
	}
)
