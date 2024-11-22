/**
 * Copyright (c) 2013-present Snowplow Analytics Ltd.
 * All rights reserved.
 * This software is made available by Snowplow Analytics, Ltd.,
 * under the terms of the Snowplow Limited Use License Agreement, Version 1.0
 * located at https://docs.snowplow.io/limited-use-license-1.0
 * BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
 * OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
 */

package validation

import (
	"testing"
)

func Test_DPLookup_RelativePaths(t *testing.T) {
	input := map[string]map[string]any{
		"/base/source-apps/a/b/file1.yml": {
			"apiVersion":   "v1",
			"resourceType": "source-application",
		},
		"/base/source-apps/a/file1.yml": {
			"apiVersion":   "v1",
			"resourceType": "source-application",
		},
		"/base/somedir/file2.yml": {
			"apiVersion":   "v1",
			"resourceType": "data-product",
			"data": map[string]any{
				"sourceApplications": []map[string]string{
					{"$ref": "../source-apps/a/../a/file1.yml"},
					{"$ref": "../source-apps/a/../a/b/../b/file1.yml"},
				},
			},
		},
	}

	lookup, err := NewDPLookup(nil, nil, input, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	v, ok := lookup.DataProducts["/base/somedir/file2.yml"]
	if !ok {
		t.Fatal("couldnt find expected")
	}

	refs := [2]string{
		v.Data.SourceApplications[0]["$ref"],
		v.Data.SourceApplications[1]["$ref"],
	}

	if refs != [2]string{"/base/source-apps/a/file1.yml", "/base/source-apps/a/b/file1.yml"} {
		t.Fatal("missing paths", refs)
	}
}

func Test_DPLookup_EventSpecBadSourceApp(t *testing.T) {
	input := map[string]map[string]any{
		"/base/somedir/file1.yml": {
			"apiVersion":   "v1",
			"resourceType": "source-application",
		},
		"/base/somedir/file2.yml": {
			"apiVersion":   "v1",
			"resourceType": "data-product",
			"data": map[string]any{
				"sourceApplications": []map[string]string{
					{"$ref": "file1.yml"},
				},
				"eventSpecifications": []map[string]any{
					{
						"excludedSourceApplications": []map[string]string{
							{"$ref": "file7.yml"},
						},
					},
				},
			},
		},
	}

	lookup, err := NewDPLookup(nil, nil, input, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	v, ok := lookup.Validations["/base/somedir/file2.yml"]
	if !ok {
		t.Fatal("couldnt find expected")
	}

	if len(v.Errors) != 1 {
		t.Fatal("expected bad event spec source app $ref error")
	}
}

func Test_DPLookup_Resolved(t *testing.T) {
	input := map[string]map[string]any{
		"/base/somedir/file1.yml": {
			"apiVersion":   "v1",
			"resourceType": "source-application",
		},
		"/base/somedir/file2.yml": {
			"apiVersion":   "v1",
			"resourceType": "data-product",
			"data": map[string]any{
				"sourceApplications": []map[string]string{
					{"$ref": "file1.yml"},
				},
			},
		},
	}

	lookup, err := NewDPLookup(nil, nil, input, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	v, ok := lookup.DataProducts["/base/somedir/file2.yml"]
	if !ok {
		t.Fatal("couldnt find expected")
	}

	if v.Data.SourceApplications[0]["$ref"] != "/base/somedir/file1.yml" {
		t.Fatal("bad path", v.Data.SourceApplications)
	}
}

func Test_DPLookup_Ignored(t *testing.T) {
	input := map[string]map[string]any{
		"somedir/file1.yml": {
			"apiVersion": "v1",
		},
		"somedir/file2.yml": {
			"apiVersion":   "v1",
			"resourceType": "nothingtoseehere",
		},
	}

	lookup, err := NewDPLookup(nil, nil, input, nil, true)

	if err != nil {
		t.Fatal(err)
	}

	v, ok := lookup.Validations["somedir/file1.yml"]

	if !ok {
		t.Fatal("validation success?")
	}

	if v.Errors[0] != "missing resourceType" {
		t.Fatal("wrong error", v.Errors)
	}

	v, ok = lookup.Validations["somedir/file2.yml"]

	if !ok {
		t.Fatal("validation success?")
	}

	if len(v.Debug) != 1 {
		t.Fatal("no debug?")
	}
}
