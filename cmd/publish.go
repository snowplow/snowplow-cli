package cmd

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:     "publish",
	Aliases: []string{"pub"},
	Short:   "Publishing commands for data structures",
	Long: `Publishing commands for data structures

Publish local data structures to BDP console.
`,
}

var devCmd = &cobra.Command{
	Use:   "dev path...",
	Short: "Publish data structures to your development environment",
	Args:  cobra.MinimumNArgs(1),
	Long: `Publish modified data structures to BDP Console and your development environment

The 'meta' section of a data structure is not versioned within BDP Console.
Changes to it will be published by this command.
	`,

	Run: func(cmd *cobra.Command, args []string) {
		apikey, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")

		dataStructuresLocal, err := DataStructuresFromPaths(args)
		slog.Info("publishing to dev from", "paths", args)

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
		err = performChangesDev(cnx, c, changes)
		if err != nil {
			LogFatal(err)
		}
		slog.Info("all done!")
	},
}

var prodCmd = &cobra.Command{
	Use:   "prod path...",
	Short: "Publish data structures to your production environment",
	Args:  cobra.MinimumNArgs(1),
	Long: `Publish data structures from your development to your production environment

Data structures found on <path...> which are deployed to your development
environment will be published to your production environment.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		apikey, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")

		dataStructuresLocal, err := DataStructuresFromPaths(args)
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

		changes, err := getChanges(dataStructuresLocal, remotesListing, "PROD")
		if err != nil {
			LogFatal(err)
		}

		err = printChangeset(changes)
		if err != nil {
			LogFatal(err)
		}
		err = performChangesProd(cnx, c, changes)
		if err != nil {
			LogFatal(err)
		}
		slog.Info("all done!")
	},
}

func init() {
	dataStructuresCmd.AddCommand(publishCmd)
	publishCmd.AddCommand(devCmd)
	publishCmd.AddCommand(prodCmd)
}
