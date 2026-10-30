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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/snowplow/snowplow-cli/internal/console"
)

// The app's Signals credentials in the project .env. They deliberately don't
// share snowplow-cli's SNOWPLOW_CONSOLE_* names, so neither picks up the other's key.
const (
	EnvSignalsAPIURL   = "SNOWPLOW_SIGNALS_API_URL"
	EnvSignalsOrgID    = "SNOWPLOW_SIGNALS_ORG_ID"
	EnvSignalsAPIKeyID = "SNOWPLOW_SIGNALS_API_KEY_ID"
	EnvSignalsAPIKey   = "SNOWPLOW_SIGNALS_API_KEY"
)

type SignalsCredentials struct {
	APIURL, OrgID, APIKeyID, APIKey string
}

func ReadSignalsCredentials(envPath string) (SignalsCredentials, error) {
	env, err := godotenv.Read(envPath)
	if err != nil {
		return SignalsCredentials{}, err
	}
	c := SignalsCredentials{
		APIURL:   env[EnvSignalsAPIURL],
		OrgID:    env[EnvSignalsOrgID],
		APIKeyID: env[EnvSignalsAPIKeyID],
		APIKey:   env[EnvSignalsAPIKey],
	}
	var missing []string
	for _, name := range []string{EnvSignalsAPIURL, EnvSignalsOrgID, EnvSignalsAPIKeyID, EnvSignalsAPIKey} {
		if env[name] == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return c, fmt.Errorf("missing %s in %s", strings.Join(missing, ", "), envPath)
	}
	return c, nil
}

type VerifyResult struct {
	OK            bool
	APIURL        string
	AttributeKeys int
	// Reason is one of: config, auth, forbidden, network, empty.
	Reason string
	Error  string
}

// VerifySignals checks the app's credentials the way the app will use them:
// exchange the key for a token, then list attribute keys on the Signals API.
// The registry always has attribute keys, so an empty list means something is wrong.
func VerifySignals(ctx context.Context, consoleHost, envPath string) VerifyResult {
	creds, err := ReadSignalsCredentials(envPath)
	if err != nil {
		return VerifyResult{Reason: "config", Error: err.Error()}
	}
	res := VerifyResult{APIURL: creds.APIURL}

	client, err := console.NewApiClient(ctx, consoleHost, creds.APIKeyID, creds.APIKey, creds.OrgID)
	if err != nil {
		// A transport error means Console wasn't reached, so the key wasn't judged.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			res.Reason, res.Error = "network", fmt.Sprintf("couldn't reach Console at %s: %v", consoleHost, err)
		} else {
			res.Reason, res.Error = "auth", fmt.Sprintf("the API key in %s was rejected by Console: %v", envPath, err)
		}
		return res
	}

	n, status, err := countAttributeKeys(ctx, creds.APIURL, client.Jwt)
	switch {
	case err != nil:
		res.Reason, res.Error = "network", err.Error()
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		res.Reason, res.Error = "forbidden", fmt.Sprintf("the Signals API refused the key (%d); it needs the SIGNALS VIEW permission", status)
	case status != http.StatusOK:
		res.Reason, res.Error = "network", fmt.Sprintf("the Signals API returned %d", status)
	case n == 0:
		res.Reason, res.Error = "empty", "the Signals API returned no attribute keys"
	default:
		res.OK = true
	}
	res.AttributeKeys = n
	return res
}

func countAttributeKeys(ctx context.Context, apiURL, jwt string) (int, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", strings.TrimSuffix(apiURL, "/")+"/api/v1/registry/attribute_keys/", nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 0, resp.StatusCode, nil
	}
	var keys []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&keys); err != nil {
		return 0, resp.StatusCode, fmt.Errorf("unexpected response from the Signals API: %w", err)
	}
	return len(keys), resp.StatusCode, nil
}
