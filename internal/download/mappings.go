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
	"strings"

	"github.com/snowplow-product/snowplow-cli/internal/console"
	"github.com/snowplow-product/snowplow-cli/internal/model"
)

func remoteDpToLocal(remoteDp console.RemoteDataProduct, saIdToRef map[string]model.Ref, eventSpecIdToRes map[string]console.RemoteEventSpec) model.DataProductCanonicalData {
	var sourceApps []model.Ref
	for _, saId := range remoteDp.SourceApplicationIds {
		ref := saIdToRef[saId]
		sourceApps = append(sourceApps, ref)
	}

	var eventSpecs []model.EventSpecCanonical
	for _, esId := range remoteDp.EventSpecifications {
		es := eventSpecIdToRes[esId.Id]
		eventSpecs = append(eventSpecs, remoteEsToLocal(es, saIdToRef))

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

func remoteEsToLocal(remoteEs console.RemoteEventSpec, saIdToRef map[string]model.Ref) model.EventSpecCanonical {
	var sourceApps []model.Ref
	for _, saId := range remoteEs.SourceApplicationIds {
		ref := saIdToRef[saId]
		sourceApps = append(sourceApps, ref)
	}

	event := model.SchemaRef{Source: remoteEs.Event.Source, Schema: remoteEs.Event.Schema}
	var trackedEntities []model.SchemaRef
	for _, te := range remoteEs.Entities.Tracked {
		trackedEntities = append(trackedEntities, model.SchemaRef{Source: te.Source, MinCardinality: te.MinCardinality, MaxCardinality: te.MaxCardinality, Schema: te.Schema})
	}
	var enrichedEntities []model.SchemaRef
	for _, ee := range remoteEs.Entities.Enriched {
		enrichedEntities = append(enrichedEntities, model.SchemaRef{Source: ee.Source, MinCardinality: ee.MinCardinality, MaxCardinality: ee.MaxCardinality, Schema: ee.Schema})
	}

	entities := model.EntitiesDef{Tracked: trackedEntities, Enriched: enrichedEntities}
	return model.EventSpecCanonical{
		ResourceName:       remoteEs.Id,
		SourceApplications: sourceApps,
		Name:               remoteEs.Name,
		Event:              event,
		Entities:           entities,
	}
}

func RemoteSasToLocalResources(remoteSas []console.RemoteSourceApplication) []model.CliResource[model.SourceAppData] {
	var res []model.CliResource[model.SourceAppData]
	for _, sa := range remoteSas {
		model := model.CliResource[model.SourceAppData]{
			ResourceType: "source-application",
			ApiVersion:   "v1",
			ResourceName: sa.Id,
			Data:         sa.ToCanonical(),
		}
		res = append(res, model)
	}
	return res
}

func LocalSasToRefs(fileNameToLocalSa map[string]model.CliResource[model.SourceAppData], dataProductsLocation string) map[string]model.Ref {
	var saIdToRef = make(map[string]model.Ref)
	for path, sa := range fileNameToLocalSa {
		saIdToRef[sa.ResourceName] = model.Ref{Ref: fmt.Sprintf(".%s", strings.TrimPrefix(path, dataProductsLocation))}
	}
	return saIdToRef
}

func GroupRemoteEsById(remoteEss []console.RemoteEventSpec) map[string]console.RemoteEventSpec {
	var esIdToRes = make(map[string]console.RemoteEventSpec)
	for _, sa := range remoteEss {
		esIdToRes[sa.Id] = sa
	}
	return esIdToRes
}

func RemoteDpsToLocalResources(remoteDps []console.RemoteDataProduct, saIdToRef map[string]model.Ref, esIdToRes map[string]console.RemoteEventSpec) []model.CliResource[model.DataProductCanonicalData] {
	var dps []model.CliResource[model.DataProductCanonicalData]
	for _, dp := range remoteDps {
		model := model.CliResource[model.DataProductCanonicalData]{
			ResourceType: "data-product",
			ApiVersion:   "v1",
			ResourceName: dp.Id,
			Data:         dp.ToCanonical(saIdToRef, esIdToRes),
		}
		dps = append(dps, model)
	}
	return dps
}
