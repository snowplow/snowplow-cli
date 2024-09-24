package ds

import (
	"log/slog"
	"os"

	"github.com/snowplow-product/snowplow-cli/internal/config"
	. "github.com/snowplow-product/snowplow-cli/internal/logging"
	"github.com/spf13/cobra"
)

var DataStructuresCmd = &cobra.Command{
	Use:     "data-structures",
	Aliases: []string{"ds"},
	Short:   "Work with Snowplow data structures",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := InitLogging(cmd); err != nil {
			return err
		}

		if err := config.InitConsoleConfig(cmd); err != nil {
			slog.Error("config failure", "error", err)
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	config.InitConsoleFlags(DataStructuresCmd)
}
