package cmd

import (
	"runtime"

	"neo/cmd/client/cli"
	"neo/internal/client"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start Neo client",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := client.MustUnmarshalConfig()
		cli := cli.NewRun(cmd, args, cfg)
		ctx := cmd.Context()
		if err := cli.Run(ctx); err != nil {
			logrus.Fatalf("Error: %v", err)
		}
		logrus.Debugf("Run finished")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().IntP("jobs", "j", runtime.NumCPU()*cli.JobsPerCPU, "exploit jobs multiplier")
}
