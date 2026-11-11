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

	"github.com/snowplow/snowplow-cli/internal/amend"
	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/download"
	"github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/model"
	"github.com/snowplow/snowplow-cli/internal/release"
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
			err := download.DownloadDataProductsAndRelatedResources(files, ctx, mockClient, false, true)
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
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
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
	wantLog := `path=/data/eventSpecifications/0 errors="Checkout Started: version 3 needs a bump\nchanged: event\nexpected version: 4\nrun ` + "`snowplow-cli dp validate offline --base-ref main --fix`" + ` to update it\n"`
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

const twoSpecsRemoteDataProducts = `{
  "data": [{
    "id": "dbc68917-3bcb-4640-a165-483de5fc033c",
    "name": "checkout",
    "status": "draft",
    "sourceApplications": [],
    "eventSpecs": [{"id": "b57f712e-ee43-4e10-8b36-48a0f6f2e628"}, {"id": "c67f712e-ee43-4e10-8b36-48a0f6f2e629"}]
  }],
  "includes": {"eventSpecs": [
    {"id": "b57f712e-ee43-4e10-8b36-48a0f6f2e628", "name": "Checkout Started", "status": "published", "version": 3, "sourceApplications": [], "dataProductId": "dbc68917-3bcb-4640-a165-483de5fc033c", "entities": {"tracked": [], "enriched": []}},
    {"id": "c67f712e-ee43-4e10-8b36-48a0f6f2e629", "name": "Checkout Finished", "status": "published", "version": 10, "sourceApplications": [], "dataProductId": "dbc68917-3bcb-4640-a165-483de5fc033c", "entities": {"tracked": [], "enriched": []}}
  ]}
}`

const twoSpecsLocalDataProduct = `apiVersion: v1
resourceType: data-product
resourceName: dbc68917-3bcb-4640-a165-483de5fc033c
data:
  name: checkout
  sourceApplications: []
  eventSpecifications:
    - resourceName: b57f712e-ee43-4e10-8b36-48a0f6f2e628
      name: Checkout Started Renamed
      version: %d
      entities:
        tracked: []
        enriched: []
    - {resourceName: c67f712e-ee43-4e10-8b36-48a0f6f2e629, name: Checkout Finished Renamed, version: %d, entities: {tracked: [], enriched: []}}
`

func TestValidateDataProductsWithClient_TwoMismatchesInOneFile(t *testing.T) {
	tmpDir := t.TempDir()
	dpFile := filepath.Join(tmpDir, "checkout.yaml")
	if err := os.WriteFile(dpFile, []byte(fmt.Sprintf(twoSpecsLocalDataProduct, 3, 9)), 0644); err != nil {
		t.Fatalf("Failed to write data product file: %v", err)
	}

	var logOutput bytes.Buffer
	ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewTextHandler(&logOutput, nil)))
	mockClient := createMockDataProductClient(&MockDataProductRemoteTransport{dataProducts: twoSpecsRemoteDataProducts})

	err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, false)
	if err == nil || !strings.Contains(err.Error(), "validation failed 2 errors") {
		t.Fatalf("Expected two version errors, got: %v", err)
	}

	err = ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, true)
	if err != nil {
		t.Fatalf("Expected --fix to succeed, got: %v", err)
	}
	fixed, err := os.ReadFile(dpFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(fixed) != fmt.Sprintf(twoSpecsLocalDataProduct, 4, 11) {
		t.Errorf("Expected both versions to be fixed, got:\n%s", fixed)
	}
	if strings.Count(logOutput.String(), "updated event specification version") != 2 {
		t.Errorf("Expected one log line per fixed spec, got:\n%s", logOutput.String())
	}

	err = ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, false)
	if err != nil {
		t.Errorf("Expected validate to pass after --fix, got: %v", err)
	}
}

type MockDataProductRecordingTransport struct {
	MockDataProductRemoteTransport
	writes []string
}

func (m *MockDataProductRecordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != "GET" && !strings.Contains(req.URL.Path, "/event-specs/v1/compatibility") {
		m.writes = append(m.writes, req.Method+" "+req.URL.Path)
	}
	return m.MockDataProductRemoteTransport.RoundTrip(req)
}

