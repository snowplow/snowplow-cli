package ds

import (
	"context"
	"log/slog"

	"github.com/snowplow-product/snowplow-cli/internal/console"
	"github.com/snowplow-product/snowplow-cli/internal/io"
	"github.com/snowplow-product/snowplow-cli/internal/util"
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
		dataStructuresFolder := util.DataStructuresFolder
		if len(args) > 0 {
			dataStructuresFolder = args[0]
		}
		format, _ := cmd.Flags().GetString("format")
		files := io.Files{DataStructuresLocation: dataStructuresFolder, ExtentionPreference: format}

		apiKeyId, _ := cmd.Flags().GetString("api-key-id")
		apiKeySecret, _ := cmd.Flags().GetString("api-key-secret")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")

		cnx := context.Background()

		c, err := console.NewApiClient(cnx, host, apiKeyId, apiKeySecret, org)
		if err != nil {
			io.LogFatalMsg("client creation fail", err)
		}

		dss, err := console.GetAllDataStructures(cnx, c)
		if err != nil {
			io.LogFatalMsg("data structure fetch failed", err)
		}

		err = files.CreateDataStructures(dss)
		if err != nil {
			io.LogFatal(err)
		}

		slog.Info("wrote data structures", "count", len(dss))
	},
}

func init() {
	DataStructuresCmd.AddCommand(downloadCmd)

	downloadCmd.PersistentFlags().StringP("output-format", "f", "yaml", "Format of the files to read/write. json or yaml are supported")
}
