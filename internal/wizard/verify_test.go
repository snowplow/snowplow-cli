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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fakeConsoleAndSignals(t *testing.T, signalsStatus int, signalsBody string) (consoleURL, signalsURL string) {
	t.Helper()
	consoleSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/msc/v1/organizations/org/credentials/v3/token" || r.Header.Get("X-API-KEY") != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"accessToken":"jwt"}`))
	}))
	signalsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/registry/attribute_keys/" || r.Header.Get("Authorization") != "Bearer jwt" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(signalsStatus)
		_, _ = w.Write([]byte(signalsBody))
	}))
	t.Cleanup(consoleSrv.Close)
	t.Cleanup(signalsSrv.Close)
	return consoleSrv.URL, signalsSrv.URL
}

func writeEnv(t *testing.T, apiURL, key string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	err := UpsertDotenv(path, map[string]string{
		EnvSignalsAPIURL: apiURL, EnvSignalsOrgID: "org", EnvSignalsAPIKeyID: "kid", EnvSignalsAPIKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestVerifySignals(t *testing.T) {
	consoleURL, signalsURL := fakeConsoleAndSignals(t, http.StatusOK, `[{"name":"domain_userid"},{"name":"domain_sessionid"}]`)

	res := VerifySignals(context.Background(), consoleURL, writeEnv(t, signalsURL, "secret"))
	if !res.OK || res.AttributeKeys != 2 || res.APIURL != signalsURL {
		t.Fatalf("unexpected result %+v", res)
	}

	res = VerifySignals(context.Background(), consoleURL, writeEnv(t, signalsURL, "wrong"))
	if res.OK || res.Reason != "auth" {
		t.Fatalf("expected an auth failure, got %+v", res)
	}
}

func TestVerifySignalsFailures(t *testing.T) {
	consoleURL, signalsURL := fakeConsoleAndSignals(t, http.StatusForbidden, ``)
	if res := VerifySignals(context.Background(), consoleURL, writeEnv(t, signalsURL, "secret")); res.Reason != "forbidden" {
		t.Fatalf("expected forbidden, got %+v", res)
	}

	consoleURL, signalsURL = fakeConsoleAndSignals(t, http.StatusOK, `[]`)
	if res := VerifySignals(context.Background(), consoleURL, writeEnv(t, signalsURL, "secret")); res.Reason != "empty" {
		t.Fatalf("expected empty, got %+v", res)
	}

	path := filepath.Join(t.TempDir(), ".env")
	_ = os.WriteFile(path, []byte("SNOWPLOW_SIGNALS_API_URL=x\n"), 0600)
	res := VerifySignals(context.Background(), consoleURL, path)
	if res.Reason != "config" {
		t.Fatalf("expected a config failure, got %+v", res)
	}
	if !strings.Contains(res.Error, "missing SNOWPLOW_SIGNALS_ORG_ID, SNOWPLOW_SIGNALS_API_KEY_ID, SNOWPLOW_SIGNALS_API_KEY in") {
		t.Fatalf("expected the missing names in a fixed order, got %q", res.Error)
	}

	// Console unreachable: the key wasn't checked, so it isn't an auth failure.
	down := httptest.NewServer(http.NotFoundHandler())
	down.Close()
	if res := VerifySignals(context.Background(), down.URL, writeEnv(t, signalsURL, "secret")); res.Reason != "network" {
		t.Fatalf("expected a network failure, got %+v", res)
	}
}
