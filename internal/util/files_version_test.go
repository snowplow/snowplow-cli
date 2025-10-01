/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package util

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/snowplow/snowplow-cli/internal/model"
)

func TestCreateDataStructuresWithVersions_IncludeVersionsTrue(t *testing.T) {
	extension := "yaml"
	vendor := "com.example"
	name := "test-schema"
	version1 := "1-0-0"
	version2 := "2-0-0"

	ds1 := DataStructure{
		Meta: DataStructureMeta{Hidden: false, SchemaType: "entity", CustomData: map[string]string{}},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  vendor,
				"name":    name,
				"format":  "jsonschema",
				"version": version1,
			},
			"schema": "string",
		},
	}

	ds2 := DataStructure{
		Meta: DataStructureMeta{Hidden: false, SchemaType: "entity", CustomData: map[string]string{}},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  vendor,
				"name":    name,
				"format":  "jsonschema",
				"version": version2,
			},
			"schema": "string",
		},
	}

	dir := t.TempDir()
	files := Files{DataStructuresLocation: dir, ExtentionPreference: extension}
	err := files.CreateDataStructuresWithVersions([]DataStructure{ds1, ds2}, false, true)

	if err != nil {
		t.Fatalf("CreateDataStructuresWithVersions failed: %s", err)
	}

	// Check that files are created with version suffixes
	filePath1 := filepath.Join(dir, vendor, fmt.Sprintf("%s_%s.%s", name, version1, extension))
	if _, err := os.Stat(filePath1); os.IsNotExist(err) {
		t.Fatalf("Expected file %s does not exist", filePath1)
	}

	filePath2 := filepath.Join(dir, vendor, fmt.Sprintf("%s_%s.%s", name, version2, extension))
	if _, err := os.Stat(filePath2); os.IsNotExist(err) {
		t.Fatalf("Expected file %s does not exist", filePath2)
	}
}

func TestCreateDataStructuresWithVersions_IncludeVersionsFalse(t *testing.T) {
	extension := "yaml"
	vendor := "com.example"
	name := "test-schema"
	version := "1-0-0"

	ds := DataStructure{
		Meta: DataStructureMeta{Hidden: false, SchemaType: "entity", CustomData: map[string]string{}},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  vendor,
				"name":    name,
				"format":  "jsonschema",
				"version": version,
			},
			"schema": "string",
		},
	}

	dir := t.TempDir()
	files := Files{DataStructuresLocation: dir, ExtentionPreference: extension}
	err := files.CreateDataStructuresWithVersions([]DataStructure{ds}, false, false)

	if err != nil {
		t.Fatalf("CreateDataStructuresWithVersions failed: %s", err)
	}

	// Check that file is created without version suffix
	filePath := filepath.Join(dir, vendor, fmt.Sprintf("%s.%s", name, extension))
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected file %s does not exist", filePath)
	}

	// Check that file with version suffix does NOT exist
	filePathWithVersion := filepath.Join(dir, vendor, fmt.Sprintf("%s_%s.%s", name, version, extension))
	if _, err := os.Stat(filePathWithVersion); !os.IsNotExist(err) {
		t.Fatalf("Expected file %s should not exist when includeVersions=false", filePathWithVersion)
	}
}

func TestCreateDataStructuresWithVersions_MultipleVersionsSameName(t *testing.T) {
	extension := "yaml"
	vendor := "com.example"
	name := "test-schema"
	version1 := "1-0-0"
	version2 := "1-0-0" // Same version, different deployments

	ds1 := DataStructure{
		Meta: DataStructureMeta{Hidden: false, SchemaType: "entity", CustomData: map[string]string{}},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  vendor,
				"name":    name,
				"format":  "jsonschema",
				"version": version1,
			},
			"schema": "string",
		},
	}

	ds2 := DataStructure{
		Meta: DataStructureMeta{Hidden: false, SchemaType: "entity", CustomData: map[string]string{}},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  vendor,
				"name":    name,
				"format":  "jsonschema",
				"version": version2,
			},
			"schema": "string",
		},
	}

	dir := t.TempDir()
	files := Files{DataStructuresLocation: dir, ExtentionPreference: extension}
	err := files.CreateDataStructuresWithVersions([]DataStructure{ds1, ds2}, false, true)

	if err != nil {
		t.Fatalf("CreateDataStructuresWithVersions failed: %s", err)
	}

	// Check that files are created with version suffixes and numeric suffixes for duplicates
	filePath1 := filepath.Join(dir, vendor, fmt.Sprintf("%s_%s-1.%s", name, version1, extension))
	if _, err := os.Stat(filePath1); os.IsNotExist(err) {
		t.Fatalf("Expected file %s does not exist", filePath1)
	}

	filePath2 := filepath.Join(dir, vendor, fmt.Sprintf("%s_%s-2.%s", name, version2, extension))
	if _, err := os.Stat(filePath2); os.IsNotExist(err) {
		t.Fatalf("Expected file %s does not exist", filePath2)
	}
}

