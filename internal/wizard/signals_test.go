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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/setup"
)

func Test_ensureAppKey_ReusesWorkingKey(t *testing.T) {
	consoleURL, signalsURL := fakeConsoleAndSignals(t, http.StatusOK, `[{"name":"domain_userid"}]`)
	envPath := writeEnv(t, signalsURL, "secret")
	before, _ := os.ReadFile(envPath)

	s := &setup.Session{AccessToken: "token", Org: console.Organization{ID: "org"}}
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
	s := &setup.Session{AccessToken: "token", Org: console.Organization{ID: "org"}}
	opts := SignalsOptions{ConsoleHost: consoleURL, ProjectDir: filepath.Dir(envPath)}
	err := ensureAppKey(context.Background(), opts, s, signalsURL, envPath)
	if err == nil || !strings.Contains(err.Error(), "failed to create the Signals app key") {
		t.Fatalf("expected a new key to be created, got %v", err)
	}
}
