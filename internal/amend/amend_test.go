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
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
    "eventSpecifications": [{"resourceName":"%s","name":"%s","entities":{"tracked":[],"enriched":[]}},{"resourceName":"%s","name":"%s","entities":{"tracked":[],"enriched":[]}}]
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
    "eventSpecifications": [{"resourceName":"8f97ac29-5fee-432c-ba12-eb2469c3b55b","name":"existing","entities":{"tracked":[],"enriched":[]}}]
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
    "eventSpecifications": [{"resourceName":"8f97ac29-5fee-432c-ba12-eb2469c3b55b","name":"existing","entities":{"tracked":[],"enriched":[]}},{"resourceName":"%s","name":"%s","entities":{"tracked":[],"enriched":[]}},{"resourceName":"%s","name":"%s","entities":{"tracked":[],"enriched":[]}}]
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

func Test_SetEventSpecVersionsYaml(t *testing.T) {
	original := `# yaml-language-server: $schema=https://raw.githubusercontent.com/snowplow/snowplow-cli/refs/heads/main/internal/validation/schema/data-product.json

apiVersion: v1 # comment before
resourceType: data-product
resourceName: 3b7c9fce-9f33-4c91-800b-8c8804f388c9
data:
  name: xxx3
  sourceApplications: []
  eventSpecifications:
    - resourceName: 15c70141-c223-4172-ab39-86a0194e44e8
      name: test1
      version: 3 # keep me
      entities: {tracked: [], enriched: []}
    - resourceName: c312c1cc-af41-450f-a086-1c2e8c04a39b
      name: test2
      version: 0
    - {resourceName: 4ac70141-c223-4172-ab39-86a0194e44e8, name: test3, version: 9}
#comment after
`
	expected := `# yaml-language-server: $schema=https://raw.githubusercontent.com/snowplow/snowplow-cli/refs/heads/main/internal/validation/schema/data-product.json

apiVersion: v1 # comment before
resourceType: data-product
resourceName: 3b7c9fce-9f33-4c91-800b-8c8804f388c9
data:
  name: xxx3
  sourceApplications: []
  eventSpecifications:
    - resourceName: 15c70141-c223-4172-ab39-86a0194e44e8
      name: test1
      version: 4 # keep me
      entities: {tracked: [], enriched: []}
    - resourceName: c312c1cc-af41-450f-a086-1c2e8c04a39b
      name: test2
      version: 0
    - {resourceName: 4ac70141-c223-4172-ab39-86a0194e44e8, name: test3, version: 10}
#comment after
`

	res, err := SetEventSpecVersions(map[int]int{0: 4, 2: 10}, []byte(original), "test.yaml")
	if err != nil {
		t.Fatalf("Can't set event spec versions %s", err)
	}
	if string(res) != expected {
		t.Fatalf("Result of setting the versions is not expected, expected:\n%s\nactual:\n%s", expected, string(res))
	}
}

func Test_SetEventSpecVersionsJson(t *testing.T) {
	original := `{
  "apiVersion": "v1",
  "resourceType": "data-product",
  "resourceName": "3b7c9fce-9f33-4c91-800b-8c8804f388c9",
  "data": {
    "name": "xxx3",
    "eventSpecifications": [
      {
        "resourceName": "15c70141-c223-4172-ab39-86a0194e44e8",
        "name": "test1",
        "version": 3
      },
      {
        "resourceName": "c312c1cc-af41-450f-a086-1c2e8c04a39b",
        "name": "test2",
        "version": 0
      }
    ]
  }
}
`
	expected := `{
  "apiVersion": "v1",
  "resourceType": "data-product",
  "resourceName": "3b7c9fce-9f33-4c91-800b-8c8804f388c9",
  "data": {
    "name": "xxx3",
    "eventSpecifications": [
      {
        "resourceName": "15c70141-c223-4172-ab39-86a0194e44e8",
        "name": "test1",
        "version": 3
      },
      {
        "resourceName": "c312c1cc-af41-450f-a086-1c2e8c04a39b",
        "name": "test2",
        "version": 12
      }
    ]
  }
}
`

	res, err := SetEventSpecVersions(map[int]int{1: 12}, []byte(original), "test.json")
	if err != nil {
		t.Fatalf("Can't set event spec versions %s", err)
	}
	if string(res) != expected {
		t.Fatalf("Result of setting the versions is not expected, expected:\n%s\nactual:\n%s", expected, string(res))
	}
}

func Test_SetEventSpecVersionsYamlMissingVersion(t *testing.T) {
	original := `apiVersion: v1
resourceType: data-product
data:
  eventSpecifications:
    - resourceName: 15c70141-c223-4172-ab39-86a0194e44e8
      name: test1
`

	_, err := SetEventSpecVersions(map[int]int{0: 1}, []byte(original), "test.yaml")
	if err == nil {
		t.Fatal("expected an error for an event spec without a version")
	}
}

