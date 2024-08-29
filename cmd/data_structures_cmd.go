package cmd

import (
	"github.com/spf13/cobra"
)

var dataStructuresCmd = &cobra.Command{
	Use:     "data-structures",
	Aliases: []string{"ds"},
	Short:   "Work with Snowplow data structures",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return InitConsoleConfig(cmd)
	},
}

func init() {
	rootCmd.AddCommand(dataStructuresCmd)
	InitConsoleFlags(dataStructuresCmd)
}
