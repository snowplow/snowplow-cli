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
)

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func GetOrganizations(ctx context.Context, accessToken, consoleHost string) ([]Organization, error) {

	apiURL := fmt.Sprintf("%s/api/msc/v1/organizations", consoleHost)

	slog.Debug("Fetching organizations", "url", apiURL)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	addStandardHeadersWithJwt(req, ctx, accessToken)

	client := &http.Client{
		Transport: &loggingRoundTripper{
			Transport: http.DefaultTransport,
		},
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

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