func TestCreateDataStructuresWithVersions_JsonFormat(t *testing.T) {
	extension := "json"
	vendor := "com.example"
	name := "test-schema"
	version := "2-0-0"

	ds := DataStructure{
		Meta: DataStructureMeta{Hidden: false, SchemaType: "entity", CustomData: map[string]string{}},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  vendor,
				"name":    name,
				"format":  "jsonschema",
				"version": version,
			},
			"schema": "string",
		},
	}

	dir := t.TempDir()
	files := Files{DataStructuresLocation: dir, ExtentionPreference: extension}
	err := files.CreateDataStructuresWithVersions([]DataStructure{ds}, false, true)

	if err != nil {
		t.Fatalf("CreateDataStructuresWithVersions failed: %s", err)
	}

	// Check that JSON file is created with version suffix
	filePath := filepath.Join(dir, vendor, fmt.Sprintf("%s_%s.%s", name, version, extension))
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected file %s does not exist", filePath)
	}
}

func TestCreateDataStructuresWithVersions_BackwardCompatibility(t *testing.T) {
	extension := "yaml"
	vendor := "com.example"
	name := "test-schema"
	version := "1-0-0"

	ds := DataStructure{
		Meta: DataStructureMeta{Hidden: false, SchemaType: "entity", CustomData: map[string]string{}},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  vendor,
				"name":    name,
				"format":  "jsonschema",
				"version": version,
			},
			"schema": "string",
		},
	}

	dir := t.TempDir()
	files := Files{DataStructuresLocation: dir, ExtentionPreference: extension}

	// Test that the old CreateDataStructures function still works (should not include versions)
	err := files.CreateDataStructures([]DataStructure{ds}, false)
	if err != nil {
		t.Fatalf("CreateDataStructures failed: %s", err)
	}

	// Check that file is created without version suffix (backward compatibility)
	filePath := filepath.Join(dir, vendor, fmt.Sprintf("%s.%s", name, extension))
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected file %s does not exist", filePath)
	}

	// Check that file with version suffix does NOT exist
	filePathWithVersion := filepath.Join(dir, vendor, fmt.Sprintf("%s_%s.%s", name, version, extension))
	if _, err := os.Stat(filePathWithVersion); !os.IsNotExist(err) {
		t.Fatalf("Expected file %s should not exist when using old CreateDataStructures function", filePathWithVersion)
	}
}

func TestCreateDataStructuresWithVersions_ComplexVersionNames(t *testing.T) {
	extension := "yaml"
	vendor := "com.example"
	name := "complex-schema"
	version := "10-15-3" // Complex version with multiple digits

	ds := DataStructure{
		Meta: DataStructureMeta{Hidden: false, SchemaType: "entity", CustomData: map[string]string{}},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  vendor,
				"name":    name,
				"format":  "jsonschema",
				"version": version,
			},
			"schema": "string",
		},
	}

	dir := t.TempDir()
	files := Files{DataStructuresLocation: dir, ExtentionPreference: extension}
	err := files.CreateDataStructuresWithVersions([]DataStructure{ds}, false, true)

	if err != nil {
		t.Fatalf("CreateDataStructuresWithVersions failed: %s", err)
	}

	// Check that file is created with complex version suffix
	filePath := filepath.Join(dir, vendor, fmt.Sprintf("%s_%s.%s", name, version, extension))
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Expected file %s does not exist", filePath)
	}
}