func Test_SetEventSpecVersionsYamlMultiByteBeforeVersion(t *testing.T) {
	original := `apiVersion: v1
resourceType: data-product
data:
  name: Café ñandú
  eventSpecifications:
    - {resourceName: 15c70141-c223-4172-ab39-86a0194e44e8, name: Café añadido 🛒, version: 3}
    - resourceName: c312c1cc-af41-450f-a086-1c2e8c04a39b
      name: Pedido ñ
      version: 7 # versión
`
	expected := `apiVersion: v1
resourceType: data-product
data:
  name: Café ñandú
  eventSpecifications:
    - {resourceName: 15c70141-c223-4172-ab39-86a0194e44e8, name: Café añadido 🛒, version: 14}
    - resourceName: c312c1cc-af41-450f-a086-1c2e8c04a39b
      name: Pedido ñ
      version: 8 # versión
`

	res, err := SetEventSpecVersions(map[int]int{0: 14, 1: 8}, []byte(original), "test.yaml")
	if err != nil {
		t.Fatalf("Can't set event spec versions %s", err)
	}
	if string(res) != expected {
		t.Fatalf("Result of setting the versions is not expected, expected:\n%s\nactual:\n%s", expected, string(res))
	}
}

func Test_SetEventSpecVersionsJsonMissingVersion(t *testing.T) {
	original := `{
  "apiVersion": "v1",
  "resourceType": "data-product",
  "data": {
    "eventSpecifications": [
      {"resourceName": "15c70141-c223-4172-ab39-86a0194e44e8", "name": "test1"}
    ]
  }
}
`

	for _, idx := range []int{0, 1} {
		res, err := SetEventSpecVersions(map[int]int{idx: 1}, []byte(original), "test.json")
		if err == nil {
			t.Fatalf("expected an error for event spec %d without a version", idx)
		}
		if res != nil {
			t.Errorf("expected no output on error, got: %s", res)
		}
	}
}

func addedResourceNames(t *testing.T, before string, after string) []string {
	t.Helper()
	re := regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	existing := map[string]bool{}
	for _, id := range re.FindAllString(before, -1) {
		existing[id] = true
	}
	var added []string
	for _, id := range re.FindAllString(after, -1) {
		if !existing[id] {
			added = append(added, id)
		}
	}
	return added
}

func Test_AddEventSpecsToFile_Versions(t *testing.T) {
	yamlWithout := `apiVersion: v1
resourceType: data-product
resourceName: 3b7c9fce-9f33-4c91-800b-8c8804f388c9
data:
    name: xxx3
    sourceApplications: []
    eventSpecifications:
        - resourceName: 15c70141-c223-4172-ab39-86a0194e44e8
          name: test1
          entities:
              tracked: []
              enriched: []
`
	yamlWith := strings.Replace(yamlWithout, "          name: test1\n", "          name: test1\n          version: 2\n", 1)
	jsonWithout := `{
  "apiVersion": "v1",
  "resourceType": "data-product",
  "resourceName": "3b7c9fce-9f33-4c91-800b-8c8804f388c9",
  "data": {
    "name": "xxx3",
    "eventSpecifications": [
      {"resourceName": "15c70141-c223-4172-ab39-86a0194e44e8", "name": "test1", "entities": {"tracked": [], "enriched": []}}
    ]
  }
}`
	jsonWith := strings.Replace(jsonWithout, `"name": "test1",`, `"name": "test1", "version": 2,`, 1)

	for _, tt := range []struct {
		name        string
		fileName    string
		content     string
		wantVersion bool
	}{
		{name: "yaml without versions", fileName: "dp.yaml", content: yamlWithout, wantVersion: false},
		{name: "yaml with versions", fileName: "dp.yaml", content: yamlWith, wantVersion: true},
		{name: "json without versions", fileName: "dp.json", content: jsonWithout, wantVersion: false},
		{name: "json with versions", fileName: "dp.json", content: jsonWith, wantVersion: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dpFile := filepath.Join(t.TempDir(), tt.fileName)
			if err := os.WriteFile(dpFile, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			if err := AddEventSpecsToFile([]string{"new spec"}, dpFile); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			written, err := os.ReadFile(dpFile)
			if err != nil {
				t.Fatal(err)
			}

			added := addedResourceNames(t, tt.content, string(written))
			if len(added) != 1 {
				t.Fatalf("expected one added event spec, got %v", added)
			}
			spec := model.EventSpecCanonical{ResourceName: added[0], Name: "new spec", Entities: model.EntitiesDef{}}
			if tt.wantVersion {
				version := 0
				spec.Version = &version
			}
			expected, err := AddEventSpecs([]model.EventSpecCanonical{spec}, []byte(tt.content), tt.fileName)
			if err != nil {
				t.Fatal(err)
			}
			if string(written) != string(expected) {
				t.Errorf("unexpected output\nwant:\n%s\ngot:\n%s", expected, written)
			}

			hasZero := strings.Contains(string(written), "version: 0") || strings.Contains(string(written), `"version":0`)
			if hasZero != tt.wantVersion {
				t.Errorf("version 0 written = %v, want %v, got:\n%s", hasZero, tt.wantVersion, written)
			}
		})
	}
}

