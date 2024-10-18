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
	"path/filepath"
	"slices"
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

func Test_MaybeResourcesfromPaths(t *testing.T) {
	saPath, _ := filepath.Abs(filepath.Join("testdata", "data-products", "source-application.yml"))
	dp1Path, _ := filepath.Abs(filepath.Join("testdata", "data-products", "data-product.yml"))
	dp2Path, _ := filepath.Abs(filepath.Join("testdata", "data-products", "sub-dir-dp", "data-product.yml"))

	paths := []string{
		filepath.Join("testdata", "data-products"),
	}

	dps, err := MaybeResourcesfromPaths(paths)
	if err != nil {
		t.Fatal(err)
	}

	keys := []string{}
	for k := range dps {
		keys = append(keys, k)
	}

	for _, p := range []string{ saPath, dp1Path, dp2Path } {
		if !slices.Contains(keys, p) {
			t.Fatal("missing path", p)
		}
	}

}
