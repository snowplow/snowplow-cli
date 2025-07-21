/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package console

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/snowplow/snowplow-cli/internal/util"
)

type APIKeyResponse struct {
	ID     string `json:"id"`
	Secret string `json:"key"`
}

type UserInfo struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func CreateAPIKey(ctx context.Context, accessToken, consoleHost, orgID string, readOnly bool) (*APIKeyResponse, error) {
	slog.Debug("Creating API key for organization", "org-id", orgID)
	apiURL := fmt.Sprintf("%s/api/msc/v1/organizations/%s/credentials/v2/api-keys", consoleHost, orgID)

	slog.Debug("Creating API key", "url", apiURL, "orgID", orgID)

	userInfo, err := GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	var description string
	if readOnly {
		description = fmt.Sprintf("%s CLI key (read-only)", userInfo.Email)
	} else {
		description = fmt.Sprintf("%s CLI key", userInfo.Email)
	}

	var permissions []map[string]any
	if readOnly {
		permissions = []map[string]any{
			{
				"capabilities": []map[string]any{
					{
						"resourceType": "DATA-PRODUCTS",
						"action":       "LIST",
						"filters":      []any{},
					},
					{
						"resourceType": "DATA-PRODUCTS",
						"action":       "VIEW",
						"filters":      []any{},
					},
				},
			},
		}
	} else {
		permissions = []map[string]any{
			{
				"capabilities": []map[string]any{
					{
						"resourceType": "*",
						"action":       "*",
						"filters":      []any{},
					},
				},
			},
		}
	}

	reqBody := map[string]any{
		"description": description,
		"permissions": permissions,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	slog.Debug("API key request", "body", string(jsonBody))

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jsonBody))
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
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	slog.Debug("API response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API key creation failed: %s", string(body))
	}

	var apiKeyResponse APIKeyResponse
	if err := json.Unmarshal(body, &apiKeyResponse); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}
	return &apiKeyResponse, nil
}

func GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	parts := strings.Split(accessToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	roles, ok := claims["https://snowplowanalytics.com/roles"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing Snowplow roles in JWT")
	}

	user, ok := roles["user"].(map[string]any)
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
