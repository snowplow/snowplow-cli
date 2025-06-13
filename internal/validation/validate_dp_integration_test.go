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

func TestValidateDataProductsWithClient_HappyPath(t *testing.T) {
	tmpDir := t.TempDir()
	writeValidDataProduct(t, tmpDir, "test-dp.yaml")
	writeValidSourceApp(t, tmpDir, "test-sa.yaml")

	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ctx := logging.ContextWithLogger(context.Background(), logger)

	mockClient := createMockDataProductClient(&MockDataProductSuccessTransport{})

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1)

	if err != nil {
		t.Errorf("Expected validation to succeed, but got error: %v", err)
	}

	logStr := logOutput.String()
	if !strings.Contains(logStr, "validation") {
		t.Error("Expected validation log messages not found")
	}
}

func TestValidateDataProductsWithClient_LocalValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()
	writeInvalidDataProduct(t, tmpDir, "invalid-dp.yaml")

	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ctx := logging.ContextWithLogger(context.Background(), logger)

	mockClient := createMockDataProductClient(&MockDataProductSuccessTransport{})

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1)

	if err == nil {
		t.Error("Expected validation to fail for invalid data product")
	}

	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("Expected validation failure error, got: %v", err)
	}

	logStr := logOutput.String()
	if !strings.Contains(logStr, "validating") {
		t.Error("Expected validation log messages")
	}
}

func TestValidateDataProductsWithClient_NetworkFailure(t *testing.T) {
	tmpDir := t.TempDir()
	writeValidDataProduct(t, tmpDir, "test-dp.yaml")
	writeValidSourceApp(t, tmpDir, "test-sa.yaml")

	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ctx := logging.ContextWithLogger(context.Background(), logger)

	mockClient := createMockDataProductClient(&MockDataProductNetworkFailureTransport{})

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1)

	if err == nil {
		t.Error("Expected validation to fail due to network error")
	}
	if !strings.Contains(err.Error(), "network") {
		t.Errorf("Expected network error, got: %v", err)
	}
}

func TestValidateDataProductsWithClient_CompatibilityCheckFailure(t *testing.T) {
	tmpDir := t.TempDir()
	writeValidDataProduct(t, tmpDir, "test-dp.yaml")
	writeValidSourceApp(t, tmpDir, "test-sa.yaml")

	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ctx := logging.ContextWithLogger(context.Background(), logger)

	mockClient := createMockDataProductClient(&MockDataProductCompatFailureTransport{})

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, true, 1)

	if err == nil {
		logStr := logOutput.String()
		t.Logf("Validation succeeded unexpectedly. Logs: %s", logStr)
		t.Fatal("Expected validation to fail due to compatibility error")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("Expected compatibility validation failure, got: %v", err)
	}
}

func TestValidateDataProductsWithClient_GitHubAnnotations(t *testing.T) {
	tmpDir := t.TempDir()
	writeValidDataProduct(t, tmpDir, "test-dp.yaml")
	writeValidSourceApp(t, tmpDir, "test-sa.yaml")

	ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	mockClient := createMockDataProductClient(&MockDataProductWarningTransport{})

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, true, false, 1)

	if err != nil {
		t.Errorf("Expected validation to succeed with warnings, but got error: %v", err)
	}
}

func TestValidateDataProductsWithClient_ConcurrencyLimits(t *testing.T) {
	tmpDir := t.TempDir()
	writeValidDataProduct(t, tmpDir, "test-dp.yaml")
	writeValidSourceApp(t, tmpDir, "test-sa.yaml")

	var logOutput bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logOutput, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	ctx := logging.ContextWithLogger(context.Background(), logger)

	mockClient := createMockDataProductClient(&MockDataProductSuccessTransport{})

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 20)

	if err != nil {
		t.Errorf("Expected validation to succeed, but got error: %v", err)
	}

	logStr := logOutput.String()
	if !strings.Contains(logStr, "concurrency set to > 10, limited to 10") {
		t.Error("Expected concurrency limit log message")
	}

	logOutput.Reset()
	err = ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 0)

	if err != nil {
		t.Errorf("Expected validation to succeed, but got error: %v", err)
	}

	logStr = logOutput.String()
	if !strings.Contains(logStr, "concurrency set to < 1, increased to 1") {
		t.Error("Expected concurrency increase log message")
	}
}

func TestValidateDataProductsWithClient_NoLoggerInContext(t *testing.T) {
	tmpDir := t.TempDir()
	writeValidDataProduct(t, tmpDir, "test-dp.yaml")
	writeValidSourceApp(t, tmpDir, "test-sa.yaml")

	ctx := context.Background()

	mockClient := createMockDataProductClient(&MockDataProductSuccessTransport{})

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1)

	t.Logf("Function completed with error: %v", err)
}

