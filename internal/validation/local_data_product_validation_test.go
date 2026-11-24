/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package validation

import (
	"errors"
	"reflect"
	"slices"
	"sync"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/model"
)

func newValidDPForCompatTesting() model.DataProduct {
	return model.DataProduct{
		Data: model.DataProductData{
			EventSpecifications: []model.EventSpec{
				{
					Event: model.SchemaRef{
						Source: "iglu:vendor/event/format/version",
						Schema: map[string]any{
							"key": "value",
						},
					},
					Entities: model.EntitiesDef{
						Tracked: []model.SchemaRef{{
							Source: "iglu:vendor/entity/format/version",
							Schema: map[string]any{
								"key":      "value",
								"otherKey": "something",
							}},
						},
					},
				},
			},
		},
	}
}

func Test_ValidateDPEventSpecCompatOk(t *testing.T) {
	dp := newValidDPForCompatTesting()

	cc := func(event console.CompatCheckable, entities []console.CompatCheckable) (*console.CompatResult, error) {
		return &console.CompatResult{Status: "compatible"}, nil
	}

	result := ValidateDPEventSpecCompat(cc, 1, dp)

	if len(result.Errors) > 0 || len(result.Warnings) > 0 {
		t.Fatal("unexpected failures")
	}
}

func Test_ValidateDPEventSpecCompatMissingEvent(t *testing.T) {
	dp := newValidDPForCompatTesting()
	dp.Data.EventSpecifications[0].Event = model.SchemaRef{}

	result := ValidateDPEventSpecCompat(nil, 1, dp).WarningsWithPaths

	expectPath := "/data/eventSpecifications/0"
	expectValue := []string{"will not run compatibility checks on entities without an event defined"}

	if resultValue, ok := result[expectPath]; !ok || !slices.Equal(expectValue, resultValue) {
		t.Fatalf("unexpected warnings %#v", result)
	}
}

func Test_ValidateDPEventSpecCompatNothingToCheckNoWarning(t *testing.T) {
	dp := newValidDPForCompatTesting()
	dp.Data.EventSpecifications[0].Event = model.SchemaRef{}
	dp.Data.EventSpecifications[0].Entities = model.EntitiesDef{}

	result := ValidateDPEventSpecCompat(nil, 1, dp)

	if len(result.WarningsWithPaths) != 0 || len(result.ErrorsWithPaths) != 0 {
		t.Fatalf("unexpected validations %#v", result)
	}
}

func Test_ValidateDPEventSpecCompatChecksSpecsAfterOneWithoutEvent(t *testing.T) {
	dp := newDPForCompatPathTesting()
	dp.Data.EventSpecifications[0].Event = model.SchemaRef{}
	cc := &recordingCompatChecker{}

	ValidateDPEventSpecCompat(cc.check, 2, dp)

	wantEvents := map[string]map[string]any{
		"iglu:vendor/event_b/format/1-0-0": {"is": "event_b"},
	}
	if cc.calls != 1 || !reflect.DeepEqual(cc.events, wantEvents) {
		t.Errorf("calls got: %d events got: %#v want: %#v", cc.calls, cc.events, wantEvents)
	}
}

func Test_ValidateDPEventSpecCompatEnrichedEntityWithoutSchemaNotSent(t *testing.T) {
	dp := newDPForCompatPathTesting()
	dp.Data.EventSpecifications[1].Entities.Enriched[0].Schema = map[string]any{}
	cc := &recordingCompatChecker{}

	ValidateDPEventSpecCompat(cc.check, 2, dp)

	sent := cc.entities["iglu:vendor/event_b/format/1-0-0"]
	if _, ok := sent["iglu:vendor/enriched_b0/format/1-0-0"]; ok {
		t.Errorf("an enriched entity without a schema was sent: %#v", sent)
	}
	if _, ok := sent["iglu:vendor/enriched_b1/format/1-0-0"]; !ok {
		t.Errorf("an enriched entity with a schema was not sent: %#v", sent)
	}
}

func Test_ValidateDPEventSpecCompatFail(t *testing.T) {
	dp := newValidDPForCompatTesting()

	cc := func(event console.CompatCheckable, entities []console.CompatCheckable) (*console.CompatResult, error) {
		return &console.CompatResult{
			Status: "incompatible",
			Sources: []console.CompatSource{
				{
					Source: dp.Data.EventSpecifications[0].Event.Source,
					Status: "incompatible",
					Properties: map[string]string{
						"key": "incompatible",
					},
				},
				{
					Source: dp.Data.EventSpecifications[0].Entities.Tracked[0].Source,
					Status: "incompatible",
					Properties: map[string]string{
						"key":      "incompatible",
						"otherKey": "undecidable",
					},
				},
			},
		}, nil
	}

	result := ValidateDPEventSpecCompat(cc, 2, dp)

	expectedErrorPaths := []string{
		"/data/eventSpecifications/0/event/schema/key",
		"/data/eventSpecifications/0/entities/tracked/0/schema/key",
	}

	for _, path := range expectedErrorPaths {
		if _, ok := result.ErrorsWithPaths[path]; !ok {
			t.Fatalf("missing error at: %s", path)
		}
	}

	expectedWarningsPath := "/data/eventSpecifications/0/entities/tracked/0/schema/otherKey"

	if _, ok := result.WarningsWithPaths[expectedWarningsPath]; !ok {
		t.Log(result.WarningsWithPaths)
		t.Fatalf("missing warning at: %s", expectedWarningsPath)
	}
}

