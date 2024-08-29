package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:     "publish",
	Aliases: []string{"pub"},
	Short:   "Publishing commands for data structures",
	Long: `Publishing commands for data structures

Publish local data structures to BDP console.
`,
}

var devCmd = &cobra.Command{
	Use:   "dev path...",
	Short: "Publish data structures to your development environment",
	Args:  cobra.MinimumNArgs(1),
	Long: `Publish modified data structures to BDP Console and your development environment

The 'meta' section of a data structure is not versioned within BDP Console.
Changes to it will be published by this command.
	`,

	Run: func(cmd *cobra.Command, args []string) {
		apikey, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")

		dataStructuresLocal, err := DataStructuresFromPaths(args)
		if err != nil {
			log.Fatal(err)
		}

		cnx := context.Background()

		c, err := NewApiClient(cnx, host, apikey, org)
		if err != nil {
			log.Fatal(err)
		}

		for _, ds := range dataStructuresLocal {
			_, err := Validate(cnx, c, ds)
			if err != nil {
				log.Fatal(err)
			}
			_, err = PublishDev(cnx, c, ds)
			if err != nil {
				log.Fatal(err)
			}
		}
	},
}

var prodCmd = &cobra.Command{
	Use:   "prod path...",
	Short: "Publish data structures to your production environment",
	Args:  cobra.MinimumNArgs(1),
	Long: `Publish data structures from your development to your production environment

Data structures found on <path...> which are deployed to your development
environment will be published to your production environment.
	`,
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

		for _, ds := range dataStructures {
			_, err = PublishProd(cnx, c, ds)
			if err != nil {
				log.Fatal(err)
			}
		}
	},
}

func init() {
	dataStructuresCmd.AddCommand(publishCmd)
	publishCmd.AddCommand(devCmd)
	publishCmd.AddCommand(prodCmd)
}
