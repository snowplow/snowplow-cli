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
	"fmt"
	"reflect"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/model"
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

	lookup, err := NewDPLookup(nil, nil, input, nil, true, 1)
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

	lookup, err := NewDPLookup(nil, nil, input, nil, true, 1)
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

	lookup, err := NewDPLookup(nil, nil, input, nil, true, 1)
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

	lookup, err := NewDPLookup(nil, nil, input, nil, true, 1)

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

func Test_DPLookup_AddVersionMismatches(t *testing.T) {
	tests := []struct {
		name     string
		mismatch model.EventSpecVersionMismatch
		want     []string
	}{
		{
			name:     "structural change",
			mismatch: model.EventSpecVersionMismatch{FileName: "/base/dp.yml", Index: 0, Name: "Checkout Started", Declared: 3, Expected: 4, Changed: []string{"event", "entities"}},
			want:     []string{"Checkout Started: version 3 needs a bump", "changed: event, entities", "expected version: 4", "run `snowplow-cli dp validate --fix` to update it"},
		},
		{
			name:     "stale version without a change",
			mismatch: model.EventSpecVersionMismatch{FileName: "/base/dp.yml", Index: 1, Name: "Checkout Started", Declared: 2, Expected: 3, Changed: []string{}},
			want:     []string{"Checkout Started: version 2 does not match", "expected version: 3", "run `snowplow-cli dp validate --fix` to update it"},
		},
		{
			name:     "entity comment set in Console",
			mismatch: model.EventSpecVersionMismatch{FileName: "/base/dp.yml", Index: 3, Name: "Checkout Started", Declared: 3, Expected: 4, Changed: []string{"entities"}, EntityCommentsRemoved: true},
			want:     []string{"Checkout Started: version 3 needs a bump", "changed: entities", "entities: a comment set in Console will be removed", "expected version: 4", "run `snowplow-cli dp validate --fix` to update it"},
		},
		{
			name:     "new event specification",
			mismatch: model.EventSpecVersionMismatch{FileName: "/base/dp.yml", Index: 2, Name: "Checkout Started", Declared: 1, Expected: 0, IsNew: true, Changed: []string{}},
			want:     []string{"Checkout Started: version 1 does not match", "new event specification", "expected version: 0", "run `snowplow-cli dp validate --fix` to update it"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := &DPLookup{Validations: map[string]DPValidations{}}
			lookup.AddVersionMismatches([]model.EventSpecVersionMismatch{tt.mismatch})

			path := fmt.Sprintf("/data/eventSpecifications/%d", tt.mismatch.Index)
			got := lookup.Validations["/base/dp.yml"].ErrorsWithPaths[path]
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got: %q want: %q", got, tt.want)
			}
			if lookup.ValidationErrorCount() != 1 {
				t.Errorf("expected 1 error, got %d", lookup.ValidationErrorCount())
			}
		})
	}
}
