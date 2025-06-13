/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package validation

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/logging"
)

func TestValidateDataStructuresWithClient_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()
	writeValidDataStructure(t, tmpDir, "test.yaml")

	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ctx := logging.ContextWithLogger(context.Background(), logger)

	mockClient := createMockClient(&MockSuccessfulTransport{})

	err := ValidateDataStructuresWithClient(ctx, mockClient, []string{tmpDir}, false)

	if err != nil {
		t.Errorf("Expected validation to succeed, but got error: %v", err)
	}

	logStr := logOutput.String()
	if !strings.Contains(logStr, "validating from") {
		t.Error("Expected 'validating from' log message not found")
	}
	if !strings.Contains(logStr, "paths") {
		t.Error("Expected 'paths' in log output")
	}
}

func TestValidateDataStructuresWithClient_LocalValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()
	writeInvalidDataStructure(t, tmpDir, "invalid.yaml")

	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ctx := logging.ContextWithLogger(context.Background(), logger)

	mockClient := createMockClient(&MockClientThatShouldNotBeCalledTransport{t: t})

	err := ValidateDataStructuresWithClient(ctx, mockClient, []string{tmpDir}, false)

	if err == nil {
		t.Error("Expected validation to fail for invalid data structure")
	}

	logStr := logOutput.String()
	if !strings.Contains(logStr, "validation") || !strings.Contains(logStr, "error") {
		t.Error("Expected validation error to be logged")
	}

	if !strings.Contains(logStr, "validating from") {
		t.Error("Expected 'validating from' log message even on failure")
	}
}

func TestValidateDataStructuresWithClient_NetworkFailure(t *testing.T) {
	tmpDir := t.TempDir()
	writeValidDataStructure(t, tmpDir, "test.yaml")

	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ctx := logging.ContextWithLogger(context.Background(), logger)

	mockClient := createMockClient(&MockNetworkFailureTransport{})

	err := ValidateDataStructuresWithClient(ctx, mockClient, []string{tmpDir}, false)

	if err == nil {
		t.Error("Expected validation to fail due to network error")
	}
	if !strings.Contains(err.Error(), "network") {
		t.Errorf("Expected network error, got: %v", err)
	}

	logStr := logOutput.String()
	if !strings.Contains(logStr, "validating from") {
		t.Error("Expected 'validating from' log message before network failure")
	}
}

func TestValidateDataStructuresWithClient_RemoteValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()
	writeValidDataStructure(t, tmpDir, "test.yaml")

	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ctx := logging.ContextWithLogger(context.Background(), logger)

	mockClient := createMockClient(&MockRemoteValidationFailureTransport{})

	err := ValidateDataStructuresWithClient(ctx, mockClient, []string{tmpDir}, false)

	if err == nil {
		t.Error("Expected validation to fail due to remote validation error")
	}
	if !strings.Contains(err.Error(), "validation failure") {
		t.Errorf("Expected validation failure error, got: %v", err)
	}

	logStr := logOutput.String()
	if !strings.Contains(logStr, "validating from") {
		t.Error("Expected 'validating from' log message")
	}
}

func TestValidateDataStructuresWithClient_GitHubAnnotations(t *testing.T) {
	tmpDir := t.TempDir()
	writeValidDataStructure(t, tmpDir, "test.yaml")

	ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	mockClient := createMockClient(&MockSuccessfulTransport{})

	err := ValidateDataStructuresWithClient(ctx, mockClient, []string{tmpDir}, true)

	if err != nil {
		t.Errorf("Expected validation to succeed, but got error: %v", err)
	}
}

func TestValidateDataStructuresWithClient_EmptyPaths(t *testing.T) {
	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ctx := logging.ContextWithLogger(context.Background(), logger)

	mockClient := createMockClient(&MockNetworkFailureTransport{})

	err := ValidateDataStructuresWithClient(ctx, mockClient, []string{}, false)

	if err == nil {
		t.Log("Unexpectedly succeeded - default data-structures folder must exist")
	}

	logStr := logOutput.String()
	if !strings.Contains(logStr, "validating from") {
		t.Error("Expected 'validating from' log message")
	}
	if !strings.Contains(logStr, "data-structures") {
		t.Error("Expected default 'data-structures' path to be logged")
	}
}

func TestValidateDataStructuresWithClient_NoLoggerInContext(t *testing.T) {
	tmpDir := t.TempDir()
	writeValidDataStructure(t, tmpDir, "test.yaml")

	ctx := context.Background()

	mockClient := createMockClient(&MockSuccessfulTransport{})

	err := ValidateDataStructuresWithClient(ctx, mockClient, []string{tmpDir}, false)

	t.Logf("Function completed with error: %v", err)
}

func writeValidDataStructure(t *testing.T, dir, filename string) {
	validDS := `apiVersion: v1
resourceType: data-structure
meta:
  schemaType: event
  hidden: false
  customData: {}
data:
  $schema: http://iglucentral.com/schemas/com.snowplowanalytics.self-desc/schema/jsonschema/1-0-0#
  self:
    vendor: com.example
    name: test_event
    format: jsonschema
    version: 1-0-0
  type: object
  properties:
    userId:
      type: string
`

	err := os.WriteFile(filepath.Join(dir, filename), []byte(validDS), 0644)
	if err != nil {
		t.Fatalf("Failed to write valid test file: %v", err)
	}
}

func writeInvalidDataStructure(t *testing.T, dir, filename string) {
	invalidDS := `apiVersion: v1
resourceType: data-structure
meta:
  schemaType: event
  hidden: false
data:
  self:
    vendor: com.example
    name: invalid_event
    format: jsonschema
  type: object
`

	err := os.WriteFile(filepath.Join(dir, filename), []byte(invalidDS), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid test file: %v", err)
	}
}

func createMockClient(roundTripper http.RoundTripper) *console.ApiClient {
	return &console.ApiClient{
		Http:    &http.Client{Transport: roundTripper},
		Jwt:     "fake-jwt-token",
		BaseUrl: "https://fake-api.example.com/api/msc/v1/organizations/fake-org",
		OrgId:   "fake-org-id",
	}
}

type MockSuccessfulTransport struct{}

func (m *MockSuccessfulTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch {
	case strings.Contains(req.URL.Path, "/data-structures/v1") && req.Method == "GET":
		body := `[]`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	case strings.Contains(req.URL.Path, "/validation-requests") && req.Method == "POST":
		body := `{"success": true, "valid": true, "errors": [], "warnings": [], "info": []}`
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	default:
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"message": "not found"}`)),
			Request:    req,
		}, nil
	}
}

type MockNetworkFailureTransport struct{}

func (m *MockNetworkFailureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("network connection failed")
}

type MockClientThatShouldNotBeCalledTransport struct {
	t *testing.T
}

func (m *MockClientThatShouldNotBeCalledTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.t.Error("HTTP client should not be called when local validation fails")
	return nil, fmt.Errorf("should not be called")
}

type MockRemoteValidationFailureTransport struct{}

func (m *MockRemoteValidationFailureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch {
	case strings.Contains(req.URL.Path, "/data-structures/v1") && req.Method == "GET":
		body := `[]`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	case strings.Contains(req.URL.Path, "/validation-requests") && req.Method == "POST":
		body := `{"success": false, "valid": false, "errors": ["Remote validation failed"], "warnings": [], "info": []}`
		return &http.Response{
			StatusCode: 201,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	default:
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader(`{"message": "not found"}`)),
			Request:    req,
		}, nil
	}
}
