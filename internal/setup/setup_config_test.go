/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package setup

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/snowplow/snowplow-cli/internal/console"
)

func createTestJWTWithOrg(userEmail, userName, userSub, orgID string) string {
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
				"organization": map[string]any{
					"id": orgID,
				},
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

func Test_getOrgIDFromJWT_Success(t *testing.T) {
	token := createTestJWTWithOrg("test@example.com", "Test User", "test-sub-123", "org-123")

	orgID, err := getOrgIDFromJWT(token)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if orgID != "org-123" {
		t.Errorf("expected org ID 'org-123', got %s", orgID)
	}
}

func Test_getOrgIDFromJWT_InvalidJWTFormat(t *testing.T) {
	token := "invalid.jwt"

	_, err := getOrgIDFromJWT(token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "invalid JWT token format") {
		t.Errorf("expected JWT format error, got: %v", err)
	}
}

func Test_getOrgIDFromJWT_TooManyParts(t *testing.T) {
	token := "header.payload.signature.extra"

	_, err := getOrgIDFromJWT(token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "invalid JWT token format") {
		t.Errorf("expected JWT format error, got: %v", err)
	}
}

func Test_getOrgIDFromJWT_InvalidBase64(t *testing.T) {
	token := "header.invalid-base64!.signature"

	_, err := getOrgIDFromJWT(token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "failed to decode JWT payload") {
		t.Errorf("expected base64 decode error, got: %v", err)
	}
}

func Test_getOrgIDFromJWT_InvalidJSON(t *testing.T) {
	invalidPayload := base64.RawURLEncoding.EncodeToString([]byte("invalid json"))
	token := fmt.Sprintf("header.%s.signature", invalidPayload)

	_, err := getOrgIDFromJWT(token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "failed to parse JWT claims") {
		t.Errorf("expected JSON parse error, got: %v", err)
	}
}

func Test_getOrgIDFromJWT_MissingSnowplowRoles(t *testing.T) {
	payload := map[string]any{"sub": "test-sub"}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := fmt.Sprintf("header.%s.signature", encodedPayload)

	_, err := getOrgIDFromJWT(token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "missing Snowplow roles in JWT") {
		t.Errorf("expected missing roles error, got: %v", err)
	}
}

func Test_getOrgIDFromJWT_MissingUserInfo(t *testing.T) {
	payload := map[string]any{
		"sub": "test-sub",
		"https://snowplowanalytics.com/roles": map[string]any{
			"admin": map[string]any{"level": "org"},
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := fmt.Sprintf("header.%s.signature", encodedPayload)

	_, err := getOrgIDFromJWT(token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "missing user info in JWT") {
		t.Errorf("expected missing user info error, got: %v", err)
	}
}

func Test_getOrgIDFromJWT_MissingOrganizationInfo(t *testing.T) {
	payload := map[string]any{
		"sub": "test-sub",
		"https://snowplowanalytics.com/roles": map[string]any{
			"user": map[string]any{
				"email": "test@example.com",
				"name":  "Test User",
			},
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := fmt.Sprintf("header.%s.signature", encodedPayload)

	_, err := getOrgIDFromJWT(token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "missing organization info in JWT") {
		t.Errorf("expected missing organization info error, got: %v", err)
	}
}

func Test_getOrgIDFromJWT_MissingOrganizationID(t *testing.T) {
	payload := map[string]any{
		"sub": "test-sub",
		"https://snowplowanalytics.com/roles": map[string]any{
			"user": map[string]any{
				"email": "test@example.com",
				"name":  "Test User",
				"organization": map[string]any{
					"name": "Test Organization",
				},
			},
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := fmt.Sprintf("header.%s.signature", encodedPayload)

	_, err := getOrgIDFromJWT(token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "missing organization ID in JWT") {
		t.Errorf("expected missing organization ID error, got: %v", err)
	}
}

func Test_getOrgIDFromJWT_RolesNotMap(t *testing.T) {
	payload := map[string]any{
		"sub":                                 "test-sub",
		"https://snowplowanalytics.com/roles": "not-a-map",
	}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := fmt.Sprintf("header.%s.signature", encodedPayload)

	_, err := getOrgIDFromJWT(token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "missing Snowplow roles in JWT") {
		t.Errorf("expected missing roles error, got: %v", err)
	}
}

func Test_getOrgIDFromJWT_UserInfoNotMap(t *testing.T) {
	payload := map[string]any{
		"sub": "test-sub",
		"https://snowplowanalytics.com/roles": map[string]any{
			"user": "not-a-map",
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := fmt.Sprintf("header.%s.signature", encodedPayload)

	_, err := getOrgIDFromJWT(token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "missing user info in JWT") {
		t.Errorf("expected missing user info error, got: %v", err)
	}
}

func Test_getOrgIDFromJWT_OrganizationInfoNotMap(t *testing.T) {
	payload := map[string]any{
		"sub": "test-sub",
		"https://snowplowanalytics.com/roles": map[string]any{
			"user": map[string]any{
				"email":        "test@example.com",
				"name":         "Test User",
				"organization": "not-a-map",
			},
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := fmt.Sprintf("header.%s.signature", encodedPayload)

	_, err := getOrgIDFromJWT(token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "missing organization info in JWT") {
		t.Errorf("expected missing organization info error, got: %v", err)
	}
}

func Test_getOrgIDFromJWT_OrganizationIDNotString(t *testing.T) {
	payload := map[string]any{
		"sub": "test-sub",
		"https://snowplowanalytics.com/roles": map[string]any{
			"user": map[string]any{
				"email": "test@example.com",
				"name":  "Test User",
				"organization": map[string]any{
					"id": 123, // Should be string, not number
				},
			},
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := fmt.Sprintf("header.%s.signature", encodedPayload)

	_, err := getOrgIDFromJWT(token)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "missing organization ID in JWT") {
		t.Errorf("expected missing organization ID error, got: %v", err)
	}
}

func Test_getOrgIDFromJWT_EmptyOrganizationID(t *testing.T) {
	payload := map[string]any{
		"sub": "test-sub",
		"https://snowplowanalytics.com/roles": map[string]any{
			"user": map[string]any{
				"email": "test@example.com",
				"name":  "Test User",
				"organization": map[string]any{
					"id": "", // Empty string
				},
			},
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	token := fmt.Sprintf("header.%s.signature", encodedPayload)

	orgID, err := getOrgIDFromJWT(token)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if orgID != "" {
		t.Errorf("expected empty org ID, got %s", orgID)
	}
}

func Test_getDefaultIndex_OrgFound(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
		{ID: "org-3", Name: "Organization 3"},
	}
	jwtOrgID := "org-2"

	result := getDefaultIndex(organizations, jwtOrgID)

	if result != 1 {
		t.Errorf("expected default index 1, got %d", result)
	}
}

func Test_getDefaultIndex_OrgNotFound(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
		{ID: "org-3", Name: "Organization 3"},
	}
	jwtOrgID := "org-4"

	result := getDefaultIndex(organizations, jwtOrgID)

	if result != -1 {
		t.Errorf("expected default index -1, got %d", result)
	}
}

func Test_getDefaultIndex_EmptyOrganizations(t *testing.T) {
	organizations := []console.Organization{}
	jwtOrgID := "org-1"

	result := getDefaultIndex(organizations, jwtOrgID)

	if result != -1 {
		t.Errorf("expected default index -1, got %d", result)
	}
}

func Test_getDefaultIndex_EmptyJwtOrgID(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
	}
	jwtOrgID := ""

	result := getDefaultIndex(organizations, jwtOrgID)

	if result != -1 {
		t.Errorf("expected default index -1, got %d", result)
	}
}

func Test_getDefaultIndex_FirstOrganization(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
	}
	jwtOrgID := "org-1"

	result := getDefaultIndex(organizations, jwtOrgID)

	if result != 0 {
		t.Errorf("expected default index 0, got %d", result)
	}
}

func Test_getDefaultIndex_LastOrganization(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
		{ID: "org-3", Name: "Organization 3"},
	}
	jwtOrgID := "org-3"

	result := getDefaultIndex(organizations, jwtOrgID)

	if result != 2 {
		t.Errorf("expected default index 2, got %d", result)
	}
}

func Test_getDefaultIndex_SingleOrganization(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
	}
	jwtOrgID := "org-1"

	result := getDefaultIndex(organizations, jwtOrgID)

	if result != 0 {
		t.Errorf("expected default index 0, got %d", result)
	}
}

func Test_getDefaultIndex_SingleOrganizationNotFound(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
	}
	jwtOrgID := "org-2"

	result := getDefaultIndex(organizations, jwtOrgID)

	if result != -1 {
		t.Errorf("expected default index -1, got %d", result)
	}
}

func Test_displayAvailableOrgs_WithDefault(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
		{ID: "org-3", Name: "Organization 3"},
	}
	jwtOrgID := "org-2"
	defaultIndex := 1

	displayAvailableOrgs(organizations, jwtOrgID, defaultIndex)

}

func Test_displayAvailableOrgs_NoDefault(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
	}
	jwtOrgID := "org-3"
	defaultIndex := -1

	displayAvailableOrgs(organizations, jwtOrgID, defaultIndex)

}

func Test_displayAvailableOrgs_EmptyOrganizations(t *testing.T) {
	organizations := []console.Organization{}
	jwtOrgID := "org-1"
	defaultIndex := -1

	displayAvailableOrgs(organizations, jwtOrgID, defaultIndex)

}

func Test_displayAvailableOrgs_SingleOrganization(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
	}
	jwtOrgID := "org-1"
	defaultIndex := 0

	displayAvailableOrgs(organizations, jwtOrgID, defaultIndex)

}

func Test_selectOrganizationForInput_EmptyInputWithDefault(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
	}
	jwtOrgID := "org-1"
	input := ""
	defaultIndex := 0

	result, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.ID != "org-1" {
		t.Errorf("expected org ID 'org-1', got %s", result.ID)
	}
	if result.Name != "Organization 1" {
		t.Errorf("expected org name 'Organization 1', got %s", result.Name)
	}
}

func Test_selectOrganizationForInput_EmptyInputNoDefault(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
	}
	jwtOrgID := "org-3"
	input := ""
	defaultIndex := -1

	_, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "invalid selection") {
		t.Errorf("expected 'invalid selection' error, got: %v", err)
	}
}

func Test_selectOrganizationForInput_ValidSelection(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
		{ID: "org-3", Name: "Organization 3"},
	}
	jwtOrgID := "org-1"
	input := "2"
	defaultIndex := 0

	result, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.ID != "org-2" {
		t.Errorf("expected org ID 'org-2', got %s", result.ID)
	}
	if result.Name != "Organization 2" {
		t.Errorf("expected org name 'Organization 2', got %s", result.Name)
	}
}

