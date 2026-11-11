/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const host = "https://console.snowplowanalytics.com"

func useTempPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nested", "login.json")
	old := Path
	Path = func() string { return path }
	t.Cleanup(func() { Path = old })
	return path
}

func useTokenServer(t *testing.T, status int, body map[string]any) *int {
	t.Helper()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got := r.FormValue("grant_type"); got != "refresh_token" {
			t.Errorf("grant_type got %s want refresh_token", got)
		}
		if got := r.FormValue("refresh_token"); got != "refresh-1" {
			t.Errorf("refresh_token got %s want refresh-1", got)
		}
		if got := r.FormValue("client_id"); got != "client-1" {
			t.Errorf("client_id got %s want client-1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
	old := tokenURL
	tokenURL = func(string) string { return server.URL }
	t.Cleanup(func() { tokenURL = old; server.Close() })
	return &calls
}

func login(expiresIn time.Duration) *Login {
	return &Login{
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		ExpiresAt:    time.Now().Add(expiresIn).UTC().Truncate(time.Second),
		OrgID:        "org-1",
		Host:         host,
		ClientID:     "client-1",
		Auth0Domain:  "id.example.com",
	}
}

func Test_Load_NoFile(t *testing.T) {
	useTempPath(t)

	got, err := Load(host)
	if err != nil || got != nil {
		t.Fatalf("got %v, %v want nil, nil", got, err)
	}
}

func Test_SaveThenLoad(t *testing.T) {
	path := useTempPath(t)
	want := login(time.Hour)

	if err := Save(want); err != nil {
		t.Fatal(err)
	}

	got, err := Load(host + "/")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != *want {
		t.Errorf("got %+v want %+v", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("mode got %o want 600", info.Mode().Perm())
	}
}

func Test_Save_RestrictsExistingFile(t *testing.T) {
	path := useTempPath(t)
	if err := Save(login(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}

	if err := Save(login(time.Hour)); err != nil {
		t.Fatal(err)
	}

	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Errorf("mode got %o want 600", info.Mode().Perm())
	}
}

func Test_Load_OtherHost(t *testing.T) {
	useTempPath(t)
	if err := Save(login(time.Hour)); err != nil {
		t.Fatal(err)
	}

	got, err := Load("https://next.console.snowplowanalytics.com")
	if err != nil || got != nil {
		t.Fatalf("got %v, %v want nil, nil", got, err)
	}
}

func Test_Load_InvalidFile(t *testing.T) {
	path := useTempPath(t)
	if err := Save(login(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(host); err == nil {
		t.Fatal("expected an error for an unparseable login file")
	}
}

func Test_Valid_UsesUnexpiredToken(t *testing.T) {
	useTempPath(t)
	calls := useTokenServer(t, http.StatusOK, nil)
	if err := Save(login(time.Hour)); err != nil {
		t.Fatal(err)
	}

	got, err := Valid(context.Background(), host)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.AccessToken != "access-1" {
		t.Errorf("got %+v want access-1", got)
	}
	if *calls != 0 {
		t.Errorf("token endpoint called %d times, want 0", *calls)
	}
}

func Test_Valid_RefreshesExpiredTokenAndSavesIt(t *testing.T) {
	useTempPath(t)
	calls := useTokenServer(t, http.StatusOK, map[string]any{
		"access_token":  "access-2",
		"refresh_token": "refresh-2",
		"token_type":    "Bearer",
		"expires_in":    3600,
	})
	if err := Save(login(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	got, err := Valid(context.Background(), host)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.AccessToken != "access-2" {
		t.Fatalf("got %+v want access-2", got)
	}
	if *calls != 1 {
		t.Errorf("token endpoint called %d times, want 1", *calls)
	}

	saved, err := Load(host)
	if err != nil {
		t.Fatal(err)
	}
	if saved.AccessToken != "access-2" || saved.RefreshToken != "refresh-2" || saved.OrgID != "org-1" {
		t.Errorf("saved %+v, want the refreshed tokens and the same organization", saved)
	}
	if time.Until(saved.ExpiresAt) < 50*time.Minute {
		t.Errorf("saved expiry %s is not about an hour away", saved.ExpiresAt)
	}
}

func Test_Valid_KeepsRefreshTokenWhenNotRotated(t *testing.T) {
	useTempPath(t)
	useTokenServer(t, http.StatusOK, map[string]any{
		"access_token": "access-2",
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
	if err := Save(login(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	got, err := Valid(context.Background(), host)
	if err != nil {
		t.Fatal(err)
	}
	if got.RefreshToken != "refresh-1" {
		t.Errorf("refresh token got %s want refresh-1", got.RefreshToken)
	}
}

func Test_Valid_ExpiredWithoutRefreshToken(t *testing.T) {
	useTempPath(t)
	expired := login(-time.Hour)
	expired.RefreshToken = ""
	if err := Save(expired); err != nil {
		t.Fatal(err)
	}

	got, err := Valid(context.Background(), host)
	if err != nil || got != nil {
		t.Fatalf("got %v, %v want nil, nil", got, err)
	}
}

func Test_Valid_RefreshRejected(t *testing.T) {
	useTempPath(t)
	useTokenServer(t, http.StatusForbidden, map[string]any{"error": "invalid_grant"})
	if err := Save(login(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	got, err := Valid(context.Background(), host)
	if err != nil || got != nil {
		t.Fatalf("got %v, %v want nil, nil", got, err)
	}
}

func Test_Valid_RefreshServerError(t *testing.T) {
	useTempPath(t)
	useTokenServer(t, http.StatusServiceUnavailable, map[string]any{"error": "temporarily_unavailable"})
	if err := Save(login(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	got, err := Valid(context.Background(), host)
	if err == nil {
		t.Fatalf("an outage must not read as logged out, got %v, nil", got)
	}
}

func Test_Save_LeavesNoTemporaryFile(t *testing.T) {
	path := useTempPath(t)
	if err := Save(login(time.Hour)); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "login.json" {
		t.Errorf("expected only login.json, got %v", entries)
	}
}

func Test_UnknownHomeDirectory(t *testing.T) {
	old := Path
	Path = func() string { return "" }
	t.Cleanup(func() { Path = old })
	t.Chdir(t.TempDir())

	if err := Save(login(time.Hour)); err == nil {
		t.Error("expected saving to fail without a home directory")
	}
	if got, err := Load(host); err != nil || got != nil {
		t.Errorf("got %v, %v want nil, nil", got, err)
	}
	if entries, _ := os.ReadDir("."); len(entries) != 0 {
		t.Errorf("nothing should be written to the working directory, got %v", entries)
	}
}