func TestSyncWorkflow_StoppedByVersionMismatchBeforeWriting(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "checkout.yaml"), []byte(versionLocalDataProduct), 0644); err != nil {
		t.Fatalf("Failed to write data product file: %v", err)
	}

	ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewJSONHandler(io.Discard, nil)))
	transport := &MockDataProductRecordingTransport{MockDataProductRemoteTransport: MockDataProductRemoteTransport{dataProducts: versionRemoteDataProducts}}
	mockClient := createMockDataProductClient(transport)

	files, err := util.MaybeResourcesfromPaths([]string{tmpDir})
	if err != nil {
		t.Fatal(err)
	}
	changes, err := release.FindChanges(ctx, mockClient, files)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes.VersionMismatches) != 1 {
		t.Fatalf("expected the sync change set to carry 1 version mismatch, got %+v", changes.VersionMismatches)
	}

	err = Validate(ctx, mockClient, files, []string{tmpDir}, tmpDir, false, false, changes.IdToFileName, changes.VersionMismatches, 1)
	if err == nil || !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("Expected the validation step of sync to fail, got: %v", err)
	}
	if len(transport.writes) != 0 {
		t.Errorf("Expected nothing written before the validation step fails, got: %v", transport.writes)
	}
}

func downloadInTest(t *testing.T, dir string, ext string, withVersions bool) string {
	t.Helper()
	ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewJSONHandler(io.Discard, nil)))
	mockClient := createMockDataProductClient(&MockDataProductRemoteTransport{dataProducts: versionRemoteDataProducts})
	files := util.Files{DataProductsLocation: dir, SourceAppsLocation: "source-apps", ExtentionPreference: ext, ImagesLocation: "images"}
	if err := download.DownloadDataProductsAndRelatedResources(files, ctx, mockClient, false, withVersions); err != nil {
		t.Fatalf("download failed: %v", err)
	}
	written, err := os.ReadFile(filepath.Join(dir, "checkout."+ext))
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	return string(written)
}

func TestDownload_DefaultMatchesOutputBeforeVersions(t *testing.T) {
	for _, ext := range []string{"yaml", "json"} {
		t.Run(ext, func(t *testing.T) {
			before, err := os.ReadFile(filepath.Join("..", "testdata", "validation", "download-before-versions."+ext))
			if err != nil {
				t.Fatal(err)
			}

			written := downloadInTest(t, t.TempDir(), ext, false)

			if written != string(before) {
				t.Errorf("default download should write what it wrote before versions existed\nwant:\n%s\ngot:\n%s", before, written)
			}
		})
	}
}

func TestDownload_KeepsVersionsOnlyWhenOptedIn(t *testing.T) {
	tests := []struct {
		name         string
		existing     string
		withVersions bool
		wantVersion  bool
	}{
		{name: "flag writes versions", withVersions: true, wantVersion: true},
		{name: "existing file with versions keeps them", existing: fmt.Sprintf(baseDataProduct, 1, "1-0-0"), wantVersion: true},
		{name: "existing file without versions writes none", existing: strings.Replace(fmt.Sprintf(baseDataProduct, 1, "1-0-0"), "      version: 1\n", "", 1), wantVersion: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			if tt.existing != "" {
				if err := os.WriteFile(filepath.Join(tmpDir, "checkout.yaml"), []byte(tt.existing), 0644); err != nil {
					t.Fatal(err)
				}
			}

			written := downloadInTest(t, tmpDir, "yaml", tt.withVersions)

			if got := strings.Contains(written, "version: 3"); got != tt.wantVersion {
				t.Errorf("version written = %v, want %v, got:\n%s", got, tt.wantVersion, written)
			}
			if !tt.wantVersion && strings.Contains(written, "version:") {
				t.Errorf("no version should be written, got:\n%s", written)
			}
		})
	}
}