func Test_selectOrganizationForInput_FirstSelection(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
	}
	jwtOrgID := "org-2"
	input := "1"
	defaultIndex := 1

	result, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.ID != "org-1" {
		t.Errorf("expected org ID 'org-1', got %s", result.ID)
	}
	if result.Name != "Organization 1" {
		t.Errorf("expected org name 'Organization 1', got %s", result.Name)
	}
}

func Test_selectOrganizationForInput_LastSelection(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
		{ID: "org-3", Name: "Organization 3"},
	}
	jwtOrgID := "org-1"
	input := "3"
	defaultIndex := 0

	result, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.ID != "org-3" {
		t.Errorf("expected org ID 'org-3', got %s", result.ID)
	}
	if result.Name != "Organization 3" {
		t.Errorf("expected org name 'Organization 3', got %s", result.Name)
	}
}

func Test_selectOrganizationForInput_SelectionTooLow(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
	}
	jwtOrgID := "org-1"
	input := "0"
	defaultIndex := 0

	_, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "selection must be between 1 and 2") {
		t.Errorf("expected range error, got: %v", err)
	}
}

func Test_selectOrganizationForInput_SelectionTooHigh(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
	}
	jwtOrgID := "org-1"
	input := "3"
	defaultIndex := 0

	_, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "selection must be between 1 and 2") {
		t.Errorf("expected range error, got: %v", err)
	}
}

