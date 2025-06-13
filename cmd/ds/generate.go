/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package ds

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"

	"github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/model"
	"github.com/snowplow/snowplow-cli/internal/util"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	nameRegexp   = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)
	vendorRegexp = regexp.MustCompile(`^[a-zA-Z0-9-_.]+$`)
)

var generateCmd = &cobra.Command{
	Use:     "generate login_click {directory ./data-structures}",
	Aliases: []string{"gen"},
	Args:    cobra.RangeArgs(1, 2),
	Short:   "Generate a new data structure locally",
	Long: `Will write a new data structure to file based on the arguments provided.

Example:
  $ snowplow-cli ds gen login_click --vendor com.example
  Will result in a new data structure getting written to './data-structures/com.example/login_click.yaml'
  The directory 'com.example' will be created automatically.

  $ snowplow-cli ds gen login_click
  Will result in a new data structure getting written to './data-structures/login_click.yaml' with
  an empty vendor field. Note that vendor is a required field and will cause a validation error if not completed.`,
	Example: `  $ snowplow-cli ds generate my-ds
  $ snowplow-cli ds generate my-ds ./my-data-structures`,
	Run: func(cmd *cobra.Command, args []string) {
		vendor, _ := cmd.Flags().GetString("vendor")
		outFmt, _ := cmd.Flags().GetString("output-format")
		event, _ := cmd.Flags().GetBool("event")
		entity, _ := cmd.Flags().GetBool("entity")
		noLsp, _ := cmd.Flags().GetBool("no-lsp")

		name := args[0]

		if ok := nameRegexp.Match([]byte(name)); !ok {
			logging.LogFatal(errors.New("name did not match [a-zA-Z0-9-_]+"))
		}

		if ok := vendorRegexp.Match([]byte(vendor)); vendor != "" && !ok {
			logging.LogFatal(errors.New("vendor did not match [a-zA-Z0-9-_.]+"))
		}

		if ok := outFmt == "json" || outFmt == "yaml"; !ok {
			logging.LogFatal(errors.New("unsupported output format. Was not yaml or json"))
		}

		outDir := filepath.Join(util.DataStructuresFolder, vendor)
		if len(args) > 1 {
			outDir = filepath.Join(args[1], vendor)
		}

		outFile := filepath.Join(outDir, name+"."+outFmt)
		if _, err := os.Stat(outFile); !os.IsNotExist(err) {
			logging.LogFatal(fmt.Errorf("file already exists, not writing %s", outFile))
		}

		var schemaType string
		if event {
			schemaType = "event"
		}
		if entity {
			schemaType = "entity"
		}

		lspComment := ""
		if !noLsp {
			lspComment = fmt.Sprint("\n# yaml-language-server: $schema=%s%s.json", util.RepoRawFileURL, util.DataStructureResourceType)
		}

		yamlOut := fmt.Sprintf(yamlTemplate, lspComment, schemaType, vendor, name)

		ds := model.DataStructure{}
		err := yaml.Unmarshal([]byte(yamlOut), &ds)
		if err != nil {
			logging.LogFatal(err)
		}

		output := yamlOut
		if outFmt == "json" {
			jsonOut, err := json.MarshalIndent(ds, "", "  ")
			if err != nil {
				logging.LogFatal(err)
			}
			output = string(jsonOut)
		}

		err = os.Mkdir(outDir, os.ModePerm)
		if err != nil && !os.IsExist(err) {
			logging.LogFatal(err)
		}
		err = os.WriteFile(outFile, []byte(output), 0644)
		if err != nil {
			logging.LogFatal(err)
		}

		slog.Info("generate", "wrote", outFile)
	},
}

var yamlTemplate = `# You might not need a custom data structure.
# Please have a look at the available list of out of the box events and entities:
# https://docs.snowplow.io/docs/collecting-data/collecting-from-own-applications/snowplow-tracker-protocol/%s

apiVersion: v1
resourceType: data-structure
meta:
  hidden: false
  schemaType: %s
  customData: {}
data:
  $schema: http://iglucentral.com/schemas/com.snowplowanalytics.self-desc/schema/jsonschema/1-0-0#
  self:
    vendor: %s
    name: %s
    format: jsonschema
    version: 1-0-0
  type: object
  properties: {}
  additionalProperties: false
`

func init() {
	DataStructuresCmd.AddCommand(generateCmd)

	generateCmd.Flags().String("vendor", "", `A vendor for the data structure.
Must conform to the regex pattern [a-zA-Z0-9-_.]+`)

	generateCmd.Flags().String("output-format", "yaml", "Format for the file (yaml|json)")

	generateCmd.Flags().Bool("event", true, "Generate data structure as an event")
	generateCmd.Flags().Bool("entity", false, "Generate data structure as an entity")
	generateCmd.Flags().Bool("no-lsp", false, "Disable LSP server functionality")
}