func Test_ValidateDPEventSpecCompatUnknownSourceSkipped(t *testing.T) {
	dp := newValidDPForCompatTesting()

	cc := func(event console.CompatCheckable, entities []console.CompatCheckable) (*console.CompatResult, error) {
		return &console.CompatResult{
			Status: "incompatible",
			Sources: []console.CompatSource{
				{
					Source: "iglu:vendor/unknown/format/version",
					Status: "incompatible",
					Properties: map[string]string{
						"key": "incompatible",
					},
				},
				{
					Source: "iglu:vendor/alsounknown/format/version",
					Status: "undecidable",
					Properties: map[string]string{
						"key": "undecidable",
					},
				},
				{
					Source: dp.Data.EventSpecifications[0].Event.Source,
					Status: "incompatible",
				},
			},
		}, nil
	}

	result := ValidateDPEventSpecCompat(cc, 1, dp)

	expectedErrors := map[string][]string{
		"/data/eventSpecifications/0/event/schema": {"definition incompatible with source data structure (iglu:vendor/event/format/version)"},
	}

	if !reflect.DeepEqual(result.ErrorsWithPaths, expectedErrors) {
		t.Fatalf("unexpected errors got: %#v want %#v", result.ErrorsWithPaths, expectedErrors)
	}

	if len(result.WarningsWithPaths) != 0 || len(result.Errors) != 0 {
		t.Fatalf("unexpected warnings %#v or errors %#v", result.WarningsWithPaths, result.Errors)
	}
}

func Test_ValidateDPEventSpecCompatCheckerError(t *testing.T) {
	dp := newValidDPForCompatTesting()

	cc := func(event console.CompatCheckable, entities []console.CompatCheckable) (*console.CompatResult, error) {
		return nil, errors.New("boom")
	}

	result := ValidateDPEventSpecCompat(cc, 1, dp)

	expected := []string{"unexpected error checking compatibility: boom"}

	if !slices.Equal(result.Errors, expected) {
		t.Fatalf("unexpected got: %#v want %#v", result.Errors, expected)
	}

	if len(result.ErrorsWithPaths) != 0 || len(result.WarningsWithPaths) != 0 {
		t.Fatalf("unexpected errors %#v or warnings %#v with paths", result.ErrorsWithPaths, result.WarningsWithPaths)
	}
}

func newDPForCompatPathTesting() model.DataProduct {
	return model.DataProduct{
		Data: model.DataProductData{
			EventSpecifications: []model.EventSpec{
				{
					Event: model.SchemaRef{
						Source: "iglu:vendor/event_a/format/1-0-0",
						Schema: map[string]any{"is": "event_a"},
					},
					Entities: model.EntitiesDef{
						Tracked: []model.SchemaRef{
							{Source: "iglu:vendor/tracked_a0/format/1-0-0", Schema: map[string]any{"is": "tracked_a0"}},
						},
					},
				},
				{
					Event: model.SchemaRef{
						Source: "iglu:vendor/event_b/format/1-0-0",
						Schema: map[string]any{"is": "event_b"},
					},
					Entities: model.EntitiesDef{
						Tracked: []model.SchemaRef{
							{Source: "iglu:vendor/tracked_b0/format/1-0-0", Schema: map[string]any{"is": "tracked_b0"}},
							{Source: "iglu:vendor/tracked_b1/format/1-0-0"},
							{Source: "iglu:vendor/tracked_b2/format/1-0-0", Schema: map[string]any{"is": "tracked_b2"}},
						},
						Enriched: []model.SchemaRef{
							{Source: "iglu:vendor/enriched_b0/format/1-0-0", Schema: map[string]any{"is": "enriched_b0"}},
							{Source: "iglu:vendor/enriched_b1/format/1-0-0", Schema: map[string]any{"is": "enriched_b1"}},
						},
					},
				},
			},
		},
	}
}

type recordingCompatChecker struct {
	mu       sync.Mutex
	calls    int
	events   map[string]map[string]any
	entities map[string]map[string]map[string]any
	results  map[string][]console.CompatSource
}

