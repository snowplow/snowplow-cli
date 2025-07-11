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
	"sort"
	"strings"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/model"
	"github.com/snowplow/snowplow-cli/internal/util"
)

func remoteSaToLocal(remoteSa console.RemoteSourceApplication) model.SourceAppData {
	var trackedEntites []model.SchemaRef
	for _, te := range remoteSa.Entities.Tracked {
		trackedEntites = append(trackedEntites,
			model.SchemaRef{
				Source:         te.Source,
				MinCardinality: te.MinCardinality,
				MaxCardinality: te.MaxCardinality,
				Schema:         te.Schema,
			},
		)
	}

	var enrichedEntities []model.SchemaRef
	for _, ee := range remoteSa.Entities.Enriched {
		enrichedEntities = append(enrichedEntities,
			model.SchemaRef{
				Source:         ee.Source,
				MinCardinality: ee.MinCardinality,
				MaxCardinality: ee.MaxCardinality,
				Schema:         ee.Schema,
			},
		)
	}

	entities := model.EntitiesDef{Tracked: trackedEntites, Enriched: enrichedEntities}

	return model.SourceAppData{
		ResourceName: remoteSa.Id,
		Name:         remoteSa.Name,
		Description:  remoteSa.Description,
		Owner:        remoteSa.Owner,
		AppIds:       remoteSa.AppIds,
		Entities:     &entities,
	}
}

func remoteSasToLocalResources(remoteSas []console.RemoteSourceApplication) []model.CliResource[model.SourceAppData] {
	var res []model.CliResource[model.SourceAppData]
	for _, sa := range remoteSas {
		model := model.CliResource[model.SourceAppData]{
			ResourceType: "source-application",
			ApiVersion:   "v1",
			ResourceName: sa.Id,
			Data:         remoteSaToLocal(sa),
		}
		res = append(res, model)
	}
	return res
}

func remoteEsToLocal(remoteEs console.RemoteEventSpec, saIdToRef map[string]model.Ref, dataProductSourceAppIds []string, triggerIdsToImagePath map[string]string) model.EventSpecCanonical {
	var excludedSourceApps []model.Ref

	excludedIds := util.SetMinus(dataProductSourceAppIds, remoteEs.SourceApplicationIds)

	for _, saId := range excludedIds {
		ref := saIdToRef[saId]
		excludedSourceApps = append(excludedSourceApps, ref)
	}

	var event model.SchemaRef
	if remoteEs.Event != nil {
		event = model.SchemaRef{Source: remoteEs.Event.Source, Schema: remoteEs.Event.Schema}
	} else {
		event = model.SchemaRef{}
	}

	var trackedEntities []model.SchemaRef
	for _, te := range remoteEs.Entities.Tracked {
		trackedEntities = append(trackedEntities, model.SchemaRef{Source: te.Source, MinCardinality: te.MinCardinality, MaxCardinality: te.MaxCardinality, Schema: te.Schema})
	}
	var enrichedEntities []model.SchemaRef
	for _, ee := range remoteEs.Entities.Enriched {
		enrichedEntities = append(enrichedEntities, model.SchemaRef{Source: ee.Source, MinCardinality: ee.MinCardinality, MaxCardinality: ee.MaxCardinality, Schema: ee.Schema})
	}

	var triggers []model.Trigger
	for _, t := range remoteEs.Triggers {
		imageRef, imageExists := triggerIdsToImagePath[t.Id]
		var image *model.Ref
		if imageExists {
			image = &model.Ref{Ref: imageRef}
		} else {
			image = nil
		}
		triggers = append(triggers, model.Trigger{
			Id:          t.Id,
			Description: t.Description,
			AppIds:      t.AppIds,
			Url:         t.Url,
			Image:       image,
		})
	}

	entities := model.EntitiesDef{Tracked: trackedEntities, Enriched: enrichedEntities}
	return model.EventSpecCanonical{
		ResourceName:               remoteEs.Id,
		ExcludedSourceApplications: excludedSourceApps,
		Name:                       remoteEs.Name,
		Description:                remoteEs.Description,
		Event:                      &event,
		Entities:                   entities,
		Triggers:                   triggers,
	}
}

