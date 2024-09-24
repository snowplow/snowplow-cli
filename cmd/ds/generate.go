package ds

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"

	"github.com/snowplow-product/snowplow-cli/internal/io"
	"github.com/snowplow-product/snowplow-cli/internal/model"
	"github.com/snowplow-product/snowplow-cli/internal/util"
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
  an empty vendor field. Note that vendor is a required field and will cause a validation error if not completed.
`,
	Run: func(cmd *cobra.Command, args []string) {
		vendor, _ := cmd.Flags().GetString("vendor")
		outFmt, _ := cmd.Flags().GetString("output-format")
		event, _ := cmd.Flags().GetBool("event")
		entity, _ := cmd.Flags().GetBool("entity")

		name := args[0]

		if ok := nameRegexp.Match([]byte(name)); !ok {
			io.LogFatal(errors.New("name did not match [a-zA-Z0-9-_]+"))
		}

		if ok := vendorRegexp.Match([]byte(vendor)); vendor != "" && !ok {
			io.LogFatal(errors.New("vendor did not match [a-zA-Z0-9-_.]+"))
		}

		if ok := outFmt == "json" || outFmt == "yaml"; !ok {
			io.LogFatal(errors.New("unsupported output format. Was not yaml or json"))
		}

		outDir := filepath.Join(util.DataStructuresFolder, vendor)
		if len(args) > 1 {
			outDir = filepath.Join(args[1], vendor)
		}

		outFile := filepath.Join(outDir, name+"."+outFmt)
		if _, err := os.Stat(outFile); !os.IsNotExist(err) {
			io.LogFatal(fmt.Errorf("file already exists, not writing %s", outFile))
		}

		var schemaType string
		if event {
			schemaType = "event"
		}
		if entity {
			schemaType = "entity"
		}

		yamlOut := fmt.Sprintf(yamlTemplate, schemaType, vendor, name)

		ds := model.DataStructure{}
		err := yaml.Unmarshal([]byte(yamlOut), &ds)
		if err != nil {
			io.LogFatal(err)
		}

		output := yamlOut
		if outFmt == "json" {
			jsonOut, err := json.MarshalIndent(ds, "", "  ")
			if err != nil {
				io.LogFatal(err)
			}
			output = string(jsonOut)
		}

		err = os.Mkdir(outDir, os.ModePerm)
		if err != nil {
			io.LogFatal(err)
		}
		err = os.WriteFile(outFile, []byte(output), 0644)
		if err != nil {
			io.LogFatal(err)
		}

		slog.Info("generate", "wrote", outFile)
	},
}

var yamlTemplate = `# You might not need a custom data structure.
# Please have a look at the available list of out of the box events and entities:
# https://docs.snowplow.io/docs/collecting-data/collecting-from-own-applications/snowplow-tracker-protocol/
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
}
