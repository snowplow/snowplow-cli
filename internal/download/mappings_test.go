/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/
package download

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/model"
)

func intPtr(i int) *int {
	return &i
}

var sampleEntity1 = console.Entity{
	Source:         "iglu:com.snplow.msc.aws/data-product-auto/jsonschema/1-0-0",
	MinCardinality: intPtr(0),
	MaxCardinality: intPtr(5),
	Schema:         map[string]any{},
}

var sampleEntity2 = console.Entity{
	Source:         "iglu:com.snplow.msc.aws/event-spec-auto/jsonschema/1-0-0",
	MinCardinality: intPtr(0),
	MaxCardinality: nil,
	Schema:         map[string]any{},
}

var sampleSa1 = console.RemoteSourceApplication{
	Id:          "6b1146d6-7b23-4dbb-b069-f568458dda8f",
	Name:        "test",
	Description: "my test sa",
	Owner:       "me@me.com",
	AppIds:      []string{"ios", "android"},
	Entities:    console.Entities{Tracked: []console.Entity{sampleEntity1, sampleEntity2}, Enriched: []console.Entity{}},
}

var sampleSaRef = model.Ref{
	Ref: "./source-apps/test.yaml",
}

var sampleSaRefs = map[string]model.Ref{
	sampleSa1.Id: sampleSaRef,
}

var sampleSaRefs2 = map[string]model.Ref{
	sampleSa1.Id: sampleSaRef,
	"8fb370f4-60af-4b7d-9438-ea08df6cdc70": model.Ref{
		Ref: "./source-apps/wrong-sa.yaml",
	},
}

var trackedEntites = []model.SchemaRef{
	{
		Source:         sampleEntity1.Source,
		MinCardinality: sampleEntity1.MinCardinality,
		MaxCardinality: sampleEntity1.MaxCardinality,
		Schema:         map[string]any{},
	},
	{
		Source:         sampleEntity2.Source,
		MinCardinality: sampleEntity2.MinCardinality,
		MaxCardinality: sampleEntity2.MaxCardinality,
		Schema:         map[string]any{},
	},
}
var entities = model.EntitiesDef{Tracked: trackedEntites, Enriched: nil}
var SaResource = []model.CliResource[model.SourceAppData]{
	{
		ApiVersion:   "v1",
		ResourceType: "source-application",
		ResourceName: sampleSa1.Id,
		Data: model.SourceAppData{
			ResourceName: sampleSa1.Id,
			Name:         sampleSa1.Name,
			Description:  sampleSa1.Description,
			Owner:        sampleSa1.Owner,
			AppIds:       sampleSa1.AppIds,
			Entities:     &entities,
		},
	},
}

var sampleRemoteTrigger = console.RemoteTrigger{
	Id:          "f843f882-45ec-4a1c-a401-f8986c5120a3",
	Description: "test trigger",
	AppIds:      []string{"one"},
	Url:         "example.com",
	VariantUrls: map[string]string{"original": "example.com/orig", "thumbnail": "example.com/small"},
}

var sampleRemoteEs = console.RemoteEventSpec{
	Id:                   "84614b3b-6039-458e-8ce2-615eaf2113e3",
	SourceApplicationIds: []string{sampleSa1.Id},
	Name:                 "test ES",
	Event: &console.EventWrapper{Event: console.Event{
		Source: "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
		Schema: map[string]any{},
	},
	},
	Entities: console.Entities{Tracked: []console.Entity{
		{
			Source:         sampleEntity1.Source,
			MinCardinality: sampleEntity1.MinCardinality,
			MaxCardinality: sampleEntity1.MaxCardinality,
			Schema:         map[string]any{},
		},
		{
			Source:         sampleEntity2.Source,
			MinCardinality: sampleEntity2.MinCardinality,
			MaxCardinality: sampleEntity2.MaxCardinality,
			Schema:         map[string]any{},
		},
	},
		Enriched: nil,
	},
	Triggers: []console.RemoteTrigger{sampleRemoteTrigger},
}

var sampleRemoteEss = []console.RemoteEventSpec{sampleRemoteEs}

