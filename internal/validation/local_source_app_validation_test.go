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
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/snowplow-product/snowplow-cli/internal/model"
)

func Test_ValidateSAMinimum(t *testing.T) {
	sa := model.SourceApp{}

	result := ValidateSAMinimum(sa).Errors

	if len(result) != 2 {
		t.Fatal("!= 2 errors?")
	}

	expected := []string{
		"resourceName must be a valid uuid",
		"data.name required",
	}

	if !slices.Equal(result, expected) {
		t.Fatal("unexpected errors", result)
	}

	sa = model.SourceApp{
		ResourceName: uuid.New().String(),
		Data: model.SourceAppData{
			Name: "name",
		},
	}

	result = ValidateSAMinimum(sa).Errors

	if len(result) != 0 {
		t.Fatal("!= 0 errors?")
	}
}

func Test_ValidateSAAppIds(t *testing.T) {
	sa := model.SourceApp{
		Data: model.SourceAppData{
			AppIds: []string{"one", "two"},
		},
	}

	result := ValidateSAAppIds(sa).Errors

	if len(result) != 0 {
		t.Error("should be valid, isnt")
	}

	sa.Data.AppIds = append(sa.Data.AppIds, "")

	result = ValidateSAAppIds(sa).Errors

	expected := []string{"data.appIds[2] can't be empty"}

	if !slices.Equal(result, expected) {
		t.Fatal("shouldn't let empty in", result)
	}
}

func Test_ValidateSAEntitesSources(t *testing.T) {
	sa := model.SourceApp{
		Data: model.SourceAppData{
			Entities: &model.EntitiesDef{
				Tracked: []model.SchemaRef{
					{Source: "something"},
				},
			},
		},
	}

	result := ValidateSAEntitiesSources(sa).Errors

	if len(result) > 0 {
		t.Fatal("errors when there shouldn't be", result)
	}

	sa.Data.Entities.Tracked = append(sa.Data.Entities.Tracked, model.SchemaRef{Source: ""})

	result = ValidateSAEntitiesSources(sa).Errors

	expected := []string{"data.entities.tracked[1].source required"}

	if !slices.Equal(result, expected) {
		t.Fatal("source wasn't required?")
	}

	sa.Data.Entities = nil

	result = ValidateSAEntitiesSources(sa).Errors

	if len(result) > 0 {
		t.Fatal("errors when there shouldn't be", result)
	}
}

func Test_ValidateSAEntitesCardinalities(t *testing.T) {
	minusOne := -1
	zero := 0
	one := 1

	sa := model.SourceApp{
		Data: model.SourceAppData{
			Entities: &model.EntitiesDef{
				Tracked: []model.SchemaRef{
					{Source: "something0", MinCardinality: &minusOne},
					{Source: "something1", MinCardinality: &zero, MaxCardinality: &minusOne},
					{Source: "something2", MinCardinality: &one, MaxCardinality: &zero},
					{Source: "something3", MaxCardinality: &one},
					{Source: "something4", MinCardinality: &one, MaxCardinality: &one},
				},
			},
		},
	}

	result := ValidateSAEntitiesCardinalities(sa).Errors

	expected := []string{
		"data.entities.tracked[0].minCardinality must be > 0",
		"data.entities.tracked[1].maxCardinality must be > minCardinality: 0",
		"data.entities.tracked[2].maxCardinality must be > minCardinality: 1",
		"data.entities.tracked[3].maxCardinality without minCardinality",
	}

	if !slices.Equal(result, expected) {
		t.Fatal("unexpected errors", result)
	}
}

func Test_ValidateSAEntitesHaveNoRules(t *testing.T) {
	sa := model.SourceApp{
		Data: model.SourceAppData{
			Entities: &model.EntitiesDef{
				Tracked: []model.SchemaRef{
					{Source: "something0"},
					{Source: "something1", Schema: map[string]any{"$schema": "anything"}},
				},
				Enriched: []model.SchemaRef{
					{Source: "something1", Schema: map[string]any{"$schema": "anything"}},
				},
			},
		},
	}

	result := ValidateSAEntitiesHaveNoRules(sa).Errors

	expected := []string{
		"data.entities.tracked[1].schema property rules unsupported for source applications",
		"data.entities.enriched[0].schema property rules unsupported for source applications",
	}

	if !slices.Equal(result, expected) {
		t.Fatal("unexpected errors", result)
	}
}
