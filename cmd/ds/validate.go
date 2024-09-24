package ds

import (
	"context"
	"errors"
	"log/slog"

	changesPkg "github.com/snowplow-product/snowplow-cli/internal/changes"
	"github.com/snowplow-product/snowplow-cli/internal/console"
	. "github.com/snowplow-product/snowplow-cli/internal/logging"
	"github.com/snowplow-product/snowplow-cli/internal/util"
	"github.com/snowplow-product/snowplow-cli/internal/validation"
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
		ghOut, _ := cmd.Flags().GetBool("gh-annotate")

		dataStructureFolders := []string{util.DataStructuresFolder}
		if len(args) > 0 {
			dataStructureFolders = args
		}

		dataStructuresLocal, err := util.DataStructuresFromPaths(dataStructureFolders)
		slog.Info("validating from", "paths", dataStructureFolders)
		if err != nil {
			LogFatal(err)
		}

		errs := validation.ValidateLocalDs(dataStructuresLocal)
		if len(errs) > 0 {
			LogFatalMultiple(errs)
		}

		cnx := context.Background()

		c, err := console.NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
		if err != nil {
			LogFatal(err)
		}

		remotesListing, err := console.GetDataStructureListing(cnx, c)
		if err != nil {
			LogFatal(err)
		}

		changes, err := changesPkg.GetChanges(dataStructuresLocal, remotesListing, "DEV")
		if err != nil {
			LogFatal(err)
		}

		err = changesPkg.PrintChangeset(changes)
		if err != nil {
			LogFatal(err)
		}

		vr, err := validation.ValidateChanges(cnx, c, changes)
		if err != nil {
			LogFatal(err)
		}

		vr.Slog()

		if ghOut {
			vr.GithubAnnotate()
		}

		if !vr.Valid {
			LogFatal(errors.New(vr.Message))
		}
	},
}

func init() {
	DataStructuresCmd.AddCommand(validateCmd)

	validateCmd.PersistentFlags().Bool("gh-annotate", false, "Output suitable for github workflow annotation (ignores -s)")
}