var sampleEsIdToEs = map[string]console.RemoteEventSpec{sampleRemoteEs.Id: sampleRemoteEs}

var sampleRemoteDp = console.RemoteDataProduct{
	Id:                   "46d47289-f3d5-4ef8-a82c-b19597e6e503",
	Name:                 "test DP",
	SourceApplicationIds: []string{sampleSa1.Id},
	Domain:               "testing",
	Owner:                "me@me.me",
	Description:          "this is a test",
	EventSpecs:           []console.EventSpecReference{{Id: sampleRemoteEss[0].Id}},
}

var sampleRemoteDps = []console.RemoteDataProduct{sampleRemoteDp}

func Test_remoteSasToLocalResources_OK(t *testing.T) {
	res := remoteSasToLocalResources([]console.RemoteSourceApplication{sampleSa1})
	if !reflect.DeepEqual(SaResource, res) {
		t.Errorf("Unexpected local source apps expected:%+v got:%+v", SaResource, res)
	}

}

func Test_localSasToRefs_OK(t *testing.T) {
	dpLocation := "my-folder"

	fileNamesToLocalSas := map[string]model.CliResource[model.SourceAppData]{
		fmt.Sprintf("%s/source-apps/test.yaml", dpLocation): SaResource[0],
	}
	res, err := localSasToRefs(fileNamesToLocalSas, dpLocation)

	if !reflect.DeepEqual(sampleSaRefs, res) || err != nil {
		t.Errorf("Unexpected source app references expected:%+v got:%+v, \nerr:%s", sampleSaRefs, res, err)
	}

}

func Test_localSasToRefs_AbsolutePath(t *testing.T) {
	dpLocation := "/tmp/data-products"

	fileNamesToLocalSas := map[string]model.CliResource[model.SourceAppData]{
		"/tmp/data-products/source-apps/test.yaml": SaResource[0],
	}
	res, err := localSasToRefs(fileNamesToLocalSas, dpLocation)

	expectedRefs := map[string]model.Ref{
		sampleSa1.Id: {Ref: "./source-apps/test.yaml"},
	}

	if !reflect.DeepEqual(expectedRefs, res) || err != nil {
		t.Errorf("localSasToRefs with absolute data products path '%s' should generate relative refs './source-apps/test.yaml', expected:%+v got:%+v, err:%s", dpLocation, expectedRefs, res, err)
	}
}

func Test_remoteDpsToLocalResources_OK(t *testing.T) {
	res := remoteDpsToLocalResources(sampleRemoteDps, sampleSaRefs2, sampleEsIdToEs, map[string]string{sampleRemoteTrigger.Id: "./images/test"})
	expected := []model.CliResource[model.DataProductCanonicalData]{{
		ApiVersion:   "v1",
		ResourceType: "data-product",
		ResourceName: sampleRemoteDp.Id,
		Data: model.DataProductCanonicalData{
			ResourceName:       sampleRemoteDp.Id,
			Name:               sampleRemoteDp.Name,
			SourceApplications: []model.Ref{sampleSaRef},
			Domain:             sampleRemoteDp.Domain,
			Owner:              sampleRemoteDp.Owner,
			Description:        sampleRemoteDp.Description,
			EventSpecifications: []model.EventSpecCanonical{{
				ResourceName:               sampleRemoteEs.Id,
				ExcludedSourceApplications: nil,
				Name:                       sampleRemoteEs.Name,
				Event: &model.SchemaRef{
					Source:         sampleRemoteEs.Event.Source,
					MinCardinality: nil,
					MaxCardinality: nil,
					Schema:         map[string]any{},
				},
				Entities: model.EntitiesDef{
					Tracked:  trackedEntites,
					Enriched: nil,
				},
				Triggers: []model.Trigger{{
					Id:          sampleRemoteTrigger.Id,
					Description: sampleRemoteTrigger.Description,
					AppIds:      sampleRemoteTrigger.AppIds,
					Url:         sampleRemoteTrigger.Url,
					Image: &model.Ref{
						Ref: "./images/test",
					},
				}},
			}},
		},
	}}

	if !reflect.DeepEqual(expected, res) {
		t.Errorf("Unexpected local data products expected:%+v got:%+v", expected, res)
	}

}

