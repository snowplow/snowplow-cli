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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSpecificDataStructureVersion_Success(t *testing.T) {
	// Mock server responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/test-org/data-structures/v1/test-hash" {
			// Mock listing response
			listingResp := map[string]any{
				"hash":   "test-hash",
				"vendor": "com.example",
				"name":   "test-schema",
				"format": "jsonschema",
				"meta": map[string]any{
					"hidden":     false,
					"schemaType": "entity",
					"customData": map[string]string{},
				},
				"deployments": []map[string]any{
					{
						"version":     "1-0-0",
						"patchLevel":  0,
						"contentHash": "hash1",
						"env":         "DEV",
						"ts":          "2023-01-01T00:00:00Z",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(listingResp)
			return
		}

		if r.URL.Path == "/api/msc/v1/organizations/test-org/data-structures/v1/schemas/versions" {
			// Mock schema versions response
			schemaVersions := []map[string]any{
				{
					"self": map[string]any{
						"vendor":  "com.example",
						"name":    "test-schema",
						"format":  "jsonschema",
						"version": "1-0-0",
					},
					"schema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"test": map[string]any{
								"type": "string",
							},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(schemaVersions)
			return
		}

		t.Errorf("Unexpected request to %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create client
	cnx := context.Background()
	client := &ApiClient{
		Http:    http.DefaultClient,
		Jwt:     "test-token",
		BaseUrl: server.URL + "/api/msc/v1/organizations/test-org",
		OrgId:   "test-org",
	}

	// Test the function
	result, err := GetSpecificDataStructureVersion(cnx, client, "test-hash", "1-0-0")

	if err != nil {
		t.Fatalf("GetSpecificDataStructureVersion failed: %v", err)
	}

	if result == nil {
		t.Fatalf("Expected result, got nil")
	}

	// Verify the result structure
	if result.ApiVersion != "v1" {
		t.Errorf("Expected ApiVersion 'v1', got '%s'", result.ApiVersion)
	}

	if result.ResourceType != "data-structure" {
		t.Errorf("Expected ResourceType 'data-structure', got '%s'", result.ResourceType)
	}

	// Verify the data contains the schema
	if result.Data == nil {
		t.Fatalf("Expected Data to be populated")
	}

	self, ok := result.Data["self"].(map[string]any)
	if !ok {
		t.Fatalf("Expected self field in Data")
	}

	if self["version"] != "1-0-0" {
		t.Errorf("Expected version '1-0-0', got '%v'", self["version"])
	}
}

func TestGetSpecificDataStructureVersion_VersionNotFound(t *testing.T) {
	// Mock server responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/test-org/data-structures/v1/test-hash" {
			// Mock listing response
			listingResp := map[string]any{
				"hash":   "test-hash",
				"vendor": "com.example",
				"name":   "test-schema",
				"format": "jsonschema",
				"meta": map[string]any{
					"hidden":     false,
					"schemaType": "entity",
					"customData": map[string]string{},
				},
				"deployments": []map[string]any{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(listingResp)
			return
		}

		if r.URL.Path == "/api/msc/v1/organizations/test-org/data-structures/v1/schemas/versions" {
			// Mock schema versions response with different version
			schemaVersions := []map[string]any{
				{
					"self": map[string]any{
						"vendor":  "com.example",
						"name":    "test-schema",
						"format":  "jsonschema",
						"version": "2-0-0", // Different version
					},
					"schema": map[string]any{
						"type": "object",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(schemaVersions)
			return
		}

		t.Errorf("Unexpected request to %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create client
	cnx := context.Background()
	client := &ApiClient{
		Http:    http.DefaultClient,
		Jwt:     "test-token",
		BaseUrl: server.URL + "/api/msc/v1/organizations/test-org",
		OrgId:   "test-org",
	}

	// Test the function with non-existent version
	result, err := GetSpecificDataStructureVersion(cnx, client, "test-hash", "1-0-0")

	if err == nil {
		t.Fatalf("Expected error for non-existent version, got nil")
	}

	if result != nil {
		t.Fatalf("Expected nil result for non-existent version, got %v", result)
	}

	expectedError := "schema data not found for version 1-0-0"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestGetAllDataStructureVersions_Success(t *testing.T) {
	// Mock server responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/test-org/data-structures/v1/test-hash/deployments" {
			// Mock deployments response
			deployments := []Deployment{
				{
					Version:     "1-0-0",
					ContentHash: "hash1",
					Env:         DEV,
				},
				{
					Version:     "2-0-0",
					ContentHash: "hash2",
					Env:         PROD,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(deployments)
			return
		}

		// Mock the GetSpecificDataStructureVersion calls
		if r.URL.Path == "/api/msc/v1/organizations/test-org/data-structures/v1/test-hash" {
			listingResp := map[string]any{
				"hash":   "test-hash",
				"vendor": "com.example",
				"name":   "test-schema",
				"format": "jsonschema",
				"meta": map[string]any{
					"hidden":     false,
					"schemaType": "entity",
					"customData": map[string]string{},
				},
				"deployments": []map[string]any{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(listingResp)
			return
		}

		if r.URL.Path == "/api/msc/v1/organizations/test-org/data-structures/v1/schemas/versions" {
			// Mock schema versions response
			schemaVersions := []map[string]any{
				{
					"self": map[string]any{
						"vendor":  "com.example",
						"name":    "test-schema",
						"format":  "jsonschema",
						"version": "1-0-0",
					},
					"schema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"test1": map[string]any{
								"type": "string",
							},
						},
					},
				},
				{
					"self": map[string]any{
						"vendor":  "com.example",
						"name":    "test-schema",
						"format":  "jsonschema",
						"version": "2-0-0",
					},
					"schema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"test2": map[string]any{
								"type": "string",
							},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(schemaVersions)
			return
		}

		t.Errorf("Unexpected request to %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create client
	cnx := context.Background()
	client := &ApiClient{
		Http:    http.DefaultClient,
		Jwt:     "test-token",
		BaseUrl: server.URL + "/api/msc/v1/organizations/test-org",
		OrgId:   "test-org",
	}

	// Test the function
	results, err := GetAllDataStructureVersions(cnx, client, "test-hash", "")

	if err != nil {
		t.Fatalf("GetAllDataStructureVersions failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// Verify the results
	versions := make(map[string]bool)
	for _, result := range results {
		self, ok := result.Data["self"].(map[string]any)
		if !ok {
			t.Fatalf("Expected self field in Data")
		}

		version := self["version"].(string)
		versions[version] = true
	}

	if !versions["1-0-0"] {
		t.Errorf("Expected version 1-0-0 to be present")
	}

	if !versions["2-0-0"] {
		t.Errorf("Expected version 2-0-0 to be present")
	}
}

func TestGetAllDataStructureVersions_WithEnvironmentFilter(t *testing.T) {
	// Mock server responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/test-org/data-structures/v1/test-hash/deployments" {
			// Mock deployments response
			deployments := []Deployment{
				{
					Version:     "1-0-0",
					ContentHash: "hash1",
					Env:         DEV,
				},
				{
					Version:     "2-0-0",
					ContentHash: "hash2",
					Env:         PROD,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(deployments)
			return
		}

		// Mock the GetSpecificDataStructureVersion calls
		if r.URL.Path == "/api/msc/v1/organizations/test-org/data-structures/v1/test-hash" {
			listingResp := map[string]any{
				"hash":   "test-hash",
				"vendor": "com.example",
				"name":   "test-schema",
				"format": "jsonschema",
				"meta": map[string]any{
					"hidden":     false,
					"schemaType": "entity",
					"customData": map[string]string{},
				},
				"deployments": []map[string]any{},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(listingResp)
			return
		}

		if r.URL.Path == "/api/msc/v1/organizations/test-org/data-structures/v1/schemas/versions" {
			// Mock schema versions response
			schemaVersions := []map[string]any{
				{
					"self": map[string]any{
						"vendor":  "com.example",
						"name":    "test-schema",
						"format":  "jsonschema",
						"version": "1-0-0",
					},
					"schema": map[string]any{
						"type": "object",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(schemaVersions)
			return
		}

		t.Errorf("Unexpected request to %s", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create client
	cnx := context.Background()
	client := &ApiClient{
		Http:    http.DefaultClient,
		Jwt:     "test-token",
		BaseUrl: server.URL + "/api/msc/v1/organizations/test-org",
		OrgId:   "test-org",
	}

	// Test the function with environment filter
	results, err := GetAllDataStructureVersions(cnx, client, "test-hash", "DEV")

	if err != nil {
		t.Fatalf("GetAllDataStructureVersions failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result (filtered by DEV environment), got %d", len(results))
	}

	// Verify the result
	self, ok := results[0].Data["self"].(map[string]any)
	if !ok {
		t.Fatalf("Expected self field in Data")
	}

	version := self["version"].(string)
	if version != "1-0-0" {
		t.Errorf("Expected version '1-0-0', got '%s'", version)
	}
}
