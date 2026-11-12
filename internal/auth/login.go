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
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type Login struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	OrgID        string    `json:"org_id"`
	Host         string    `json:"host"`
	ClientID     string    `json:"client_id"`
	Auth0Domain  string    `json:"auth0_domain"`
}

const expiryMargin = time.Minute

const defaultLifetime = time.Hour

var refreshMu sync.Mutex

const refreshTimeout = 30 * time.Second

var lockWait = 5 * time.Second

var Path = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "snowplow", "login.json")
}

var tokenURL = func(auth0Domain string) string {
	return fmt.Sprintf("https://%s/oauth/token", auth0Domain)
}

var revokeURL = func(auth0Domain string) string {
	return fmt.Sprintf("https://%s/oauth/revoke", auth0Domain)
}

func Save(login *Login) error {
	return withLock(func(path string) error { return write(login, path) })
}

func withLock(fn func(path string) error) error {
	path := Path()
	if path == "" {
		return errors.New("failed to write login file: home directory not found")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	lock := path + ".lock"
	deadline := time.Now().Add(lockWait)
	for {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			_ = f.Close()
			break
		}
		if !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("failed to lock login file: %w", err)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("failed to lock login file: %s exists. If no other snowplow-cli or snowtype command is running, delete it and try again", lock)
		}
		time.Sleep(25 * time.Millisecond)
	}
	defer func() { _ = os.Remove(lock) }()

	return fn(path)
}

func write(login *Login, path string) error {
	saved := *login
	if !saved.ExpiresAt.IsZero() {
		saved.ExpiresAt = saved.ExpiresAt.UTC().Truncate(time.Millisecond)
	}
	data, err := json.MarshalIndent(&saved, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".login-*.json")
	if err != nil {
		return fmt.Errorf("failed to write login file: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write login file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write login file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to write login file: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("failed to write login file: %w", err)
	}
	return nil
}

func Load(host string) (*Login, error) {
	login, err := loadAny()
	if err != nil || login == nil {
		return nil, err
	}
	if login.AccessToken == "" || strings.TrimRight(login.Host, "/") != strings.TrimRight(host, "/") {
		slog.Debug("saved login ignored: it has no token or is for another host", "file", Path(), "login-host", login.Host, "host", host)
		return nil, nil
	}
	return login, nil
}

func loadAny() (*Login, error) {
	data, err := os.ReadFile(Path())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var login Login
	if err := json.Unmarshal(data, &login); err != nil {
		return nil, fmt.Errorf("failed to parse login file %s: %w", Path(), err)
	}
	return &login, nil
}

func Valid(ctx context.Context, host string) (*Login, error) {
	refreshMu.Lock()
	defer refreshMu.Unlock()

	login, err := Load(host)
	if err != nil || login == nil {
		return nil, err
	}
	if login.ExpiresAt.IsZero() || time.Until(login.ExpiresAt) > expiryMargin {
		return login, nil
	}
	if login.RefreshToken == "" || login.ClientID == "" || login.Auth0Domain == "" {
		return nil, nil
	}

	conf := &oauth2.Config{
		ClientID: login.ClientID,
		Endpoint: oauth2.Endpoint{TokenURL: tokenURL(login.Auth0Domain), AuthStyle: oauth2.AuthStyleInParams},
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Timeout: refreshTimeout})
	token, err := conf.TokenSource(ctx, &oauth2.Token{RefreshToken: login.RefreshToken}).Token()
	if err != nil {
		var retrieveErr *oauth2.RetrieveError
		if errors.As(err, &retrieveErr) && rejected(retrieveErr) {
			slog.Debug("saved login rejected at refresh, logging in again is needed",
				"error-code", retrieveErr.ErrorCode, "error-description", retrieveErr.ErrorDescription)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to refresh login: %w", err)
	}

	usedRefreshToken := login.RefreshToken
	login.AccessToken = token.AccessToken
	login.ExpiresAt = token.Expiry
	if token.Expiry.IsZero() {
		login.ExpiresAt = time.Now().Add(defaultLifetime)
	}
	if token.RefreshToken != "" {
		login.RefreshToken = token.RefreshToken
	}
	err = withLock(func(path string) error {
		current, err := loadAny()
		if err != nil || current == nil || current.RefreshToken != usedRefreshToken {
			return nil
		}
		return write(login, path)
	})
	if err != nil {
		slog.Warn("Refreshed login could not be saved, so the next command may need 'snowplow-cli login'", "error", err)
	}
	return login, nil
}

func rejected(err *oauth2.RetrieveError) bool {
	if err.Response == nil {
		return false
	}
	status := err.Response.StatusCode
	return status >= 400 && status < 500 && status != http.StatusTooManyRequests
}

type LogoutResult struct {
	LoggedIn bool
	Revoked  bool
}

func Logout(ctx context.Context) (LogoutResult, error) {
	path := Path()
	if path == "" {
		return LogoutResult{}, nil
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return LogoutResult{}, nil
	}

	var login *Login
	err := withLock(func(path string) error {
		login, _ = loadAny()
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to remove login file: %w", err)
		}
		return nil
	})
	if err != nil {
		return LogoutResult{}, err
	}

	result := LogoutResult{LoggedIn: true}
	if login != nil && login.RefreshToken != "" && login.ClientID != "" && login.Auth0Domain != "" {
		if err := revoke(ctx, login); err != nil {
			slog.Debug("refresh token not revoked", "error", err)
		} else {
			result.Revoked = true
		}
	}
	return result, nil
}

func revoke(ctx context.Context, login *Login) error {
	form := url.Values{"client_id": {login.ClientID}, "token": {login.RefreshToken}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, revokeURL(login.Auth0Domain), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: refreshTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("revoke returned %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return nil
}
