package cmd

import (
	"fmt"

	cli "github.com/spf13/cobra"

	"git.coinninja.net/backend/thunderdome/conf"
)

// Version command
func init() {
	rootCmd.AddCommand(&cli.Command{
		Use:   "version",
		Short: "Show version",
		Long:  `Show version`,
		Run: func(cmd *cli.Command, args []string) {
			fmt.Println(conf.Executable + " - " + conf.GitVersion)
		},
	})
}
