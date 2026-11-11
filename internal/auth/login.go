/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

// Package auth stores the Console login made with `snowplow-cli login`.
// Snowtype reads and writes the same file, so one login serves both tools.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
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

// expiryMargin avoids sending a token that expires while a command runs its first requests
const expiryMargin = time.Minute

// defaultLifetime is assumed when the provider does not say when a token expires
const defaultLifetime = time.Hour

var refreshMu sync.Mutex

const refreshTimeout = 30 * time.Second

// lockWait is long enough for another process to finish a write, after which
// a lock left behind by a crash is taken over
var lockWait = 5 * time.Second

// Path is a variable holding the function to get the login file path (for testability)
// It returns "" when the home directory is unknown, so that tokens are never
// written relative to the working directory.
var Path = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "snowplow", "login.json")
}

// tokenURL is a variable holding the function to get the token endpoint (for testability)
var tokenURL = func(auth0Domain string) string {
	return fmt.Sprintf("https://%s/oauth/token", auth0Domain)
}

// Save replaces the file in one step, so that another process reading it
// never sees a partial write. Two processes refreshing at the same moment can
// still invalidate each other's refresh token when the provider rotates them;
// logging in again recovers.
func Save(login *Login) error {
	return withLock(func(path string) error { return write(login, path) })
}

// withLock runs fn while holding login.json.lock, the lock Snowtype takes
// too, so that a check of the file and the write that follows it cannot be
// interleaved with a write by another process.
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
			// Left behind by a process that crashed
			_ = os.Remove(lock)
			continue
		}
		time.Sleep(25 * time.Millisecond)
	}
	defer func() { _ = os.Remove(lock) }()

	return fn(path)
}

func write(login *Login, path string) error {
	data, err := json.MarshalIndent(login, "", "  ")
	if err != nil {
		return err
	}

	// CreateTemp makes the file with mode 0600, before any token is written
	tmp, err := os.CreateTemp(filepath.Dir(path), ".login-*.json")
	if err != nil {
		return fmt.Errorf("failed to write login file: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write login file: %w", err)
	}
	// Synced so that a crash cannot leave an empty file behind the rename
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

// Load returns nil when nobody has logged in to host.
func Load(host string) (*Login, error) {
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
	if login.AccessToken == "" || strings.TrimRight(login.Host, "/") != strings.TrimRight(host, "/") {
		slog.Debug("saved login ignored: it has no token or is for another host", "file", Path(), "login-host", login.Host, "host", host)
		return nil, nil
	}
	return &login, nil
}

// Valid returns the login for host with an access token that can be used now,
// refreshing and saving it when it has expired. It returns nil when there is
// no login or it can no longer be refreshed.
func Valid(ctx context.Context, host string) (*Login, error) {
	// One refresh at a time in this process: a second caller then finds the
	// refreshed token on disk instead of spending the same refresh token.
	refreshMu.Lock()
	defer refreshMu.Unlock()

	login, err := Load(host)
	if err != nil || login == nil {
		return nil, err
	}
	// A login without an expiry is used as it is, as oauth2 does
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
	// Bounded, as the refresh holds refreshMu and the default client never times out
	ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Timeout: refreshTimeout})
	token, err := conf.TokenSource(ctx, &oauth2.Token{RefreshToken: login.RefreshToken}).Token()
	if err != nil {
		var retrieveErr *oauth2.RetrieveError
		// A rejected refresh means logging in again. Anything else, such as
		// a rate limit or an outage, is reported as it is.
		if errors.As(err, &retrieveErr) && rejected(retrieveErr) {
			slog.Debug("saved login rejected at refresh, logging in again is needed",
				"error-code", retrieveErr.ErrorCode, "error-description", retrieveErr.ErrorDescription)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to refresh login: %w", err)
	}

	// A login made while the refresh was in flight is newer: leave it in place
	usedRefreshToken := login.RefreshToken
	login.AccessToken = token.AccessToken
	login.ExpiresAt = token.Expiry
	if token.Expiry.IsZero() {
		// Without an expiry every command would refresh again
		login.ExpiresAt = time.Now().Add(defaultLifetime)
	}
	if token.RefreshToken != "" {
		login.RefreshToken = token.RefreshToken
	}
	// Checked and written under the lock, so no login can land in between
	err = withLock(func(path string) error {
		if current, err := Load(host); err == nil && current != nil && current.RefreshToken != usedRefreshToken {
			return nil
		}
		return write(login, path)
	})
	if err != nil {
		return nil, err
	}
	return login, nil
}

// rejected mirrors console.TokenError.Rejected: any client error except a rate limit.
func rejected(err *oauth2.RetrieveError) bool {
	if err.Response == nil {
		return false
	}
	status := err.Response.StatusCode
	return status >= 400 && status < 500 && status != http.StatusTooManyRequests
}