func TestValidateDataProductsAgainstBase_FindsSpecsAcrossBaseFiles(t *testing.T) {
	newSpec := `    - resourceName: d77f712e-ee43-4e10-8b36-48a0f6f2e630
      name: Checkout Abandoned
      version: %d
`
	emptyDataProduct := `apiVersion: v1
resourceType: data-product
resourceName: ec68917a-3bcb-4640-a165-483de5fc033d
data:
  name: other
  sourceApplications: []
  eventSpecifications: []
`

	tests := []struct {
		name    string
		arrange func(t *testing.T, tracking string, dpFile string)
		wantErr bool
	}{
		{
			name: "renamed file keeps its versions",
			arrange: func(t *testing.T, tracking string, dpFile string) {
				if err := os.Rename(dpFile, filepath.Join(tracking, "renamed.yaml")); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "file moved to another directory keeps its versions",
			arrange: func(t *testing.T, tracking string, dpFile string) {
				if err := os.MkdirAll(filepath.Join(tracking, "nested"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(dpFile, filepath.Join(tracking, "nested", "checkout.yaml")); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "spec moved to another data product file keeps its version",
			arrange: func(t *testing.T, tracking string, dpFile string) {
				moved := strings.Replace(fmt.Sprintf(baseDataProduct, 3, "1-0-0"), "dbc68917-3bcb-4640-a165-483de5fc033c", "fd68917a-3bcb-4640-a165-483de5fc033e", 1)
				if err := os.WriteFile(filepath.Join(tracking, "moved.yaml"), []byte(moved), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(dpFile, []byte(emptyDataProduct), 0644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "truly new spec at 0 passes",
			arrange: func(t *testing.T, tracking string, dpFile string) {
				if err := os.WriteFile(dpFile, []byte(fmt.Sprintf(baseDataProduct, 3, "1-0-0")+fmt.Sprintf(newSpec, 0)), 0644); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "truly new spec not at 0 fails",
			arrange: func(t *testing.T, tracking string, dpFile string) {
				if err := os.WriteFile(dpFile, []byte(fmt.Sprintf(baseDataProduct, 3, "1-0-0")+fmt.Sprintf(newSpec, 1)), 0644); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracking, dpFile := setupBaseRepo(t)
			tt.arrange(t, tracking, dpFile)
			ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewJSONHandler(io.Discard, nil)))

			err := ValidateDataProductsAgainstBase(ctx, []string{tracking}, tracking, false, "main", false)

			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, want error %v", err, tt.wantErr)
			}
		})
	}
}

func TestFixEventSpecVersions_LogsFilesInOrder(t *testing.T) {
	tmpDir := t.TempDir()
	names := []string{"b.yaml", "a.yaml", "c.yaml"}
	var mismatches []model.EventSpecVersionMismatch
	for _, name := range names {
		f := filepath.Join(tmpDir, name)
		if err := os.WriteFile(f, []byte(fmt.Sprintf(baseDataProduct, 3, "1-0-0")), 0644); err != nil {
			t.Fatal(err)
		}
		mismatches = append(mismatches, model.EventSpecVersionMismatch{FileName: f, Index: 0, Name: name, Declared: 3, Expected: 4})
	}

	var logOutput bytes.Buffer
	ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewTextHandler(&logOutput, nil)))

	if err := fixEventSpecVersions(ctx, mismatches, nil); err != nil {
		t.Fatal(err)
	}

	a := strings.Index(logOutput.String(), "name=a.yaml")
	b := strings.Index(logOutput.String(), "name=b.yaml")
	c := strings.Index(logOutput.String(), "name=c.yaml")
	if a < 0 || a >= b || b >= c {
		t.Errorf("expected one log line per file in sorted order, got:\n%s", logOutput.String())
	}
}

func TestDownload_UnreadableFileKeepsOtherVersions(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "checkout.yaml"), []byte(fmt.Sprintf(baseDataProduct, 1, "1-0-0")), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "broken.yaml"), []byte("key: [unclosed\n"), 0644); err != nil {
		t.Fatal(err)
	}

	written := downloadInTest(t, tmpDir, "yaml", false)

	if !strings.Contains(written, "version: 3") {
		t.Errorf("a broken file should not drop the versions of another data product, got:\n%s", written)
	}
}

func TestDownload_SymbolicLinkDirectoryKeepsVersions(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "data-products")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "checkout.yaml"), []byte(fmt.Sprintf(baseDataProduct, 1, "1-0-0")), 0644); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(tmpDir, "linked")
	if err := os.Symlink(target, linked); err != nil {
		t.Fatal(err)
	}

	written := downloadInTest(t, linked, "yaml", false)

	if !strings.Contains(written, "version: 3") {
		t.Errorf("a data products directory reached through a symbolic link should keep its versions, got:\n%s", written)
	}
}

