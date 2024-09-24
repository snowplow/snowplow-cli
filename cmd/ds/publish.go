package ds

import (
	"context"
	"errors"
	"log/slog"

	changesPkg "github.com/snowplow-product/snowplow-cli/internal/changes"
	"github.com/snowplow-product/snowplow-cli/internal/console"
	"github.com/snowplow-product/snowplow-cli/internal/io"
	"github.com/snowplow-product/snowplow-cli/internal/util"
	"github.com/snowplow-product/snowplow-cli/internal/validation"
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
		ghOut, _ := cmd.Flags().GetBool("gh-annotate")
		managedFrom, _ := cmd.Flags().GetString("managed-from")

		dataStructureFolders := []string{util.DataStructuresFolder}
		if len(args) > 0 {
			dataStructureFolders = args
		}

		dataStructuresLocal, err := util.DataStructuresFromPaths(dataStructureFolders)

		if err != nil {
			io.LogFatal(err)
		}

		errs := validation.ValidateLocalDs(dataStructuresLocal)
		if len(errs) > 0 {
			io.LogFatalMultiple(errs)
		}

		slog.Info("publishing to dev from", "paths", dataStructureFolders)

		cnx := context.Background()

		c, err := console.NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
		if err != nil {
			io.LogFatal(err)
		}

		remotesListing, err := console.GetDataStructureListing(cnx, c)
		if err != nil {
			io.LogFatal(err)
		}

		changes, err := changesPkg.GetChanges(dataStructuresLocal, remotesListing, "DEV")
		if err != nil {
			io.LogFatal(err)
		}

		err = changesPkg.PrintChangeset(changes)
		if err != nil {
			io.LogFatal(err)
		}

		vr, err := validation.ValidateChanges(cnx, c, changes)
		if err != nil {
			io.LogFatal(err)
		}

		vr.Slog()

		if ghOut {
			vr.GithubAnnotate()
		}

		if !vr.Valid {
			io.LogFatal(errors.New(vr.Message))
		}

		if !dryRun {
			err = changesPkg.PerformChangesDev(cnx, c, changes, managedFrom)
			if err != nil {
				io.LogFatal(err)
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
		managedFrom, _ := cmd.Flags().GetString("managed-from")

		dataStructureFolders := []string{util.DataStructuresFolder}
		if len(args) > 0 {
			dataStructureFolders = args
		}

		dataStructuresLocal, err := util.DataStructuresFromPaths(dataStructureFolders)
		if err != nil {
			io.LogFatal(err)
		}

		errs := validation.ValidateLocalDs(dataStructuresLocal)
		if len(errs) > 0 {
			io.LogFatalMultiple(errs)
		}

		slog.Info("publishing to prod from", "paths", dataStructureFolders)

		cnx := context.Background()

		c, err := console.NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
		if err != nil {
			io.LogFatal(err)
		}

		remotesListing, err := console.GetDataStructureListing(cnx, c)
		if err != nil {
			io.LogFatal(err)
		}

		changes, err := changesPkg.GetChanges(dataStructuresLocal, remotesListing, "PROD")
		if err != nil {
			io.LogFatal(err)
		}

		err = changesPkg.PrintChangeset(changes)
		if err != nil {
			io.LogFatal(err)
		}
		if !dryRun {
			err = changesPkg.PerformChangesProd(cnx, c, changes, managedFrom)
			if err != nil {
				io.LogFatal(err)
			}
			slog.Info("all done!")
		}
	},
}

func init() {
	DataStructuresCmd.AddCommand(publishCmd)
	publishCmd.AddCommand(devCmd)
	publishCmd.AddCommand(prodCmd)

	devCmd.PersistentFlags().BoolP("dry-run", "d", false, "Only print planned changes without performing them")
	prodCmd.PersistentFlags().BoolP("dry-run", "d", false, "Only print planned changes without performing them")

	devCmd.PersistentFlags().Bool("gh-annotate", false, "Output suitable for github workflow annotation (ignores -s)")
}
