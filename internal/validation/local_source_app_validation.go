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
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/model"
)

//go:embed schema/source-application.json
var saSchema string

func ValidateSAShape(sa map[string]any) (validations DPValidations, ok bool) {
	sch := jsonschema.MustCompileString("data://source-application.json", saSchema)
	return validateWithSchema(sch, sa)
}

func cardinalityCheck(key string, i int, s model.SchemaRef) DPValidations {
	errors := map[string][]string{}
	if s.MinCardinality != nil {
		if *s.MinCardinality < 0 {
			path := fmt.Sprintf("/data/entities/%s/%d/minCardinality", key, i)
			errors[path] = append(errors[path], "must be > 0")
		}
		if s.MaxCardinality != nil {
			if *s.MaxCardinality < *s.MinCardinality {
				path := fmt.Sprintf("/data/entities/%s/%d/maxCardinality", key, i)
				errors[path] = append(errors[path], fmt.Sprintf("must be > minCardinality: %d", *s.MinCardinality))
			}
		}
	} else {
		if s.MaxCardinality != nil {
			path := fmt.Sprintf("/data/entities/%s/%d/maxCardinality", key, i)
			errors[path] = append(errors[path], "without minCardinality")
		}
	}
	return DPValidations{ErrorsWithPaths: errors}
}

func ValidateSAEntitiesCardinalities(sa model.SourceApp) DPValidations {
	result := DPValidations{}

	if sa.Data.Entities == nil {
		return result
	}

	for i, e := range sa.Data.Entities.Tracked {
		result.concat(cardinalityCheck("tracked", i, e))
	}

	for i, e := range sa.Data.Entities.Enriched {
		result.concat(cardinalityCheck("enriched", i, e))
	}

	return result
}

func validIgluUri(uri string) bool {
	uriRegex := regexp.MustCompile("^iglu:[a-zA-Z0-9-_.]+/[a-zA-Z0-9-_]+/[a-zA-Z0-9-_]+/[0-9]+-[0-9]+-[0-9]+$")
	return uriRegex.MatchString(uri)
}

func checkSchemaDeployed(sdc console.SchemaDeployChecker, i int, key string, uri string) (path string, errors []string) {
	result := []string{}
	errorPath := fmt.Sprintf("/data/entities/%s/%d/source", key, i)

	if !validIgluUri(uri) {
		return errorPath, append(
			result,
			"invalid iglu uri should follow the format iglu:vendor/name/format/version, eg: iglu:io.snowplow/login/jsonschema/1-0-0",
		)
	}

	found, alternatives, err := sdc.IsDSDeployed(uri)
	if err != nil {
		result = append(
			result,
			fmt.Sprintf("error while resolving %s", err.Error()),
		)
	}
	if !found {
		if len(alternatives) > 0 {
			var available string
			if len(alternatives) > 5 {
				available = fmt.Sprintf("%s, ...%d more", strings.Join(alternatives[0:5], ", "), len(alternatives)-5)
			} else {
				available = strings.Join(alternatives, ", ")
			}
			result = append(
				result,
				fmt.Sprintf("could not find deployment of %s, available versions (%s)", uri, available),
			)
		} else {
			result = append(
				result,
				fmt.Sprintf("could not find deployment of %s", uri),
			)
		}
	}

	return errorPath, result
}

func ValidateSAEntitiesSchemaDeployed(sdc console.SchemaDeployChecker, sa model.SourceApp) DPValidations {
	errors := map[string][]string{}

	if sa.Data.Entities == nil {
		return DPValidations{}
	}

	for i, e := range sa.Data.Entities.Tracked {
		path, err := checkSchemaDeployed(sdc, i, "tracked", e.Source)
		if len(err) > 0 {
			errors[path] = err
		}
	}

	for i, e := range sa.Data.Entities.Enriched {
		path, err := checkSchemaDeployed(sdc, i, "enriched", e.Source)
		if len(err) > 0 {
			errors[path] = err
		}
	}

	return DPValidations{ErrorsWithPaths: errors}
}
