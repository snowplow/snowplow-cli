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
	"errors"
	"slices"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/model"
	"gopkg.in/yaml.v3"
)

func Test_ValidateSAMinimum(t *testing.T) {
	mvSa := `
apiVersion: v1
resourceType: source-application
resourceName: 1111-111
data:
  name:
`

	var sa map[string]any
	_ = yaml.Unmarshal([]byte(mvSa), &sa)

	result, ok := ValidateSAShape(sa)

	if ok {
		t.Fatal("valid but shouldn't be")
	}

	if _, ok := result.ErrorsWithPaths["/data/name"]; !ok {
		t.Fatal("expected errors at /data/name")
	}

	if _, ok := result.ErrorsWithPaths["/resourceName"]; !ok {
		t.Fatal("expected errors at /resourceName")
	}
}

func Test_ValidateSAAppIds(t *testing.T) {
	mvSa := `
apiVersion: v1
resourceType: source-application
resourceName: 791a4198-e1ca-4fcf-9c1f-bc882830a34f
data:
  name: name
  appIds:
  -
  - two
`

	var sa map[string]any
	_ = yaml.Unmarshal([]byte(mvSa), &sa)

	result, ok := ValidateSAShape(sa)

	t.Log(result)

	if ok {
		t.Fatal("valid but shouldn't be")
	}

	if _, ok := result.ErrorsWithPaths["/data/appIds/0"]; !ok {
		t.Fatal("expected errors at /data/name")
	}
}

func Test_ValidateSAEntitesSourcesOk(t *testing.T) {
	mvSa := `
apiVersion: v1
resourceType: source-application
resourceName: 791a4198-e1ca-4fcf-9c1f-bc882830a34f
data:
  name: name
  appIds: []
  entities:
    enriched: []
    tracked:
    - source: iglu:vendor/name/format/1-0-0
`

	var sa map[string]any
	_ = yaml.Unmarshal([]byte(mvSa), &sa)

	result, ok := ValidateSAShape(sa)

	t.Log(result)

	if !ok {
		t.Fatal("should be valid")
	}
}

func Test_ValidateSAEntitesSourcesBad(t *testing.T) {
	mvSa := `
apiVersion: v1
resourceType: source-application
resourceName: 791a4198-e1ca-4fcf-9c1f-bc882830a34f
data:
  name: name
  appIds: []
  entities:
    enriched: []
    tracked:
    - source:
`

	var sa map[string]any
	_ = yaml.Unmarshal([]byte(mvSa), &sa)

	result, ok := ValidateSAShape(sa)

	t.Log(result)

	if ok {
		t.Fatal("should be invalid")
	}

	if _, ok := result.ErrorsWithPaths["/data/entities/tracked/0/source"]; !ok {
		t.Fatal("expected errors at /data/entities/tracked/0/source")
	}
}

func Test_ValidateSAEntitesCardinalities(t *testing.T) {
	minusOne := -1
	zero := 0
	one := 1

	sa := model.SourceApp{
		Data: model.SourceAppData{
			Entities: &model.EntitiesDef{
				Tracked: []model.SchemaRef{
					{Source: "something0", MinCardinality: &minusOne},
					{Source: "something1", MinCardinality: &zero, MaxCardinality: &minusOne},
					{Source: "something2", MinCardinality: &one, MaxCardinality: &zero},
					{Source: "something3", MaxCardinality: &one},
					{Source: "something4", MinCardinality: &one, MaxCardinality: &one},
				},
			},
		},
	}

	result := ValidateSAEntitiesCardinalities(sa).ErrorsWithPaths

	expected := map[string][]string{
		"/data/entities/tracked/0/minCardinality": {"must be > 0"},
		"/data/entities/tracked/1/maxCardinality": {"must be > minCardinality: 0"},
		"/data/entities/tracked/2/maxCardinality": {"must be > minCardinality: 1"},
		"/data/entities/tracked/3/maxCardinality": {"without minCardinality"},
	}

	for k, v := range expected {
		if errs, ok := result[k]; ok {
			if !slices.Equal(errs, v) {
				t.Fatalf("unexpected got: %#v want %#v", errs, v)
			}
		} else {
			t.Fatalf("missing err at path %s", k)
		}
	}
}

