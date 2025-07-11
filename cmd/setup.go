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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/browser"
	"github.com/snowplow/snowplow-cli/internal/config"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/util"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

const snowplowAudience = "https://snowplowanalytics.com/api/"

type UserInfo struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type APIKeyResponse struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

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

		if err := config.InitConsoleConfig(cmd); err != nil {
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

		if clientID == "" {
			return fmt.Errorf("client-id is required. Use --client-id flag")
		}

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
			os.Exit(1)
		}

		yellow := color.New(color.FgYellow, color.Bold)
		green := color.New(color.FgGreen)
		cyan := color.New(color.FgCyan)

		authURL := deviceAuth.VerificationURIComplete
		if authURL == "" {
			authURL = deviceAuth.VerificationURI
		}

		fmt.Printf("- Press Enter to open ")
		cyan.Printf("%s", authURL)
		fmt.Printf(" in your browser...")

		fmt.Scanln()

		browserOpened := false
		if err := browser.OpenURL(authURL); err == nil {
			browserOpened = true
			green.Printf("\n✓ Opened %s in your browser\n", auth0Domain)
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

		slog.Debug("Extracting organization ID from JWT token")
		jwtOrgID, err := getOrgIDFromJWT(token.AccessToken)
		if err != nil {
			return fmt.Errorf("failed to get organization ID: %w", err)
		}

		organizations, err := getOrganizations(ctx, token.AccessToken, consoleHost)
		if err != nil {
			return fmt.Errorf("failed to fetch organizations: %w", err)
		}

		if len(organizations) == 0 {
			return fmt.Errorf("no organizations found for your account")
		}

		var selectedOrg *Organization

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

		slog.Debug("Creating API key for organization", "org-id", selectedOrg.ID)
		apiKey, err := createAPIKey(ctx, token.AccessToken, consoleHost, selectedOrg.ID, readOnly)
		if err != nil {
			return fmt.Errorf("failed to create API key: %w", err)
		}

		userInfo, err := getUserInfo(ctx, token.AccessToken)
		if err != nil {
			return fmt.Errorf("failed to get user info: %w", err)
		}

		slog.Debug("Saving configuration to file", "config-path", getConfigPath())
		if err := saveConfig(selectedOrg.ID, apiKey.ID, apiKey.Secret, consoleHost); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		green.Printf("\n✓ Logged in to Snowplow Console as %s\n", userInfo.Email)
		green.Printf("✓ Authentication credentials saved\n")
		if readOnly {
			green.Printf("✓ API key created with read-only permissions for %s\n", selectedOrg.Name)
		} else {
			green.Printf("✓ API key created with admin permissions for %s\n", selectedOrg.Name)
		}
		green.Printf("✓ Configuration saved to %s\n", getConfigPath())

		return nil
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
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

func init() {
	config.InitConsoleFlags(SetupCmd)

	SetupCmd.Flags().String("client-id", "YOUR_PROD_CLIENT_ID_PLACEHOLDER", "Auth0 Client ID for device auth")
	SetupCmd.Flags().String("auth0-domain", "id.snowplowanalytics.com", "Auth0 domain")
	SetupCmd.Flags().Bool("read-only", false, "Create a read-only API key")
}

func createAPIKey(ctx context.Context, accessToken, consoleHost, orgID string, readOnly bool) (*APIKeyResponse, error) {
	apiURL := fmt.Sprintf("%s/api/msc/v1/organizations/%s/credentials/v2/api-keys", consoleHost, orgID)

	slog.Debug("Creating API key", "url", apiURL, "orgID", orgID)

	userInfo, err := getUserInfo(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	var description string
	if readOnly {
		description = fmt.Sprintf("%s CLI key (read-only)", userInfo.Email)
	} else {
		description = fmt.Sprintf("%s CLI key", userInfo.Email)
	}

	var permissions []map[string]interface{}
	if readOnly {
		permissions = []map[string]interface{}{
			{
				"capabilities": []map[string]interface{}{
					{
						"resourceType": "DATA-PRODUCTS",
						"action":       "LIST",
						"filters":      []interface{}{},
					},
					{
						"resourceType": "DATA-PRODUCTS",
						"action":       "VIEW",
						"filters":      []interface{}{},
					},
				},
			},
		}
	} else {
		permissions = []map[string]interface{}{
			{
				"capabilities": []map[string]interface{}{
					{
						"resourceType": "*",
						"action":       "*",
						"filters":      []interface{}{},
					},
				},
			},
		}
	}

	reqBody := map[string]interface{}{
		"description": description,
		"permissions": permissions,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	slog.Debug("API key request", "body", string(jsonBody))

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-SNOWPLOW-CLI", util.VersionInfo)

	slog.Debug("Request headers",
		"Content-Type", req.Header.Get("Content-Type"),
		"Authorization", "Bearer "+accessToken[:min(20, len(accessToken))]+"...",
		"X-SNOWPLOW-CLI", req.Header.Get("X-SNOWPLOW-CLI"))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	slog.Debug("API response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API key creation failed: %s", string(body))
	}

	var apiResp map[string]interface{}
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	id, ok := apiResp["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response: missing id field")
	}

	key, ok := apiResp["key"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response: missing key field")
	}

	return &APIKeyResponse{
		ID:     id,
		Secret: key,
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	roles, ok := claims["https://snowplowanalytics.com/roles"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing Snowplow roles in JWT")
	}

	user, ok := roles["user"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("missing user info in JWT")
	}

	email, _ := user["email"].(string)
	name, _ := user["name"].(string)
	sub, _ := claims["sub"].(string)

	return &UserInfo{
		Sub:   sub,
		Email: email,
		Name:  name,
	}, nil
}

func getOrganizations(ctx context.Context, accessToken, consoleHost string) ([]Organization, error) {
	apiURL := fmt.Sprintf("%s/api/msc/v1/organizations", consoleHost)

	slog.Debug("Fetching organizations", "url", apiURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-SNOWPLOW-CLI", util.VersionInfo)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	slog.Debug("Organizations API response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch organizations: %s", string(body))
	}

	var organizations []Organization
	if err := json.Unmarshal(body, &organizations); err != nil {
		return nil, fmt.Errorf("failed to parse organizations response: %w", err)
	}

	return organizations, nil
}

func selectOrganization(organizations []Organization, jwtOrgID string) (*Organization, error) {
	cyan := color.New(color.FgCyan)
	yellow := color.New(color.FgYellow, color.Bold)

	fmt.Printf("\n")
	cyan.Printf("You have access to %d organizations:\n\n", len(organizations))

	var defaultIndex int = -1
	for i, org := range organizations {
		if org.ID == jwtOrgID {
			defaultIndex = i
			break
		}
	}

	for i, org := range organizations {
		if i == defaultIndex {
			yellow.Printf("  %d. %s (default)\n", i+1, org.Name)
		} else {
			fmt.Printf("  %d. %s\n", i+1, org.Name)
		}
	}

	fmt.Printf("\n")
	if defaultIndex >= 0 {
		fmt.Printf("Select organization [1-%d] (default: %d): ", len(organizations), defaultIndex+1)
	} else {
		fmt.Printf("Select organization [1-%d]: ", len(organizations))
	}

	var input string
	fmt.Scanln(&input)

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

func saveConfig(orgID, apiKeyID, apiKeySecret, consoleHost string) error {
	configPath := getConfigPath()

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	var existingConfig map[string]interface{}
	if existingData, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(existingData, &existingConfig); err != nil {
			slog.Warn("Failed to parse existing config, creating new one", "error", err)
			existingConfig = make(map[string]interface{})
		}
	} else {
		existingConfig = make(map[string]interface{})
	}

	if existingConfig["console"] == nil {
		existingConfig["console"] = make(map[string]interface{})
	}

	consoleConfig, ok := existingConfig["console"].(map[string]interface{})
	if !ok {
		consoleConfig = make(map[string]interface{})
		existingConfig["console"] = consoleConfig
	}

	consoleConfig["api-key"] = apiKeySecret
	consoleConfig["api-key-id"] = apiKeyID
	consoleConfig["org-id"] = orgID

	// Only save host if it's not the default production value
	if consoleHost != "https://console.snowplowanalytics.com" {
		consoleConfig["host"] = consoleHost
	}

	// Note: We don't save auth0-domain or client-id to config file
	// These should be provided via environment variables or command line flags

	data, err := yaml.Marshal(existingConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "snowplow", "snowplow.yml")
}

func getOrgIDFromJWT(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	roles, ok := claims["https://snowplowanalytics.com/roles"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("missing Snowplow roles in JWT")
	}

	user, ok := roles["user"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("missing user info in JWT")
	}

	organization, ok := user["organization"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("missing organization info in JWT")
	}

	orgID, ok := organization["id"].(string)
	if !ok {
		return "", fmt.Errorf("missing organization ID in JWT")
	}

	return orgID, nil
}
