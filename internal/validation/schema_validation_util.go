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

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func validateWithSchema(sch *jsonschema.Schema, dp map[string]any) (DPValidations, bool) {
	errorOut := map[string][]string{}
	if err := sch.Validate(dp); err != nil {
		if e, ok := err.(*jsonschema.ValidationError); ok {
			aggr := map[string][]string{}
			for _, ie := range e.BasicOutput().Errors {
				path := ie.InstanceLocation
				if path == "" {
					path = "/"
				}
				if ie.Error != "" {
					aggr[path] = append(aggr[path], ie.Error)
				}
			}
			for path, msgs := range aggr {
				errorOut[path] = append(errorOut[path], msgs...)
			}
		}
	}

	return DPValidations{ErrorsWithPaths: errorOut}, len(errorOut) == 0
}
