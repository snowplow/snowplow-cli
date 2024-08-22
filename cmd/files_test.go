package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
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

	dir := filepath.Join("..", "out", "test-ds2")
	files := Files{DataStructuresLocation: dir, ExtentionPreference: extension}
	err := files.createDataStructures([]DataStructure{ds1, ds2})

	if err != nil {
		t.Fatalf("Can't create directory %s", err)
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

	dir := filepath.Join("..", "out", "test-ds2")
	files := Files{DataStructuresLocation: dir, ExtentionPreference: extension}
	err := files.createDataStructures([]DataStructure{ds1, ds2})

	if err != nil {
		t.Fatalf("Can't create directory %s", err)
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
