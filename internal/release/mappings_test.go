/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package release

import (
	"reflect"
	"sort"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/model"
	kjson "k8s.io/apimachinery/pkg/util/json"
)

func TestLocalSaToRemote(t *testing.T) {
	input := model.SourceApp{
		ResourceName: "test-app",
		Data: model.SourceAppData{
			Name:        "Test App",
			Description: "Test Description",
			Owner:       "Test Owner",
			AppIds:      []string{"app1", "app2"},
			Entities: &model.EntitiesDef{
				Tracked: []model.SchemaRef{
					{
						Source:         "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
						MinCardinality: new(0),
						MaxCardinality: new(1),
						Schema:         map[string]any{},
					},
				},
				Enriched: []model.SchemaRef{
					{
						Source:         "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
						MinCardinality: new(0),
						MaxCardinality: new(1),
						Schema:         map[string]any{},
					},
				},
			},
		},
	}

	expected := console.RemoteSourceApplication{
		Id:          "test-app",
		Name:        "Test App",
		Description: "Test Description",
		Owner:       "Test Owner",
		AppIds:      []string{"app1", "app2"},
		Entities: console.Entities{
			Tracked: []console.Entity{
				{
					Source:         "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
					MinCardinality: new(0),
					MaxCardinality: new(1),
					Schema:         map[string]any{},
				},
			},
			Enriched: []console.Entity{
				{
					Source:         "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
					MinCardinality: new(0),
					MaxCardinality: new(1),
					Schema:         map[string]any{},
				},
			},
		},
	}

	result := localSaToRemote(input)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("localSaToRemote() = %v, want %v", result, expected)
	}
}

func TestLocalDpToRemote(t *testing.T) {
	input := model.DataProduct{
		ResourceName: "test-dp",
		Data: model.DataProductData{
			Name: "Test DP",
			SourceApplications: []map[string]string{
				{"id": "app1"},
				{"id": "app2"},
			},
			Domain:      "test-domain",
			Owner:       "test-owner",
			Description: "test-description",
		},
	}

	expected := console.RemoteDataProduct{
		Id:                   "test-dp",
		Name:                 "Test DP",
		SourceApplicationIds: []string{"app1", "app2"},
		Domain:               "test-domain",
		Owner:                "test-owner",
		Description:          "test-description",
	}

	result := LocalDpToRemote(input)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("LocalDpToRemote() = %v, want %v", result, expected)
	}
}

func TestLocalEventSpecToRemote(t *testing.T) {
	input := model.EventSpec{
		ResourceName: "test-event",
		Name:         "Test Event",
		ExcludedSourceApplications: []map[string]string{
			{"id": "app2"},
		},
		Event: model.SchemaRef{
			Source: "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
			Schema: map[string]any{},
		},
		Entities: model.EntitiesDef{
			Tracked: []model.SchemaRef{
				{
					Source:         "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
					MinCardinality: new(0),
					MaxCardinality: new(1),
					Schema:         map[string]any{},
				},
			},
			Enriched: []model.SchemaRef{
				{
					Source:         "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
					MinCardinality: new(0),
					MaxCardinality: new(1),
					Schema:         map[string]any{},
				},
			},
		},
	}

	dpSourceApps := []string{"app1", "app2", "app3"}
	dpId := "dp1"

	result := LocalEventSpecToRemote(input, dpSourceApps, dpId)

	expectedSourceApps := []string{"app1", "app3"}
	sort.Strings(result.SourceApplicationIds)
	sort.Strings(expectedSourceApps)
	if !reflect.DeepEqual(result.SourceApplicationIds, expectedSourceApps) {
		t.Errorf("SourceApplicationIds = %v, want %v", result.SourceApplicationIds, expectedSourceApps)
	}

	if result.Id != "test-event" {
		t.Errorf("Id = %v, want test-event", result.Id)
	}

	if result.DataProductId != dpId {
		t.Errorf("DataProductId = %v, want %v", result.DataProductId, dpId)
	}
}

func TestEsToDiff(t *testing.T) {
	input := console.RemoteEventSpec{
		Id:                   "test-event",
		Name:                 "Test Event",
		SourceApplicationIds: []string{"app1", "app3"},
		Event: &console.EventWrapper{
			Source: "test-source",
			Schema: map[string]any{},
		},
		Entities: console.Entities{
			Tracked: []console.Entity{
				{
					Source:         "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
					MinCardinality: new(0),
					MaxCardinality: new(1),
					Schema:         map[string]any{},
				},
			},
		},
		DataProductId: "dp1",
	}

	result, err := esToDiff(input)
	if err != nil {
		t.Fatalf("esToDiff() error = %v", err)
	}

	// Test source app IDs are sorted
	expectedSourceApps := []string{"app1", "app3"}
	if !reflect.DeepEqual(result.SourceApplicationIds, expectedSourceApps) {
		t.Errorf("SourceApplicationIds = %v, want %v", result.SourceApplicationIds, expectedSourceApps)
	}

	// Verify event JSON
	var eventData map[string]any
	if err := kjson.Unmarshal([]byte(result.Event), &eventData); err != nil {
		t.Fatalf("Failed to unmarshal event JSON: %v", err)
	}
}

