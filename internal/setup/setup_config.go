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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/browser"
	"github.com/snowplow/snowplow-cli/internal/config"
	"github.com/snowplow/snowplow-cli/internal/console"
	"golang.org/x/oauth2"
)

const snowplowAudience = "https://snowplowanalytics.com/api/"

func SetupConfig(clientID, auth0Domain, consoleHost string, readOnly, isDotenv bool, ctx context.Context) error {
	slog.Debug("Starting Snowplow CLI setup",
		"auth0-domain", auth0Domain,
		"console-host", consoleHost,
		"audience", snowplowAudience,
		"read-only", readOnly)

	slog.Debug("Initiating device authentication flow")

	oauthConfig := &oauth2.Config{
		ClientID: clientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:       fmt.Sprintf("https://%s/oauth/authorize", auth0Domain),
			TokenURL:      fmt.Sprintf("https://%s/oauth/token", auth0Domain),
			DeviceAuthURL: fmt.Sprintf("https://%s/oauth/device/code", auth0Domain),
		},
		Scopes: []string{"openid", "profile", "email", "offline_access"},
	}

	deviceAuth, err := oauthConfig.DeviceAuth(ctx,
		oauth2.SetAuthURLParam("audience", snowplowAudience))
	if err != nil {
		handleAuthError(err, clientID, auth0Domain)
		return fmt.Errorf("failed to initiate device authentication flow: %w", err)
	}

	yellow := color.New(color.FgYellow, color.Bold)
	green := color.New(color.FgGreen)
	cyan := color.New(color.FgCyan)

	authURL := deviceAuth.VerificationURIComplete
	if authURL == "" {
		authURL = deviceAuth.VerificationURI
	}

	fmt.Printf("You must sign in to continue. Would you like to sign in (Y/n)?")

	var openBrowser string
	browserOpened := false

	_, _ = fmt.Scanln(&openBrowser)

	if openBrowser == "" || strings.ToLower(openBrowser) == "y" {
		if err := browser.OpenURL(authURL); err == nil {
			browserOpened = true
			green.Printf("\n✓ Opened %s in your browser\n", authURL)
		}
	}

	if !browserOpened {
		fmt.Printf("\n! Please manually open this URL in your browser:\n")
	} else {
		fmt.Printf("If the browser didn't open, manually visit:\n")
	}
	cyan.Printf("  %s\n", authURL)

	if deviceAuth.VerificationURIComplete == "" {
		fmt.Printf("Then enter the code: ")
		yellow.Printf("%s\n", deviceAuth.UserCode)
	} else {
		fmt.Printf("(Your verification code ")
		yellow.Printf("%s", deviceAuth.UserCode)
		fmt.Printf(" should be pre-filled)\n")
	}

	fmt.Printf("\n⠋ Waiting for authentication...")

	token, err := pollForTokenWithSpinner(ctx, oauthConfig, deviceAuth)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	green.Printf("\n✓ Authentication complete.\n")
	slog.Debug("Authentication successful, proceeding to organization selection")

	jwtOrgID, err := getOrgIDFromJWT(token.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to get organization ID: %w", err)
	}

	organizations, err := console.GetOrganizations(ctx, token.AccessToken, consoleHost)
	if err != nil {
		return fmt.Errorf("failed to fetch organizations: %w", err)
	}

	if len(organizations) == 0 {
		return fmt.Errorf("no organizations found for your account")
	}

	var selectedOrg *console.Organization

	if len(organizations) == 1 {
		selectedOrg = &organizations[0]
		green.Printf("✓ Found organization: %s\n", selectedOrg.Name)
	} else {
		selectedOrg, err = selectOrganization(organizations, jwtOrgID)
		if err != nil {
			return fmt.Errorf("failed to select organization: %w", err)
		}
		green.Printf("✓ Selected organization: %s\n", selectedOrg.Name)
	}

	slog.Debug("Organization selected", "org-id", selectedOrg.ID, "org-name", selectedOrg.Name)

	apiKey, err := console.CreateAPIKey(ctx, token.AccessToken, consoleHost, selectedOrg.ID, readOnly)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	userInfo, err := console.GetUserInfo(ctx, token.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	if err := config.PersistConfig(selectedOrg.ID, apiKey.ID, apiKey.Secret, consoleHost, isDotenv); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	green.Printf("✓ Logged in to Snowplow Console as %s\n", userInfo.Email)
	green.Printf("✓ Authentication credentials saved\n")
	if readOnly {
		green.Printf("✓ API key created with read-only permissions for %s\n", selectedOrg.Name)
	} else {
		green.Printf("✓ API key created with admin permissions for %s\n", selectedOrg.Name)
	}

	return nil
}

