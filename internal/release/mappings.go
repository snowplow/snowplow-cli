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
	"encoding/json"
	"reflect"
	"sort"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/model"
	"github.com/snowplow/snowplow-cli/internal/util"
)

func localSaToRemote(local model.SourceApp) console.RemoteSourceApplication {
	entities := console.Entities{}
	if local.Data.Entities != nil {
		trackedEntities := []console.Entity{}
		for _, te := range local.Data.Entities.Tracked {
			trackedEntities = append(trackedEntities,
				console.Entity{
					Source:         te.Source,
					MinCardinality: te.MinCardinality,
					MaxCardinality: te.MaxCardinality,
					Schema:         te.Schema,
				},
			)
		}

		enrichedEntities := []console.Entity{}
		for _, ee := range local.Data.Entities.Enriched {
			enrichedEntities = append(enrichedEntities,
				console.Entity{
					Source:         ee.Source,
					MinCardinality: ee.MinCardinality,
					MaxCardinality: ee.MaxCardinality,
					Schema:         ee.Schema,
				},
			)
		}

		entities = console.Entities{Tracked: trackedEntities, Enriched: enrichedEntities}
	}

	return console.RemoteSourceApplication{
		Id:          local.ResourceName,
		Name:        local.Data.Name,
		Description: local.Data.Description,
		Owner:       local.Data.Owner,
		AppIds:      local.Data.AppIds,
		Entities:    entities,
	}
}

func LocalDpToRemote(local model.DataProduct) console.RemoteDataProduct {

	sourceAppIds := []string{}
	for _, sa := range local.Data.SourceApplications {
		sourceAppIds = append(sourceAppIds, sa["id"])
	}

	return console.RemoteDataProduct{
		Id:                   local.ResourceName,
		Name:                 local.Data.Name,
		SourceApplicationIds: sourceAppIds,
		Domain:               local.Data.Domain,
		Owner:                local.Data.Owner,
		Description:          local.Data.Description,
	}
}

func LocalEventSpecToRemote(es model.EventSpec, dpSourceApps []string, dpId string) console.RemoteEventSpec {
	excludedSourceAppIds := []string{}
	for _, esa := range es.ExcludedSourceApplications {
		excludedSourceAppIds = append(excludedSourceAppIds, esa["id"])
	}

	event := console.Event{
		Source: es.Event.Source,
		Schema: es.Event.Schema,
	}
	trackedEntities := []console.Entity{}
	for _, te := range es.Entities.Tracked {
		trackedEntities = append(trackedEntities, console.Entity{
			Source:         te.Source,
			MinCardinality: te.MinCardinality,
			MaxCardinality: te.MaxCardinality,
			Schema:         te.Schema,
		})
	}
	enrichedEntities := []console.Entity{}
	for _, ee := range es.Entities.Enriched {
		enrichedEntities = append(enrichedEntities, console.Entity{
			Source:         ee.Source,
			MinCardinality: ee.MinCardinality,
			MaxCardinality: ee.MaxCardinality,
			Schema:         ee.Schema,
		})
	}
	entities := console.Entities{
		Tracked:  trackedEntities,
		Enriched: enrichedEntities,
	}

	sourceApps := util.SetMinus(dpSourceApps, excludedSourceAppIds)

	triggers := []console.RemoteTrigger{}
	for _, t := range es.Triggers {
		triggers = append(triggers, console.RemoteTrigger{
			Id:          t.Id,
			Description: t.Description,
			AppIds:      t.AppIds,
			Url:         t.Url,
			// Variant URLs are not available until the image is uploaded
			// They will be populated later if needed
			VariantUrls: map[string]string{},
		})
	}

	return console.RemoteEventSpec{
		Id:                   es.ResourceName,
		SourceApplicationIds: sourceApps,
		Name:                 es.Name,
		Description:          es.Description,
		Event:                &console.EventWrapper{Event: event},
		Entities:             entities,
		DataProductId:        dpId,
		Triggers:             triggers,
	}
}

type RemoteEventSpecDiff struct {
	SourceApplicationIds []string
	Name                 string
	Description          string
	Event                string
	Entities             EntitiesDiff
	DataProductId        string
}

type EventDiff struct {
	Source string
	Schema []byte
}

type EntitiesDiff struct {
	Tracked  []string
	Enriched []string
}

