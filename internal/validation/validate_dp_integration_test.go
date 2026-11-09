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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/download"
	"github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/util"
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

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, false)

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

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, false)

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

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, false)

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

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, true, 1, false)

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

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, true, false, 1, false)

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

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 20, false)

	if err != nil {
		t.Errorf("Expected validation to succeed, but got error: %v", err)
	}

	logStr := logOutput.String()
	if !strings.Contains(logStr, "concurrency set to > 10, limited to 10") {
		t.Error("Expected concurrency limit log message")
	}

	logOutput.Reset()
	err = ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 0, false)

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

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, false)

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

const versionRemoteDataProducts = `{
  "data": [{
    "id": "dbc68917-3bcb-4640-a165-483de5fc033c",
    "name": "checkout",
    "status": "draft",
    "sourceApplications": [],
    "eventSpecs": [{"id": "b57f712e-ee43-4e10-8b36-48a0f6f2e628"}]
  }],
  "includes": {"eventSpecs": [{
    "id": "b57f712e-ee43-4e10-8b36-48a0f6f2e628",
    "name": "Checkout Started",
    "description": "A checkout started",
    "status": "published",
    "version": 3,
    "sourceApplications": [],
    "dataProductId": "dbc68917-3bcb-4640-a165-483de5fc033c",
    "event": {
      "source": "iglu:com.example/checkout/jsonschema/1-0-0",
      "schema": {"type": "object", "additionalProperties": false, "properties": {"a": {"type": "string", "maxLength": 10}}, "required": ["a"]}
    },
    "entities": {"tracked": [{"source": "iglu:com.example/cart/jsonschema/1-0-0", "minCardinality": 0, "comment": "a note"}]}
  }]}
}`

const versionLocalDataProduct = `apiVersion: v1
resourceType: data-product
resourceName: dbc68917-3bcb-4640-a165-483de5fc033c
data:
  name: checkout
  sourceApplications: []
  eventSpecifications:
    # the spec Console knows
    - resourceName: b57f712e-ee43-4e10-8b36-48a0f6f2e628
      name: Checkout Started
      version: 3
      description: A checkout started
      event:
        source: iglu:com.example/checkout/jsonschema/1-0-1
        schema: {type: object, additionalProperties: false, properties: {a: {type: string, maxLength: 10}}, required: [a]}
      entities:
        tracked:
          - source: iglu:com.example/cart/jsonschema/1-0-0
            minCardinality: 0
        enriched: []
`

type MockDataProductRemoteTransport struct {
	dataProducts string
}

func (m *MockDataProductRemoteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.Path, "/data-products/v2") && req.Method == "GET" {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(m.dataProducts)),
			Request:    req,
		}, nil
	}
	return (&MockDataProductSuccessTransport{}).RoundTrip(req)
}

func TestValidateDataProductsWithClient_DownloadThenValidate(t *testing.T) {
	for _, ext := range []string{"yaml", "json"} {
		t.Run(ext, func(t *testing.T) {
			tmpDir := t.TempDir()
			ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewJSONHandler(io.Discard, nil)))
			mockClient := createMockDataProductClient(&MockDataProductRemoteTransport{dataProducts: versionRemoteDataProducts})

			files := util.Files{DataProductsLocation: tmpDir, SourceAppsLocation: "source-apps", ExtentionPreference: ext, ImagesLocation: "images"}
			err := download.DownloadDataProductsAndRelatedResources(files, ctx, mockClient, false)
			if err != nil {
				t.Fatalf("download failed: %v", err)
			}

			written, err := os.ReadFile(filepath.Join(tmpDir, "checkout."+ext))
			if err != nil {
				t.Fatalf("failed to read downloaded file: %v", err)
			}
			wantVersion := "version: 3"
			if ext == "json" {
				wantVersion = `"version": 3`
			}
			if !strings.Contains(string(written), wantVersion) {
				t.Errorf("downloaded file should carry the Console version, got:\n%s", written)
			}

			err = ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, false)
			if err != nil {
				t.Errorf("Expected an unchanged download to validate, but got error: %v", err)
			}
		})
	}
}

func TestValidateDataProductsWithClient_VersionMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	dpFile := filepath.Join(tmpDir, "checkout.yaml")
	if err := os.WriteFile(dpFile, []byte(versionLocalDataProduct), 0644); err != nil {
		t.Fatalf("Failed to write data product file: %v", err)
	}

	var logOutput bytes.Buffer
	ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewTextHandler(&logOutput, nil)))
	mockClient := createMockDataProductClient(&MockDataProductRemoteTransport{dataProducts: versionRemoteDataProducts})

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, false)

	if err == nil || !strings.Contains(err.Error(), "validation failed 1 errors") {
		t.Fatalf("Expected a version validation failure, got: %v", err)
	}
	wantLog := `path=/data/eventSpecifications/0 errors="Checkout Started: version 3 needs a bump\nchanged: event, entities\nexpected version: 4\nrun ` + "`snowplow-cli dp validate --fix`" + ` to update it\n"`
	if !strings.Contains(logOutput.String(), wantLog) {
		t.Errorf("Expected version error in logs\nwant: %s\ngot: %s", wantLog, logOutput.String())
	}
}

