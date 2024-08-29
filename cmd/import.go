/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

// importCmd represents the import command
var importCmd = &cobra.Command{
	Use:   "import",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		dataStructuresFolder, _ := cmd.Flags().GetString("data-structures")
		format, _ := cmd.Flags().GetString("format")
		files := Files{dataStructuresFolder, format}

		apikey, _ := cmd.Flags().GetString("api-key")
		host, _ := cmd.Flags().GetString("host")
		org, _ := cmd.Flags().GetString("org-id")

		cnx := context.Background()

		fmt.Println("import called")

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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return InitConsoleConfig(cmd)
	},
}

func init() {
	rootCmd.AddCommand(importCmd)
	InitConsoleFlags(importCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// importCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// importCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
