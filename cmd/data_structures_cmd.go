package cmd

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var dataStructuresCmd = &cobra.Command{
	Use:     "data-structures",
	Aliases: []string{"ds"},
	Short:   "Work with Snowplow data structures",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := InitLogging(cmd); err != nil {
			return err
		}

		if err := InitConsoleConfig(cmd); err != nil {
			slog.Error("config failure", "error", err)
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dataStructuresCmd)
	InitConsoleFlags(dataStructuresCmd)
}
