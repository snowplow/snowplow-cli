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
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/snowplow/snowplow-cli/internal/config"
	"github.com/snowplow/snowplow-cli/internal/console"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/spf13/cobra"
)

var StatusCmd = &cobra.Command{
	Use:          "status",
	Short:        "Check Snowplow CLI configuration and connectivity",
	Long:         `Verify that the CLI is properly configured and can connect to Snowplow Console`,
	Example:      `  $ snowplow-cli status`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := snplog.InitLogging(cmd); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := config.InitConsoleConfig(cmd); err != nil {
			slog.Info("Status check failed: configuration error",
				"error", err.Error(),
				"status", "not_configured",
				"action", "run 'snowplow-cli setup' to configure authentication")
			os.Exit(1)
		}

		apiKey, _ := cmd.Flags().GetString("api-key")
		apiKeyID, _ := cmd.Flags().GetString("api-key-id")
		orgID, _ := cmd.Flags().GetString("org-id")
		host, _ := cmd.Flags().GetString("host")

		slog.Info("API credentials configured",
			"api_key_id", apiKeyID,
			"org_id", orgID,
			"host", host)

		client, err := console.NewApiClient(ctx, host, apiKeyID, apiKey, orgID)
		if err != nil {
			slog.Info("Status check failed: API connectivity error",
				"error", err.Error(),
				"host", host,
				"org_id", orgID,
				"status", "api_error",
				"action", "check network connectivity or run 'snowplow-cli setup' to refresh credentials")
			os.Exit(1)
		}

		organizations, err := getStatusOrganizations(ctx, client)
		if err != nil {
			slog.Info("Status check failed: API connectivity error",
				"error", err.Error(),
				"host", host,
				"org_id", orgID,
				"status", "api_error",
				"action", "check network connectivity or run 'snowplow-cli setup' to refresh credentials")
			os.Exit(1)
		}

		var selectedOrg *console.Organization
		for _, org := range organizations {
			if org.ID == orgID {
				selectedOrg = &org
				break
			}
		}

		if selectedOrg == nil {
			slog.Info("Status check failed: organization not found",
				"org_id", orgID,
				"status", "org_not_found",
				"action", "run 'snowplow-cli setup' to reconfigure with a valid organization")
			os.Exit(1)
		}

		slog.Info("Status check passed",
			"status", "healthy",
			"host", host,
			"org_id", selectedOrg.ID,
			"org_name", selectedOrg.Name,
			"api_key_id", apiKeyID)

		return nil
	},
}

func getStatusOrganizations(ctx context.Context, client *console.ApiClient) ([]console.Organization, error) {
	baseAPIURL := client.BaseUrl[:strings.LastIndex(client.BaseUrl, "/organizations/")] + "/organizations"

	resp, err := console.DoConsoleRequest("GET", baseAPIURL, client, ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = resp.Body.Close()
	}()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var organizations []console.Organization
	if err := json.NewDecoder(resp.Body).Decode(&organizations); err != nil {
		return nil, fmt.Errorf("failed to parse organizations response: %w", err)
	}

	return organizations, nil
}

func init() {
	config.InitConsoleFlags(StatusCmd)
}
