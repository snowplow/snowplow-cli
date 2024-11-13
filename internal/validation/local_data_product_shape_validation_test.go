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
	"testing"

	"gopkg.in/yaml.v3"
)

func Test_ValidateDPShape_Ok(t *testing.T) {
	inputRaw := `
apiVersion: v1
resourceType: data-product
resourceName: d066a9a7-e6c4-4d72-9a93-1418746f5279
data:
  name: Data Product number 1
  sourceApplications:
    - $ref: ./source-application.yml
  domain: Data Products
  description: This Data Product describes a data product
  eventSpecifications:
    - resourceName: e066a9a7-e6c4-4d72-9a93-1418746f5278
      excludedSourceApplications:
        - $ref: ./source-application.yml
      name: event spec 1
      triggers:
        - description: number 1 trigger
      event:
        source: iglu:io.snowplow/button_click_custom/jsonschema/1-1-0
`

	var input map[string]any
	if err := yaml.Unmarshal([]byte(inputRaw), &input); err != nil {
		t.Fatal(err)
	}

	_, ok := ValidateDPShape(input)

	if !ok {
		t.Fatal("errors? there should be no errors")
	}
}

func Test_ValidateDPShape_AllWrong(t *testing.T) {
	inputRaw := `
apiVersion: v1
resourceType: data-product
resourceN: d066a9a7-e6c4-4d72-9a93-1418746f5279
data:
  ame: Data Product number 1
  eventSpecifications:
    - name: event spec 1
      event:
`

	var input map[string]any
	if err := yaml.Unmarshal([]byte(inputRaw), &input); err != nil {
		t.Fatal(err)
	}

	_, ok := ValidateDPShape(input)

	if ok {
		t.Fatal("errors? there should be errors")
	}
}

var inputValidRuleBase = `
apiVersion: v1
resourceType: data-product
resourceName: d066a9a7-e6c4-4d72-9a93-1418746f5279
data:
  name: Data Product number 1
  eventSpecifications:
    - name: event spec 1
      resourceName: d066a9a7-e6c4-4d72-9a93-1418746f5279
      event:
        source: iglu:vendor/name/format/1-0-0
        schema: %s
`

var dpRulesTests = []struct {
	in  string
	valid bool
}{
	{`{ type: string }`, false},
	{`{ type: object, additionalProperties: true }`, false},
	{`
          type: object
          additionalProperties: false
          required: [a]
          properties: { a: { type: string } }`, true},
}

func Test_ValidateDPShape_Rules(t *testing.T) {

	for _, tt := range dpRulesTests {
		inputRaw := fmt.Sprintf(inputValidRuleBase, tt.in)
		t.Run(tt.in, func(t *testing.T) {
			var input map[string]any
			if err := yaml.Unmarshal([]byte(inputRaw), &input); err != nil {
				t.Fatal(err)
			}

			_, ok := ValidateDPShape(input)

			if ok != tt.valid { 
				t.Errorf("%s got: %v want: %v", tt.in, ok, tt.valid)
			}
		})
	}

}
