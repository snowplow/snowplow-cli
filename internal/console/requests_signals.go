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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/snowplow/snowplow-cli/internal/util"
)

type SignalsEnvironment struct {
	Status   string `json:"status"`
	Pipeline struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"pipeline"`
	Config *struct {
		PersonalizationApiHost string `json:"personalizationApiHost"`
	} `json:"config"`
}

func (e SignalsEnvironment) APIURL() string {
	if e.Config == nil || e.Config.PersonalizationApiHost == "" {
		return ""
	}
	return "https://" + e.Config.PersonalizationApiHost
}

var SignalsReadCapabilities = []Capability{
	{ResourceType: "SIGNALS", Action: "LIST", Filters: []any{}},
	{ResourceType: "SIGNALS", Action: "VIEW", Filters: []any{}},
}

// userClient makes Console requests with the token from the device login,
// for the calls made before snowplow-cli has an API key of its own.
func userClient(consoleHost, orgID, accessToken string) *ApiClient {
	return &ApiClient{
		Http:    &http.Client{Transport: &loggingRoundTripper{Transport: http.DefaultTransport}, Timeout: 30 * time.Second},
		Jwt:     accessToken,
		BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/%s", consoleHost, orgID),
		OrgId:   orgID,
	}
}

func GetSignalsEnvironments(ctx context.Context, accessToken, consoleHost, orgID string) ([]SignalsEnvironment, error) {
	client := userClient(consoleHost, orgID, accessToken)
	url := fmt.Sprintf("%s/signals/v1", client.BaseUrl)
	slog.Debug("Fetching Signals environments", "url", url)

	resp, err := DoConsoleRequest("GET", url, client, ctx, nil)
	if err != nil {
		return nil, err
	}
	defer util.LoggingCloser(ctx, resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch Signals environments: %d %s", resp.StatusCode, string(body))
	}

	var envs []SignalsEnvironment
	if err := json.Unmarshal(body, &envs); err != nil {
		return nil, fmt.Errorf("failed to parse Signals environments: %w", err)
	}
	return envs, nil
}

// SignalsSetupCapabilities covers the Snowplow MCP tools the signals-setup
// skill uses. Update it when the skill's tools change.
var SignalsSetupCapabilities = []Capability{
	// Every signals_* tool, and list_signals_environments.
	{ResourceType: "SIGNALS", Action: "LIST", Filters: []any{}},
	{ResourceType: "SIGNALS", Action: "VIEW", Filters: []any{}},
	{ResourceType: "SIGNALS", Action: "EDIT", Filters: []any{}},
	// list_pipelines, get_pipeline_metrics, get_collector_config.
	{ResourceType: "ENVIRONMENTS", Action: "LIST", Filters: []any{}},
	{ResourceType: "ENVIRONMENTS", Action: "VIEW", Filters: []any{}},
	// Reading, creating and editing tracking plans, event specs and source
	// apps. Publishing an event spec is an edit.
	{ResourceType: "DATA-PRODUCTS", Action: "LIST", Filters: []any{}},
	{ResourceType: "DATA-PRODUCTS", Action: "VIEW", Filters: []any{}},
	{ResourceType: "DATA-PRODUCTS", Action: "CREATE", Filters: []any{}},
	{ResourceType: "DATA-PRODUCTS", Action: "EDIT", Filters: []any{}},
	// Schemas for custom events: CREATE deploys a new one to dev, EDIT a new
	// version, and PUBLISH to prod, where Signals reads events from.
	{ResourceType: "DATA-STRUCTURES", Action: "LIST", Filters: []any{}},
	{ResourceType: "DATA-STRUCTURES", Action: "VIEW", Filters: []any{}},
	{ResourceType: "DATA-STRUCTURES", Action: "CREATE", Filters: []any{}},
	{ResourceType: "DATA-STRUCTURES", Action: "EDIT", Filters: []any{}},
	{ResourceType: "DATA-STRUCTURES", Action: "PUBLISH", Filters: []any{}},
}