func TestDownload_UnreadableFileWithoutVersionsMatchesOutputBeforeVersions(t *testing.T) {
	before, err := os.ReadFile(filepath.Join("..", "testdata", "validation", "download-before-versions.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "broken.json"), []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}

	written := downloadInTest(t, tmpDir, "yaml", false)

	if written != string(before) {
		t.Errorf("default download should write what it wrote before versions existed\nwant:\n%s\ngot:\n%s", before, written)
	}
}

func TestValidateDataProductsAgainstBase_SymbolicLinksAtBase(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	repo := t.TempDir()
	tracking := filepath.Join(repo, "tracking")
	shared := filepath.Join(repo, "shared")
	for _, dir := range []string{tracking, shared} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(shared, "checkout.yaml"), []byte(fmt.Sprintf(baseDataProduct, 3, "1-0-0")), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "shared", "checkout.yaml"), filepath.Join(tracking, "checkout.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "..", "outside.yaml"), filepath.Join(tracking, "outside.yaml")); err != nil {
		t.Fatal(err)
	}
	gitInTest(t, repo, "init", "-q", "-b", "main")
	gitInTest(t, repo, "add", ".")
	gitInTest(t, repo, "commit", "-q", "-m", "base")

	outsideNow := `apiVersion: v1
resourceType: data-product
resourceName: ec68917a-3bcb-4640-a165-483de5fc033d
data:
  name: other
  sourceApplications: []
  eventSpecifications:
    - resourceName: d77f712e-ee43-4e10-8b36-48a0f6f2e630
      name: Checkout Abandoned
      version: 0
`
	if err := os.Remove(filepath.Join(tracking, "outside.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tracking, "outside.yaml"), []byte(outsideNow), 0644); err != nil {
		t.Fatal(err)
	}
	ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewJSONHandler(io.Discard, nil)))

	err := ValidateDataProductsAgainstBase(ctx, []string{tracking}, tracking, false, "main", false)
	if err != nil {
		t.Fatalf("Expected the linked file to keep its version and the outside link to be skipped, got: %v", err)
	}

	if err := os.Remove(filepath.Join(tracking, "checkout.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tracking, "checkout.yaml"), []byte(fmt.Sprintf(baseDataProduct, 3, "1-0-1")), 0644); err != nil {
		t.Fatal(err)
	}
	err = ValidateDataProductsAgainstBase(ctx, []string{tracking}, tracking, false, "main", false)
	if err == nil || !strings.Contains(err.Error(), "validation failed 1 errors") {
		t.Errorf("Expected the base version behind the link to be compared, got: %v", err)
	}
}

func TestValidateDataProductsWithClient_AddedSpecAtZeroPasses(t *testing.T) {
	tmpDir := t.TempDir()
	downloadInTest(t, tmpDir, "yaml", true)
	dpFile := filepath.Join(tmpDir, "checkout.yaml")

	if err := amend.AddEventSpecsToFile([]string{"Checkout Abandoned"}, dpFile); err != nil {
		t.Fatalf("add-es failed: %v", err)
	}
	written, err := os.ReadFile(dpFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(written), "name: Checkout Abandoned\n          version: 0\n") {
		t.Fatalf("expected the added spec to declare version 0, got:\n%s", written)
	}

	ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewJSONHandler(io.Discard, nil)))
	mockClient := createMockDataProductClient(&MockDataProductRemoteTransport{dataProducts: versionRemoteDataProducts})

	err = ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, false)
	if err != nil {
		t.Errorf("Expected a newly added spec at version 0 to validate, got: %v", err)
	}
}

const missingVersionsLocalYaml = `apiVersion: v1
resourceType: data-product
resourceName: dbc68917-3bcb-4640-a165-483de5fc033c
data:
  name: checkout
  sourceApplications: []
  eventSpecifications:
    # unchanged in Console
    - resourceName: b57f712e-ee43-4e10-8b36-48a0f6f2e628
      name: Checkout Started # café
%[1]s      entities:
        tracked: []
        enriched: []
    - {resourceName: c67f712e-ee43-4e10-8b36-48a0f6f2e629, name: Checkout Finished Renamed, %[2]sentities: {tracked: [], enriched: []}}
    - {resourceName: d77f712e-ee43-4e10-8b36-48a0f6f2e630, name: Pedido añadido 🛒, %[3]sentities: {tracked: [], enriched: []}}
    - resourceName: e87f712e-ee43-4e10-8b36-48a0f6f2e631
      name: Checkout Abandoned
      version: %[4]d
      entities:
        tracked: []
        enriched: []
`

