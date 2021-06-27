package cmd

import (
	"time"

	"neo/cmd/client/cli"
	"neo/internal/client"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an exploit",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := client.MustUnmarshalConfig()
		cli := cli.NewAdd(cmd, args, cfg)
		ctx := cmd.Context()
		if err := cli.Run(ctx); err != nil {
			logrus.Fatalf("Error adding exploit: %v", err)
		}
		logrus.Debugf("Add finished")
	},
}

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.PersistentFlags().String("id", "", "exploit name")
	addCmd.PersistentFlags().BoolP("dir", "d", false, "add exploit as a directory")
	addCmd.PersistentFlags().DurationP("interval", "i", time.Second*15, "run interval")
	addCmd.PersistentFlags().DurationP("timeout", "t", time.Second*15, "timeout for a single run")
	addCmd.PersistentFlags().BoolP("endless", "e", false, "mark script as endless")
}
