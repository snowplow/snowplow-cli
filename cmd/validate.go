package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
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
		if err != nil {
			log.Fatal(err)
		}

		cnx := context.Background()

		c, err := NewApiClient(cnx, host, apikey, org)
		if err != nil {
			log.Fatal(err)
		}

		remotesListing, err := GetDataStructureListing(cnx, c)
		if err != nil {
			log.Fatal(err)
		}

		changes, err := getChanges(maps.Values(dataStructuresLocal), remotesListing, "DEV")
		if err != nil {
			log.Fatal(err)
		}

		err = printChangeset(changes)
		if err != nil {
			log.Fatal(err)
		}

		err = validate(cnx, c, changes)
		if err != nil {
			log.Fatal(err)
		}

	},
}

func init() {
	dataStructuresCmd.AddCommand(validateCmd)
}
