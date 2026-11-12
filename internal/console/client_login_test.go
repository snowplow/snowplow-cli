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
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/snowplow/snowplow-cli/internal/auth"
)

func useLoginFile(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "login.json")
	old := auth.Path
	auth.Path = func() string { return path }
	t.Cleanup(func() { auth.Path = old })
}

func Test_NewClient_UsesSavedLoginWithoutApiKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("no request expected, the saved token is used as is, got: %s", r.URL.Path)
	}))
	defer server.Close()

	useLoginFile(t)
	err := auth.Save(&auth.Login{
		AccessToken: "login-token",
		ExpiresAt:   time.Now().Add(time.Hour),
		OrgID:       "orgid",
		Host:        server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	client, err := NewApiClient(context.Background(), server.URL, "", "", "orgid")
	if err != nil {
		t.Fatal(err)
	}
	if client.Jwt != "login-token" {
		t.Errorf("jwt got %s want login-token", client.Jwt)
	}
	if want := server.URL + "/api/msc/v1/organizations/orgid"; client.BaseUrl != want {
		t.Errorf("base url got %s want %s", client.BaseUrl, want)
	}
}

func Test_NewClient_NotLoggedIn(t *testing.T) {
	useLoginFile(t)

	_, err := NewApiClient(context.Background(), "https://console.example.com", "", "", "orgid")
	if err == nil || !strings.Contains(err.Error(), "snowplow-cli login") {
		t.Errorf("expected an error pointing at 'snowplow-cli login', got: %v", err)
	}
}

func Test_NewClient_ExpiredLoginWithoutRefresh(t *testing.T) {
	useLoginFile(t)
	err := auth.Save(&auth.Login{
		AccessToken: "login-token",
		ExpiresAt:   time.Now().Add(-time.Hour),
		OrgID:       "orgid",
		Host:        "https://console.example.com",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewApiClient(context.Background(), "https://console.example.com", "", "", "orgid")
	if err == nil || !strings.Contains(err.Error(), "snowplow-cli login") {
		t.Errorf("expected an error pointing at 'snowplow-cli login', got: %v", err)
	}
}
