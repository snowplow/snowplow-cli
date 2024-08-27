/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"log"

	"github.com/spf13/cobra"
)

// prodCmd represents the prod command
var prodCmd = &cobra.Command{
	Use:   "prod",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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
	syncCmd.AddCommand(prodCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// prodCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// prodCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
