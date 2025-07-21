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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/snowplow/snowplow-cli/internal/util"
)

func createTestJWTForOrg(userEmail, userName, userSub string) string {
	header := map[string]any{
		"alg": "HS256",
		"typ": "JWT",
	}

	payload := map[string]any{
		"sub": userSub,
		"https://snowplowanalytics.com/roles": map[string]any{
			"user": map[string]any{
				"email": userEmail,
				"name":  userName,
			},
		},
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
	}

	headerBytes, _ := json.Marshal(header)
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)

	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	signature := base64.RawURLEncoding.EncodeToString([]byte("fake-signature"))

	return fmt.Sprintf("%s.%s.%s", encodedHeader, encodedPayload, signature)
}

func Test_GetOrganizations_Success(t *testing.T) {
	accessToken := createTestJWTForOrg("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations" && r.Method == "GET" {
			if r.Header.Get("authorization") != "Bearer "+accessToken {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}
			if r.Header.Get("X-SNOWPLOW-CLI") != util.VersionInfo {
				t.Errorf("bad version header, got: %s", r.Header.Get("X-SNOWPLOW-CLI"))
			}

			resp := `[
				{
					"id": "org-1",
					"name": "Test Organization 1"
				},
				{
					"id": "org-2", 
					"name": "Test Organization 2"
				}
			]`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	organizations, err := GetOrganizations(ctx, accessToken, server.URL)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(organizations) != 2 {
		t.Errorf("expected 2 organizations, got %d", len(organizations))
	}

	if organizations[0].ID != "org-1" {
		t.Errorf("expected first org ID 'org-1', got %s", organizations[0].ID)
	}
	if organizations[0].Name != "Test Organization 1" {
		t.Errorf("expected first org name 'Test Organization 1', got %s", organizations[0].Name)
	}

	if organizations[1].ID != "org-2" {
		t.Errorf("expected second org ID 'org-2', got %s", organizations[1].ID)
	}
	if organizations[1].Name != "Test Organization 2" {
		t.Errorf("expected second org name 'Test Organization 2', got %s", organizations[1].Name)
	}
}

func Test_GetOrganizations_EmptyResponse(t *testing.T) {
	accessToken := createTestJWTForOrg("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations" && r.Method == "GET" {
			if r.Header.Get("authorization") != "Bearer "+accessToken {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp := `[]`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	organizations, err := GetOrganizations(ctx, accessToken, server.URL)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(organizations) != 0 {
		t.Errorf("expected 0 organizations, got %d", len(organizations))
	}
}

func Test_GetOrganizations_SingleOrganization(t *testing.T) {
	accessToken := createTestJWTForOrg("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations" && r.Method == "GET" {
			if r.Header.Get("authorization") != "Bearer "+accessToken {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp := `[
				{
					"id": "single-org-id",
					"name": "Single Organization"
				}
			]`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	organizations, err := GetOrganizations(ctx, accessToken, server.URL)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(organizations) != 1 {
		t.Errorf("expected 1 organization, got %d", len(organizations))
	}

	if organizations[0].ID != "single-org-id" {
		t.Errorf("expected org ID 'single-org-id', got %s", organizations[0].ID)
	}
	if organizations[0].Name != "Single Organization" {
		t.Errorf("expected org name 'Single Organization', got %s", organizations[0].Name)
	}
}

func Test_GetOrganizations_ServerError(t *testing.T) {
	accessToken := createTestJWTForOrg("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations" && r.Method == "GET" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"error": "Internal server error"}`)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := GetOrganizations(ctx, accessToken, server.URL)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "failed to fetch organizations") {
		t.Errorf("expected 'failed to fetch organizations' error, got: %v", err)
	}
}

func Test_GetOrganizations_Unauthorized(t *testing.T) {
	accessToken := createTestJWTForOrg("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations" && r.Method == "GET" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error": "Unauthorized"}`)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := GetOrganizations(ctx, accessToken, server.URL)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "failed to fetch organizations") {
		t.Errorf("expected 'failed to fetch organizations' error, got: %v", err)
	}
}

func Test_GetOrganizations_Forbidden(t *testing.T) {
	accessToken := createTestJWTForOrg("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations" && r.Method == "GET" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"error": "Forbidden"}`)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := GetOrganizations(ctx, accessToken, server.URL)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "failed to fetch organizations") {
		t.Errorf("expected 'failed to fetch organizations' error, got: %v", err)
	}
}

func Test_GetOrganizations_InvalidResponseJSON(t *testing.T) {
	accessToken := createTestJWTForOrg("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations" && r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `invalid json response`)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := GetOrganizations(ctx, accessToken, server.URL)

	if err == nil {
		t.Error("expected error with invalid JSON response")
	}
	if !strings.Contains(err.Error(), "failed to parse organizations response") {
		t.Errorf("expected JSON parse error, got: %v", err)
	}
}

func Test_GetOrganizations_MalformedJSONArray(t *testing.T) {
	accessToken := createTestJWTForOrg("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations" && r.Method == "GET" {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `[{"id": "org-1", "name": "Test"}`) // Missing closing bracket
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := GetOrganizations(ctx, accessToken, server.URL)

	if err == nil {
		t.Error("expected error with malformed JSON array")
	}
	if !strings.Contains(err.Error(), "failed to parse organizations response") {
		t.Errorf("expected JSON parse error, got: %v", err)
	}
}

func Test_GetOrganizations_NetworkError(t *testing.T) {
	accessToken := createTestJWTForOrg("test@example.com", "Test User", "test-sub-123")

	ctx := context.Background()
	_, err := GetOrganizations(ctx, accessToken, "http://invalid-host-that-does-not-exist")

	if err == nil {
		t.Error("expected network error")
	}
}

func Test_GetOrganizations_ContextCancellation(t *testing.T) {
	accessToken := createTestJWTForOrg("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := GetOrganizations(ctx, accessToken, server.URL)
	if err == nil {
		t.Error("expected error due to context cancellation")
	}
}

func Test_GetOrganizations_PartialOrganizationData(t *testing.T) {
	accessToken := createTestJWTForOrg("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations" && r.Method == "GET" {
			resp := `[
				{
					"id": "org-1"
				},
				{
					"id": "org-2",
					"name": "Complete Organization"
				}
			]`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	organizations, err := GetOrganizations(ctx, accessToken, server.URL)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(organizations) != 2 {
		t.Errorf("expected 2 organizations, got %d", len(organizations))
	}

	// First organization should have empty name
	if organizations[0].ID != "org-1" {
		t.Errorf("expected first org ID 'org-1', got %s", organizations[0].ID)
	}
	if organizations[0].Name != "" {
		t.Errorf("expected first org name to be empty, got %s", organizations[0].Name)
	}

	// Second organization should be complete
	if organizations[1].ID != "org-2" {
		t.Errorf("expected second org ID 'org-2', got %s", organizations[1].ID)
	}
	if organizations[1].Name != "Complete Organization" {
		t.Errorf("expected second org name 'Complete Organization', got %s", organizations[1].Name)
	}
}

func Test_GetOrganizations_ExtraJSONFields(t *testing.T) {
	accessToken := createTestJWTForOrg("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations" && r.Method == "GET" {
			// Response with extra fields that should be ignored
			resp := `[
				{
					"id": "org-1",
					"name": "Test Organization",
					"description": "This field should be ignored",
					"created_at": "2024-01-01T00:00:00Z",
					"owner": "test@example.com"
				}
			]`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	organizations, err := GetOrganizations(ctx, accessToken, server.URL)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(organizations) != 1 {
		t.Errorf("expected 1 organization, got %d", len(organizations))
	}

	// Should only parse the id and name fields
	if organizations[0].ID != "org-1" {
		t.Errorf("expected org ID 'org-1', got %s", organizations[0].ID)
	}
	if organizations[0].Name != "Test Organization" {
		t.Errorf("expected org name 'Test Organization', got %s", organizations[0].Name)
	}
}

func Test_GetOrganizations_EmptyFieldValues(t *testing.T) {
	accessToken := createTestJWTForOrg("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations" && r.Method == "GET" {
			// Response with empty field values
			resp := `[
				{
					"id": "",
					"name": ""
				},
				{
					"id": "org-2",
					"name": ""
				}
			]`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	organizations, err := GetOrganizations(ctx, accessToken, server.URL)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(organizations) != 2 {
		t.Errorf("expected 2 organizations, got %d", len(organizations))
	}

	// Should handle empty strings gracefully
	if organizations[0].ID != "" {
		t.Errorf("expected empty org ID, got %s", organizations[0].ID)
	}
	if organizations[0].Name != "" {
		t.Errorf("expected empty org name, got %s", organizations[0].Name)
	}

	if organizations[1].ID != "org-2" {
		t.Errorf("expected org ID 'org-2', got %s", organizations[1].ID)
	}
	if organizations[1].Name != "" {
		t.Errorf("expected empty org name, got %s", organizations[1].Name)
	}
}
