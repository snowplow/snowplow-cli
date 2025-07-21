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

func createTestJWT(userEmail, userName, userSub string) string {
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

func Test_CreateAPIKey_FullAccess(t *testing.T) {
	accessToken := createTestJWT("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/test-org-id/credentials/v2/api-keys" && r.Method == "POST" {
			if r.Header.Get("Authorization") != "Bearer "+accessToken {
				t.Errorf("bad auth token, got: %s", r.Header.Get("Authorization"))
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("bad content type, got: %s", r.Header.Get("Content-Type"))
			}
			if r.Header.Get("X-SNOWPLOW-CLI") != util.VersionInfo {
				t.Errorf("bad version header, got: %s", r.Header.Get("X-SNOWPLOW-CLI"))
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("failed to read request body: %v", err)
			}

			var reqBody map[string]any
			if err := json.Unmarshal(body, &reqBody); err != nil {
				t.Errorf("failed to parse request body: %v", err)
			}

			expectedDesc := "test@example.com CLI key"
			if reqBody["description"] != expectedDesc {
				t.Errorf("expected description %s, got %s", expectedDesc, reqBody["description"])
			}

			permissions, ok := reqBody["permissions"].([]any)
			if !ok {
				t.Errorf("permissions should be an array")
			}
			if len(permissions) != 1 {
				t.Errorf("expected 1 permission, got %d", len(permissions))
			}

			resp := `{"id": "test-key-id", "key": "test-key-secret"}`
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, resp)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	result, err := CreateAPIKey(ctx, accessToken, server.URL, "test-org-id", false)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.ID != "test-key-id" {
		t.Errorf("expected ID test-key-id, got %s", result.ID)
	}
	if result.Secret != "test-key-secret" {
		t.Errorf("expected Secret test-key-secret, got %s", result.Secret)
	}
}

func Test_CreateAPIKey_ReadOnly(t *testing.T) {
	accessToken := createTestJWT("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/test-org-id/credentials/v2/api-keys" && r.Method == "POST" {
			if r.Header.Get("Authorization") != "Bearer "+accessToken {
				t.Errorf("bad auth token, got: %s", r.Header.Get("Authorization"))
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("failed to read request body: %v", err)
			}

			var reqBody map[string]any
			if err := json.Unmarshal(body, &reqBody); err != nil {
				t.Errorf("failed to parse request body: %v", err)
			}

			expectedDesc := "test@example.com CLI key (read-only)"
			if reqBody["description"] != expectedDesc {
				t.Errorf("expected description %s, got %s", expectedDesc, reqBody["description"])
			}

			permissions, ok := reqBody["permissions"].([]any)
			if !ok {
				t.Errorf("permissions should be an array")
			}
			if len(permissions) != 1 {
				t.Errorf("expected 1 permission, got %d", len(permissions))
			}

			resp := `{"id": "readonly-key-id", "key": "readonly-key-secret"}`
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, resp)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	result, err := CreateAPIKey(ctx, accessToken, server.URL, "test-org-id", true)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.ID != "readonly-key-id" {
		t.Errorf("expected ID readonly-key-id, got %s", result.ID)
	}
	if result.Secret != "readonly-key-secret" {
		t.Errorf("expected Secret readonly-key-secret, got %s", result.Secret)
	}
}

func Test_CreateAPIKey_ServerError(t *testing.T) {
	accessToken := createTestJWT("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/test-org-id/credentials/v2/api-keys" && r.Method == "POST" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"error": "Internal server error"}`)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := CreateAPIKey(ctx, accessToken, server.URL, "test-org-id", false)

	if err == nil {
		t.Error("expected error but got none")
	}
}

func Test_CreateAPIKey_Unauthorized(t *testing.T) {
	accessToken := createTestJWT("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/test-org-id/credentials/v2/api-keys" && r.Method == "POST" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error": "Unauthorized"}`)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := CreateAPIKey(ctx, accessToken, server.URL, "test-org-id", false)

	if err == nil {
		t.Error("expected error but got none")
	}
}

func Test_CreateAPIKey_InvalidJWT(t *testing.T) {
	ctx := context.Background()
	invalidToken := "invalid.jwt.token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach server with invalid JWT")
	}))
	defer server.Close()

	_, err := CreateAPIKey(ctx, invalidToken, server.URL, "test-org-id", false)
	if err == nil {
		t.Error("expected error with invalid JWT")
	}
}

func Test_CreateAPIKey_MalformedJWT(t *testing.T) {
	ctx := context.Background()
	malformedToken := "header.payload"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach server with malformed JWT")
	}))
	defer server.Close()

	_, err := CreateAPIKey(ctx, malformedToken, server.URL, "test-org-id", false)
	if err == nil {
		t.Error("expected error with malformed JWT")
	}
	if !strings.Contains(err.Error(), "failed to get user info") {
		t.Errorf("expected user info error, got: %v", err)
	}
}

func Test_CreateAPIKey_InvalidResponseJSON(t *testing.T) {
	accessToken := createTestJWT("test@example.com", "Test User", "test-sub-123")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/test-org-id/credentials/v2/api-keys" && r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `invalid json response`)
			return
		}
		t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	ctx := context.Background()
	_, err := CreateAPIKey(ctx, accessToken, server.URL, "test-org-id", false)
	if err == nil {
		t.Error("expected error with invalid JSON response")
	}
	if !strings.Contains(err.Error(), "failed to parse API response") {
		t.Errorf("expected JSON parse error, got: %v", err)
	}
}

