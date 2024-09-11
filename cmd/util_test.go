package cmd

import (
	"os"
	"strings"
	"testing"
)

func Test_DataStructuresFromPaths(t *testing.T) {
	path := strings.Join([]string{"testdata", "util"}, string(os.PathSeparator))
	paths := []string{path}

	ds, err := DataStructuresFromPaths(paths)

	if err != nil {
		t.Fatal(err)
	}

	jsonpath := strings.Join([]string{"testdata", "util", "vendor.one", "someds.json"}, string(os.PathSeparator))

	if json, ok := ds[jsonpath]; ok {
		if json.Meta.SchemaType != "event" {
			t.Fatal("json unexpected unmarshalling")
		}
	} else {
		t.Fatal("didn't find the json one")
	}

	yamlpath := strings.Join([]string{"testdata", "util", "vendor.two", "someds.yaml"}, string(os.PathSeparator))

	if yaml, ok := ds[yamlpath]; ok {
		if yaml.Meta.SchemaType != "event" {
			t.Fatal("yaml unexpected unmarshalling")
		}
	} else {
		t.Fatal("didn't find the yaml one")
	}
}

func Test_DataStructuresFromPathsFailNotASchema(t *testing.T) {
	path := strings.Join([]string{"testdata", "not-a-schema"}, string(os.PathSeparator))
	paths := []string{path}

	_, err := DataStructuresFromPaths(paths)

	if err == nil {
		t.Fatal(err)
	}
}
