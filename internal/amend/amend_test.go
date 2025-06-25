/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package amend

import (
	"fmt"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/model"
)

var EventSpecs = []model.EventSpecCanonical{{
	ResourceName:               "15c70141-c223-4172-ab39-86a0194e44e8",
	ExcludedSourceApplications: nil,
	Name:                       "test1",
}, {
	ResourceName:               "c312c1cc-af41-450f-a086-1c2e8c04a39b",
	ExcludedSourceApplications: nil,
	Name:                       "test2",
},
}

func Test_AddYamlEmpty(t *testing.T) {
	original := `# yaml-language-server: $schema=https://raw.githubusercontent.com/snowplow/snowplow-cli/refs/heads/main/internal/validation/schema/data-product.json

apiVersion: v1 # comment before
resourceType: data-product
resourceName: 3b7c9fce-9f33-4c91-800b-8c8804f388c9
data:
    name: xxx3
    sourceApplications: []
    eventSpecifications: [] # lost comment ;(
#comment after
`
	expected := fmt.Sprintf(`# yaml-language-server: $schema=https://raw.githubusercontent.com/snowplow/snowplow-cli/refs/heads/main/internal/validation/schema/data-product.json

apiVersion: v1 # comment before
resourceType: data-product
resourceName: 3b7c9fce-9f33-4c91-800b-8c8804f388c9
data:
    name: xxx3
    sourceApplications: []
    eventSpecifications:
        - resourceName: %s
          name: %s
          entities:
              tracked: []
              enriched: []
        - resourceName: %s
          name: %s
          entities:
              tracked: []
              enriched: []
#comment after
`, EventSpecs[0].ResourceName, EventSpecs[0].Name, EventSpecs[1].ResourceName, EventSpecs[1].Name)

	res, err := AddEventSpecs(EventSpecs, []byte(original), "test.yaml")
	if err != nil {
		t.Fatalf("Can't add event specs to data product %s", err)
	}
	if string(res) != expected {
		t.Fatalf("Result of adding the event specs is not expected, expected:\n%s\nactual:\n%s", expected, string(res))
	}
}

func Test_AddYamlEmptyMissaligned(t *testing.T) {
	original := `# yaml-language-server: $schema=https://raw.githubusercontent.com/snowplow/snowplow-cli/refs/heads/main/internal/validation/schema/data-product.json

apiVersion: v1 # comment before
resourceType: data-product
resourceName: 3b7c9fce-9f33-4c91-800b-8c8804f388c9
data:
    name: xxx3
    sourceApplications: []
    eventSpecifications: 
    [] # lost comment ;(
#comment after
`
	expected := fmt.Sprintf(`# yaml-language-server: $schema=https://raw.githubusercontent.com/snowplow/snowplow-cli/refs/heads/main/internal/validation/schema/data-product.json

apiVersion: v1 # comment before
resourceType: data-product
resourceName: 3b7c9fce-9f33-4c91-800b-8c8804f388c9
data:
    name: xxx3
    sourceApplications: []
    eventSpecifications:
        - resourceName: %s
          name: %s
          entities:
              tracked: []
              enriched: []
        - resourceName: %s
          name: %s
          entities:
              tracked: []
              enriched: []
#comment after
`, EventSpecs[0].ResourceName, EventSpecs[0].Name, EventSpecs[1].ResourceName, EventSpecs[1].Name)

	res, err := AddEventSpecs(EventSpecs, []byte(original), "test.yaml")
	if err != nil {
		t.Fatalf("Can't add event specs to data product %s", err)
	}
	if string(res) != expected {
		t.Fatalf("Result of adding the event specs is not expected, expected:\n%s\nactual:\n%s", expected, string(res))
	}
}

func Test_AddYaml_FlowExisting(t *testing.T) {
	original := `# yaml-language-server: $schema=https://raw.githubusercontent.com/snowplow/snowplow-cli/refs/heads/main/internal/validation/schema/data-product.json

apiVersion: v1 # comment before
resourceType: data-product
resourceName: 3b7c9fce-9f33-4c91-800b-8c8804f388c9
data:
    name: xxx3
    sourceApplications: []
    eventSpecifications: [{resourceName: 1953c1ea-efcc-443d-90eb-a672478f2eae, name: existing, entities: {tracked: [], enriched: []}}] # lost comment ;(
#comment after
`
	expected := fmt.Sprintf(`# yaml-language-server: $schema=https://raw.githubusercontent.com/snowplow/snowplow-cli/refs/heads/main/internal/validation/schema/data-product.json

apiVersion: v1 # comment before
resourceType: data-product
resourceName: 3b7c9fce-9f33-4c91-800b-8c8804f388c9
data:
    name: xxx3
    sourceApplications: []
    eventSpecifications: [{resourceName: 1953c1ea-efcc-443d-90eb-a672478f2eae, name: existing, entities: {tracked: [], enriched: []}}, {resourceName: %s, name: %s, entities: {tracked: [], enriched: []}}, {resourceName: %s, name: %s, entities: {tracked: [], enriched: []}}]
#comment after
`, EventSpecs[0].ResourceName, EventSpecs[0].Name, EventSpecs[1].ResourceName, EventSpecs[1].Name)

	res, err := AddEventSpecs(EventSpecs, []byte(original), "test.yaml")
	if err != nil {
		t.Fatalf("Can't add event specs to data product %s", err)
	}
	if string(res) != expected {
		t.Fatalf("Result of adding the event specs is not expected, expected:\n%s\nactual:\n%s", expected, string(res))
	}
}