func (r *recordingCompatChecker) check(event console.CompatCheckable, entities []console.CompatCheckable) (*console.CompatResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.events == nil {
		r.events = map[string]map[string]any{}
		r.entities = map[string]map[string]map[string]any{}
	}
	r.calls++
	r.events[event.Source] = event.Schema
	r.entities[event.Source] = map[string]map[string]any{}
	for _, e := range entities {
		r.entities[event.Source][e.Source] = e.Schema
	}

	return &console.CompatResult{Status: "undecidable", Sources: r.results[event.Source]}, nil
}

func Test_ValidateDPEventSpecCompatSendsEventWithItsEntities(t *testing.T) {
	cc := &recordingCompatChecker{}

	result := ValidateDPEventSpecCompat(cc.check, 2, newDPForCompatPathTesting())

	wantEvents := map[string]map[string]any{
		"iglu:vendor/event_a/format/1-0-0": {"is": "event_a"},
		"iglu:vendor/event_b/format/1-0-0": {"is": "event_b"},
	}
	wantEntities := map[string]map[string]map[string]any{
		"iglu:vendor/event_a/format/1-0-0": {
			"iglu:vendor/tracked_a0/format/1-0-0": {"is": "tracked_a0"},
		},
		"iglu:vendor/event_b/format/1-0-0": {
			"iglu:vendor/tracked_b0/format/1-0-0":  {"is": "tracked_b0"},
			"iglu:vendor/tracked_b2/format/1-0-0":  {"is": "tracked_b2"},
			"iglu:vendor/enriched_b0/format/1-0-0": {"is": "enriched_b0"},
			"iglu:vendor/enriched_b1/format/1-0-0": {"is": "enriched_b1"},
		},
	}

	if cc.calls != 2 {
		t.Errorf("compat checker calls got: %d want: 2", cc.calls)
	}
	if !reflect.DeepEqual(cc.events, wantEvents) {
		t.Errorf("events got: %#v want: %#v", cc.events, wantEvents)
	}
	if !reflect.DeepEqual(cc.entities, wantEntities) {
		t.Errorf("entities got: %#v want: %#v", cc.entities, wantEntities)
	}
	if len(result.Errors) != 0 || len(result.ErrorsWithPaths) != 0 || len(result.WarningsWithPaths) != 0 {
		t.Errorf("unexpected validations %#v", result)
	}
}

func Test_ValidateDPEventSpecCompatEntitiesWithoutEventNotSent(t *testing.T) {
	dp := newDPForCompatPathTesting()
	dp.Data.EventSpecifications[1].Event = model.SchemaRef{}
	cc := &recordingCompatChecker{}

	result := ValidateDPEventSpecCompat(cc.check, 2, dp)

	wantEvents := map[string]map[string]any{
		"iglu:vendor/event_a/format/1-0-0": {"is": "event_a"},
	}
	wantWarnings := map[string][]string{
		"/data/eventSpecifications/1": {"will not run compatibility checks on entities without an event defined"},
	}

	if cc.calls != 1 || !reflect.DeepEqual(cc.events, wantEvents) {
		t.Errorf("calls got: %d events got: %#v want: %#v", cc.calls, cc.events, wantEvents)
	}
	if !reflect.DeepEqual(result.WarningsWithPaths, wantWarnings) {
		t.Errorf("warnings got: %#v want: %#v", result.WarningsWithPaths, wantWarnings)
	}
	if len(result.Errors) != 0 || len(result.ErrorsWithPaths) != 0 {
		t.Errorf("unexpected errors %#v or errors with paths %#v", result.Errors, result.ErrorsWithPaths)
	}
}

func Test_ValidateDPEventSpecCompatUndecidableIsWarning(t *testing.T) {
	cc := &recordingCompatChecker{results: map[string][]console.CompatSource{
		"iglu:vendor/event_a/format/1-0-0": {
			{Source: "iglu:vendor/event_a/format/1-0-0", Status: "undecidable"},
			{Source: "iglu:vendor/tracked_a0/format/1-0-0", Status: "compatible"},
		},
		"iglu:vendor/event_b/format/1-0-0": {
			{Source: "iglu:vendor/event_b/format/1-0-0", Status: "compatible"},
			{Source: "iglu:vendor/enriched_b1/format/1-0-0", Status: "undecidable"},
		},
	}}

	result := ValidateDPEventSpecCompat(cc.check, 2, newDPForCompatPathTesting())

	wantWarnings := map[string][]string{
		"/data/eventSpecifications/0/event/schema": {
			"definition has unknown compatibility with source data structure (iglu:vendor/event_a/format/1-0-0)",
		},
		"/data/eventSpecifications/1/entities/enriched/1/schema": {
			"definition has unknown compatibility with source data structure (iglu:vendor/enriched_b1/format/1-0-0)",
		},
	}

	if !reflect.DeepEqual(result.WarningsWithPaths, wantWarnings) {
		t.Errorf("warnings got: %#v want: %#v", result.WarningsWithPaths, wantWarnings)
	}
	if len(result.Errors) != 0 || len(result.ErrorsWithPaths) != 0 {
		t.Errorf("unexpected errors %#v or errors with paths %#v", result.Errors, result.ErrorsWithPaths)
	}
}

