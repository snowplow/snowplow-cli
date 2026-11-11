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
	"fmt"

	"github.com/snowplow/snowplow-cli/internal/config"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/setup"
	"github.com/spf13/cobra"
)

var LoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Snowplow Console with your browser",
	Long: `Authenticate with Snowplow Console using device authentication flow and save the login

No API key is created. Commands use the saved login when no API key is
configured, and Snowtype shares the same login.`,
	Example:      `  $ snowplow-cli login`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := snplog.InitLogging(cmd); err != nil {
			return err
		}

		if err := config.InitConsoleConfigForSetup(cmd); err != nil {
			return err
		}

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

		if clientID == "" {
			return fmt.Errorf("client-id is required. Use --client-id flag")
		}

		return setup.LoginConfig(clientID, auth0Domain, consoleHost, cmd.Context())
	},
}

func init() {
	config.InitConsoleFlags(LoginCmd)

	LoginCmd.Flags().String("client-id", setup.DefaultClientID, "Auth0 Client ID for device auth")
	LoginCmd.Flags().String("auth0-domain", setup.DefaultAuth0Domain, "Auth0 domain")
}