func Test_AddYaml_BlockExisting(t *testing.T) {
	original := `# yaml-language-server: $schema=https://raw.githubusercontent.com/snowplow/snowplow-cli/refs/heads/main/internal/validation/schema/data-product.json

apiVersion: v1 # comment before
resourceType: data-product
resourceName: 3b7c9fce-9f33-4c91-800b-8c8804f388c9
data:
    name: xxx3
    sourceApplications: []
    eventSpecifications:
        - resourceName: 34d1054b-5265-46ac-a719-b0aa93495eda
          name: existing
          entities:
              tracked: []
              enriched: []
#comment after
`
	expected := fmt.Sprintf(`# yaml-language-server: $schema=https://raw.githubusercontent.com/snowplow/snowplow-cli/refs/heads/main/internal/validation/schema/data-product.json

apiVersion: v1 # comment before
resourceType: data-product
resourceName: 3b7c9fce-9f33-4c91-800b-8c8804f388c9
data:
    name: xxx3
    sourceApplications: []
    eventSpecifications:
        - resourceName: 34d1054b-5265-46ac-a719-b0aa93495eda
          name: existing
          entities:
              tracked: []
              enriched: []
        - resourceName: %s
          name: %s
          entities:
              tracked: []
              enriched: []
        - resourceName: %s
          name: %s
          entities:
              tracked: []
              enriched: []
#comment after
`, EventSpecs[0].ResourceName, EventSpecs[0].Name, EventSpecs[1].ResourceName, EventSpecs[1].Name)

	res, err := AddEventSpecs(EventSpecs, []byte(original), "test.yaml")
	if err != nil {
		t.Fatalf("Can't add event specs to data product %s", err)
	}
	if string(res) != expected {
		t.Fatalf("Result of adding the event specs is not expected, expected:\n%s\nactual:\n%s", expected, string(res))
	}
}

func Test_AddJson_Empty(t *testing.T) {
	original := `{
  "apiVersion": "v1",
  "resourceType": "data-product",
  "resourceName": "7b763497-cc48-4421-b0a6-5574add5b758",
  "data": {
    "name": "xxx3",
    "sourceApplications": [],
    "eventSpecifications": []
  }
}
`
	expected := fmt.Sprintf(`{
  "apiVersion": "v1",
  "resourceType": "data-product",
  "resourceName": "7b763497-cc48-4421-b0a6-5574add5b758",
  "data": {
    "name": "xxx3",
    "sourceApplications": [],
    "eventSpecifications": [{"resourceName":"%s","name":"%s","event":{},"entities":{}},{"resourceName":"%s","name":"%s","event":{},"entities":{}}]
  }
}
`, EventSpecs[0].ResourceName, EventSpecs[0].Name, EventSpecs[1].ResourceName, EventSpecs[1].Name)

	res, err := AddEventSpecs(EventSpecs, []byte(original), "test.json")
	if err != nil {
		t.Fatalf("Can't add event specs to data product %s", err)
	}
	if string(res) != expected {
		t.Fatalf("Result of adding the event specs is not expected, expected:\n%s\nactual:\n%s", expected, string(res))
	}
}

func Test_AddJson_Existing(t *testing.T) {
	original := `{
  "apiVersion": "v1",
  "resourceType": "data-product",
  "resourceName": "7b763497-cc48-4421-b0a6-5574add5b758",
  "data": {
    "name": "xxx3",
    "sourceApplications": [],
    "eventSpecifications": [{"resourceName":"8f97ac29-5fee-432c-ba12-eb2469c3b55b","name":"existing","event":{},"entities":{}}]
  }
}
`
	expected := fmt.Sprintf(`{
  "apiVersion": "v1",
  "resourceType": "data-product",
  "resourceName": "7b763497-cc48-4421-b0a6-5574add5b758",
  "data": {
    "name": "xxx3",
    "sourceApplications": [],
    "eventSpecifications": [{"resourceName":"8f97ac29-5fee-432c-ba12-eb2469c3b55b","name":"existing","event":{},"entities":{}},{"resourceName":"%s","name":"%s","event":{},"entities":{}},{"resourceName":"%s","name":"%s","event":{},"entities":{}}]
  }
}
`, EventSpecs[0].ResourceName, EventSpecs[0].Name, EventSpecs[1].ResourceName, EventSpecs[1].Name)

	res, err := AddEventSpecs(EventSpecs, []byte(original), "test.json")
	if err != nil {
		t.Fatalf("Can't add event specs to data product %s", err)
	}
	if string(res) != expected {
		t.Fatalf("Result of adding the event specs is not expected, expected:\n%s\nactual:\n%s", expected, string(res))
	}
}
