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
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/model"
	"gopkg.in/yaml.v3"
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

func Test_DPLookup_RefErrorsOrder(t *testing.T) {
	input := map[string]map[string]any{
		"/base/sa/app.yml": {
			"apiVersion":   "v1",
			"resourceType": "source-application",
		},
		"/base/dp/file.yml": {
			"apiVersion":   "v1",
			"resourceType": "data-product",
			"data": map[string]any{
				"sourceApplications": []any{
					map[string]any{"$ref": "../sa/gone1.yml"},
					map[string]any{"$ref": "../sa/app.yml"},
					map[string]any{"$ref": "../sa/gone2.yml"},
				},
				"eventSpecifications": []any{
					map[string]any{
						"resourceName": "spec-1",
						"excludedSourceApplications": []any{
							map[string]any{"$ref": "../sa/other1.yml"},
						},
					},
					map[string]any{
						"resourceName": "spec-2",
						"excludedSourceApplications": []any{
							map[string]any{"$ref": "../sa/other2.yml"},
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

	want := []string{
		"source application $ref not found ../sa/gone1.yml, available list [../sa/app.yml]",
		"source application $ref not found ../sa/gone2.yml, available list [../sa/app.yml]",
		"event spec source app not in parent data product list (event spec: spec-1, source app: ../sa/other1.yml), available list [../sa/app.yml]",
		"event spec source app not in parent data product list (event spec: spec-2, source app: ../sa/other2.yml), available list [../sa/app.yml]",
	}

	got := lookup.Validations["/base/dp/file.yml"].Errors
	if !slices.Equal(got, want) {
		t.Fatalf("got: %q want: %q", got, want)
	}
}

func Test_DPLookup_SourceAppWithoutRefSkipped(t *testing.T) {
	input := map[string]map[string]any{
		"/base/somedir/file1.yml": {
			"apiVersion":   "v1",
			"resourceType": "source-application",
		},
		"/base/somedir/file2.yml": {
			"apiVersion":   "v1",
			"resourceType": "data-product",
			"data": map[string]any{
				"sourceApplications": []any{
					map[string]any{"name": "no ref here"},
					map[string]any{"$ref": "file1.yml"},
				},
				"eventSpecifications": []any{
					map[string]any{
						"excludedSourceApplications": []any{
							map[string]any{"$ref": "file1.yml"},
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

	if errs := lookup.Validations["/base/somedir/file2.yml"].Errors; len(errs) != 0 {
		t.Fatal("unexpected errors", errs)
	}

	sas := lookup.DataProducts["/base/somedir/file2.yml"].Data.SourceApplications
	want := []map[string]string{
		{"name": "no ref here"},
		{"$ref": "/base/somedir/file1.yml"},
	}
	if !reflect.DeepEqual(sas, want) {
		t.Fatalf("got: %q want: %q", sas, want)
	}
}

func Test_DPLookup_ExcludedSourceAppWithoutRefSkipped(t *testing.T) {
	input := map[string]map[string]any{
		"/base/somedir/file1.yml": {
			"apiVersion":   "v1",
			"resourceType": "source-application",
		},
		"/base/somedir/file2.yml": {
			"apiVersion":   "v1",
			"resourceType": "data-product",
			"data": map[string]any{
				"sourceApplications": []any{
					map[string]any{"$ref": "file1.yml"},
				},
				"eventSpecifications": []any{
					map[string]any{
						"resourceName": "spec-1",
						"excludedSourceApplications": []any{
							map[string]any{"name": "no ref here"},
							map[string]any{"$ref": "file7.yml"},
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

	want := []string{
		"event spec source app not in parent data product list (event spec: spec-1, source app: file7.yml), available list [file1.yml]",
	}

	got := lookup.Validations["/base/somedir/file2.yml"].Errors
	if !slices.Equal(got, want) {
		t.Fatalf("got: %q want: %q", got, want)
	}
}

func Test_DPLookup_ExcludedSourceAppInParent(t *testing.T) {
	input := map[string]map[string]any{
		"/base/somedir/file1.yml": {
			"apiVersion":   "v1",
			"resourceType": "source-application",
		},
		"/base/somedir/file2.yml": {
			"apiVersion":   "v1",
			"resourceType": "source-application",
		},
		"/base/somedir/file3.yml": {
			"apiVersion":   "v1",
			"resourceType": "data-product",
			"data": map[string]any{
				"sourceApplications": []any{
					map[string]any{"$ref": "file1.yml"},
					map[string]any{"$ref": "file2.yml"},
				},
				"eventSpecifications": []any{
					map[string]any{
						"resourceName": "spec-1",
						"excludedSourceApplications": []any{
							map[string]any{"$ref": "file1.yml"},
							map[string]any{"$ref": "file7.yml"},
							map[string]any{"$ref": "file2.yml"},
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

	want := []string{
		"event spec source app not in parent data product list (event spec: spec-1, source app: file7.yml), available list [file1.yml file2.yml]",
	}

	got := lookup.Validations["/base/somedir/file3.yml"].Errors
	if !slices.Equal(got, want) {
		t.Fatalf("got: %q want: %q", got, want)
	}

	excluded := lookup.DataProducts["/base/somedir/file3.yml"].Data.EventSpecifications[0].ExcludedSourceApplications
	if len(excluded) != 3 {
		t.Fatalf("expected 3 excluded source apps, got: %q", excluded)
	}
	gotFound := []string{excluded[0]["$ref"], excluded[2]["$ref"]}
	wantFound := []string{"/base/somedir/file1.yml", "/base/somedir/file2.yml"}
	if !slices.Equal(gotFound, wantFound) {
		t.Fatalf("got: %q want: %q", gotFound, wantFound)
	}
}

func Test_DPLookup_CompatCheckForAllOrChanged(t *testing.T) {
	validDP := `
apiVersion: v1
resourceType: data-product
resourceName: dbc68917-3bcb-4640-a165-483de5fc033c
data:
  name: test_data_product
  sourceApplications: []
  eventSpecifications:
    - resourceName: b57f712e-ee43-4e10-8b36-48a0f6f2e628
      name: Test Event
      event:
        source: iglu:com.example/test_event/jsonschema/1-0-0
        schema:
          type: object
          additionalProperties: false
          properties:
            testField:
              type: string
      entities:
        tracked: []
        enriched: []
`
	file := "/base/somedir/dp.yml"
	skipped := []string{"skipping compatibility check for /base/somedir/dp.yml, since it was not changed, use --full to include it in validation"}

	tests := []struct {
		name        string
		validateAll bool
		changed     map[string]string
		wantCalls   int32
		wantDebug   []string
	}{
		{"all, not changed", true, nil, 1, nil},
		{"all, changed", true, map[string]string{"dbc68917-3bcb-4640-a165-483de5fc033c": file}, 1, nil},
		{"not all, changed", false, map[string]string{"dbc68917-3bcb-4640-a165-483de5fc033c": file}, 1, nil},
		{"not all, not changed", false, nil, 0, skipped},
		{"not all, other file changed", false, map[string]string{"other": "/base/somedir/other.yml"}, 0, skipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var dp map[string]any
			if err := yaml.Unmarshal([]byte(validDP), &dp); err != nil {
				t.Fatal(err)
			}

			var calls atomic.Int32
			cc := func(event console.CompatCheckable, entities []console.CompatCheckable) (*console.CompatResult, error) {
				calls.Add(1)
				return &console.CompatResult{Status: "compatible"}, nil
			}

			lookup, err := NewDPLookup(cc, nil, map[string]map[string]any{file: dp}, tt.changed, tt.validateAll, 1)
			if err != nil {
				t.Fatal(err)
			}

			if calls.Load() != tt.wantCalls {
				t.Errorf("compat checker calls got: %d want: %d", calls.Load(), tt.wantCalls)
			}
			if got := lookup.Validations[file].Debug; !slices.Equal(got, tt.wantDebug) {
				t.Errorf("got: %q want: %q", got, tt.wantDebug)
			}
			if lookup.ValidationErrorCount() != 0 {
				t.Errorf("unexpected errors %#v", lookup.Validations[file])
			}
		})
	}
}

func Test_DPLookup_NoCompatCheckWhenShapeInvalid(t *testing.T) {
	invalidDP := `
apiVersion: v1
resourceType: data-product
resourceName: dbc68917-3bcb-4640-a165-483de5fc033c
data:
  sourceApplications: []
  eventSpecifications:
    - resourceName: b57f712e-ee43-4e10-8b36-48a0f6f2e628
      name: Test Event
      event:
        source: iglu:com.example/test_event/jsonschema/1-0-0
        schema:
          type: object
          additionalProperties: false
          properties:
            testField:
              type: string
      entities:
        tracked: []
        enriched: []
`
	file := "/base/somedir/dp.yml"

	var dp map[string]any
	if err := yaml.Unmarshal([]byte(invalidDP), &dp); err != nil {
		t.Fatal(err)
	}

	var calls atomic.Int32
	cc := func(event console.CompatCheckable, entities []console.CompatCheckable) (*console.CompatResult, error) {
		calls.Add(1)
		return &console.CompatResult{Status: "compatible"}, nil
	}

	lookup, err := NewDPLookup(cc, nil, map[string]map[string]any{file: dp}, nil, true, 1)
	if err != nil {
		t.Fatal(err)
	}

	if calls.Load() != 0 {
		t.Errorf("compat checker calls got: %d want: 0", calls.Load())
	}
	if len(lookup.Validations[file].ErrorsWithPaths) == 0 {
		t.Errorf("expected shape errors, got: %#v", lookup.Validations[file])
	}
}

type countingSdc struct {
	calls int
}

func (c *countingSdc) IsDSDeployed(uri string) (bool, []string, error) {
	c.calls++
	return true, nil, nil
}

func Test_DPLookup_SchemaDeployCheckOnlyWhenShapeValid(t *testing.T) {
	validSA := `
apiVersion: v1
resourceType: source-application
resourceName: 9567c7f6-356e-4f73-a7ec-e5097e4d2f42
data:
  name: Website
  appIds: [web]
  entities:
    tracked:
      - source: iglu:com.example/test_entity/jsonschema/1-0-0
    enriched: []
`
	file := "/base/somedir/sa.yml"

	tests := []struct {
		name      string
		yaml      string
		wantCalls int
		wantValid bool
	}{
		{"shape valid", validSA, 1, true},
		{"shape invalid", strings.Replace(validSA, "  appIds: [web]\n", "", 1), 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sa map[string]any
			if err := yaml.Unmarshal([]byte(tt.yaml), &sa); err != nil {
				t.Fatal(err)
			}

			sdc := &countingSdc{}

			lookup, err := NewDPLookup(nil, sdc, map[string]map[string]any{file: sa}, nil, true, 1)
			if err != nil {
				t.Fatal(err)
			}

			if sdc.calls != tt.wantCalls {
				t.Errorf("schema deploy checker calls got: %d want: %d", sdc.calls, tt.wantCalls)
			}
			if valid := lookup.ValidationErrorCount() == 0; valid != tt.wantValid {
				t.Errorf("valid got: %t want: %t, validations %#v", valid, tt.wantValid, lookup.Validations[file])
			}
		})
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
			lookup.AddVersionMismatches([]model.EventSpecVersionMismatch{tt.mismatch}, "snowplow-cli dp validate --fix")

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
