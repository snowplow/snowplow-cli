/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package cmd

import (
	"context"
	"fmt"

	"github.com/snowplow/snowplow-cli/internal/config"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/setup"
	"github.com/spf13/cobra"
)

var SetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up Snowplow CLI with device authentication",
	Long:  `Authenticate with Snowplow Console using device authentication flow and create an API key`,
	Example: `  $ snowplow-cli setup
  $ snowplow-cli setup --read-only`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := snplog.InitLogging(cmd); err != nil {
			return err
		}

		if err := config.InitConsoleConfigForSetup(cmd); err != nil {
			return err
		}

		ctx := context.Background()

		clientID, err := cmd.Flags().GetString("client-id")
		if err != nil {
			return err
		}

		auth0Domain, err := cmd.Flags().GetString("auth0-domain")
		if err != nil {
			return err
		}

		consoleHost, err := cmd.Flags().GetString("host")
		if err != nil {
			return err
		}

		readOnly, err := cmd.Flags().GetBool("read-only")
		if err != nil {
			return err
		}

		isDotenv, err := cmd.Flags().GetBool("dotenv")
		if err != nil {
			return err
		}

		if clientID == "" {
			return fmt.Errorf("client-id is required. Use --client-id flag")
		}

		return setup.SetupConfig(clientID, auth0Domain, consoleHost, readOnly, isDotenv, ctx)
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	config.InitConsoleFlags(SetupCmd)

	SetupCmd.Flags().String("client-id", "EXQ3csSDr6D7wTIiebNPhXpgkSsOzCzi", "Auth0 Client ID for device auth")
	SetupCmd.Flags().String("auth0-domain", "id.snowplowanalytics.com", "Auth0 domain")
	SetupCmd.Flags().Bool("read-only", false, "Create a read-only API key")
	SetupCmd.Flags().Bool("dotenv", false, "Store as .env file in current working directory")
}