func Test_AddEventSpecVersionsYaml(t *testing.T) {
	original := `# yaml-language-server: $schema=https://raw.githubusercontent.com/snowplow/snowplow-cli/refs/heads/main/internal/validation/schema/data-product.json

apiVersion: v1 # comment before
resourceType: data-product
resourceName: 3b7c9fce-9f33-4c91-800b-8c8804f388c9
data:
  name: Café ñandú
  sourceApplications: []
  eventSpecifications:
    - resourceName: 15c70141-c223-4172-ab39-86a0194e44e8
      name: "Café añadido 🛒" # keep me
      # about the entities
      entities: {tracked: [], enriched: []}
    - resourceName: c312c1cc-af41-450f-a086-1c2e8c04a39b
      name: test2
      version: 7
    - {resourceName: 4ac70141-c223-4172-ab39-86a0194e44e8, name: 'ñ, 🛒', entities: {tracked: []}}
    - {resourceName: 5bc70141-c223-4172-ab39-86a0194e44e8, name: last  }
    - name: first
      resourceName: 6cc70141-c223-4172-ab39-86a0194e44e8
    - resourceName: 7dc70141-c223-4172-ab39-86a0194e44e8
      name: only key left
#comment after
`
	expected := `# yaml-language-server: $schema=https://raw.githubusercontent.com/snowplow/snowplow-cli/refs/heads/main/internal/validation/schema/data-product.json

apiVersion: v1 # comment before
resourceType: data-product
resourceName: 3b7c9fce-9f33-4c91-800b-8c8804f388c9
data:
  name: Café ñandú
  sourceApplications: []
  eventSpecifications:
    - resourceName: 15c70141-c223-4172-ab39-86a0194e44e8
      name: "Café añadido 🛒" # keep me
      version: 4
      # about the entities
      entities: {tracked: [], enriched: []}
    - resourceName: c312c1cc-af41-450f-a086-1c2e8c04a39b
      name: test2
      version: 7
    - {resourceName: 4ac70141-c223-4172-ab39-86a0194e44e8, name: 'ñ, 🛒', version: 0, entities: {tracked: []}}
    - {resourceName: 5bc70141-c223-4172-ab39-86a0194e44e8, name: last, version: 2  }
    - name: first
      version: 1
      resourceName: 6cc70141-c223-4172-ab39-86a0194e44e8
    - resourceName: 7dc70141-c223-4172-ab39-86a0194e44e8
      name: only key left
      version: 5
#comment after
`

	res, err := AddEventSpecVersions(map[int]int{0: 4, 2: 0, 3: 2, 4: 1, 5: 5}, []byte(original), "test.yaml")
	if err != nil {
		t.Fatalf("Can't add event spec versions %s", err)
	}
	if string(res) != expected {
		t.Fatalf("Result of adding the versions is not expected, expected:\n%s\nactual:\n%s", expected, string(res))
	}
}

func Test_AddEventSpecVersionsYamlLastLineWithoutNewline(t *testing.T) {
	original := "apiVersion: v1\r\ndata:\r\n  eventSpecifications:\r\n    - resourceName: 15c70141-c223-4172-ab39-86a0194e44e8\r\n      name: test1\r\n    - resourceName: c312c1cc-af41-450f-a086-1c2e8c04a39b\r\n      name: test2"
	expected := "apiVersion: v1\r\ndata:\r\n  eventSpecifications:\r\n    - resourceName: 15c70141-c223-4172-ab39-86a0194e44e8\r\n      name: test1\r\n      version: 3\r\n    - resourceName: c312c1cc-af41-450f-a086-1c2e8c04a39b\r\n      name: test2\r\n      version: 0"

	res, err := AddEventSpecVersions(map[int]int{0: 3, 1: 0}, []byte(original), "test.yml")
	if err != nil {
		t.Fatalf("Can't add event spec versions %s", err)
	}
	if string(res) != expected {
		t.Fatalf("Result of adding the versions is not expected, expected:\n%q\nactual:\n%q", expected, string(res))
	}
}