const missingVersionsLocalJson = `{
  "apiVersion": "v1",
  "resourceType": "data-product",
  "resourceName": "dbc68917-3bcb-4640-a165-483de5fc033c",
  "data": {
    "name": "checkout",
    "sourceApplications": [],
    "eventSpecifications": [
      {
        "resourceName": "b57f712e-ee43-4e10-8b36-48a0f6f2e628",
        "name": "Checkout Started"%[1]s,
        "entities": {"tracked": [], "enriched": []}
      },
      {"resourceName": "c67f712e-ee43-4e10-8b36-48a0f6f2e629", "name": "Checkout Finished Renamed"%[2]s, "entities": {"tracked": [], "enriched": []}},
      {"resourceName": "d77f712e-ee43-4e10-8b36-48a0f6f2e630", "name": "Pedido añadido 🛒"%[3]s, "entities": {"tracked": [], "enriched": []}},
      {
        "resourceName": "e87f712e-ee43-4e10-8b36-48a0f6f2e631",
        "name": "Checkout Abandoned",
        "version": %[4]d,
        "entities": {"tracked": [], "enriched": []}
      }
    ]
  }
}
`

const missingVersionsFixedJson = `{
  "apiVersion": "v1",
  "resourceType": "data-product",
  "resourceName": "dbc68917-3bcb-4640-a165-483de5fc033c",
  "data": {
    "name": "checkout",
    "sourceApplications": [],
    "eventSpecifications": [
      {
        "resourceName": "b57f712e-ee43-4e10-8b36-48a0f6f2e628",
        "name": "Checkout Started",
        "version": 3,
        "entities": {
          "tracked": [],
          "enriched": []
        }
      },
      {
        "resourceName": "c67f712e-ee43-4e10-8b36-48a0f6f2e629",
        "name": "Checkout Finished Renamed",
        "version": 11,
        "entities": {
          "tracked": [],
          "enriched": []
        }
      },
      {
        "resourceName": "d77f712e-ee43-4e10-8b36-48a0f6f2e630",
        "name": "Pedido añadido 🛒",
        "version": 0,
        "entities": {
          "tracked": [],
          "enriched": []
        }
      },
      {
        "resourceName": "e87f712e-ee43-4e10-8b36-48a0f6f2e631",
        "name": "Checkout Abandoned",
        "version": 0,
        "entities": {
          "tracked": [],
          "enriched": []
        }
      }
    ]
  }
}
`

func TestValidateDataProductsWithClient_FixAddsMissingVersions(t *testing.T) {
	for _, tt := range []struct {
		ext      string
		original string
		want     string
	}{
		{
			ext:      "yaml",
			original: fmt.Sprintf(missingVersionsLocalYaml, "", "", "", 2),
			want:     fmt.Sprintf(missingVersionsLocalYaml, "      version: 3\n", "version: 11, ", "version: 0, ", 0),
		},
		{
			ext:      "json",
			original: fmt.Sprintf(missingVersionsLocalJson, "", "", "", 2),
			want:     missingVersionsFixedJson,
		},
	} {
		t.Run(tt.ext, func(t *testing.T) {
			tmpDir := t.TempDir()
			dpFile := filepath.Join(tmpDir, "checkout."+tt.ext)
			if err := os.WriteFile(dpFile, []byte(tt.original), 0644); err != nil {
				t.Fatal(err)
			}

			var logOutput bytes.Buffer
			ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewTextHandler(&logOutput, nil)))
			mockClient := createMockDataProductClient(&MockDataProductRemoteTransport{dataProducts: twoSpecsRemoteDataProducts})

			err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, true)
			if err != nil {
				t.Fatalf("Expected --fix to succeed, got: %v", err)
			}
			fixed, err := os.ReadFile(dpFile)
			if err != nil {
				t.Fatal(err)
			}
			if string(fixed) != tt.want {
				t.Errorf("Expected only the versions to change\nwant:\n%s\ngot:\n%s", tt.want, fixed)
			}
			if strings.Count(logOutput.String(), "added event specification version") != 3 || strings.Count(logOutput.String(), "updated event specification version") != 1 {
				t.Errorf("Expected one log line per added or updated version, got:\n%s", logOutput.String())
			}

			err = ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, false)
			if err != nil {
				t.Errorf("Expected validate to pass after --fix, got: %v", err)
			}
			err = ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, true)
			if err != nil {
				t.Errorf("Expected a second --fix to succeed, got: %v", err)
			}
			again, err := os.ReadFile(dpFile)
			if err != nil {
				t.Fatal(err)
			}
			if string(again) != tt.want {
				t.Errorf("Expected a second --fix to change nothing, got:\n%s", again)
			}
		})
	}
}

