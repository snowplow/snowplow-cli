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
	"os"
	"strings"
	"testing"
)

func Test_DataStructuresFromPaths(t *testing.T) {
	path := strings.Join([]string{"..", "testdata", "util"}, string(os.PathSeparator))
	paths := []string{path}

	ds, err := DataStructuresFromPaths(paths)

	if err != nil {
		t.Fatal(err)
	}

	jsonpath := strings.Join([]string{"..", "testdata", "util", "vendor.one", "someds.json"}, string(os.PathSeparator))

	if json, ok := ds[jsonpath]; ok {
		if json.Meta.SchemaType != "event" {
			t.Fatal("json unexpected unmarshalling")
		}
	} else {
		t.Fatal("didn't find the json one")
	}

	yamlpath := strings.Join([]string{"..", "testdata", "util", "vendor.two", "someds.yaml"}, string(os.PathSeparator))

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
