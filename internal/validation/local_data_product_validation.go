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

	"github.com/snowplow-product/snowplow-cli/internal/console"
	"github.com/snowplow-product/snowplow-cli/internal/model"
)

func ValidateDPEventSpecCompat(cc console.CompatChecker, dp model.DataProduct) DPValidations {
	pathErrors := map[string][]string{}
	pathWarnings := map[string][]string{}

	errors := []string{}

	for i, spec := range dp.Data.EventSpecifications {
		var event *console.CompatCheckable
		entities := []console.CompatCheckable{}
		pathLookup := map[string]string{}
		haveEntitySchemaToCheck := false

		if len(spec.Event.Schema) > 0 {
			pathLookup[spec.Event.Source] = fmt.Sprintf("/data/eventSpecifications/%d/event/schema", i)
			event = &console.CompatCheckable{
				Source: spec.Event.Source,
				Schema: spec.Event.Schema,
			}
		}
		for j, ent := range spec.Entities.Tracked {
			if len(ent.Schema) > 0 {
				haveEntitySchemaToCheck = true
				pathLookup[ent.Source] = fmt.Sprintf("/data/eventSpecifications/%d/entities/tracked/%d/schema", i, j)
				entities = append(entities, console.CompatCheckable{
					Source: ent.Source,
					Schema: ent.Schema,
				})
			}
		}
		for j, ent := range spec.Entities.Enriched {
			if len(ent.Schema) > 0 {
				haveEntitySchemaToCheck = true
				pathLookup[ent.Source] = fmt.Sprintf("/data/eventSpecifications/%d/entities/enriched/%d/schema", i, j)
				entities = append(entities, console.CompatCheckable{
					Source: ent.Source,
					Schema: ent.Schema,
				})
			}
		}

		if event == nil {
			if haveEntitySchemaToCheck {
				path := fmt.Sprintf("/data/eventSpecifications/%d", i)
				pathWarnings[path] = append(
					pathWarnings[path],
					"will not run compatibility checks on entities without an event defined",
				)
			}
			continue
		}

		result, err := cc(*event, entities)
		if err != nil {
			errors = append(
				errors,
				fmt.Sprintf("unexpected error checking compatibility got: %s", err.Error()),
			)
			continue
		}

		for _, s := range result.Sources {
			if path, ok := pathLookup[s.Source]; ok {
				if s.Status == console.CompatIncompatible {
					pathErrors[path] = append(
						pathErrors[path],
						fmt.Sprintf("definition incompatible with source data structure (%s)", s.Source),
					)
				}
				if s.Status == console.CompatUndecidable {
					pathErrors[path] = append(
						pathErrors[path],
						fmt.Sprintf("definition has unknown compatibility with source data structure (%s)", s.Source),
					)
				}
				for k, v := range s.Properties {
					if v == console.CompatIncompatible {
						lp := fmt.Sprintf("%s/%s", path, k)
						pathErrors[lp] = append(
							pathErrors[lp],
							fmt.Sprintf("definition incompatible with .%s in source data structure (%s)", k, s.Source),
						)
					}
					if v == console.CompatUndecidable {
						lp := fmt.Sprintf("%s/%s", path, k)
						pathWarnings[lp] = append(
							pathWarnings[lp],
							fmt.Sprintf("definition has unknown compatibility with .%s in source data structure (%s)", k, s.Source),
						)
					}
				}
			}
		}
	}

	return DPValidations{Errors: errors, ErrorsWithPaths: pathErrors, WarningsWithPaths: pathWarnings}
}