func TestValidateDataProductsWithClient_NoVersionsWithoutFixUnchanged(t *testing.T) {
	for _, tt := range []struct {
		ext      string
		original string
	}{
		{ext: "yaml", original: strings.Replace(fmt.Sprintf(missingVersionsLocalYaml, "", "", "", 0), "      version: 0\n", "", 1)},
		{ext: "json", original: strings.Replace(fmt.Sprintf(missingVersionsLocalJson, "", "", "", 0), "        \"version\": 0,\n", "", 1)},
	} {
		t.Run(tt.ext, func(t *testing.T) {
			tmpDir := t.TempDir()
			dpFile := filepath.Join(tmpDir, "checkout."+tt.ext)
			if err := os.WriteFile(dpFile, []byte(tt.original), 0644); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(tt.original, "version") {
				t.Fatalf("fixture should declare no versions:\n%s", tt.original)
			}

			var logOutput bytes.Buffer
			ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewTextHandler(&logOutput, nil)))
			mockClient := createMockDataProductClient(&MockDataProductRemoteTransport{dataProducts: twoSpecsRemoteDataProducts})

			err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, false)
			if err != nil {
				t.Fatalf("Expected validate without versions to pass, got: %v", err)
			}
			after, err := os.ReadFile(dpFile)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != tt.original {
				t.Errorf("Expected the file to be left untouched, got:\n%s", after)
			}
			if strings.Contains(logOutput.String(), "version") {
				t.Errorf("Expected no version output without --fix, got:\n%s", logOutput.String())
			}
		})
	}
}

func TestValidateDataProductsWithClient_FixOnDownloadMatchesDownloadWithVersions(t *testing.T) {
	for _, ext := range []string{"yaml", "json"} {
		t.Run(ext, func(t *testing.T) {
			tmpDir := t.TempDir()
			downloadInTest(t, tmpDir, ext, false)

			ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewJSONHandler(io.Discard, nil)))
			mockClient := createMockDataProductClient(&MockDataProductRemoteTransport{dataProducts: versionRemoteDataProducts})
			if err := ValidateDataProductsWithClient(ctx, mockClient, []string{tmpDir}, tmpDir, false, false, 1, true); err != nil {
				t.Fatalf("Expected --fix to succeed, got: %v", err)
			}
			fixed, err := os.ReadFile(filepath.Join(tmpDir, "checkout."+ext))
			if err != nil {
				t.Fatal(err)
			}

			downloaded := downloadInTest(t, tmpDir, ext, true)
			if string(fixed) != downloaded {
				t.Errorf("Expected --fix to write the version where dp download --versions does\nfixed:\n%s\ndownloaded:\n%s", fixed, downloaded)
			}
		})
	}
}

func writeRepoFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func missingVersionsBaseFile(resourceName string, specs string) string {
	return fmt.Sprintf(`apiVersion: v1
resourceType: data-product
resourceName: %s
data:
  name: checkout
  sourceApplications: []
  eventSpecifications:
%s`, resourceName, specs)
}

