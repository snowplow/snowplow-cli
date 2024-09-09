package cmd

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [paths...] default: [./data-structures]",
	Short: "Validate data structures with BDP Console",
	Args:  cobra.ArbitraryArgs,
	Long:  `Sends all data structures from <path> for validation by BDP Console.`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKeyId, _ := cmd.Flags().GetString("api-key-id")
		apiKeySecret, _ := cmd.Flags().GetString("api-key-secret")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")

		dataStructureFolders := []string{DataStructuresFolder}
		if len(args) > 0 {
			dataStructureFolders = args
		}

		dataStructuresLocal, err := DataStructuresFromPaths(dataStructureFolders)
		slog.Info("validating from", "paths", dataStructureFolders)
		if err != nil {
			LogFatal(err)
		}

		cnx := context.Background()

		c, err := NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
		if err != nil {
			LogFatal(err)
		}

		remotesListing, err := GetDataStructureListing(cnx, c)
		if err != nil {
			LogFatal(err)
		}

		changes, err := getChanges(dataStructuresLocal, remotesListing, "DEV")
		if err != nil {
			LogFatal(err)
		}

		err = printChangeset(changes)
		if err != nil {
			LogFatal(err)
		}

		err = validate(cnx, c, changes)
		if err != nil {
			LogFatal(err)
		}
	},
}

func init() {
	dataStructuresCmd.AddCommand(validateCmd)
}
