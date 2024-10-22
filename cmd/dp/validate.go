/**
 * Copyright (c) 2013-present Snowplow Analytics Ltd.
 * All rights reserved.
 * This software is made available by Snowplow Analytics, Ltd.,
 * under the terms of the Snowplow Limited Use License Agreement, Version 1.0
 * located at https://docs.snowplow.io/limited-use-license-1.0
 * BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
 * OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
 */

package dp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/snowplow-product/snowplow-cli/internal/console"
	snplog "github.com/snowplow-product/snowplow-cli/internal/logging"
	"github.com/snowplow-product/snowplow-cli/internal/util"
	"github.com/snowplow-product/snowplow-cli/internal/validation"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [paths...]",
	Short: "Validate data structures with BDP Console",
	Args:  cobra.ArbitraryArgs,
	Long:  `Sends all data products and source applications from <path> for validation by BDP Console.`,
	Example: `  $ snowplow-cli dp validate ./data-products ./source-applications
  $ snowplow-cli dp validate ./src`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKeyId, _ := cmd.Flags().GetString("api-key-id")
		apiKeySecret, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")
		ghOut, _ := cmd.Flags().GetBool("gh-annotate")

		searchPaths := []string{}

		if len(args) == 0 {
			searchPaths = append(searchPaths, "data-products")
			slog.Debug("validation", "msg", "no path provided, using default (./data-products)")
		}

		searchPaths = append(searchPaths, args...)

		files, err := util.MaybeResourcesfromPaths(searchPaths)
		if err != nil {
			snplog.LogFatal(err)
		}

		possibleFiles := []string{}
		for n := range files {
			possibleFiles = append(possibleFiles, n)
		}

		arg0, err := os.Executable()
		if err != nil {
			snplog.LogFatal(err)
		}
		basePath := filepath.Dir(arg0)

		cnx := context.Background()

		c, err := console.NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
		if err != nil {
			snplog.LogFatal(err)
		}

		schemaResolver, err := console.NewSchemaDeployChecker(cnx, c)
		if err != nil {
			snplog.LogFatal(err)
		}

		lookup, err := validation.NewDPLookup(schemaResolver, files)
		if err != nil {
			snplog.LogFatal(err)
		}

		slog.Debug("validation", "msg", "from", "paths", searchPaths, "files", possibleFiles)

		err = lookup.SlogValidations(basePath)
		if err != nil {
			snplog.LogFatal(err)
		}

		if ghOut {
			err := lookup.GhAnnotateValidations(basePath)
			if err != nil {
				snplog.LogFatal(err)
			}
		}

		numErrors := lookup.ValidationErrorCount()

		if numErrors > 0 {
			snplog.LogFatal(fmt.Errorf("validation failed %d errors", numErrors))
		} else {
			dpCount := 0
			for range lookup.DataProducts {
				dpCount++
			}
			saCount := 0
			for range lookup.SourceApps {
				saCount++
			}
			slog.Info("validation", "msg", "success", "data products found", dpCount, "source applications found", saCount)
		}
	},
}

func init() {
	DataProductsCmd.AddCommand(validateCmd)

	validateCmd.PersistentFlags().Bool("gh-annotate", false, "Output suitable for github workflow annotation (ignores -s)")
}
