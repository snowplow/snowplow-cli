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
	"sync"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/model"
)

type result struct {
	source     string
	status     console.CompatStatus
	props      map[string]string
	err        error
	pathLookup map[string]string
}

func ValidateDPEventSpecCompat(cc console.CompatChecker, concurrency int, dp model.DataProduct) DPValidations {
	pathErrors := map[string][]string{}
	pathWarnings := map[string][]string{}
	errors := []string{}

	resultsChan := make(chan result)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

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

		wg.Add(1)
		go func(event console.CompatCheckable, entities []console.CompatCheckable, pathLookup map[string]string) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			cc_result, err := cc(event, entities)
			if err != nil {
				resultsChan <- result{
					err: fmt.Errorf("unexpected error checking compatibility: %s", err.Error()),
				}
				return
			}
			for _, s := range cc_result.Sources {
				resultsChan <- result{
					source:     s.Source,
					status:     s.Status,
					props:      s.Properties,
					err:        nil,
					pathLookup: pathLookup,
				}
			}
		}(*event, entities, pathLookup)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for res := range resultsChan {
		if res.err != nil {
			errors = append(errors, res.err.Error())
			continue
		}
		if path, ok := res.pathLookup[res.source]; ok {
			if res.status == console.CompatIncompatible {
				pathErrors[path] = append(
					pathErrors[path],
					fmt.Sprintf("definition incompatible with source data structure (%s)", res.source),
				)
			}
			if res.status == console.CompatUndecidable {
				pathWarnings[path] = append(
					pathWarnings[path],
					fmt.Sprintf("definition has unknown compatibility with source data structure (%s)", res.source),
				)
			}
			for k, v := range res.props {
				lp := fmt.Sprintf("%s/%s", path, k)
				if v == console.CompatIncompatible {
					pathErrors[lp] = append(
						pathErrors[lp],
						fmt.Sprintf("definition incompatible with .%s in source data structure (%s)", k, res.source),
					)
				}
				if v == console.CompatUndecidable {
					pathWarnings[lp] = append(
						pathWarnings[lp],
						fmt.Sprintf("definition has unknown compatibility with .%s in source data structure (%s)", k, res.source),
					)
				}
			}
		}
	}

	return DPValidations{Errors: errors, ErrorsWithPaths: pathErrors, WarningsWithPaths: pathWarnings}
}
