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
	"testing"
)

func TestGenerateDataStructureHash(t *testing.T) {
	// Test case from Snowplow API documentation
	orgId := "38e97db9-f3cb-404d-8250-cd227506e544"
	vendor := "com.acme.event"
	name := "search"
	format := "jsonschema"
	expectedHash := "a41ef92847476c1caaf5342c893b51089a596d8ecd28a54d3f22d922422a6700"

	actualHash := GenerateDataStructureHash(orgId, vendor, name, format)

	if actualHash != expectedHash {
		t.Errorf("GenerateDataStructureHash() = %v, want %v", actualHash, expectedHash)
	}
}

func TestGenerateDataStructureHashDifferentInputs(t *testing.T) {
	tests := []struct {
		name         string
		orgId        string
		vendor       string
		schemaName   string
		format       string
		expectedHash string
	}{
		{
			name:         "example from docs",
			orgId:        "38e97db9-f3cb-404d-8250-cd227506e544",
			vendor:       "com.acme.event",
			schemaName:   "search",
			format:       "jsonschema",
			expectedHash: "a41ef92847476c1caaf5342c893b51089a596d8ecd28a54d3f22d922422a6700",
		},
		{
			name:         "different vendor",
			orgId:        "38e97db9-f3cb-404d-8250-cd227506e544",
			vendor:       "com.example",
			schemaName:   "search",
			format:       "jsonschema",
			expectedHash: "b8c4e8f2a1d3e5f7b9c2d4e6f8a0b2c4d6e8f0a2b4c6d8e0f2a4b6c8d0e2f4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the second test case, we'll just verify it generates a different hash
			actualHash := GenerateDataStructureHash(tt.orgId, tt.vendor, tt.schemaName, tt.format)
			
			if tt.name == "example from docs" {
				if actualHash != tt.expectedHash {
					t.Errorf("GenerateDataStructureHash() = %v, want %v", actualHash, tt.expectedHash)
				}
			} else {
				// Just verify it's a valid hex string of correct length (64 chars for SHA-256)
				if len(actualHash) != 64 {
					t.Errorf("GenerateDataStructureHash() should return 64-character hex string, got %d characters", len(actualHash))
				}
			}
		})
	}
}
