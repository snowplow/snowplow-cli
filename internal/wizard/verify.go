/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package wizard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/snowplow/snowplow-cli/internal/console"
)

// The app's Signals credentials in the project .env. They deliberately don't
// share snowplow-cli's SNOWPLOW_CONSOLE_* names, so neither picks up the other's key.
const (
	envSignalsAPIURL   = "SNOWPLOW_SIGNALS_API_URL"
	envSignalsOrgID    = "SNOWPLOW_SIGNALS_ORG_ID"
	envSignalsAPIKeyID = "SNOWPLOW_SIGNALS_API_KEY_ID"
	envSignalsAPIKey   = "SNOWPLOW_SIGNALS_API_KEY"
)

type signalsCredentials struct {
	APIURL, OrgID, APIKeyID, APIKey string
}

func readSignalsCredentials(envPath string) (signalsCredentials, error) {
	env, err := godotenv.Read(envPath)
	if err != nil {
		return signalsCredentials{}, err
	}
	var missing []string
	for _, name := range []string{envSignalsAPIURL, envSignalsOrgID, envSignalsAPIKeyID, envSignalsAPIKey} {
		if env[name] == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return signalsCredentials{}, fmt.Errorf("missing %s in %s", strings.Join(missing, ", "), envPath)
	}
	return signalsCredentials{
		APIURL:   env[envSignalsAPIURL],
		OrgID:    env[envSignalsOrgID],
		APIKeyID: env[envSignalsAPIKeyID],
		APIKey:   env[envSignalsAPIKey],
	}, nil
}

// VerifyReason says why the check failed. Agents read it from --json-output.
type VerifyReason string

const (
	ReasonConfig  VerifyReason = "config"  // values missing from .env
	ReasonAuth    VerifyReason = "auth"    // Console or Signals rejected the key
	ReasonNetwork VerifyReason = "network" // Console or the Signals API couldn't be checked
)

type VerifyResult struct {
	OK     bool
	APIURL string
	Reason VerifyReason
	Error  string
}

func VerifySignals(ctx context.Context, consoleHost, envPath string) VerifyResult {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	creds, err := readSignalsCredentials(envPath)
	if err != nil {
		return VerifyResult{Reason: ReasonConfig, Error: err.Error()}
	}
	res := VerifyResult{APIURL: creds.APIURL}

	client, err := console.NewApiClient(ctx, consoleHost, creds.APIKeyID, creds.APIKey, creds.OrgID)
	if err != nil {
		var tokenErr *console.TokenError
		if errors.As(err, &tokenErr) && tokenErr.Rejected() {
			res.Reason, res.Error = ReasonAuth, fmt.Sprintf("the API key in %s was rejected by Console: %v", envPath, err)
		} else {
			res.Reason, res.Error = ReasonNetwork, fmt.Sprintf("couldn't check the key with Console at %s: %v", consoleHost, err)
		}
		return res
	}

	status, err := signalsStatus(ctx, client, creds.APIURL)
	switch {
	case err != nil:
		res.Reason, res.Error = ReasonNetwork, err.Error()
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		res.Reason, res.Error = ReasonAuth, fmt.Sprintf("the Signals API rejected the key in %s (%d)", envPath, status)
	case status != http.StatusOK:
		res.Reason, res.Error = ReasonNetwork, fmt.Sprintf("the Signals API returned %d", status)
	default:
		res.OK = true
	}
	return res
}

// signalsStatus makes an authenticated call to check the key validity.
// Using /attribute_keys for simplicity (health is public)
// The Console client's HTTP client logs the request under --debug.
func signalsStatus(ctx context.Context, client *console.ApiClient, apiURL string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", strings.TrimSuffix(apiURL, "/")+"/api/v1/registry/attribute_keys/", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+client.Jwt)
	resp, err := client.Http.Do(req)
	if err != nil {
		return 0, err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode, nil
}
