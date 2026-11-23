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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
)

const apiKeyDocsURL = "https://docs.snowplow.io/docs/using-the-snowplow-console/managing-console-api-authentication/"

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

var cliCapabilities = slices.Concat(
	actionsOn("DATA-PRODUCTS", "LIST", "VIEW", "CREATE", "EDIT"),
	actionsOn("DATA-STRUCTURES", "LIST", "VIEW", "CREATE", "EDIT", "PUBLISH"),
	actionsOn("ENVIRONMENTS", "LIST"),
)

var cliReadOnlyCapabilities = slices.Concat(
	actionsOn("DATA-PRODUCTS", "LIST", "VIEW"),
	actionsOn("DATA-STRUCTURES", "LIST", "VIEW"),
	actionsOn("ENVIRONMENTS", "LIST"),
)

func actionsOn(resourceType string, actions ...string) []Capability {
	result := make([]Capability, len(actions))
	for i, action := range actions {
		result[i] = Capability{ResourceType: resourceType, Action: action, Filters: []any{}}
	}
	return result
}

func describeCapabilities(capabilities []Capability) string {
	var resourceTypes []string
	actions := map[string][]string{}
	for _, c := range capabilities {
		if _, seen := actions[c.ResourceType]; !seen {
			resourceTypes = append(resourceTypes, c.ResourceType)
		}
		actions[c.ResourceType] = append(actions[c.ResourceType], c.Action)
	}
	parts := make([]string, len(resourceTypes))
	for i, resourceType := range resourceTypes {
		parts[i] = fmt.Sprintf("%s (%s)", resourceType, strings.Join(actions[resourceType], ", "))
	}
	return strings.Join(parts, ", ")
}

func CreateAPIKey(ctx context.Context, accessToken, consoleHost, orgID string, readOnly bool) (*APIKeyResponse, error) {
	slog.Debug("Creating API key for organization", "org-id", orgID)

	userInfo, err := GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	if readOnly {
		return CreateScopedAPIKey(ctx, accessToken, consoleHost, orgID, userInfo.Email+" CLI key (read-only)", cliReadOnlyCapabilities)
	}
	return CreateScopedAPIKey(ctx, accessToken, consoleHost, orgID, userInfo.Email+" CLI key", cliCapabilities)
}

func CreateScopedAPIKey(ctx context.Context, accessToken, consoleHost, orgID, description string, capabilities []Capability) (*APIKeyResponse, error) {
	return postAPIKey(ctx, userClient(consoleHost, orgID, accessToken), description, capabilities)
}

func postAPIKey(ctx context.Context, client *ApiClient, description string, capabilities []Capability) (*APIKeyResponse, error) {
	apiURL := client.BaseUrl + "/credentials/v3/api-keys"
	slog.Debug("Creating API key", "url", apiURL)

	jsonBody, err := json.Marshal(map[string]any{
		"description": description,
		"permissions": []map[string]any{{"capabilities": capabilities}},
	})
	if err != nil {
		return nil, err
	}

	req, err := ConsoleRequest("POST", apiURL, client, ctx, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Http.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeBody(ctx, resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	slog.Debug("API response", "status", resp.StatusCode)

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("you are not allowed to create API keys, or you don't have all the permissions the key needs: %s. Ask an organization admin for them, create an API key with the permissions you have (%s), or run 'snowplow-cli login' to work without an API key",
			describeCapabilities(capabilities), apiKeyDocsURL)
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API key creation failed: %d %s", resp.StatusCode, string(body))
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
		return nil, errors.New("invalid JWT token format")
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
		return nil, errors.New("missing Snowplow roles in JWT")
	}

	user, ok := roles["user"].(map[string]any)
	if !ok {
		return nil, errors.New("missing user info in JWT")
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
