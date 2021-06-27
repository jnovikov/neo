package cmd

import (
	"neo/cmd/client/cli"
	"neo/internal/client"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// enableCmd represents the enable command
var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable a disabled exploit by id",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := client.MustUnmarshalConfig()
		cli := cli.NewSetDisabled(cmd, args, cfg, false)
		ctx := cmd.Context()
		if err := cli.Run(ctx); err != nil {
			logrus.Fatalf("Error enabling exploit: %v", err)
		}
		logrus.Debugf("Enable finished")
	},
}

func init() {
	rootCmd.AddCommand(enableCmd)
}
