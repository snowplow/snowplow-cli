package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCreatesDataStructuresFolderWithFiles(t *testing.T) {
	ds1 := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity", CustomData: map[string]string{
			"additionalProp1": "string",
			"additionalProp2": "string",
			"additionalProp3": "string",
		},
		},
		Data: DataStrucutreData{
			Self: DataStructureSelf{
				Vendor: "test.my.vendor",
				Name:   "my-test-ds",
				Format: "string",
				Version: DataStructureVersion{
					Model:    1073741824,
					Revision: 1073741824,
					Addition: 1073741824,
				},
			},
			Schema: "string"},
	}
	ds2 := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity", CustomData: map[string]string{
			"additionalProp1": "string",
			"additionalProp2": "string",
			"additionalProp3": "string",
		},
		},
		Data: DataStrucutreData{
			Self: DataStructureSelf{
				Vendor: "com.test.vendor",
				Name:   "ds2",
				Format: "string",
				Version: DataStructureVersion{
					Model:    1073741824,
					Revision: 1073741824,
					Addition: 1073741824,
				},
			},
			Schema: "string"},
	}

	dir := filepath.Join("..", "out", "test-ds2")
	files := Files{DataStructuresLocation: dir, ExtentionPreference: "yaml"}
	err := files.createDataStructures([]DataStructure{ds1, ds2})

	if err != nil {
		t.Fatalf("Can't create directory %s", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("%s does not exists", dir)
	}

	filePath1 := filepath.Join(dir, ds1.Data.Self.Vendor, fmt.Sprintf("%s.%s", ds1.Data.Self.Name, "yaml"))
	if _, err := os.Stat(filePath1); os.IsNotExist(err) {
		t.Fatalf("%s does not exists", filePath1)
	}

	filePath2 := filepath.Join(dir, ds2.Data.Self.Vendor, fmt.Sprintf("%s.%s", ds2.Data.Self.Name, "yaml"))
	if _, err := os.Stat(filePath2); os.IsNotExist(err) {
		t.Fatalf("%s does not exists", filePath2)
	}

}
