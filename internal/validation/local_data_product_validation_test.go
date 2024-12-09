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

	"github.com/snowplow-product/snowplow-cli/internal/console"
	"github.com/snowplow-product/snowplow-cli/internal/model"
)

func newValidDPForCompatTesting() model.DataProduct {
	return model.DataProduct{
		Data: model.DataProductData{
			EventSpecifications: []model.EventSpec{
				{
					Event: model.SchemaRef{
						Source: "iglu:vendor/event/format/version",
						Schema: map[string]any{
							"key": "value",
						},
					},
					Entities: model.EntitiesDef{
						Tracked: []model.SchemaRef{{
							Source: "iglu:vendor/entity/format/version",
							Schema: map[string]any{
								"key":      "value",
								"otherKey": "something",
							}},
						},
					},
				},
			},
		},
	}
}

func Test_ValidateDPEventSpecCompatOk(t *testing.T) {
	dp := newValidDPForCompatTesting()

	cc := func(event console.CompatCheckable, entities []console.CompatCheckable) (*console.CompatResult, error) {
		return &console.CompatResult{Status: "compatible"}, nil
	}

	result := ValidateDPEventSpecCompat(cc, dp)

	if len(result.Errors) > 0 || len(result.Warnings) > 0 {
		t.Fatal("unexpected failures")
	}
}

func Test_ValidateDPEventSpecCompatMissingEvent(t *testing.T) {
	dp := newValidDPForCompatTesting()
	dp.Data.EventSpecifications[0].Event = model.SchemaRef{}

	result := ValidateDPEventSpecCompat(nil, dp).WarningsWithPaths

	expectPath := "/data/eventSpecifications/0"
	expectValue := []string{"will not run compatibility check without an event defined"}

	if resultValue, ok := result[expectPath]; !ok || slices.Equal(expectValue, resultValue) {
		t.Fatal("unexpected")
	}
}

func Test_ValidateDPEventSpecCompatFail(t *testing.T) {
	dp := newValidDPForCompatTesting()

	cc := func(event console.CompatCheckable, entities []console.CompatCheckable) (*console.CompatResult, error) {
		return &console.CompatResult{
			Status: "incompatible",
			Sources: []console.CompatSource{
				{
					Source: dp.Data.EventSpecifications[0].Event.Source,
					Status: "incompatible",
					Properties: map[string]string{
						"key": "incompatible",
					},
				},
				{
					Source: dp.Data.EventSpecifications[0].Entities.Tracked[0].Source,
					Status: "incompatible",
					Properties: map[string]string{
						"key":      "incompatible",
						"otherKey": "undecidable",
					},
				},
			},
		}, nil
	}

	result := ValidateDPEventSpecCompat(cc, dp)

	expectedErrorPaths := []string{
		"/data/eventSpecifications/0/event/schema/key",
		"/data/eventSpecifications/0/entities/tracked/0/schema/key",
	}

	for _, path := range expectedErrorPaths {
		if _, ok := result.ErrorsWithPaths[path]; !ok {
			t.Fatalf("missing error at: %s", path)
		}
	}

	expectedWarningsPath := "/data/eventSpecifications/0/entities/tracked/0/schema/otherKey"

	if _, ok := result.WarningsWithPaths[expectedWarningsPath]; !ok {
		t.Log(result.WarningsWithPaths)
		t.Fatalf("missing warning at: %s", expectedWarningsPath)
	}
}
