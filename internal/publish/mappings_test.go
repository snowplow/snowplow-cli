/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/
package publish

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/snowplow-product/snowplow-cli/internal/console"
	"github.com/snowplow-product/snowplow-cli/internal/model"
)

func intPtr(i int) *int {
	return &i
}

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
						MinCardinality: intPtr(0),
						MaxCardinality: intPtr(1),
						Schema:         map[string]any{},
					},
				},
				Enriched: []model.SchemaRef{
					{
						Source:         "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
						MinCardinality: intPtr(0),
						MaxCardinality: intPtr(1),
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
					MinCardinality: intPtr(0),
					MaxCardinality: intPtr(1),
					Schema:         map[string]any{},
				},
			},
			Enriched: []console.Entity{
				{
					Source:         "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
					MinCardinality: intPtr(0),
					MaxCardinality: intPtr(1),
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
					MinCardinality: intPtr(0),
					MaxCardinality: intPtr(1),
					Schema:         map[string]any{},
				},
			},
			Enriched: []model.SchemaRef{
				{
					Source:         "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
					MinCardinality: intPtr(0),
					MaxCardinality: intPtr(1),
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
			Event: console.Event{
				Source: "test-source",
				Schema: map[string]any{},
			},
		},
		Entities: console.Entities{
			Tracked: []console.Entity{
				{
					Source:         "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
					MinCardinality: intPtr(0),
					MaxCardinality: intPtr(1),
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
	var eventData map[string]interface{}
	if err := json.Unmarshal([]byte(result.Event), &eventData); err != nil {
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
