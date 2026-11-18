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
	v := DPValidations{
		Errors:            []string{},
		ErrorsWithPaths:   map[string][]string{},
		WarningsWithPaths: map[string][]string{},
	}

	resultsChan := make(chan result)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	for i, spec := range dp.Data.EventSpecifications {
		event, entities, pathLookup := eventSpecCompatCheckables(i, spec)

		if event == nil {
			if len(entities) > 0 {
				path := fmt.Sprintf("/data/eventSpecifications/%d", i)
				v.WarningsWithPaths[path] = append(
					v.WarningsWithPaths[path],
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

			for _, res := range compatResults(cc, event, entities, pathLookup) {
				resultsChan <- res
			}
		}(*event, entities, pathLookup)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for res := range resultsChan {
		v.addCompatResult(res)
	}

	return v
}

func eventSpecCompatCheckables(i int, spec model.EventSpec) (*console.CompatCheckable, []console.CompatCheckable, map[string]string) {
	var event *console.CompatCheckable
	entities := []console.CompatCheckable{}
	pathLookup := map[string]string{}

	if len(spec.Event.Schema) > 0 {
		pathLookup[spec.Event.Source] = fmt.Sprintf("/data/eventSpecifications/%d/event/schema", i)
		event = &console.CompatCheckable{
			Source: spec.Event.Source,
			Schema: spec.Event.Schema,
		}
	}
	for j, ent := range spec.Entities.Tracked {
		if len(ent.Schema) > 0 {
			pathLookup[ent.Source] = fmt.Sprintf("/data/eventSpecifications/%d/entities/tracked/%d/schema", i, j)
			entities = append(entities, console.CompatCheckable{
				Source: ent.Source,
				Schema: ent.Schema,
			})
		}
	}
	for j, ent := range spec.Entities.Enriched {
		if len(ent.Schema) > 0 {
			pathLookup[ent.Source] = fmt.Sprintf("/data/eventSpecifications/%d/entities/enriched/%d/schema", i, j)
			entities = append(entities, console.CompatCheckable{
				Source: ent.Source,
				Schema: ent.Schema,
			})
		}
	}

	return event, entities, pathLookup
}

func compatResults(cc console.CompatChecker, event console.CompatCheckable, entities []console.CompatCheckable, pathLookup map[string]string) []result {
	cc_result, err := cc(event, entities)
	if err != nil {
		return []result{{
			err: fmt.Errorf("unexpected error checking compatibility: %s", err.Error()),
		}}
	}

	results := []result{}
	for _, s := range cc_result.Sources {
		results = append(results, result{
			source:     s.Source,
			status:     s.Status,
			props:      s.Properties,
			err:        nil,
			pathLookup: pathLookup,
		})
	}

	return results
}

func (v *DPValidations) addCompatResult(res result) {
	if res.err != nil {
		v.Errors = append(v.Errors, res.err.Error())
		return
	}

	path, ok := res.pathLookup[res.source]
	if !ok {
		return
	}

	if res.status == console.CompatIncompatible {
		v.ErrorsWithPaths[path] = append(
			v.ErrorsWithPaths[path],
			fmt.Sprintf("definition incompatible with source data structure (%s)", res.source),
		)
	}
	if res.status == console.CompatUndecidable {
		v.WarningsWithPaths[path] = append(
			v.WarningsWithPaths[path],
			fmt.Sprintf("definition has unknown compatibility with source data structure (%s)", res.source),
		)
	}
	for k, status := range res.props {
		lp := fmt.Sprintf("%s/%s", path, k)
		if status == console.CompatIncompatible {
			v.ErrorsWithPaths[lp] = append(
				v.ErrorsWithPaths[lp],
				fmt.Sprintf("definition incompatible with .%s in source data structure (%s)", k, res.source),
			)
		}
		if status == console.CompatUndecidable {
			v.WarningsWithPaths[lp] = append(
				v.WarningsWithPaths[lp],
				fmt.Sprintf("definition has unknown compatibility with .%s in source data structure (%s)", k, res.source),
			)
		}
	}
}
