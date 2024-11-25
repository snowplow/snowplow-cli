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
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/flytam/filenamify"
	"github.com/google/uuid"
	snplog "github.com/snowplow-product/snowplow-cli/internal/logging"
	"github.com/snowplow-product/snowplow-cli/internal/util"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func mkSafeFilename(name string) (string, error) {
	appn, err := filenamify.FilenamifyV2(name)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(strings.TrimSpace(strings.ToLower(appn)), " ", "-"), nil
}

var generateCmd = &cobra.Command{
	Use:     "generate [paths...]",
	Short:   "Generate new data products and source applications locally",
	Aliases: []string{"gen"},
	Args:    cobra.NoArgs,
	Long: `Will write new data products and/or source application to file based on the arguments provided.

Example:
  $ snowplow-cli dp gen --source-app "Mobile app"
  Will result in a new source application getting written to './data-products/source-applications/mobile-app.yaml'

  $ snowplow-cli dp gen --data-product "Ad tracking" --output-format json --data-products-directory dir1
  Will result in a new data product getting written to './dir1/ad-tracking.json'
`,
	Example: `  $ snowplow-cli dp generate --source-app "Mobile app" --source-app "Web app" --data-product "Signup flow"`,
	Run: func(cmd *cobra.Command, args []string) {
		outFmt, _ := cmd.Flags().GetString("output-format")

		sourceAppDirectory, _ := cmd.Flags().GetString("source-apps-directory")
		sourceApps, _ := cmd.Flags().GetStringArray("source-app")

		dataproductDirectory, _ := cmd.Flags().GetString("data-products-directory")
		dataProducts, _ := cmd.Flags().GetStringArray("data-product")

		err := os.MkdirAll(sourceAppDirectory, os.ModePerm)
		if err != nil && !os.IsExist(err) {
			snplog.LogFatal(err)
		}
		err = os.MkdirAll(dataproductDirectory, os.ModePerm)
		if err != nil && !os.IsExist(err) {
			snplog.LogFatal(err)
		}

		for _, app := range sourceApps {
			appn, err := mkSafeFilename(app)
			if err != nil {
				snplog.LogFatal(err)
			}
			filename := filepath.Join(sourceAppDirectory, appn+"."+outFmt)
			err = write(buildSaTpl(app), filename, outFmt)
			if err != nil {
				snplog.LogFatal(err)
			}
			slog.Info("generate", "msg", "wrote", "kind", "source app", "file", filename)
		}

		for _, app := range dataProducts {
			appn, err := mkSafeFilename(app)
			if err != nil {
				snplog.LogFatal(err)
			}
			filename := filepath.Join(dataproductDirectory, appn+"."+outFmt)
			err = write(buildDpTpl(app), filename, outFmt)
			if err != nil {
				snplog.LogFatal(err)
			}
			slog.Info("generate", "msg", "wrote", "kind", "data product", "file", filename)
		}

	},
}

func write(tpl any, fname string, format string) error {

	var out []byte
	var err error

	if format == "yaml" {
		out, err = yaml.Marshal(tpl)
		if err != nil {
			return err
		}
	}

	if format == "json" {
		out, err = json.MarshalIndent(tpl, "", "  ")
		if err != nil {
			return err
		}
	}

	err = os.WriteFile(fname, out, 0644)
	if err != nil {
		return err
	}
	return nil
}

func buildDpTpl(name string) any {
	return map[string]any{
		"apiVersion":   "v1",
		"resourceType": "data-product",
		"resourceName": uuid.NewString(),
		"data": map[string]any{
			"name":                name,
			"sourceApplications":  []string{},
			"eventSpecifications": []any{},
		},
	}
}

func buildSaTpl(name string) any {
	return map[string]any{
		"apiVersion":   "v1",
		"resourceType": "source-application",
		"resourceName": uuid.NewString(),
		"data": map[string]any{
			"name":   name,
			"appIds": []string{},
			"entities": map[string]any{
				"tracked":  []any{},
				"enriched": []any{},
			},
		},
	}
}

func init() {
	DataProductsCmd.AddCommand(generateCmd)

	generateCmd.Flags().String("output-format", "yaml", "File format (yaml|json)")
	generateCmd.Flags().String("data-products-directory", util.DataProductsFolder, "Directory to write data products to")
	generateCmd.Flags().String("source-apps-directory", filepath.Join(util.DataProductsFolder, util.SourceAppsFolder), "Directory to write source apps to")

	generateCmd.Flags().StringArray("source-app", []string{}, "Name of source app to generate")
	generateCmd.Flags().StringArray("data-product", []string{}, "Name of data product to generate")
}