func esToDiff(es console.RemoteEventSpec) (*RemoteEventSpecDiff, error) {

	tracked := []string{}
	for _, te := range es.Entities.Tracked {
		te.Comment = nil
		entityJson, err := json.Marshal(te)
		if err != nil {
			return nil, err
		}
		tracked = append(tracked, string(entityJson))
	}

	enriched := []string{}
	for _, ee := range es.Entities.Enriched {
		ee.Comment = nil
		entityJson, err := json.Marshal(ee)
		if err != nil {
			return nil, err
		}
		enriched = append(enriched, string(entityJson))
	}

	var sourceApps []string
	if es.SourceApplicationIds == nil {
		sourceApps = []string{}
	} else {
		// the order is not kept during the inversion
		sort.Strings(es.SourceApplicationIds)
		sourceApps = es.SourceApplicationIds
	}

	// the local fiels are getting serialized by mapstructure
	// while the remote ones are getting serialized by json
	// this lead to different numeric types
	// serializeing them back to json help the comparison
	eventJson, err := json.Marshal(es.Event)
	if err != nil {
		return nil, err
	}

	return &RemoteEventSpecDiff{
		SourceApplicationIds: sourceApps,
		Name:                 es.Name,
		Description:          es.Description,
		Event:                string(eventJson),
		Entities:             EntitiesDiff{Tracked: tracked, Enriched: enriched},
		DataProductId:        es.DataProductId,
	}, nil
}

type RemoteDataProductDiff struct {
	Name                 string
	SourceApplicationIds []string
	Domain               string
	Owner                string
	Description          string
}

func dpToDiff(dp console.RemoteDataProduct) RemoteDataProductDiff {
	sort.Strings(dp.SourceApplicationIds)
	return RemoteDataProductDiff{
		Name:                 dp.Name,
		SourceApplicationIds: dp.SourceApplicationIds,
		Domain:               dp.Domain,
		Owner:                dp.Owner,
		Description:          dp.Description,
	}
}

func saToDiff(sa console.RemoteSourceApplication) console.RemoteSourceApplication {
	sa.LockStatus = ""
	sa.ManagedFrom = ""
	sa.Entities = console.Entities{
		Tracked:  entitiesWithoutComments(sa.Entities.Tracked),
		Enriched: entitiesWithoutComments(sa.Entities.Enriched),
	}
	return sa
}

func entitiesWithoutComments(entities []console.Entity) []console.Entity {
	if entities == nil {
		return nil
	}
	res := []console.Entity{}
	for _, e := range entities {
		e.Comment = nil
		res = append(res, e)
	}
	return res
}

func changedStructuralFields(remote console.RemoteEventSpec, sent console.RemoteEventSpec) ([]string, error) {
	changed := []string{}

	if remote.Name != sent.Name {
		changed = append(changed, "name")
	}

	remoteEvent, err := normalizeStructure(structuralEvent(remote.Event, false))
	if err != nil {
		return nil, err
	}
	sentEvent, err := normalizeStructure(structuralEvent(sent.Event, true))
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(remoteEvent, sentEvent) {
		changed = append(changed, "event")
	}

	remoteEntities, err := normalizeStructure(structuralEntities(remote.Entities, false))
	if err != nil {
		return nil, err
	}
	sentEntities, err := normalizeStructure(structuralEntities(sent.Entities, true))
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(remoteEntities, sentEntities) {
		changed = append(changed, "entities")
	}

	return changed, nil
}

func structuralEvent(event *console.EventWrapper, isSent bool) any {
	if event == nil {
		return nil
	}
	if isSent && event.Source == "" && len(event.Schema) == 0 {
		return nil
	}
	res := map[string]any{"source": event.Source}
	if (isSent && len(event.Schema) > 0) || (!isSent && event.Schema != nil) {
		res["schema"] = event.Schema
	}
	return res
}

func structuralEntities(entities console.Entities, isSent bool) any {
	return map[string]any{
		"tracked":  structuralEntityList(entities.Tracked, isSent),
		"enriched": structuralEntityList(entities.Enriched, isSent),
	}
}

func structuralEntityList(entities []console.Entity, isSent bool) any {
	if entities == nil {
		return nil
	}
	res := []any{}
	for _, e := range entities {
		minCardinality := 0
		if e.MinCardinality != nil {
			minCardinality = *e.MinCardinality
		}
		entity := map[string]any{
			"source":         e.Source,
			"minCardinality": minCardinality,
		}
		if e.MaxCardinality != nil {
			entity["maxCardinality"] = *e.MaxCardinality
		}
		if !isSent && e.Comment != nil {
			entity["comment"] = *e.Comment
		}
		if (isSent && len(e.Schema) > 0) || (!isSent && e.Schema != nil) {
			entity["schema"] = e.Schema
		}
		res = append(res, entity)
	}
	return res
}

func normalizeStructure(structure any) (any, error) {
	structureJson, err := json.Marshal(structure)
	if err != nil {
		return nil, err
	}
	var res any
	err = json.Unmarshal(structureJson, &res)
	return res, err
}
