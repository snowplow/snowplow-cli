/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package setup

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/fatih/color"
	"github.com/snowplow/snowplow-cli/internal/auth/device"
	"github.com/snowplow/snowplow-cli/internal/config"
	"github.com/snowplow/snowplow-cli/internal/console"
)

func SetupConfig(clientID, auth0Domain, consoleHost string, readOnly, isDotenv bool, ctx context.Context) error {
	slog.Debug("Starting Snowplow CLI setup",
		"auth0-domain", auth0Domain,
		"console-host", consoleHost,
		"audience", device.Audience,
		"read-only", readOnly)

	session, err := device.Login(ctx, clientID, auth0Domain, consoleHost, false)
	if err != nil {
		return err
	}

	green := color.New(color.FgGreen)

	apiKey, err := console.CreateAPIKey(ctx, session.AccessToken, consoleHost, session.Org.ID, readOnly)
	if errors.Is(err, console.ErrAPIKeyForbidden) {
		return fmt.Errorf("failed to create API key: %w. Ask an organization admin for them, create an API key with the permissions you have (%s), or run 'snowplow-cli login' to work without an API key", err, console.APIKeyDocsURL)
	}
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	if err := config.PersistConfig(session.Org.ID, apiKey.ID, apiKey.Secret, consoleHost, isDotenv); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	green.Printf("✓ Logged in to Snowplow Console as %s\n", session.Email)
	green.Printf("✓ Authentication credentials saved\n")
	if readOnly {
		green.Printf("✓ API key created with read-only permissions for %s\n", session.Org.Name)
	} else {
		green.Printf("✓ API key created with permissions to manage tracking plans and data structures for %s\n", session.Org.Name)
	}

	return nil
}