func TestValidateDataProductsWithClient_VersionFix(t *testing.T) {
	tmpDir := t.TempDir()
	dpFile := filepath.Join(tmpDir, "checkout.yaml")
	if err := os.WriteFile(dpFile, []byte(versionLocalDataProduct), 0644); err != nil {
		t.Fatalf("Failed to write data product file: %v", err)
	}

	ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewJSONHandler(io.Discard, nil)))
	mockClient := createMockDataProductClient(&MockDataProductRemoteTransport{dataProducts: versionRemoteDataProducts})

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, true)
	if err != nil {
		t.Fatalf("Expected --fix to succeed, got: %v", err)
	}

	fixed, err := os.ReadFile(dpFile)
	if err != nil {
		t.Fatalf("failed to read fixed file: %v", err)
	}
	want := strings.Replace(versionLocalDataProduct, "version: 3", "version: 4", 1)
	if string(fixed) != want {
		t.Errorf("Expected only the version to change\nwant:\n%s\ngot:\n%s", want, fixed)
	}

	err = ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, false)
	if err != nil {
		t.Errorf("Expected validate to pass after --fix, got: %v", err)
	}
}

const baseDataProduct = `apiVersion: v1
resourceType: data-product
resourceName: dbc68917-3bcb-4640-a165-483de5fc033c
data:
  name: checkout
  sourceApplications: []
  eventSpecifications:
    - resourceName: b57f712e-ee43-4e10-8b36-48a0f6f2e628
      name: Checkout Started
      version: %d
      description: A checkout started
      event:
        source: iglu:com.example/checkout/jsonschema/%s
      entities:
        tracked: []
        enriched: []
`

func gitInTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.email=test@example.com", "-c", "user.name=test", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func setupBaseRepo(t *testing.T) (string, string) {
	t.Helper()
	repo := t.TempDir()
	tracking := filepath.Join(repo, "tracking")
	if err := os.MkdirAll(tracking, 0755); err != nil {
		t.Fatal(err)
	}
	dpFile := filepath.Join(tracking, "checkout.yaml")

	gitInTest(t, repo, "init", "-q", "-b", "main")
	if err := os.WriteFile(dpFile, []byte(fmt.Sprintf(baseDataProduct, 3, "1-0-0")), 0644); err != nil {
		t.Fatal(err)
	}
	gitInTest(t, repo, "add", ".")
	gitInTest(t, repo, "commit", "-q", "-m", "base")
	gitInTest(t, repo, "checkout", "-q", "-b", "feature")

	gitInTest(t, repo, "checkout", "-q", "main")
	if err := os.WriteFile(dpFile, []byte(fmt.Sprintf(baseDataProduct, 8, "3-0-0")), 0644); err != nil {
		t.Fatal(err)
	}
	gitInTest(t, repo, "commit", "-q", "-am", "main moves on")
	gitInTest(t, repo, "checkout", "-q", "feature")

	return tracking, dpFile
}

func TestValidateDataProductsAgainstBase_UnchangedUsesMergeBase(t *testing.T) {
	tracking, _ := setupBaseRepo(t)
	ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewJSONHandler(io.Discard, nil)))

	err := ValidateDataProductsAgainstBase(ctx, []string{tracking}, tracking, false, "main", false)
	if err != nil {
		t.Errorf("Expected an unchanged spec to match the merge base, not the tip of main, got: %v", err)
	}
}

func TestValidateDataProductsAgainstBase_StructuralChangeNeedsBump(t *testing.T) {
	tracking, dpFile := setupBaseRepo(t)
	if err := os.WriteFile(dpFile, []byte(fmt.Sprintf(baseDataProduct, 3, "1-0-1")), 0644); err != nil {
		t.Fatal(err)
	}

	var logOutput bytes.Buffer
	ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewTextHandler(&logOutput, nil)))

	t.Chdir(tracking)
	err := ValidateDataProductsAgainstBase(ctx, []string{"."}, tracking, false, "main", false)
	if err == nil || !strings.Contains(err.Error(), "validation failed 1 errors") {
		t.Fatalf("Expected a version validation failure, got: %v", err)
	}
	wantLog := `path=/data/eventSpecifications/0 errors="Checkout Started: version 3 needs a bump\nchanged: event\nexpected version: 4\nrun ` + "`snowplow-cli dp validate --fix`" + ` to update it\n"`
	if !strings.Contains(logOutput.String(), wantLog) {
		t.Errorf("Expected version error in logs\nwant: %s\ngot: %s", wantLog, logOutput.String())
	}
}

func TestValidateDataProductsAgainstBase_Fix(t *testing.T) {
	tracking, dpFile := setupBaseRepo(t)
	if err := os.WriteFile(dpFile, []byte(fmt.Sprintf(baseDataProduct, 3, "1-0-1")), 0644); err != nil {
		t.Fatal(err)
	}
	ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewJSONHandler(io.Discard, nil)))

	err := ValidateDataProductsAgainstBase(ctx, []string{tracking}, tracking, false, "main", true)
	if err != nil {
		t.Fatalf("Expected --fix to succeed, got: %v", err)
	}
	fixed, err := os.ReadFile(dpFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(fixed) != fmt.Sprintf(baseDataProduct, 4, "1-0-1") {
		t.Errorf("Expected only the version to change, got:\n%s", fixed)
	}

	err = ValidateDataProductsAgainstBase(ctx, []string{tracking}, tracking, false, "main", false)
	if err != nil {
		t.Errorf("Expected validate to pass after --fix, got: %v", err)
	}
}

func TestValidateDataProductsAgainstBase_InvalidRef(t *testing.T) {
	tracking, _ := setupBaseRepo(t)
	ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewJSONHandler(io.Discard, nil)))

	err := ValidateDataProductsAgainstBase(ctx, []string{tracking}, tracking, false, "origin/does-not-exist", false)
	if err == nil || !strings.Contains(err.Error(), "merge-base") {
		t.Errorf("Expected a git error for an unknown ref, got: %v", err)
	}
}
