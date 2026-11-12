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
	"log/slog"

	"github.com/snowplow/snowplow-cli/internal/auth"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/spf13/cobra"
)

var LogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of Snowplow Console",
	Long: `Revoke the login saved by 'snowplow-cli login' and delete it

Snowtype shares the same login, so this logs it out too. An API key, if one
is configured, is not affected and keeps working.`,
	Example:      `  $ snowplow-cli logout`,
	SilenceUsage: true,
	Args:         cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := snplog.InitLogging(cmd); err != nil {
			return err
		}

		result, err := auth.Logout(cmd.Context())
		if err != nil {
			return err
		}

		switch {
		case !result.LoggedIn:
			slog.Info("Not logged in")
		case result.Revoked:
			slog.Info("Logged out", "file", auth.Path())
		default:
			slog.Warn("Logged out on this computer, but the login could not be revoked with Snowplow. It stays valid until it expires", "file", auth.Path())
		}
		return nil
	},
}