func Test_selectOrganizationForInput_NegativeSelection(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
	}
	jwtOrgID := "org-1"
	input := "-1"
	defaultIndex := 0

	_, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "selection must be between 1 and 2") {
		t.Errorf("expected range error, got: %v", err)
	}
}

func Test_selectOrganizationForInput_InvalidInput(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
	}
	jwtOrgID := "org-1"
	input := "abc"
	defaultIndex := 0

	_, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "invalid selection: abc") {
		t.Errorf("expected invalid selection error, got: %v", err)
	}
}

func Test_selectOrganizationForInput_NumberWithLetters(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
	}
	jwtOrgID := "org-1"
	input := "1abc"
	defaultIndex := 0

	// fmt.Sscanf with %d will parse "1abc" as "1", so this should succeed
	result, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should select the first organization since "1abc" parses as "1"
	if result.ID != "org-1" {
		t.Errorf("expected org ID 'org-1', got %s", result.ID)
	}
	if result.Name != "Organization 1" {
		t.Errorf("expected org name 'Organization 1', got %s", result.Name)
	}
}

func Test_selectOrganizationForInput_FloatInput(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
	}
	jwtOrgID := "org-1"
	input := "1.5"
	defaultIndex := 0

	result, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.ID != "org-1" {
		t.Errorf("expected org ID 'org-1', got %s", result.ID)
	}
	if result.Name != "Organization 1" {
		t.Errorf("expected org name 'Organization 1', got %s", result.Name)
	}
}

func Test_selectOrganizationForInput_SingleOrganization(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
	}
	jwtOrgID := "org-1"
	input := "1"
	defaultIndex := 0

	result, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result.ID != "org-1" {
		t.Errorf("expected org ID 'org-1', got %s", result.ID)
	}
	if result.Name != "Organization 1" {
		t.Errorf("expected org name 'Organization 1', got %s", result.Name)
	}
}

func Test_selectOrganizationForInput_SingleOrganizationInvalidSelection(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
	}
	jwtOrgID := "org-1"
	input := "2"
	defaultIndex := 0

	_, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "selection must be between 1 and 1") {
		t.Errorf("expected range error, got: %v", err)
	}
}

func Test_selectOrganizationForInput_EmptyOrganizations(t *testing.T) {
	organizations := []console.Organization{}
	jwtOrgID := "org-1"
	input := "1"
	defaultIndex := -1

	_, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "selection must be between 1 and 0") {
		t.Errorf("expected range error, got: %v", err)
	}
}

func Test_selectOrganizationForInput_WhitespaceInput(t *testing.T) {
	organizations := []console.Organization{
		{ID: "org-1", Name: "Organization 1"},
		{ID: "org-2", Name: "Organization 2"},
	}
	jwtOrgID := "org-1"
	input := " "
	defaultIndex := 0

	_, err := selectOrganizationForInput(organizations, jwtOrgID, input, defaultIndex)

	if err == nil {
		t.Error("expected error but got none")
	}
	if !strings.Contains(err.Error(), "invalid selection:  ") {
		t.Errorf("expected invalid selection error, got: %v", err)
	}
}
