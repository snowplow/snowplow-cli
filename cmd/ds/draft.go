package ds

import (
	"github.com/snowplow-product/snowplow-cli/internal/io"
	"github.com/snowplow-product/snowplow-cli/internal/util"
	"github.com/spf13/cobra"
)

// draftCmd represents the draft command
var draftCmd = &cobra.Command{
	Use:   "draft my_new_data_structure {directory ./data-structures}",
	Short: "Create a new file that represents a minimal data structure",
	Long:  `Creates a new file with all the required fields populated by sample data`,
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		dataStructuresFolder := util.DataStructuresFolder
		if len(args) > 1 {
			dataStructuresFolder = args[1]
		}
		format, _ := cmd.Flags().GetString("format")
		err := util.CreateNewDataStructureFile(args[0], dataStructuresFolder, format)
		if err != nil {
			io.LogFatal(err)
		}
	},
}

func init() {
	DataStructuresCmd.AddCommand(draftCmd)
	draftCmd.PersistentFlags().StringP("format", "f", "yaml", "Format of the files to read/write. json or yaml are supported")
}
