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
	"reflect"
	"sort"
	"strings"
	"testing"

	. "github.com/snowplow/snowplow-cli/internal/model"
)

func TestCreatesDataStructuresFolderWithFiles(t *testing.T) {
	extension := "yaml"
	vendor1 := "test.my.vendor"
	name1 := "my-test-ds"
	ds1 := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity", CustomData: map[string]string{
			"additionalProp1": "string",
			"additionalProp2": "string",
			"additionalProp3": "string",
		},
		},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  vendor1,
				"name":    name1,
				"format":  "string",
				"version": "1-2-0",
			},
			"schema": "string"},
	}
	vendor2 := "com.test.vendor"
	name2 := "ds2"
	ds2 := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity", CustomData: map[string]string{
			"additionalProp1": "string",
			"additionalProp2": "string",
			"additionalProp3": "string",
		},
		},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  vendor2,
				"name":    name2,
				"format":  "string",
				"version": "1-0-1",
			},
			"schema": "string"},
	}

	dir := t.TempDir()
	files := Files{DataStructuresLocation: dir, ExtentionPreference: extension}
	err := files.CreateDataStructures([]DataStructure{ds1, ds2}, false)

	if err != nil {
		t.Fatalf("CreateDataStructures failed: %s", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("%s does not exists", dir)
	}

	filePath1 := filepath.Join(dir, vendor1, fmt.Sprintf("%s.%s", name1, extension))
	if _, err := os.Stat(filePath1); os.IsNotExist(err) {
		t.Fatalf("%s does not exists", filePath1)
	}

	filePath2 := filepath.Join(dir, vendor2, fmt.Sprintf("%s.%s", name2, extension))
	if _, err := os.Stat(filePath2); os.IsNotExist(err) {
		t.Fatalf("%s does not exists", filePath2)
	}

}

func TestCreatesDataStructuresFolderWithFilesJson(t *testing.T) {
	extension := "json"
	vendor1 := "test.my.vendor"
	name1 := "my-test-ds"
	ds1 := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity", CustomData: map[string]string{
			"additionalProp1": "string",
			"additionalProp2": "string",
			"additionalProp3": "string",
		},
		},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  vendor1,
				"name":    name1,
				"format":  "string",
				"version": "1-2-0",
			},
			"schema": "string"}}
	vendor2 := "com.test.vendor"
	name2 := "ds2"
	ds2 := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity", CustomData: map[string]string{
			"additionalProp1": "string",
			"additionalProp2": "string",
			"additionalProp3": "string",
		},
		},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  vendor2,
				"name":    name2,
				"format":  "string",
				"version": "1-0-1",
			},
			"schema": "string"},
	}

	dir := t.TempDir()
	files := Files{DataStructuresLocation: dir, ExtentionPreference: extension}
	err := files.CreateDataStructures([]DataStructure{ds1, ds2}, false)

	if err != nil {
		t.Fatalf("CreateDataStructures failed: %s", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("%s does not exists", dir)
	}

	filePath1 := filepath.Join(dir, vendor1, fmt.Sprintf("%s.%s", name1, extension))
	if _, err := os.Stat(filePath1); os.IsNotExist(err) {
		t.Fatalf("%s does not exists", filePath1)
	}

	filePath2 := filepath.Join(dir, vendor2, fmt.Sprintf("%s.%s", name2, extension))
	if _, err := os.Stat(filePath2); os.IsNotExist(err) {
		t.Fatalf("%s does not exists", filePath2)
	}

}

func Test_createUniqueNames_OK(t *testing.T) {
	input1 := []idFileName{
		{Id: "id2", FileName: "NaMe"},
		{Id: "id5", FileName: "hey"},
		{Id: "id1", FileName: "Name"},
		{Id: "id3", FileName: "Test"},
		{Id: "id4", FileName: "🐌Hey"},
	}
	expected1 := []idFileName{
		{Id: "id1", FileName: "name-1"},
		{Id: "id2", FileName: "name-2"},
		{Id: "id3", FileName: "test"},
		{Id: "id4", FileName: "hey-1"},
		{Id: "id5", FileName: "hey-2"},
	}

	res := createUniqueNames(input1)

	sort.Slice(res, func(i, j int) bool {
		return res[i].Id < res[j].Id
	})

	if !reflect.DeepEqual(res, expected1) {
		t.Fatalf("Not expected result, expected: %+v, actual: %+v", expected1, res)
	}
}

