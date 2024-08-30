package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "dps-cli",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&cfgFile, "config", "",
		`config file (defaults to $HOME/.snowplow.{yaml|json|toml}
then $XDG_CONFIG_HOME/snowplow/.snowplow.{yaml|json|toml})
then $HOME/.config/snowplow/.snowplow.{yaml|json|toml})`,
	)
	rootCmd.PersistentFlags().Bool("debug", false, "Log output level to Debug")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Log output level to Warn")
	rootCmd.PersistentFlags().BoolP("silent", "s", false, "Disable output")
}