func Test_ValidateDPEventSpecCompatIncompatiblePaths(t *testing.T) {
	cc := &recordingCompatChecker{results: map[string][]console.CompatSource{
		"iglu:vendor/event_a/format/1-0-0": {
			{Source: "iglu:vendor/event_a/format/1-0-0", Status: "compatible"},
			{Source: "iglu:vendor/tracked_a0/format/1-0-0", Status: "incompatible"},
		},
		"iglu:vendor/event_b/format/1-0-0": {
			{Source: "iglu:vendor/event_b/format/1-0-0", Status: "incompatible"},
			{Source: "iglu:vendor/tracked_b0/format/1-0-0", Status: "compatible"},
			{Source: "iglu:vendor/tracked_b2/format/1-0-0", Status: "incompatible"},
			{Source: "iglu:vendor/enriched_b0/format/1-0-0", Status: "compatible"},
			{Source: "iglu:vendor/enriched_b1/format/1-0-0", Status: "incompatible"},
		},
	}}

	result := ValidateDPEventSpecCompat(cc.check, 2, newDPForCompatPathTesting())

	wantErrors := map[string][]string{
		"/data/eventSpecifications/0/entities/tracked/0/schema": {
			"definition incompatible with source data structure (iglu:vendor/tracked_a0/format/1-0-0)",
		},
		"/data/eventSpecifications/1/event/schema": {
			"definition incompatible with source data structure (iglu:vendor/event_b/format/1-0-0)",
		},
		"/data/eventSpecifications/1/entities/tracked/2/schema": {
			"definition incompatible with source data structure (iglu:vendor/tracked_b2/format/1-0-0)",
		},
		"/data/eventSpecifications/1/entities/enriched/1/schema": {
			"definition incompatible with source data structure (iglu:vendor/enriched_b1/format/1-0-0)",
		},
	}

	if !reflect.DeepEqual(result.ErrorsWithPaths, wantErrors) {
		t.Errorf("errors got: %#v want: %#v", result.ErrorsWithPaths, wantErrors)
	}
	if len(result.Errors) != 0 || len(result.WarningsWithPaths) != 0 {
		t.Errorf("unexpected errors %#v or warnings %#v", result.Errors, result.WarningsWithPaths)
	}
}

func Test_ValidateDPEventSpecCompatPropertyResults(t *testing.T) {
	cc := &recordingCompatChecker{results: map[string][]console.CompatSource{
		"iglu:vendor/event_b/format/1-0-0": {
			{
				Source:     "iglu:vendor/event_b/format/1-0-0",
				Status:     "compatible",
				Properties: map[string]string{"price": "incompatible", "currency": "compatible"},
			},
			{
				Source:     "iglu:vendor/tracked_b2/format/1-0-0",
				Status:     "compatible",
				Properties: map[string]string{"sku": "undecidable"},
			},
			{
				Source:     "iglu:vendor/enriched_b1/format/1-0-0",
				Status:     "compatible",
				Properties: map[string]string{"country": "incompatible", "city": "undecidable"},
			},
		},
	}}

	result := ValidateDPEventSpecCompat(cc.check, 2, newDPForCompatPathTesting())

	wantErrors := map[string][]string{
		"/data/eventSpecifications/1/event/schema/price": {
			"definition incompatible with .price in source data structure (iglu:vendor/event_b/format/1-0-0)",
		},
		"/data/eventSpecifications/1/entities/enriched/1/schema/country": {
			"definition incompatible with .country in source data structure (iglu:vendor/enriched_b1/format/1-0-0)",
		},
	}
	wantWarnings := map[string][]string{
		"/data/eventSpecifications/1/entities/tracked/2/schema/sku": {
			"definition has unknown compatibility with .sku in source data structure (iglu:vendor/tracked_b2/format/1-0-0)",
		},
		"/data/eventSpecifications/1/entities/enriched/1/schema/city": {
			"definition has unknown compatibility with .city in source data structure (iglu:vendor/enriched_b1/format/1-0-0)",
		},
	}

	if !reflect.DeepEqual(result.ErrorsWithPaths, wantErrors) {
		t.Errorf("errors got: %#v want: %#v", result.ErrorsWithPaths, wantErrors)
	}
	if !reflect.DeepEqual(result.WarningsWithPaths, wantWarnings) {
		t.Errorf("warnings got: %#v want: %#v", result.WarningsWithPaths, wantWarnings)
	}
	if len(result.Errors) != 0 {
		t.Errorf("unexpected errors %#v", result.Errors)
	}
}
