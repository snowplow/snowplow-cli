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
	"log/slog"
	"net/http"
	"strings"
)

type APIKeyResponse struct {
	ID     string `json:"id"`
	Secret string `json:"key"`
}

type Capability struct {
	ResourceType string `json:"resourceType"`
	Action       string `json:"action"`
	Filters      []any  `json:"filters"`
}

type UserInfo struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func CreateAPIKey(ctx context.Context, accessToken, consoleHost, orgID string, readOnly bool) (*APIKeyResponse, error) {
	slog.Debug("Creating API key for organization", "org-id", orgID)

	userInfo, err := GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	description := fmt.Sprintf("%s CLI key", userInfo.Email)
	capabilities := []Capability{{ResourceType: "*", Action: "*", Filters: []any{}}}
	if readOnly {
		description = fmt.Sprintf("%s CLI key (read-only)", userInfo.Email)
		capabilities = []Capability{
			{ResourceType: "DATA-PRODUCTS", Action: "LIST", Filters: []any{}},
			{ResourceType: "DATA-PRODUCTS", Action: "VIEW", Filters: []any{}},
		}
	}

	return postAPIKey(ctx, accessToken, fmt.Sprintf("%s/api/msc/v1/organizations/%s/credentials/v2/api-keys", consoleHost, orgID), description, capabilities)
}

func CreateScopedAPIKey(ctx context.Context, accessToken, consoleHost, orgID, description string, capabilities []Capability) (*APIKeyResponse, error) {
	return postAPIKey(ctx, accessToken, fmt.Sprintf("%s/api/msc/v1/organizations/%s/credentials/v3/api-keys", consoleHost, orgID), description, capabilities)
}

func postAPIKey(ctx context.Context, accessToken, apiURL, description string, capabilities []Capability) (*APIKeyResponse, error) {
	slog.Debug("Creating API key", "url", apiURL)

	jsonBody, err := json.Marshal(map[string]any{
		"description": description,
		"permissions": []map[string]any{{"capabilities": capabilities}},
	})
	if err != nil {
		return nil, err
	}

	body, status, err := userTokenRequest(ctx, "POST", apiURL, accessToken, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	slog.Debug("API response", "status", status, "body", string(body))

	if status != http.StatusCreated {
		return nil, fmt.Errorf("API key creation failed: %d %s", status, string(body))
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
