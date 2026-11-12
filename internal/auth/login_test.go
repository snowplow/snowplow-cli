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
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

func Test_Valid_RefreshesTokenAboutToExpire(t *testing.T) {
	useTempPath(t)
	calls := useTokenServer(t, http.StatusOK, map[string]any{
		"access_token": "access-2",
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
	if err := Save(login(30 * time.Second)); err != nil {
		t.Fatal(err)
	}

	got, err := Valid(context.Background(), host)
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != "access-2" || *calls != 1 {
		t.Errorf("got %s after %d refreshes, want access-2 after 1", got.AccessToken, *calls)
	}
}

func Test_Valid_RefreshWithoutExpiryIsNotRepeated(t *testing.T) {
	useTempPath(t)
	calls := useTokenServer(t, http.StatusOK, map[string]any{
		"access_token": "access-2",
		"token_type":   "Bearer",
	})
	if err := Save(login(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if _, err := Valid(context.Background(), host); err != nil {
			t.Fatal(err)
		}
	}
	if *calls != 1 {
		t.Errorf("token endpoint called %d times, want 1", *calls)
	}
}

func Test_Valid_ExpiredWithoutClient(t *testing.T) {
	useTempPath(t)
	calls := useTokenServer(t, http.StatusOK, nil)
	expired := login(-time.Hour)
	expired.ClientID = ""
	if err := Save(expired); err != nil {
		t.Fatal(err)
	}

	got, err := Valid(context.Background(), host)
	if err != nil || got != nil {
		t.Fatalf("got %v, %v want nil, nil", got, err)
	}
	if *calls != 0 {
		t.Errorf("token endpoint called %d times, want 0", *calls)
	}
}

func Test_Valid_RefreshRateLimited(t *testing.T) {
	useTempPath(t)
	useTokenServer(t, http.StatusTooManyRequests, map[string]any{"error": "too_many_requests"})
	if err := Save(login(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	if got, err := Valid(context.Background(), host); err == nil {
		t.Fatalf("a rate limit must not read as logged out, got %v, nil", got)
	}
}

func Test_Valid_RefreshGrantNotEnabled(t *testing.T) {
	useTempPath(t)
	useTokenServer(t, http.StatusForbidden, map[string]any{"error": "unauthorized_client"})
	if err := Save(login(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	got, err := Valid(context.Background(), host)
	if err != nil || got != nil {
		t.Fatalf("got %v, %v want nil, nil", got, err)
	}
}

func Test_Valid_LoginWithoutExpiryIsUsed(t *testing.T) {
	useTempPath(t)
	calls := useTokenServer(t, http.StatusOK, nil)
	noExpiry := login(time.Hour)
	noExpiry.ExpiresAt = time.Time{}
	noExpiry.RefreshToken = ""
	if err := Save(noExpiry); err != nil {
		t.Fatal(err)
	}

	got, err := Valid(context.Background(), host)
	if err != nil || got == nil || got.AccessToken != "access-1" {
		t.Fatalf("got %v, %v want access-1", got, err)
	}
	if *calls != 0 {
		t.Errorf("token endpoint called %d times, want 0", *calls)
	}
}

func Test_Save_FileFormatSharedWithSnowtype(t *testing.T) {
	path := useTempPath(t)
	want := login(time.Hour)
	if err := Save(want); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	expected := map[string]any{
		"access_token":  "access-1",
		"refresh_token": "refresh-1",
		"expires_at":    want.ExpiresAt.Format(time.RFC3339),
		"org_id":        "org-1",
		"host":          host,
		"client_id":     "client-1",
		"auth0_domain":  "id.example.com",
	}
	if len(got) != len(expected) {
		t.Errorf("keys got %v want %v", got, expected)
	}
	for key, value := range expected {
		if got[key] != value {
			t.Errorf("%s got %v want %v", key, got[key], value)
		}
	}
}

func Test_Valid_DoesNotReplaceLoginMadeDuringRefresh(t *testing.T) {
	useTempPath(t)
	newer := login(time.Hour)
	newer.AccessToken = "other-account"
	newer.RefreshToken = "other-refresh"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := Save(newer); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access-2", "token_type": "Bearer", "expires_in": 3600})
	}))
	old := tokenURL
	tokenURL = func(string) string { return server.URL }
	t.Cleanup(func() { tokenURL = old; server.Close() })

	if err := Save(login(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := Valid(context.Background(), host); err != nil {
		t.Fatal(err)
	}

	saved, err := Load(host)
	if err != nil {
		t.Fatal(err)
	}
	if saved.AccessToken != "other-account" || saved.RefreshToken != "other-refresh" {
		t.Errorf("the newer login was replaced, got %+v", saved)
	}
}

func Test_Save_WaitsForLock(t *testing.T) {
	path := useTempPath(t)
	if err := Save(login(time.Hour)); err != nil {
		t.Fatal(err)
	}
	lock := path + ".lock"
	if err := os.WriteFile(lock, nil, 0600); err != nil {
		t.Fatal(err)
	}
	released := time.Now().Add(150 * time.Millisecond)
	go func() {
		time.Sleep(time.Until(released))
		_ = os.Remove(lock)
	}()

	later := login(time.Hour)
	later.AccessToken = "after-wait"
	if err := Save(later); err != nil {
		t.Fatal(err)
	}

	if time.Now().Before(released) {
		t.Error("saved while another process held the lock")
	}
	if saved, _ := Load(host); saved == nil || saved.AccessToken != "after-wait" {
		t.Errorf("got %+v want after-wait", saved)
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Error("the lock was left behind")
	}
}

func Test_Save_NeverTakesAnotherProcessLock(t *testing.T) {
	path := useTempPath(t)
	old := lockWait
	lockWait = 50 * time.Millisecond
	t.Cleanup(func() { lockWait = old })

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	lock := path + ".lock"
	if err := os.WriteFile(lock, []byte("held"), 0600); err != nil {
		t.Fatal(err)
	}

	err := Save(login(time.Hour))
	if err == nil || !strings.Contains(err.Error(), lock) {
		t.Errorf("expected an error naming the lock file, got: %v", err)
	}
	if held, _ := os.ReadFile(lock); string(held) != "held" {
		t.Error("the other process's lock was removed or replaced")
	}
	if saved, _ := Load(host); saved != nil {
		t.Error("the login was written without the lock")
	}
}

func Test_Save_WritesExpiryAsUTCMilliseconds(t *testing.T) {
	path := useTempPath(t)
	local := login(time.Hour)
	local.ExpiresAt = time.Date(2030, 1, 2, 3, 4, 5, 123456789, time.FixedZone("CET", 3600))
	if err := Save(local); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := `"expires_at": "2030-01-02T02:04:05.123Z"`; !strings.Contains(string(data), want) {
		t.Errorf("want %s in %s", want, data)
	}
}

func Test_Valid_DoesNotReplaceLoginForAnotherHostMadeDuringRefresh(t *testing.T) {
	useTempPath(t)
	other := login(time.Hour)
	other.AccessToken = "other-host-token"
	other.RefreshToken = "other-host-refresh"
	other.Host = "https://next.console.snowplowanalytics.com"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := Save(other); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access-2", "token_type": "Bearer", "expires_in": 3600})
	}))
	old := tokenURL
	tokenURL = func(string) string { return server.URL }
	t.Cleanup(func() { tokenURL = old; server.Close() })

	if err := Save(login(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, err := Valid(context.Background(), host)
	if err != nil || got == nil || got.AccessToken != "access-2" {
		t.Fatalf("the command in hand should still get the refreshed token, got %v, %v", got, err)
	}

	saved, err := Load(other.Host)
	if err != nil {
		t.Fatal(err)
	}
	if saved == nil || saved.AccessToken != "other-host-token" {
		t.Errorf("the login for the other host was replaced, got %+v", saved)
	}
}

func Test_Valid_ReturnsRefreshedTokenWhenItCannotBeSaved(t *testing.T) {
	path := useTempPath(t)
	oldWait := lockWait
	lockWait = 50 * time.Millisecond
	t.Cleanup(func() { lockWait = oldWait })
	useTokenServer(t, http.StatusOK, map[string]any{
		"access_token": "access-2",
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
	if err := Save(login(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".lock", nil, 0600); err != nil {
		t.Fatal(err)
	}

	got, err := Valid(context.Background(), host)
	if err != nil {
		t.Fatalf("a refresh that worked must not fail the command: %v", err)
	}
	if got == nil || got.AccessToken != "access-2" {
		t.Errorf("got %+v want access-2", got)
	}
}

func useRevokeServer(t *testing.T, status int) *[]url.Values {
	t.Helper()
	var calls []url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		calls = append(calls, r.PostForm)
		w.WriteHeader(status)
	}))
	old := revokeURL
	revokeURL = func(string) string { return server.URL }
	t.Cleanup(func() { revokeURL = old; server.Close() })
	return &calls
}

func Test_Logout_RevokesAndRemovesLogin(t *testing.T) {
	path := useTempPath(t)
	calls := useRevokeServer(t, http.StatusOK)
	if err := Save(login(time.Hour)); err != nil {
		t.Fatal(err)
	}

	result, err := Logout(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !result.LoggedIn || !result.Revoked {
		t.Errorf("got %+v want logged in and revoked", result)
	}
	if len(*calls) != 1 || (*calls)[0].Get("token") != "refresh-1" || (*calls)[0].Get("client_id") != "client-1" {
		t.Errorf("revoke calls got %v want one for refresh-1 and client-1", *calls)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("the login file was not removed")
	}
	if _, err := os.Stat(path + ".lock"); !os.IsNotExist(err) {
		t.Error("the lock was left behind")
	}
	if got, _ := Load(host); got != nil {
		t.Errorf("still logged in: %+v", got)
	}
}

func Test_Logout_NotLoggedIn(t *testing.T) {
	path := useTempPath(t)
	calls := useRevokeServer(t, http.StatusOK)

	result, err := Logout(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.LoggedIn || result.Revoked {
		t.Errorf("got %+v want not logged in", result)
	}
	if len(*calls) != 0 {
		t.Errorf("revoke called %d times, want 0", len(*calls))
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Error("logging out with no login must not create the config directory")
	}
}

func Test_Logout_RemovesLoginWhenRevokeFails(t *testing.T) {
	path := useTempPath(t)
	useRevokeServer(t, http.StatusServiceUnavailable)
	if err := Save(login(time.Hour)); err != nil {
		t.Fatal(err)
	}

	result, err := Logout(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !result.LoggedIn || result.Revoked {
		t.Errorf("got %+v want logged in and not revoked", result)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("the login file was not removed")
	}
}

func Test_Logout_WithoutRefreshTokenSkipsRevoke(t *testing.T) {
	useTempPath(t)
	calls := useRevokeServer(t, http.StatusOK)
	noRefresh := login(time.Hour)
	noRefresh.RefreshToken = ""
	if err := Save(noRefresh); err != nil {
		t.Fatal(err)
	}

	result, err := Logout(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !result.LoggedIn || result.Revoked {
		t.Errorf("got %+v want logged in and not revoked", result)
	}
	if len(*calls) != 0 {
		t.Errorf("revoke called %d times, want 0", len(*calls))
	}
}

func Test_Logout_RemovesUnreadableLogin(t *testing.T) {
	path := useTempPath(t)
	calls := useRevokeServer(t, http.StatusOK)
	if err := Save(login(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0600); err != nil {
		t.Fatal(err)
	}

	result, err := Logout(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !result.LoggedIn || result.Revoked {
		t.Errorf("got %+v want logged in and not revoked", result)
	}
	if len(*calls) != 0 {
		t.Errorf("revoke called %d times, want 0", len(*calls))
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("the unreadable login file was not removed")
	}
}

func Test_Logout_LogsOutEveryHost(t *testing.T) {
	path := useTempPath(t)
	useRevokeServer(t, http.StatusOK)
	other := login(time.Hour)
	other.Host = "https://next.console.snowplowanalytics.com"
	if err := Save(other); err != nil {
		t.Fatal(err)
	}

	result, err := Logout(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !result.LoggedIn || !result.Revoked {
		t.Errorf("got %+v want logged in and revoked", result)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("the login file was not removed")
	}
}

func Test_Logout_FailsWhenLocked(t *testing.T) {
	path := useTempPath(t)
	calls := useRevokeServer(t, http.StatusOK)
	oldWait := lockWait
	lockWait = 50 * time.Millisecond
	t.Cleanup(func() { lockWait = oldWait })
	if err := Save(login(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".lock", nil, 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := Logout(context.Background()); err == nil {
		t.Error("expected an error while another process holds the lock")
	}
	if _, err := os.Stat(path); err != nil {
		t.Error("the login file must stay when the lock cannot be taken")
	}
	if len(*calls) != 0 {
		t.Errorf("revoke called %d times, want 0", len(*calls))
	}
}