type mockSdc struct {
	deployed []string
}

func (mi *mockSdc) IsDSDeployed(uri string) (bool, []string, error) {
	return slices.Contains(mi.deployed, uri), nil, nil
}

func Test_ValidateSAEntitiesSchemaDeployed(t *testing.T) {
	sa := model.SourceApp{
		Data: model.SourceAppData{
			Entities: &model.EntitiesDef{
				Tracked: []model.SchemaRef{
					{Source: "iglu:vendor/name/format/2-0-0"},
					{Source: "iglu:invalid/***/format/2-0-0"},
					{Source: "iglu:vendor/name/format/1-0-0"},
				},
				Enriched: []model.SchemaRef{
					{Source: "iglu:vendor/name/format/1-0-0"},
				},
			},
		},
	}

	sdc := &mockSdc{[]string{"iglu:vendor/name/format/1-0-0"}}

	result := ValidateSAEntitiesSchemaDeployed(sdc, sa).ErrorsWithPaths

	expectedOne := "/data/entities/tracked/0/source"
	expectedTwo := "/data/entities/tracked/1/source"

	t.Log(result)

	for _, p := range []string{expectedOne, expectedTwo} {
		if _, ok := result[p]; !ok {
			t.Fatalf("expected errors at %s", p)
		}
	}
}

type mockSdcWithAlternatives struct {
	found        bool
	alternatives []string
	err          error
}

func (m *mockSdcWithAlternatives) IsDSDeployed(uri string) (bool, []string, error) {
	return m.found, m.alternatives, m.err
}

func Test_checkSchemaDeployed(t *testing.T) {
	uri := "iglu:vendor/name/format/2-0-0"
	cases := []struct {
		name     string
		sdc      *mockSdcWithAlternatives
		expected []string
	}{
		{"deployed", &mockSdcWithAlternatives{found: true}, []string{}},
		{"not deployed", &mockSdcWithAlternatives{}, []string{"could not find deployment of " + uri}},
		{
			"not deployed, empty list of other versions",
			&mockSdcWithAlternatives{alternatives: []string{}},
			[]string{"could not find deployment of " + uri},
		},
		{
			"other versions deployed",
			&mockSdcWithAlternatives{alternatives: []string{"a", "b"}},
			[]string{"could not find deployment of " + uri + ", available versions (a, b)"},
		},
		{
			"five other versions deployed",
			&mockSdcWithAlternatives{alternatives: []string{"a", "b", "c", "d", "e"}},
			[]string{"could not find deployment of " + uri + ", available versions (a, b, c, d, e)"},
		},
		{
			"many other versions deployed",
			&mockSdcWithAlternatives{alternatives: []string{"a", "b", "c", "d", "e", "f", "g"}},
			[]string{"could not find deployment of " + uri + ", available versions (a, b, c, d, e, ...2 more)"},
		},
		{
			"lookup failed",
			&mockSdcWithAlternatives{err: errors.New("boom")},
			[]string{"error while resolving boom", "could not find deployment of " + uri},
		},
		{
			"lookup failed but deployed",
			&mockSdcWithAlternatives{found: true, err: errors.New("boom")},
			[]string{"error while resolving boom"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path, result := checkSchemaDeployed(c.sdc, 3, "tracked", uri)
			if path != "/data/entities/tracked/3/source" {
				t.Fatalf("unexpected path: %s", path)
			}
			if !slices.Equal(result, c.expected) {
				t.Fatalf("unexpected got: %#v want %#v", result, c.expected)
			}
		})
	}
}