func Test_AddEventSpecVersionsYamlHashInNameBeforeClosingBrace(t *testing.T) {
	original := "data:\n  eventSpecifications:\n    - {resourceName: a, name: \"Checkout #1\"\n      }\n"
	expected := "data:\n  eventSpecifications:\n    - {resourceName: a, name: \"Checkout #1\", version: 3\n      }\n"

	res, err := AddEventSpecVersions(map[int]int{0: 3}, []byte(original), "test.yaml")
	if err != nil {
		t.Fatalf("Can't add event spec versions %s", err)
	}
	if string(res) != expected {
		t.Fatalf("Result of adding the versions is not expected, expected:\n%s\nactual:\n%s", expected, string(res))
	}
}

func Test_AddEventSpecVersionsYamlRefusesUnsafeEdits(t *testing.T) {
	for _, tt := range []struct {
		name     string
		original string
	}{
		{name: "already declares a version", original: "data:\n  eventSpecifications:\n    - name: test1\n      version: 2\n"},
		{name: "index out of range", original: "data:\n  eventSpecifications: []\n"},
		{name: "no name", original: "data:\n  eventSpecifications:\n    - resourceName: 15c70141-c223-4172-ab39-86a0194e44e8\n"},
		{name: "literal name", original: "data:\n  eventSpecifications:\n    - name: |\n        test1\n      resourceName: a\n"},
		{name: "name continued on the next line", original: "data:\n  eventSpecifications:\n    - name: test\n        one\n      resourceName: a\n"},
		{name: "name on the next line", original: "data:\n  eventSpecifications:\n    - name:\n        test1\n      resourceName: a\n"},
		{name: "comment before a closing brace on the next line", original: "data:\n  eventSpecifications:\n    - {resourceName: a, name: test1 # note\n      }\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res, err := AddEventSpecVersions(map[int]int{0: 1}, []byte(tt.original), "test.yaml")
			if err == nil {
				t.Fatalf("expected an error, got:\n%s", res)
			}
		})
	}
}

func Test_AddEventSpecVersionsJson(t *testing.T) {
	original := `{
  "apiVersion": "v1",
  "resourceType": "data-product",
  "resourceName": "3b7c9fce-9f33-4c91-800b-8c8804f388c9",
  "data": {
    "name": "Café ñandú",
    "eventSpecifications": [
      {
        "resourceName": "15c70141-c223-4172-ab39-86a0194e44e8",
        "name": "Café añadido 🛒",
        "entities": {"tracked": [], "enriched": []}
      },
      {
        "resourceName": "c312c1cc-af41-450f-a086-1c2e8c04a39b",
        "name": "test2",
        "version": 7
      },
      {"resourceName": "4ac70141-c223-4172-ab39-86a0194e44e8", "name":"ñ, 🛒"},
      {
        "name" : "first",
        "resourceName": "6cc70141-c223-4172-ab39-86a0194e44e8"
      }
    ]
  }
}
`
	expected := `{
  "apiVersion": "v1",
  "resourceType": "data-product",
  "resourceName": "3b7c9fce-9f33-4c91-800b-8c8804f388c9",
  "data": {
    "name": "Café ñandú",
    "eventSpecifications": [
      {
        "resourceName": "15c70141-c223-4172-ab39-86a0194e44e8",
        "name": "Café añadido 🛒",
        "version": 4,
        "entities": {"tracked": [], "enriched": []}
      },
      {
        "resourceName": "c312c1cc-af41-450f-a086-1c2e8c04a39b",
        "name": "test2",
        "version": 7
      },
      {"resourceName": "4ac70141-c223-4172-ab39-86a0194e44e8", "name":"ñ, 🛒", "version":0},
      {
        "name" : "first",
        "version" : 1,
        "resourceName": "6cc70141-c223-4172-ab39-86a0194e44e8"
      }
    ]
  }
}
`

	res, err := AddEventSpecVersions(map[int]int{0: 4, 2: 0, 3: 1}, []byte(original), "test.json")
	if err != nil {
		t.Fatalf("Can't add event spec versions %s", err)
	}
	if string(res) != expected {
		t.Fatalf("Result of adding the versions is not expected, expected:\n%s\nactual:\n%s", expected, string(res))
	}
}

func Test_AddEventSpecVersionsJsonRefusesUnsafeEdits(t *testing.T) {
	original := `{"data": {"eventSpecifications": [{"name": "test1", "version": 2}, {"resourceName": "15c70141-c223-4172-ab39-86a0194e44e8"}]}}`

	for _, idx := range []int{0, 1, 2, -1} {
		res, err := AddEventSpecVersions(map[int]int{idx: 1}, []byte(original), "test.json")
		if err == nil {
			t.Fatalf("expected an error for event spec %d, got: %s", idx, res)
		}
		if res != nil {
			t.Errorf("expected no output on error, got: %s", res)
		}
	}
}
