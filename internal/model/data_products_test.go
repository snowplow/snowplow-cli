/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package model

import (
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestEntitiesDefMarshalJSONNilSlices(t *testing.T) {
	got, err := json.Marshal(EntitiesDef{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"tracked":[],"enriched":[]}` {
		t.Fatalf("unexpected json: %s", got)
	}
}

func TestEntitiesDefMarshalJSONKeepsValues(t *testing.T) {
	got, err := json.Marshal(EntitiesDef{
		Tracked:  []SchemaRef{{Source: "iglu:com.acme/a/jsonschema/1-0-0"}},
		Enriched: []SchemaRef{{Source: "iglu:com.acme/b/jsonschema/1-0-0"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `{"tracked":[{"source":"iglu:com.acme/a/jsonschema/1-0-0"}],"enriched":[{"source":"iglu:com.acme/b/jsonschema/1-0-0"}]}`
	if string(got) != want {
		t.Fatalf("unexpected json: %s", got)
	}
}

func TestEventSpecCanonicalMarshalEvent(t *testing.T) {
	one := 1
	cases := []struct {
		name      string
		event     *SchemaRef
		wantEvent bool
	}{
		{"nil", nil, false},
		{"empty", &SchemaRef{}, false},
		{"source", &SchemaRef{Source: "iglu:com.acme/a/jsonschema/1-0-0"}, true},
		{"min cardinality", &SchemaRef{MinCardinality: &one}, true},
		{"max cardinality", &SchemaRef{MaxCardinality: &one}, true},
		{"schema", &SchemaRef{Schema: map[string]any{"type": "object"}}, true},
	}
	for _, c := range cases {
		es := EventSpecCanonical{ResourceName: "r", Name: "n", Event: c.event}

		jsonBytes, err := json.Marshal(es)
		if err != nil {
			t.Fatalf("%s: unexpected json error: %v", c.name, err)
		}
		if got := strings.Contains(string(jsonBytes), `"event":`); got != c.wantEvent {
			t.Errorf("%s: json has event = %v, want %v: %s", c.name, got, c.wantEvent, jsonBytes)
		}

		yamlBytes, err := yaml.Marshal(es)
		if err != nil {
			t.Fatalf("%s: unexpected yaml error: %v", c.name, err)
		}
		if got := strings.Contains(string(yamlBytes), "event:"); got != c.wantEvent {
			t.Errorf("%s: yaml has event = %v, want %v: %s", c.name, got, c.wantEvent, yamlBytes)
		}
	}
}
