package cmd

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate path...",
	Short: "Validate data structures with BDP Console",
	Args: cobra.MinimumNArgs(1),
	Long: `Sends all data structures from <path> for validation by BDP Console.`,
	Run: func(cmd *cobra.Command, args []string) {
		apikey, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")

		dataStructuresLocal, err := DataStructuresFromPaths(args)
		slog.Info("validating from", "paths", args)
		if err != nil {
			LogFatal(err)
		}

		cnx := context.Background()

		c, err := NewApiClient(cnx, host, apikey, org)
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
