package cmd

import (
	cli "github.com/spf13/cobra"
	"go.uber.org/zap"

	"git.coinninja.net/backend/thunderdome/conf"
)

func init() {
	rootCmd.AddCommand(monitorCmd)
}

var (
	monitorCmd = &cli.Command{
		Use:   "monitor",
		Short: "Monitor",
		Long:  `Monitor`,
		Run: func(cmd *cli.Command, args []string) { // Initialize the databse

			startMonitor()

			<-conf.Stop.Chan() // Wait until StopChan
			conf.Stop.Wait()   // Wait until everyone cleans up
			_ = zap.L().Sync() // Flush the logger

		},
	}
)

func startMonitor() {
	_, err := NewMonitor()
	if err != nil {
		logger.Fatalw("Could not create Monitor", "error", err)
	}
}
