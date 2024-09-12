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
	Use:   "dev [paths...] default: [./data-structures]",
	Short: "Publish data structures to your development environment",
	Args:  cobra.ArbitraryArgs,
	Long: `Publish modified data structures to BDP Console and your development environment

The 'meta' section of a data structure is not versioned within BDP Console.
Changes to it will be published by this command.
	`,

	Run: func(cmd *cobra.Command, args []string) {
		apiKeyId, _ := cmd.Flags().GetString("api-key-id")
		apiKeySecret, _ := cmd.Flags().GetString("api-key-secret")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		dataStructureFolders := []string{DataStructuresFolder}
		if len(args) > 0 {
			dataStructureFolders = args
		}

		dataStructuresLocal, err := DataStructuresFromPaths(dataStructureFolders)

		if err != nil {
			LogFatal(err)
		}

		errs := ValidateLocalDs(dataStructuresLocal)
		if len(errs) > 0{
			LogFatalMultiple(errs)
		}

		slog.Info("publishing to dev from", "paths", dataStructureFolders)

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
		if !dryRun {
			err = performChangesDev(cnx, c, changes)
			if err != nil {
				LogFatal(err)
			}
			slog.Info("all done!")
		}
	},
}

var prodCmd = &cobra.Command{
	Use:   "prod [paths...] default: [./data-structures]",
	Short: "Publish data structures to your production environment",
	Args:  cobra.ArbitraryArgs,
	Long: `Publish data structures from your development to your production environment

Data structures found on <path...> which are deployed to your development
environment will be published to your production environment.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKeyId, _ := cmd.Flags().GetString("api-key-id")
		apiKeySecret, _ := cmd.Flags().GetString("api-key-secret")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		dataStructureFolders := []string{DataStructuresFolder}
		if len(args) > 0 {
			dataStructureFolders = args
		}

		dataStructuresLocal, err := DataStructuresFromPaths(dataStructureFolders)
		if err != nil {
			LogFatal(err)
		}

		errs := ValidateLocalDs(dataStructuresLocal)
		if len(errs) > 0{
			LogFatalMultiple(errs)
		}

		slog.Info("publishing to prod from", "paths", dataStructureFolders)

		cnx := context.Background()

		c, err := NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
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
		if !dryRun {
			err = performChangesProd(cnx, c, changes)
			if err != nil {
				LogFatal(err)
			}
			slog.Info("all done!")
		}
	},
}

func init() {
	dataStructuresCmd.AddCommand(publishCmd)
	publishCmd.AddCommand(devCmd)
	publishCmd.AddCommand(prodCmd)

	devCmd.PersistentFlags().BoolP("dry-run", "d", false, "Only print planned changes without performing them")
	prodCmd.PersistentFlags().BoolP("dry-run", "d", false, "Only print planned changes without performing them")

}
