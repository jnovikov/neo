package cmd

import (
	"neo/cmd/client/cli"
	"neo/internal/client"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// singleRunCmd represents the single command
var singleRunCmd = &cobra.Command{
	Use:   "single",
	Short: "Run an exploit once on all teams immediately",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := client.MustUnmarshalConfig()
		cli := cli.NewSingleRun(cmd, args, cfg)
		ctx := cmd.Context()
		if err := cli.Run(ctx); err != nil {
			logrus.Fatalf("Error in single run: %v", err)
		}
		logrus.Debugf("Single run finished")
	},
}

func init() {
	rootCmd.AddCommand(singleRunCmd)
}
