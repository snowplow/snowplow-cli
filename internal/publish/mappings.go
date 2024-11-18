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

	"github.com/snowplow-product/snowplow-cli/internal/console"
	"github.com/snowplow-product/snowplow-cli/internal/model"
	"github.com/snowplow-product/snowplow-cli/internal/util"
)

func localSaToRemote(local model.SourceApp) console.RemoteSourceApplication {
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

	entities := console.Entities{Tracked: trackedEntities, Enriched: enrichedEntities}

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
	for _, ee := range es.Entities.Tracked {
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
	return console.RemoteEventSpec{
		Id:                   es.ResourceName,
		SourceApplicationIds: sourceApps,
		Name:                 es.Name,
		Event:                event,
		Entities:             entities,
		DataProductId:        dpId,
	}
}
