package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"neo/cmd/client/cli"
	"neo/internal/client"
)

// broadcastCmd represents the broadcast command
var broadcastCmd = &cobra.Command{
	Use:   "broadcast",
	Short: "Run a command on all connected clients",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := client.MustUnmarshalConfig()
		cli := cli.NewBroadcast(cmd, args, cfg)
		ctx := cmd.Context()
		if err := cli.Run(ctx); err != nil {
			logrus.Fatalf("Error broadcasting command: %v", err)
		}
		logrus.Debugf("Broadcast finished")
	},
}

func init() {
	rootCmd.AddCommand(broadcastCmd)
	broadcastCmd.Flags().StringP("command", "r", "", "command to run")
}
