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
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/setup"
)

// userToken is a stand-in for the device login's access token, with the
// claims console.GetUserInfo reads.
func userToken() string {
	enc := base64.RawURLEncoding.EncodeToString
	payload := `{"sub":"u1","https://snowplowanalytics.com/roles":{"user":{"email":"dev@example.com","name":"Dev"}}}`
	return enc([]byte(`{"alg":"none"}`)) + "." + enc([]byte(payload)) + ".sig"
}

func Test_ensureAppKey_ReusesWorkingKey(t *testing.T) {
	consoleURL, signalsURL := fakeConsoleAndSignals(t, http.StatusOK, `[{"name":"domain_userid"}]`)
	envPath := writeEnv(t, signalsURL, "secret")
	before, _ := os.ReadFile(envPath)

	s := &setup.Session{AccessToken: userToken(), Org: console.Organization{ID: "org"}}
	opts := SignalsOptions{ConsoleHost: consoleURL, ProjectDir: filepath.Dir(envPath)}
	if err := ensureAppKey(context.Background(), opts, s, signalsURL, envPath); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.ReadFile(envPath); string(after) != string(before) {
		t.Fatalf(".env changed:\n%s", after)
	}
}

func Test_ensureAppKey_ReplacesKeyWithoutSignalsAccess(t *testing.T) {
	consoleURL, signalsURL := fakeConsoleAndSignals(t, http.StatusForbidden, ``)
	envPath := writeEnv(t, signalsURL, "secret")

	// The fake Console has no key creation endpoint, so reaching it fails.
	s := &setup.Session{AccessToken: userToken(), Org: console.Organization{ID: "org"}}
	opts := SignalsOptions{ConsoleHost: consoleURL, ProjectDir: filepath.Dir(envPath)}
	err := ensureAppKey(context.Background(), opts, s, signalsURL, envPath)
	if err == nil || !strings.Contains(err.Error(), "failed to create the Signals app key") {
		t.Fatalf("expected a new key to be created, got %v", err)
	}
}

func Test_ensureCLIKey_KeepsKeyWithoutSignalsAccessUnlessAsked(t *testing.T) {
	created := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/msc/v1/organizations/org/credentials/v3/token":
			_, _ = w.Write([]byte(`{"accessToken":"jwt"}`))
		case r.URL.Path == "/api/msc/v1/organizations/org/credentials/v2/api-keys/kid":
			_, _ = w.Write([]byte(`{"permissions":[{"organizationId":"org","capabilities":[{"resourceType":"DATA-PRODUCTS","action":"VIEW"}]}]}`))
		case r.Method == "POST":
			created = true
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	// Without a terminal to answer in, setup stops instead of replacing the key.
	orig := interactive
	interactive = func() bool { return false }
	t.Cleanup(func() { interactive = orig })

	s := &setup.Session{AccessToken: userToken(), Org: console.Organization{ID: "org"}}
	opts := SignalsOptions{ConsoleHost: srv.URL, SavedOrgID: "org", SavedAPIKeyID: "kid", SavedAPIKey: "secret"}
	err := ensureCLIKey(context.Background(), opts, s)
	if err == nil || !strings.Contains(err.Error(), "can't set up Signals and tracking") {
		t.Fatalf("expected setup to stop, got %v", err)
	}
	if created {
		t.Fatal("no key should be created without asking")
	}
}

func Test_describeCapabilities(t *testing.T) {
	caps := []console.Capability{
		{ResourceType: "SIGNALS", Action: "CREATE"},
		{ResourceType: "SIGNALS", Action: "EDIT"},
		{ResourceType: "DATA-PRODUCTS", Action: "EDIT"},
	}
	if got := describeCapabilities(caps); got != "SIGNALS (CREATE, EDIT), DATA-PRODUCTS (EDIT)" {
		t.Fatalf("got %q", got)
	}
}
