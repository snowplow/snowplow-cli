/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package login

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/snowplow/snowplow-cli/internal/auth"
	"github.com/snowplow/snowplow-cli/internal/auth/device"
)

func LoginConfig(clientID, auth0Domain, consoleHost string, hasAPIKey bool, ctx context.Context) error {
	session, err := device.Login(ctx, clientID, auth0Domain, consoleHost, false)
	if err != nil {
		return err
	}

	err = auth.Save(&auth.Login{
		AccessToken:  session.AccessToken,
		RefreshToken: session.RefreshToken,
		ExpiresAt:    session.Expiry,
		OrgID:        session.Org.ID,
		Host:         consoleHost,
		ClientID:     clientID,
		Auth0Domain:  auth0Domain,
	})
	if err != nil {
		return fmt.Errorf("failed to save login: %w", err)
	}

	if session.RefreshToken == "" {
		slog.Warn("No refresh token was issued, so this login ends when its token expires", "expires-at", session.Expiry)
	}
	slog.Info("Logged in to Snowplow Console", "email", session.Email, "org-id", session.Org.ID)
	slog.Info("Login saved", "file", auth.Path())
	if hasAPIKey {
		slog.Warn("An API key is configured and takes precedence over this login. Remove it to use the login")
	}

	return nil
}