func remoteDpToLocal(remoteDp console.RemoteDataProduct, saIdToRef map[string]model.Ref, eventSpecIdToRes map[string]console.RemoteEventSpec, triggerIdsToImagePath map[string]string) model.DataProductCanonicalData {
	var sourceApps []model.Ref
	for _, saId := range remoteDp.SourceApplicationIds {
		ref := saIdToRef[saId]
		sourceApps = append(sourceApps, ref)
	}

	var eventSpecs []model.EventSpecCanonical
	for _, esId := range remoteDp.EventSpecs {
		es := eventSpecIdToRes[esId.Id]
		eventSpecs = append(eventSpecs, remoteEsToLocal(es, saIdToRef, remoteDp.SourceApplicationIds, triggerIdsToImagePath))

	}
	return model.DataProductCanonicalData{
		ResourceName:        remoteDp.Id,
		Name:                remoteDp.Name,
		SourceApplications:  sourceApps,
		Domain:              remoteDp.Domain,
		Owner:               remoteDp.Owner,
		Description:         remoteDp.Description,
		EventSpecifications: eventSpecs,
	}
}

func localSasToRefs(fileNameToLocalSa map[string]model.CliResource[model.SourceAppData], dataProductsLocation string) map[string]model.Ref {
	var saIdToRef = make(map[string]model.Ref)
	for path, sa := range fileNameToLocalSa {
		saIdToRef[sa.ResourceName] = model.Ref{Ref: fmt.Sprintf(".%s", strings.TrimPrefix(path, dataProductsLocation))}
	}
	return saIdToRef
}

func groupRemoteEsById(remoteEss []console.RemoteEventSpec) map[string]console.RemoteEventSpec {
	var esIdToRes = make(map[string]console.RemoteEventSpec)
	for _, sa := range remoteEss {
		esIdToRes[sa.Id] = sa
	}
	return esIdToRes
}

func remoteDpsToLocalResources(remoteDps []console.RemoteDataProduct, saIdToRef map[string]model.Ref, esIdToRes map[string]console.RemoteEventSpec, triggerIdsToImagePath map[string]string) []model.CliResource[model.DataProductCanonicalData] {
	var dps []model.CliResource[model.DataProductCanonicalData]
	for _, dp := range remoteDps {
		model := model.CliResource[model.DataProductCanonicalData]{
			ResourceType: "data-product",
			ApiVersion:   "v1",
			ResourceName: dp.Id,
			Data:         remoteDpToLocal(dp, saIdToRef, esIdToRes, triggerIdsToImagePath),
		}
		dps = append(dps, model)
	}
	return dps
}

type imgUrlFile struct {
	url      string
	filename string
}

func remoteEsToTriggerIdToUrlAndFilename(remoteEss []console.RemoteEventSpec) map[string]imgUrlFile {
	esNameToTriggerId := make(map[string][]string)
	for _, es := range remoteEss {
		for _, t := range es.Triggers {
			name := util.ResourceNameToFileName(es.Name)
			esNameToTriggerId[name] = append(esNameToTriggerId[name], t.Id)
		}
	}
	filenameById := make(map[string]string)
	for esName, tIds := range esNameToTriggerId {
		sort.Strings(tIds)
		for i, tId := range tIds {
			if len(tIds) > 1 {
				filenameById[tId] = fmt.Sprintf("%s_%d", esName, i+1)
			} else {
				filenameById[tId] = esName
			}
		}
	}

	triggerIdToUrl := make(map[string]imgUrlFile)
	for _, es := range remoteEss {
		for _, t := range es.Triggers {
			for variant, url := range t.VariantUrls {
				if variant == "original" {
					filename := filenameById[t.Id]
					triggerIdToUrl[t.Id] = imgUrlFile{url: url, filename: filename}
				}
			}
		}
	}
	return triggerIdToUrl
}
