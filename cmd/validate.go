package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate path...",
	Short: "Validate data structures with BDP Console",
	Args: cobra.MinimumNArgs(1),
	Long: `Sends all data structures from <path> for validation by BDP Console.`,
	Run: func(cmd *cobra.Command, args []string) {
		apikey, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")

		dataStructures, err := DataStructuresFromPaths(args)
		if err != nil {
			log.Fatal(err)
		}

		cnx := context.Background()

		c, err := NewApiClient(cnx, host, apikey, org)
		if err != nil {
			log.Fatal(err)
		}

		for f, ds := range dataStructures {
			_, err := Validate(cnx, c, ds)
			if err != nil {
				log.Printf("%s: %s\n", f, err)
			}
		}
	},
}

func init() {
	dataStructuresCmd.AddCommand(validateCmd)
}
