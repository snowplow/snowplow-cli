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
	"reflect"
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

	for _, p := range []string{saPath, dp1Path, dp2Path} {
		if !slices.Contains(keys, p) {
			t.Fatal("missing path", p)
		}
	}

	if got := dps[saPath]["resourceType"]; got != "source-application" {
		t.Fatalf("unexpected resourceType of %s: %v", saPath, got)
	}
	if got := dps[dp1Path]["resourceType"]; got != "data-product" {
		t.Fatalf("unexpected resourceType of %s: %v", dp1Path, got)
	}
}

func Test_MaybeResourcesfromPathsUnreadable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.yaml"), []byte("a: [unclosed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{filepath.Join(dir, "broken.yaml"), filepath.Join(dir, "broken.json"), filepath.Join(dir, "missing")} {
		res, err := MaybeResourcesfromPaths([]string{path})
		if err == nil {
			t.Errorf("%s: expected an error, got %v", path, res)
		}
		if res != nil {
			t.Errorf("%s: expected no resources on error, got %v", path, res)
		}
	}
}

func Test_DataStructuresFromPathsExtensions(t *testing.T) {
	dir := t.TempDir()
	ds := "apiVersion: v1\nresourceType: data-structure\nmeta:\n  schemaType: event\n"
	dsJson := `{"apiVersion": "v1", "resourceType": "data-structure", "meta": {"schemaType": "event"}}`
	for name, content := range map[string]string{"a.yaml": ds, "b.yml": ds, "c.json": dsJson, "d.txt": ds, "e": ds} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	got, err := DataStructuresFromPaths([]string{dir})
	if err != nil {
		t.Fatal(err)
	}

	keys := []string{}
	for k, v := range got {
		keys = append(keys, filepath.Base(k))
		if v.Meta.SchemaType != "event" {
			t.Errorf("%s: unexpected unmarshalling: %+v", k, v)
		}
	}
	slices.Sort(keys)
	if !slices.Equal(keys, []string{"a.yaml", "b.yml", "c.json"}) {
		t.Fatalf("unexpected files read: %v", keys)
	}
}

func Test_StringSlicesEqual(t *testing.T) {
	tests := []struct {
		one, other []string
		want       bool
	}{
		{nil, nil, true},
		{nil, []string{}, true},
		{[]string{}, nil, true},
		{[]string{"a"}, []string{"a"}, true},
		{[]string{"a"}, nil, false},
		{nil, []string{"a"}, false},
		{[]string{"a"}, []string{"b"}, false},
		{[]string{"a", "b"}, []string{"b", "a"}, false},
	}

	for _, tt := range tests {
		if got := StringSlicesEqual(tt.one, tt.other); got != tt.want {
			t.Errorf("StringSlicesEqual(%#v, %#v) = %t, want %t", tt.one, tt.other, got, tt.want)
		}
	}
}

func Test_ResourceNameToFileName(t *testing.T) {
	longName := strings.Repeat("a", 300)
	suffix := "...(truncated)"
	expectedLongNameTruncated := longName[:200-len(suffix)] + "...(truncated)"
	tests := map[string]string{
		"cool_name":                "cool_name",
		"Normal data product name": "normal-data-product-name",
		"/щ💡test":                  "-test",
		"lpt1":                     "unnamed",
		longName:                   expectedLongNameTruncated,
		strings.Repeat("b", 200):   strings.Repeat("b", 200),
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			if got := ResourceNameToFileName(input); got != expected {
				t.Fatalf("%s result not as expected, expected: %s, actual: %s", input, expected, got)
			}
		})
	}
}

func Test_setMinus_OK(t *testing.T) {
	set1 := []string{"1", "2", "3"}
	set2 := []string{"1", "2", "3"}
	set3 := []string{"1"}
	set4 := []string{"1", "4"}

	expected2 := []string{"2", "3"}

	res1 := SetMinus(set1, set2)
	res2 := SetMinus(set1, set3)
	res3 := SetMinus(set1, set4)

	slices.Sort(res2)
	slices.Sort(res3)

	if len(res1) != 0 {
		t.Fatalf("result not as expected, expected: [], actual: %s", res1)
	}

	if !reflect.DeepEqual(res2, expected2) {
		t.Fatalf("result not as expected, expected: %v, actual: %v", expected2, res2)
	}

	if !reflect.DeepEqual(res3, expected2) {
		t.Fatalf("result not as expected, expected: %s, actual: %s", expected2, res3)
	}
}