func pollForTokenWithSpinner(ctx context.Context, config *oauth2.Config, deviceAuth *oauth2.DeviceAuthResponse) (*oauth2.Token, error) {
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerIndex := 0

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.Until(deviceAuth.Expiry)
	if timeout <= 0 {
		timeout = 15 * time.Minute
	}
	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tokenChan := make(chan *oauth2.Token, 1)
	errChan := make(chan error, 1)

	go func() {
		token, err := config.DeviceAccessToken(pollCtx, deviceAuth)
		if err != nil {
			errChan <- err
			return
		}
		tokenChan <- token
	}()

	for {
		select {
		case token := <-tokenChan:
			fmt.Printf("\r")
			return token, nil
		case err := <-errChan:
			fmt.Printf("\r")
			return nil, err
		case <-ticker.C:
			fmt.Printf("\r%s Waiting for authentication...", spinner[spinnerIndex])
			spinnerIndex = (spinnerIndex + 1) % len(spinner)
		}
	}
}

func selectOrganization(organizations []console.Organization, jwtOrgID string) (*console.Organization, error) {
	defaultIndex := getDefaultIndex(organizations, jwtOrgID)
	displayAvailableOrgs(organizations, jwtOrgID, defaultIndex)
	var input string
	_, _ = fmt.Scanln(&input)
	return selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)
}

func selectOrganizationForInput(organizations []console.Organization, jwtOrgID, input string, defaultIndex int) (*console.Organization, error) {
	if input == "" && defaultIndex >= 0 {
		return &organizations[defaultIndex], nil
	}

	var selection int
	if _, err := fmt.Sscanf(input, "%d", &selection); err != nil {
		return nil, fmt.Errorf("invalid selection: %s", input)
	}

	if selection < 1 || selection > len(organizations) {
		return nil, fmt.Errorf("selection must be between 1 and %d", len(organizations))
	}

	return &organizations[selection-1], nil
}

func displayAvailableOrgs(organizations []console.Organization, jwtOrgID string, defaultIndex int) {
	cyan := color.New(color.FgCyan)
	yellow := color.New(color.FgYellow, color.Bold)

	fmt.Printf("\n")
	cyan.Printf("You have access to %d organizations:\n\n", len(organizations))

	for i, org := range organizations {
		if i == defaultIndex {
			yellow.Printf("  %d. %s (default)\n", i+1, org.Name)
		} else {
			fmt.Printf("  %d. %s\n", i+1, org.Name)
		}
	}

	if defaultIndex >= 0 {
		fmt.Printf("Select organization [1-%d] (default: %d): ", len(organizations), defaultIndex+1)
	} else {
		fmt.Printf("Select organization [1-%d]: ", len(organizations))
	}
}

func getDefaultIndex(organizations []console.Organization, jwtOrgID string) int {
	var defaultIndex = -1
	for i, org := range organizations {
		if org.ID == jwtOrgID {
			defaultIndex = i
			break
		}
	}
	return defaultIndex
}

func handleAuthError(err error, clientID, auth0Domain string) {
	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) {
		slog.Error("Authentication failed",
			"error", err.Error(),
			"client-id", clientID,
			"auth0-domain", auth0Domain,
			"status_code", retrieveErr.Response.StatusCode,
			"error_code", retrieveErr.ErrorCode,
			"error_description", retrieveErr.ErrorDescription,
			"response_body", string(retrieveErr.Body))
	} else {
		slog.Error("Authentication failed",
			"error", err.Error(),
			"client-id", clientID,
			"auth0-domain", auth0Domain)
	}
}

func getOrgIDFromJWT(token string) (string, error) {
	slog.Debug("Extracting organization ID from JWT token")
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	roles, ok := claims["https://snowplowanalytics.com/roles"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("missing Snowplow roles in JWT")
	}

	user, ok := roles["user"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("missing user info in JWT")
	}

	organization, ok := user["organization"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("missing organization info in JWT")
	}

	orgID, ok := organization["id"].(string)
	if !ok {
		return "", fmt.Errorf("missing organization ID in JWT")
	}

	return orgID, nil
}