func TestValidateDataProductsAgainstBase_FixAddsMissingVersions(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	repo := t.TempDir()
	tracking := filepath.Join(repo, "tracking")
	if err := os.MkdirAll(tracking, 0755); err != nil {
		t.Fatal(err)
	}

	unversionedAtBase := "    - resourceName: b57f712e-ee43-4e10-8b36-48a0f6f2e628\n      name: Checkout Started\n"
	versionedAtBase := "    - resourceName: c67f712e-ee43-4e10-8b36-48a0f6f2e629\n      name: Checkout Finished\n      version: 3\n"
	duplicated := "    - resourceName: f97f712e-ee43-4e10-8b36-48a0f6f2e632\n      name: Shared\n"
	gitInTest(t, repo, "init", "-q", "-b", "main")
	writeRepoFiles(t, tracking, map[string]string{
		"checkout.yaml": missingVersionsBaseFile("dbc68917-3bcb-4640-a165-483de5fc033c", unversionedAtBase+versionedAtBase+duplicated),
		"other.yaml":    missingVersionsBaseFile("aaa68917-3bcb-4640-a165-483de5fc033c", duplicated),
	})
	gitInTest(t, repo, "add", ".")
	gitInTest(t, repo, "commit", "-q", "-m", "base")

	localWithoutVersions := func(finishedName string, added string) string {
		return missingVersionsBaseFile("dbc68917-3bcb-4640-a165-483de5fc033c",
			unversionedAtBase+
				"    - resourceName: c67f712e-ee43-4e10-8b36-48a0f6f2e629\n      name: "+finishedName+"\n"+
				duplicated+
				"    - resourceName: d77f712e-ee43-4e10-8b36-48a0f6f2e630\n      name: Pedido añadido 🛒\n"+added)
	}
	other := missingVersionsBaseFile("aaa68917-3bcb-4640-a165-483de5fc033c", duplicated)

	for _, tt := range []struct {
		name         string
		finishedName string
		wantFinished int
	}{
		{name: "unchanged since the base", finishedName: "Checkout Finished", wantFinished: 3},
		{name: "renamed since the base", finishedName: "Checkout Finished Renamed", wantFinished: 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			original := localWithoutVersions(tt.finishedName, "")
			writeRepoFiles(t, tracking, map[string]string{"checkout.yaml": original, "other.yaml": other})

			var logOutput bytes.Buffer
			ctx := logging.ContextWithLogger(context.Background(), slog.New(slog.NewTextHandler(&logOutput, nil)))

			err := ValidateDataProductsAgainstBase(ctx, []string{tracking}, tracking, false, "main", false)
			if err != nil {
				t.Fatalf("Expected validate without versions to pass, got: %v", err)
			}
			if strings.Contains(logOutput.String(), "version") {
				t.Errorf("Expected no version output without --fix, got:\n%s", logOutput.String())
			}

			err = ValidateDataProductsAgainstBase(ctx, []string{tracking}, tracking, false, "main", true)
			if err != nil {
				t.Fatalf("Expected --fix to succeed, got: %v", err)
			}
			fixed, err := os.ReadFile(filepath.Join(tracking, "checkout.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			want := strings.Replace(original, "      name: "+tt.finishedName+"\n", fmt.Sprintf("      name: %s\n      version: %d\n", tt.finishedName, tt.wantFinished), 1)
			want = strings.Replace(want, "      name: Pedido añadido 🛒\n", "      name: Pedido añadido 🛒\n      version: 0\n", 1)
			if string(fixed) != want {
				t.Errorf("Expected versions only where they are known offline\nwant:\n%s\ngot:\n%s", want, fixed)
			}
			untouched, err := os.ReadFile(filepath.Join(tracking, "other.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if string(untouched) != other {
				t.Errorf("Expected the file holding only a duplicated spec to be left untouched, got:\n%s", untouched)
			}
			for _, want := range []string{
				`msg="event specification version not added" file=` + filepath.Join(tracking, "checkout.yaml") + ` name="Checkout Started" reason="no version declared at the base, run dp validate --fix against Snowplow Console to add it"`,
				`msg="event specification version not added" file=` + filepath.Join(tracking, "other.yaml") + ` name=Shared reason="defined in more than one file"`,
			} {
				if !strings.Contains(logOutput.String(), want) {
					t.Errorf("Expected log line\nwant: %s\ngot:\n%s", want, logOutput.String())
				}
			}

			err = ValidateDataProductsAgainstBase(ctx, []string{tracking}, tracking, false, "main", false)
			if err != nil {
				t.Errorf("Expected validate to pass after --fix, got: %v", err)
			}
			err = ValidateDataProductsAgainstBase(ctx, []string{tracking}, tracking, false, "main", true)
			if err != nil {
				t.Errorf("Expected a second --fix to succeed, got: %v", err)
			}
			again, err := os.ReadFile(filepath.Join(tracking, "checkout.yaml"))
			if err != nil {
				t.Fatal(err)
			}
			if string(again) != want {
				t.Errorf("Expected a second --fix to change nothing, got:\n%s", again)
			}
		})
	}
}
