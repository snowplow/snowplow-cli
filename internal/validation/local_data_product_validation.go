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
	errors := []string{}
	warnings := []string{}

	for i, spec := range dp.Data.EventSpecifications {
		var event *console.CompatCheckable
		entities := []console.CompatCheckable{}
		pathLookup := map[string]string{}

		if len(spec.Event.Schema) > 0 {
			pathLookup[spec.Event.Source] = fmt.Sprintf("data.eventSpecifications[%d].event.schema", i)
			event = &console.CompatCheckable{
				Source: spec.Event.Source,
				Schema: spec.Event.Schema,
			}
		}
		for j, ent := range spec.Entities.Tracked {
			if len(ent.Schema) > 0 {
				pathLookup[ent.Source] = fmt.Sprintf("data.eventSpecifications[%d].entities.tracked[%d].schema", i, j)
				entities = append(entities, console.CompatCheckable{
					Source: ent.Source,
					Schema: ent.Schema,
				})
			}
		}
		for j, ent := range spec.Entities.Enriched {
			if len(ent.Schema) > 0 {
				pathLookup[ent.Source] = fmt.Sprintf("data.eventSpecifications[%d].entities.enriched[%d].schema", i, j)
				entities = append(entities, console.CompatCheckable{
					Source: ent.Source,
					Schema: ent.Schema,
				})
			}
		}

		if event == nil {
			warnings = append(
				warnings,
				fmt.Sprintf("data.eventSpecifications[%d] will not run compatibility check without an event defined", i),
			)
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
				for k, v := range s.Properties {
					if v == console.CompatIncompatible {
						errors = append(
							errors,
							fmt.Sprintf("%s.%s definition incompatible with .%s in source data structure (%s)", path, k, k, s.Source),
						)
					}
					if v == console.CompatUndecidable {
						warnings = append(
							warnings,
							fmt.Sprintf("%s.%s definition has unknown compatibility with .%s in source data structure (%s)", path, k, k, s.Source),
						)
					}
				}
			}
		}
	}

	return DPValidations{errors, warnings, []string{}, []string{}}
}
