/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package signals

import (
	"errors"
	"log/slog"

	"github.com/snowplow/snowplow-cli/internal/config"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/wizard"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check that the Signals credentials in .env work",
	Long: `Check the app's Signals credentials the way the app uses them: exchange the
key in .env for a token, then make an authenticated call to the Signals API.

With --json-output the result is a single JSON object, for coding agents.
The command exits non-zero when the check fails.`,
	Example: `  $ snowplow-cli signals status
  $ snowplow-cli signals status --json-output`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := snplog.InitLogging(cmd); err != nil {
			return err
		}
		_ = config.InitConsoleConfigForSetup(cmd)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		host, err := cmd.Flags().GetString("host")
		if err != nil {
			return err
		}
		envPath, err := cmd.Flags().GetString("env")
		if err != nil {
			return err
		}

		res := wizard.VerifySignals(cmd.Context(), host, envPath)
		if !res.OK {
			slog.Error("Signals check failed", "ok", false, "apiUrl", res.APIURL, "reason", res.Reason, "error", res.Error)
			return errors.New(res.Error)
		}
		slog.Info("Signals credentials work", "ok", true, "apiUrl", res.APIURL)
		return nil
	},
}

func init() {
	statusCmd.Flags().String("env", ".env", "The .env file holding the SNOWPLOW_SIGNALS_* values")
}