func Test_remoteEsToTriggerIdToUrl_OK(t *testing.T) {
	var sampleRemoteEs = []console.RemoteEventSpec{{
		Id:                   "84614b3b-6039-458e-8ce2-615eaf2113e3",
		SourceApplicationIds: []string{sampleSa1.Id},
		Name:                 "test ES",
		Event: &console.EventWrapper{Event: console.Event{
			Source: "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
			Schema: map[string]any{},
		},
		},
		Triggers: []console.RemoteTrigger{{
			Id:          "11d52733-bf70-4758-b64d-c4eb694003e3",
			Description: "test",
			AppIds:      []string{"one", "two"},
			Url:         "http://example.com",
			VariantUrls: map[string]string{"original": "http://example.com/nice", "fake": "http://example.com/bad"},
		}, {
			Id:          "90591260-7277-4319-8f66-30ad31c6bd85",
			Description: "test",
			AppIds:      []string{"three"},
			Url:         "http://example.com",
			VariantUrls: map[string]string{"original": "http://example.com/test", "fake": "http://example.com/bad2"},
		},
		},
	},
		{
			Id:                   "39b421d4-5f1a-4ca1-8f89-559cf72f0d91",
			SourceApplicationIds: []string{sampleSa1.Id},
			Name:                 "test ES",
			Event: &console.EventWrapper{Event: console.Event{
				Source: "iglu:com.yalo.schemas.events.channel/YaloMessage/jsonschema/1-0-0",
				Schema: map[string]any{},
			},
			},
			Triggers: []console.RemoteTrigger{{
				Id:          "291a4a6b-b46a-4d16-87d1-c3464139b8d9",
				Description: "test",
				AppIds:      []string{"one", "two"},
				Url:         "http://example.com",
				VariantUrls: map[string]string{"original": "http://example.com/other_dp", "fake": "http://example.com/bad"},
			},
			},
		},
	}

	res := remoteEsToTriggerIdToUrlAndFilename(sampleRemoteEs)
	expected := map[string]imgUrlFile{
		"11d52733-bf70-4758-b64d-c4eb694003e3": imgUrlFile{"http://example.com/nice", "test-es_1"},
		"291a4a6b-b46a-4d16-87d1-c3464139b8d9": imgUrlFile{"http://example.com/other_dp", "test-es_2"},
		"90591260-7277-4319-8f66-30ad31c6bd85": imgUrlFile{"http://example.com/test", "test-es_3"}}

	if !reflect.DeepEqual(expected, res) {
		t.Errorf("Unexpected trigger urls expected:%+v got:%+v", expected, res)
	}

}

func Test_remoteDpToLocal_SourceAppOrderingConsistency(t *testing.T) {
	saRefs := map[string]model.Ref{
		"c-third":  {Ref: "./source-apps/third.yaml"},
		"a-first":  {Ref: "./source-apps/first.yaml"},
		"b-second": {Ref: "./source-apps/second.yaml"},
	}

	dp1 := console.RemoteDataProduct{
		Id:                   "test-dp",
		Name:                 "Test DP",
		SourceApplicationIds: []string{"c-third", "a-first", "b-second"},
		EventSpecs:           []console.EventSpecReference{},
	}

	dp2 := console.RemoteDataProduct{
		Id:                   "test-dp",
		Name:                 "Test DP",
		SourceApplicationIds: []string{"b-second", "c-third", "a-first"},
		EventSpecs:           []console.EventSpecReference{},
	}

	result1 := remoteDpToLocal(dp1, saRefs, map[string]console.RemoteEventSpec{}, map[string]string{})
	result2 := remoteDpToLocal(dp2, saRefs, map[string]console.RemoteEventSpec{}, map[string]string{})

	expectedRefs := []model.Ref{
		{Ref: "./source-apps/first.yaml"},
		{Ref: "./source-apps/second.yaml"},
		{Ref: "./source-apps/third.yaml"},
	}

	if !reflect.DeepEqual(result1.SourceApplications, expectedRefs) {
		t.Errorf("First result SourceApplications = %v, want %v", result1.SourceApplications, expectedRefs)
	}

	if !reflect.DeepEqual(result2.SourceApplications, expectedRefs) {
		t.Errorf("Second result SourceApplications = %v, want %v", result2.SourceApplications, expectedRefs)
	}

	if !reflect.DeepEqual(result1.SourceApplications, result2.SourceApplications) {
		t.Errorf("Results differ based on input order. Result1=%v, Result2=%v",
			result1.SourceApplications, result2.SourceApplications)
	}
}