func TestCreateDataStructures_CaseInsensitiveConflicts(t *testing.T) {
	extension := "yaml"

	ds1 := DataStructure{
		Meta: DataStructureMeta{Hidden: false, SchemaType: "event", CustomData: map[string]string{}},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "com.Example",
				"name":    "Article",
				"format":  "jsonschema",
				"version": "1-0-0",
			},
			"schema": "string",
		},
	}

	ds2 := DataStructure{
		Meta: DataStructureMeta{Hidden: false, SchemaType: "event", CustomData: map[string]string{}},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "com.example",
				"name":    "article",
				"format":  "jsonschema",
				"version": "1-0-0",
			},
			"schema": "string",
		},
	}

	ds3 := DataStructure{
		Meta: DataStructureMeta{Hidden: false, SchemaType: "event", CustomData: map[string]string{}},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "com.example",
				"name":    "user",
				"format":  "jsonschema",
				"version": "1-0-0",
			},
			"schema": "string",
		},
	}

	dir := t.TempDir()
	files := Files{DataStructuresLocation: dir, ExtentionPreference: extension}
	err := files.CreateDataStructures([]DataStructure{ds1, ds2, ds3}, false)

	if err != nil {
		t.Fatalf("CreateDataStructures failed with case-insensitive vendor and schema name conflicts: %s", err)
	}

	vendorDirs := []string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("Cannot read data structures directory after creating conflicting case vendors com.Example and com.example: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			vendorDirs = append(vendorDirs, entry.Name())
		}
	}

	if len(vendorDirs) != 2 {
		t.Fatalf("Case-insensitive vendor conflicts not resolved properly - expected exactly 2 unique vendor directories for com.Example and com.example, got %d directories: %v", len(vendorDirs), vendorDirs)
	}

	for _, vendorDir := range vendorDirs {
		vendorPath := filepath.Join(dir, vendorDir)
		files, err := os.ReadDir(vendorPath)
		if err != nil {
			t.Fatalf("Cannot read vendor directory %s after resolving case conflicts: %v", vendorDir, err)
		}

		if len(files) == 0 {
			t.Fatalf("Vendor directory %s is empty after creating data structures with case conflicts", vendorDir)
		}

		fileNames := make(map[string]bool)
		for _, file := range files {
			lowerName := strings.ToLower(file.Name())
			if fileNames[lowerName] {
				t.Fatalf("Case-insensitive filename conflict not resolved - found duplicate file %s in vendor directory %s after processing Article and article schemas", file.Name(), vendorDir)
			}
			fileNames[lowerName] = true
		}
	}
}

func TestRespectsNoLsp(t *testing.T) {
	extension := "yaml"
	vendor1 := "test.my.vendor"
	name1 := "my-test-ds"
	ds1 := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity", CustomData: map[string]string{
			"additionalProp1": "string",
			"additionalProp2": "string",
			"additionalProp3": "string",
		},
		},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  vendor1,
				"name":    name1,
				"format":  "string",
				"version": "1-2-0",
			},
			"schema": "string"},
	}

	dir := t.TempDir()
	files := Files{DataStructuresLocation: dir, ExtentionPreference: extension}
	err := files.CreateDataStructures([]DataStructure{ds1}, false)

	if err != nil {
		t.Fatalf("CreateDataStructures failed: %s", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("%s does not exists", dir)
	}

	filePath1 := filepath.Join(dir, vendor1, fmt.Sprintf("%s.%s", name1, extension))
	if _, err := os.Stat(filePath1); os.IsNotExist(err) {
		t.Fatalf("%s does not exists", filePath1)
	}

	// Read the file contents and check for the LSP schema URL
	fileContent, err := os.ReadFile(filePath1)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filePath1, err)
	}

	// Check that the file contains the RepoRawFileURL (which is part of the LSP schema URL)
	if !strings.Contains(string(fileContent), RepoRawFileURL) {
		t.Fatalf("Expected file to contain LSP schema URL with %s, but it didn't", RepoRawFileURL)
	}

	err = files.CreateDataStructures([]DataStructure{ds1}, true)

	if err != nil {
		t.Fatalf("CreateDataStructures failed: %s", err)
	}

	fileContent, err = os.ReadFile(filePath1)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", filePath1, err)
	}

	if strings.Contains(string(fileContent), RepoRawFileURL) {
		t.Fatalf("Expected file to not contain LSP schema URL with %s, but it did", RepoRawFileURL)
	}

}
