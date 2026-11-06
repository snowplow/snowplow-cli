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

// The app's key can also edit Signals, so the app can set up attributes through a Signals SDK.
var SignalsAppCapabilities = []Capability{
	{ResourceType: "SIGNALS", Action: "LIST", Filters: []any{}},
	{ResourceType: "SIGNALS", Action: "VIEW", Filters: []any{}},
	{ResourceType: "SIGNALS", Action: "EDIT", Filters: []any{}},
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