func Test_CreateAPIKey_NetworkError(t *testing.T) {
	accessToken := createTestJWT("test@example.com", "Test User", "test-sub-123")

	ctx := context.Background()
	_, err := CreateAPIKey(ctx, accessToken, "http://invalid-host-that-does-not-exist", "test-org-id", false)
	if err == nil {
		t.Error("expected network error")
	}
}

func Test_GetUserInfo_ValidJWT(t *testing.T) {
	accessToken := createTestJWT("test@example.com", "Test User", "test-sub-123")

	ctx := context.Background()
	userInfo, err := GetUserInfo(ctx, accessToken)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if userInfo.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", userInfo.Email)
	}
	if userInfo.Name != "Test User" {
		t.Errorf("expected name 'Test User', got %s", userInfo.Name)
	}
	if userInfo.Sub != "test-sub-123" {
		t.Errorf("expected sub test-sub-123, got %s", userInfo.Sub)
	}
}

func Test_GetUserInfo_MinimalJWT(t *testing.T) {
	accessToken := createTestJWT("minimal@example.com", "", "minimal-sub")

	ctx := context.Background()
	userInfo, err := GetUserInfo(ctx, accessToken)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if userInfo.Email != "minimal@example.com" {
		t.Errorf("expected email minimal@example.com, got %s", userInfo.Email)
	}
	if userInfo.Name != "" {
		t.Errorf("expected empty name, got %s", userInfo.Name)
	}
	if userInfo.Sub != "minimal-sub" {
		t.Errorf("expected sub minimal-sub, got %s", userInfo.Sub)
	}
}

func Test_GetUserInfo_EmptyValues(t *testing.T) {
	accessToken := createTestJWT("", "", "")

	ctx := context.Background()
	userInfo, err := GetUserInfo(ctx, accessToken)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if userInfo.Email != "" {
		t.Errorf("expected empty email, got %s", userInfo.Email)
	}
	if userInfo.Name != "" {
		t.Errorf("expected empty name, got %s", userInfo.Name)
	}
	if userInfo.Sub != "" {
		t.Errorf("expected empty sub, got %s", userInfo.Sub)
	}
}

func Test_GetUserInfo_InvalidJWTFormat(t *testing.T) {
	ctx := context.Background()
	_, err := GetUserInfo(ctx, "header.payload")

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "invalid JWT token format") {
		t.Errorf("expected JWT format error, got: %v", err)
	}
}

func Test_GetUserInfo_TooManyJWTParts(t *testing.T) {
	ctx := context.Background()
	_, err := GetUserInfo(ctx, "header.payload.signature.extra")

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "invalid JWT token format") {
		t.Errorf("expected JWT format error, got: %v", err)
	}
}

func Test_GetUserInfo_InvalidBase64(t *testing.T) {
	ctx := context.Background()
	_, err := GetUserInfo(ctx, "header.invalid-base64!.signature")

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "failed to decode JWT payload") {
		t.Errorf("expected base64 decode error, got: %v", err)
	}
}

func Test_GetUserInfo_InvalidJSON(t *testing.T) {
	invalidPayload := base64.RawURLEncoding.EncodeToString([]byte("invalid json"))
	token := fmt.Sprintf("header.%s.signature", invalidPayload)

	ctx := context.Background()
	_, err := GetUserInfo(ctx, token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "failed to parse JWT claims") {
		t.Errorf("expected JSON parse error, got: %v", err)
	}
}

func Test_GetUserInfo_MissingSnowplowRoles(t *testing.T) {
	payload := map[string]any{"sub": "test-sub"}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := fmt.Sprintf("header.%s.signature", encodedPayload)

	ctx := context.Background()
	_, err := GetUserInfo(ctx, token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "missing Snowplow roles in JWT") {
		t.Errorf("expected missing roles error, got: %v", err)
	}
}

func Test_GetUserInfo_MissingUserInfo(t *testing.T) {
	payload := map[string]any{
		"sub": "test-sub",
		"https://snowplowanalytics.com/roles": map[string]any{
			"admin": map[string]any{"level": "org"},
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := fmt.Sprintf("header.%s.signature", encodedPayload)

	ctx := context.Background()
	_, err := GetUserInfo(ctx, token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "missing user info in JWT") {
		t.Errorf("expected missing user info error, got: %v", err)
	}
}

func Test_GetUserInfo_RolesNotMap(t *testing.T) {
	payload := map[string]any{
		"sub":                                 "test-sub",
		"https://snowplowanalytics.com/roles": "not-a-map",
	}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := fmt.Sprintf("header.%s.signature", encodedPayload)

	ctx := context.Background()
	_, err := GetUserInfo(ctx, token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "missing Snowplow roles in JWT") {
		t.Errorf("expected missing roles error, got: %v", err)
	}
}

func Test_GetUserInfo_UserInfoNotMap(t *testing.T) {
	payload := map[string]any{
		"sub": "test-sub",
		"https://snowplowanalytics.com/roles": map[string]any{
			"user": "not-a-map",
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := fmt.Sprintf("header.%s.signature", encodedPayload)

	ctx := context.Background()
	_, err := GetUserInfo(ctx, token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "missing user info in JWT") {
		t.Errorf("expected missing user info error, got: %v", err)
	}
}
