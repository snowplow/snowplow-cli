package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

var downloadCmd = &cobra.Command{
	Use:   "download {directory ./data-structures}",
	Short: "Download all data structures from BDP Console",
	Args:  cobra.MaximumNArgs(1),
	Long: `Downloads the latest versions of all data structures from BDP Console.

Will retrieve schema contents from your development environment.
If no directory is provided then defaults to 'data-structures' in the current directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		dataStructuresFolder := "data-structures"
		if len(args) > 0 {
			dataStructuresFolder = args[0]
		}
		format, _ := cmd.Flags().GetString("format")
		files := Files{dataStructuresFolder, format}

		apikey, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")

		cnx := context.Background()

		c, err := NewApiClient(cnx, host, apikey, org)
		if err != nil {
			log.Fatal(err)
		}

		dss, err := GetAllDataStructures(cnx, c)
		if err != nil {
			log.Fatal(err)
		}

		files.createDataStructures(dss)
	},
}

func init() {
	dataStructuresCmd.AddCommand(downloadCmd)

	downloadCmd.PersistentFlags().StringP("format", "f", "yaml", "Format of the files to read/write. json or yaml are supported")
}
