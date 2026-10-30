/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package verify

import (
	"errors"
	"log/slog"

	"github.com/snowplow/snowplow-cli/internal/wizard"
	"github.com/spf13/cobra"
)

var signalsCmd = &cobra.Command{
	Use:   "signals",
	Short: "Check that the Signals credentials in .env work",
	Long: `Check the app's Signals credentials the way the app uses them: exchange the
key in .env for a token, then list attribute keys on the Signals API.

With --json-output the result is a single JSON object, for coding agents.
The command exits non-zero when the check fails.`,
	Example: `  $ snowplow-cli verify signals
  $ snowplow-cli verify signals --json-output`,
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		host, _ := cmd.Flags().GetString("host")
		envPath, _ := cmd.Flags().GetString("env")

		res := wizard.VerifySignals(cmd.Context(), host, envPath)
		if !res.OK {
			slog.Error("Signals check failed", "ok", false, "apiUrl", res.APIURL, "reason", res.Reason, "error", res.Error)
			return errors.New(res.Error)
		}
		slog.Info("Signals credentials work", "ok", true, "apiUrl", res.APIURL, "attributeKeys", res.AttributeKeys)
		return nil
	},
}

func init() {
	signalsCmd.Flags().String("env", ".env", "The .env file holding the SNOWPLOW_SIGNALS_* values")
}