func writeValidDataProduct(t *testing.T, dir, filename string) {
	validDP := `apiVersion: v1
resourceType: data-product
resourceName: dbc68917-3bcb-4640-a165-483de5fc033c
data:
  name: test_data_product
  description: Test data product for validation
  owner: test-team@example.com
  sourceApplications:
    - $ref: ./test-sa.yaml
  eventSpecifications:
    - resourceName: b57f712e-ee43-4e10-8b36-48a0f6f2e628
      name: Test Event
      description: A test event specification
      event:
        source: iglu:com.example/test_event/jsonschema/1-0-0
        schema:
          type: object
          additionalProperties: false
          properties:
            testField:
              type: string
          required: ["testField"]
      entities:
        tracked: []
        enriched: []
      triggers:
        - id: a29e6ba7-769a-4919-b9a3-e6bb7bd97d67
          description: Test trigger
      excludedSourceApplications: []
`

	err := os.WriteFile(filepath.Join(dir, filename), []byte(validDP), 0644)
	if err != nil {
		t.Fatalf("Failed to write valid data product file: %v", err)
	}
}

func writeValidSourceApp(t *testing.T, dir, filename string) {
	validSA := `apiVersion: v1
resourceType: source-application
resourceName: c85f1787-6a42-48c5-afeb-455b6380cb62
data:
  name: test_source_app
  description: Test source application
  owner: test-team@example.com
  appIds:
    - test-app-123
  entities:
    tracked: []
    enriched: []
`

	err := os.WriteFile(filepath.Join(dir, filename), []byte(validSA), 0644)
	if err != nil {
		t.Fatalf("Failed to write valid source app file: %v", err)
	}
}

func writeInvalidDataProduct(t *testing.T, dir, filename string) {
	invalidDP := `apiVersion: v1
resourceType: data-product
data:
  description: Invalid data product
`

	err := os.WriteFile(filepath.Join(dir, filename), []byte(invalidDP), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid data product file: %v", err)
	}
}

func createMockDataProductClient(roundTripper http.RoundTripper) *console.ApiClient {
	return &console.ApiClient{
		Http:    &http.Client{Transport: roundTripper},
		Jwt:     "fake-jwt-token",
		BaseUrl: "https://fake-api.example.com/api/msc/v1/organizations/fake-org",
		OrgId:   "fake-org-id",
	}
}

type MockDataProductSuccessTransport struct{}

func (m *MockDataProductSuccessTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch {
	case strings.Contains(req.URL.Host, "iglu.snplow.net") && strings.Contains(req.URL.Path, "/api/schemas"):
		body := `[]`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	case strings.Contains(req.URL.Path, "/data-products/v2") && req.Method == "GET":
		body := `{"data": [], "includes": {"eventSpecs": []}}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	case strings.Contains(req.URL.Path, "/source-apps/v1") && req.Method == "GET":
		body := `{"data": []}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	case strings.Contains(req.URL.Path, "/data-structures/v1") && req.Method == "GET":
		body := `[]`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	case strings.Contains(req.URL.Path, "/event-specs/v1/compatibility") && req.Method == "POST":
		body := `{"status": "compatible", "sources": [], "message": ""}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	default:
		body := `{"success": true}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	}
}

type MockDataProductNetworkFailureTransport struct{}

func (m *MockDataProductNetworkFailureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("network connection failed")
}

type MockDataProductCompatFailureTransport struct{}

func (m *MockDataProductCompatFailureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Printf("[MOCK] Request: %s %s\n", req.Method, req.URL.String())

	switch {
	case strings.Contains(req.URL.Host, "iglu.snplow.net") && strings.Contains(req.URL.Path, "/api/schemas"):
		body := `[]`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	case strings.Contains(req.URL.Path, "/data-products/v2") && req.Method == "GET":
		body := `{"data": [], "includes": {"eventSpecs": []}}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	case strings.Contains(req.URL.Path, "/source-apps/v1") && req.Method == "GET":
		body := `{"data": []}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	case strings.Contains(req.URL.Path, "/data-structures/v1") && req.Method == "GET":
		body := `[]`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	case strings.Contains(req.URL.Path, "/event-specs/v1/compatibility") && req.Method == "POST":
		body := `{"status": "incompatible", "sources": [{"source": "iglu:com.example/test_event/jsonschema/1-0-0", "status": "incompatible", "properties": {}}], "message": "Schema compatibility error"}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	default:
		body := `{"success": true}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	}
}

type MockDataProductWarningTransport struct{}

func (m *MockDataProductWarningTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch {
	case strings.Contains(req.URL.Host, "iglu.snplow.net") && strings.Contains(req.URL.Path, "/api/schemas"):
		// Mock Iglu Central listing - return empty array
		body := `[]`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	case strings.Contains(req.URL.Path, "/data-products/v2") && req.Method == "GET":
		body := `{"data": [], "includes": {"eventSpecs": []}}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	case strings.Contains(req.URL.Path, "/source-apps/v1") && req.Method == "GET":
		body := `{"data": []}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	case strings.Contains(req.URL.Path, "/data-structures/v1") && req.Method == "GET":
		body := `[]`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	case strings.Contains(req.URL.Path, "/event-specs/v1/compatibility") && req.Method == "POST":
		body := `{"status": "compatible", "sources": [{"source": "test", "status": "compatible", "properties": {"warning": "Minor compatibility warning"}}], "message": ""}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil

	default:
		// Return success for any other endpoints
		body := `{"success": true}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	}
}
