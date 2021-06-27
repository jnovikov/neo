package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "client",
	Short: "Neo client",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringP("config", "c", "client_config.yml", "config file")
	rootCmd.PersistentFlags().BoolP("verbose", "v", true, "enable debug logging")
	rootCmd.PersistentFlags().String("host", "127.0.0.1", "server host")

	mustBindPersistent(rootCmd, "config")
	mustBindPersistent(rootCmd, "host")
	mustBindPersistent(rootCmd, "verbose")
}

func mustBindPersistent(c *cobra.Command, flag string) {
	if err := viper.BindPFlag(flag, c.PersistentFlags().Lookup(flag)); err != nil {
		logrus.Fatalf("Error binding flag %s: %v", flag, err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigFile(viper.GetString("config"))
	viper.SetConfigType("yaml")

	viper.SetEnvPrefix("NEO")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		logrus.Info("Using config file:", viper.ConfigFileUsed())
	}

	if viper.GetBool("verbose") {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	logrus.Debugf("Got configuration: %+v", viper.AllSettings())
}