func Test_remoteEsToLocal_ExclusionOrderingConsistency(t *testing.T) {
	saRefs := map[string]model.Ref{
		"exclude-b": {Ref: "./source-apps/exclude-b.yaml"},
		"exclude-a": {Ref: "./source-apps/exclude-a.yaml"},
		"include-x": {Ref: "./source-apps/include-x.yaml"},
	}

	es := console.RemoteEventSpec{
		Id:                   "test-es",
		Name:                 "Test ES",
		SourceApplicationIds: []string{"include-x"},
		Event: &console.EventWrapper{Event: console.Event{
			Source: "iglu:test/schema/jsonschema/1-0-0",
			Schema: map[string]any{},
		}},
		Entities: console.Entities{Tracked: []console.Entity{}, Enriched: []console.Entity{}},
	}

	dpSourceApps1 := []string{"exclude-b", "exclude-a", "include-x"}
	dpSourceApps2 := []string{"include-x", "exclude-a", "exclude-b"}

	result1 := remoteEsToLocal(es, saRefs, dpSourceApps1, map[string]string{})
	result2 := remoteEsToLocal(es, saRefs, dpSourceApps2, map[string]string{})

	expectedExclusions := []model.Ref{
		{Ref: "./source-apps/exclude-a.yaml"},
		{Ref: "./source-apps/exclude-b.yaml"},
	}

	if !reflect.DeepEqual(result1.ExcludedSourceApplications, expectedExclusions) {
		t.Errorf("First result exclusions = %v, want %v", result1.ExcludedSourceApplications, expectedExclusions)
	}

	if !reflect.DeepEqual(result2.ExcludedSourceApplications, expectedExclusions) {
		t.Errorf("Second result exclusions = %v, want %v", result2.ExcludedSourceApplications, expectedExclusions)
	}

	if !reflect.DeepEqual(result1.ExcludedSourceApplications, result2.ExcludedSourceApplications) {
		t.Errorf("Exclusion results differ based on input order. Result1=%v, Result2=%v",
			result1.ExcludedSourceApplications, result2.ExcludedSourceApplications)
	}
}

func Test_remoteDpToLocal_EmptySourceApps(t *testing.T) {
	dp := console.RemoteDataProduct{
		Id:                   "test-dp",
		Name:                 "Test DP",
		SourceApplicationIds: []string{},
		EventSpecs:           []console.EventSpecReference{},
	}

	result := remoteDpToLocal(dp, map[string]model.Ref{}, map[string]console.RemoteEventSpec{}, map[string]string{})

	if len(result.SourceApplications) != 0 {
		t.Errorf("Expected empty SourceApplications, got %v", result.SourceApplications)
	}
}

func Test_remoteEsToLocal_NoExclusions(t *testing.T) {
	saRefs := map[string]model.Ref{
		"app1": {Ref: "./source-apps/app1.yaml"},
		"app2": {Ref: "./source-apps/app2.yaml"},
	}

	es := console.RemoteEventSpec{
		Id:                   "test-es",
		Name:                 "Test ES",
		SourceApplicationIds: []string{"app1", "app2"},
		Event: &console.EventWrapper{Event: console.Event{
			Source: "iglu:test/schema/jsonschema/1-0-0",
			Schema: map[string]any{},
		}},
		Entities: console.Entities{Tracked: []console.Entity{}, Enriched: []console.Entity{}},
	}

	dpSourceApps := []string{"app2", "app1"}

	result := remoteEsToLocal(es, saRefs, dpSourceApps, map[string]string{})

	if len(result.ExcludedSourceApplications) != 0 {
		t.Errorf("Expected no exclusions when ES includes all DP source apps, got %v", result.ExcludedSourceApplications)
	}
}
