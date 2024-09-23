package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	nameRegexp   = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)
	vendorRegexp = regexp.MustCompile(`^[a-zA-Z0-9-_.]+$`)
)

var generateCmd = &cobra.Command{
	Use:     "generate --name login_click --vendor com.example",
	Aliases: []string{"gen"},
	Short:   "Generate a new data structure locally",
	Long:    `Uses a default template to build a minimal but valid data structure ready to be built into something useful.`,
	Run: func(cmd *cobra.Command, args []string) {
		name, _ := cmd.Flags().GetString("name")
		vendor, _ := cmd.Flags().GetString("vendor")
		outFmt, _ := cmd.Flags().GetString("output-format")
		outFile, _ := cmd.Flags().GetString("output-file")
		event, _ := cmd.Flags().GetBool("event")
		entity, _ := cmd.Flags().GetBool("entity")

		if ok := nameRegexp.Match([]byte(name)); !ok {
			LogFatal(errors.New("name did not match [a-zA-Z0-9-_]+"))
		}

		if ok := vendorRegexp.Match([]byte(vendor)); !ok {
			LogFatal(errors.New("vendor did not match [a-zA-Z0-9-_.]+"))
		}

		if ok := outFmt == "json" || outFmt == "yaml"; !ok {
			LogFatal(errors.New("unsupported output format. Was not yaml or json"))
		}

		var schemaType string
		if event {
			schemaType = "event"
		}
		if entity {
			schemaType = "entity"
		}

		yamlOut := fmt.Sprintf(yamlTemplate, schemaType, vendor, name)

		ds := DataStructure{}
		err := yaml.Unmarshal([]byte(yamlOut), &ds)
		if err != nil {
			LogFatal(err)
		}

		output := yamlOut
		if outFmt == "json" {
			jsonOut, err := json.MarshalIndent(ds, "", "  ")
			if err != nil {
				LogFatal(err)
			}
			output = string(jsonOut)
		}

		if outFile == "" {
			fmt.Print(output)
		} else {
			err = os.WriteFile(outFile, []byte(output), 0644)
			if err != nil {
				LogFatal(err)
			}
		}

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
	dataStructuresCmd.AddCommand(generateCmd)

	generateCmd.Flags().String("name", "", `A name for the data structure.
Must conform to the regex pattern [a-zA-Z0-9-_]+`)
	if err := generateCmd.MarkFlagRequired("name"); err != nil {
		panic(err)
	}

	generateCmd.Flags().String("vendor", "", `A vendor for the data structure.
Must conform to the regex pattern [a-zA-Z0-9-_.]+`)
	if err := generateCmd.MarkFlagRequired("vendor"); err != nil {
		panic(err)
	}

	generateCmd.Flags().String("output-format", "yaml", "Format for the file (yaml|json)")
	generateCmd.Flags().String("output-file", "", "Location to write the file defaults to stdout")

	generateCmd.Flags().Bool("event", true, "Generate data structure as an event")
	generateCmd.Flags().Bool("entity", false, "Generate data structure as an entity")
}