func TestDpToDiff(t *testing.T) {
	input := console.RemoteDataProduct{
		Id:                   "test-dp",
		Name:                 "Test DP",
		SourceApplicationIds: []string{"app3", "app1", "app2"},
		Domain:               "test-domain",
		Owner:                "test-owner",
		Description:          "test-description",
	}

	result := dpToDiff(input)

	// Verify source app IDs are sorted
	expectedSourceApps := []string{"app1", "app2", "app3"}
	if !reflect.DeepEqual(result.SourceApplicationIds, expectedSourceApps) {
		t.Errorf("SourceApplicationIds = %v, want %v", result.SourceApplicationIds, expectedSourceApps)
	}

	if result.Name != input.Name {
		t.Errorf("Name = %v, want %v", result.Name, input.Name)
	}

	if result.Domain != input.Domain {
		t.Errorf("Domain = %v, want %v", result.Domain, input.Domain)
	}

	if result.Owner != input.Owner {
		t.Errorf("Owner = %v, want %v", result.Owner, input.Owner)
	}

	if result.Description != input.Description {
		t.Errorf("Description = %v, want %v", result.Description, input.Description)
	}
}

func Test_saToDiff(t *testing.T) {
	input := console.RemoteSourceApplication{
		Id:          "id",
		LockStatus:  "locked",
		ManagedFrom: "somewhere",
	}

	result := saToDiff(input)

	if result.LockStatus != "" && result.ManagedFrom != "" {
		t.Errorf("failed to setup remote source app for sensible comparison")
	}
}

func Test_saToDiff_IgnoresEntityComments(t *testing.T) {
	comment := "a note"
	remote := console.RemoteSourceApplication{
		Id: "id",
		Entities: console.Entities{
			Tracked:  []console.Entity{{Source: "iglu:com.example/a/jsonschema/1-0-0", Comment: &comment}},
			Enriched: []console.Entity{},
		},
	}
	local := console.RemoteSourceApplication{
		Id: "id",
		Entities: console.Entities{
			Tracked:  []console.Entity{{Source: "iglu:com.example/a/jsonschema/1-0-0"}},
			Enriched: []console.Entity{},
		},
	}

	if !reflect.DeepEqual(saToDiff(remote), saToDiff(local)) {
		t.Errorf("an entity comment should not make a source app differ")
	}
	if remote.Entities.Tracked[0].Comment == nil {
		t.Errorf("saToDiff should not change its input")
	}
}

func Test_structuralEntity_LeavesCommentOutOfWhatIsSent(t *testing.T) {
	comment := "a note"
	entity := console.Entity{Source: "iglu:com.example/a/jsonschema/1-0-0", Comment: &comment}

	remote := structuralEntity(entity, false)
	if remote["comment"] != comment {
		t.Errorf("comment = %v, want %q for an entity read from Console", remote["comment"], comment)
	}

	sent := structuralEntity(entity, true)
	if _, ok := sent["comment"]; ok {
		t.Errorf("comment = %v, want none for an entity sent to Console", sent["comment"])
	}
}

func Test_structuralEvent(t *testing.T) {
	schema := map[string]any{"type": "object"}

	for _, tt := range []struct {
		name   string
		event  *console.EventWrapper
		isSent bool
		want   any
	}{
		{name: "no event sent", event: nil, isSent: true, want: nil},
		{name: "no event read", event: nil, isSent: false, want: nil},
		{name: "empty event sent", event: &console.EventWrapper{}, isSent: true, want: nil},
		{name: "empty event read", event: &console.EventWrapper{}, isSent: false, want: map[string]any{"source": ""}},
		{name: "source only sent", event: &console.EventWrapper{Source: "s"}, isSent: true, want: map[string]any{"source": "s"}},
		{name: "schema only sent", event: &console.EventWrapper{Schema: schema}, isSent: true, want: map[string]any{"source": "", "schema": schema}},
		{name: "empty schema sent", event: &console.EventWrapper{Source: "s", Schema: map[string]any{}}, isSent: true, want: map[string]any{"source": "s"}},
		{name: "empty schema read", event: &console.EventWrapper{Source: "s", Schema: map[string]any{}}, isSent: false, want: map[string]any{"source": "s", "schema": map[string]any{}}},
		{name: "schema read", event: &console.EventWrapper{Source: "s", Schema: schema}, isSent: false, want: map[string]any{"source": "s", "schema": schema}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := structuralEvent(tt.event, tt.isSent); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_structuralEntity_Schema(t *testing.T) {
	schema := map[string]any{"type": "object"}

	for _, tt := range []struct {
		name       string
		schema     map[string]any
		isSent     bool
		wantSchema bool
	}{
		{name: "schema sent", schema: schema, isSent: true, wantSchema: true},
		{name: "empty schema sent", schema: map[string]any{}, isSent: true, wantSchema: false},
		{name: "no schema sent", schema: nil, isSent: true, wantSchema: false},
		{name: "schema read", schema: schema, isSent: false, wantSchema: true},
		{name: "empty schema read", schema: map[string]any{}, isSent: false, wantSchema: true},
		{name: "no schema read", schema: nil, isSent: false, wantSchema: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := structuralEntity(console.Entity{Source: "s", Schema: tt.schema}, tt.isSent)
			if _, ok := got["schema"]; ok != tt.wantSchema {
				t.Errorf("has schema = %t, want %t: %#v", ok, tt.wantSchema, got)
			}
		})
	}
}
