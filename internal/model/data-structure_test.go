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
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDataStructureJsonParseSuccess(t *testing.T) {
	jsonString := string(`{
	  "apiVersion": "v1",
      "meta": {
        "hidden": true,
        "schemaType": "entity",
        "customData": {
          "additionalProp1": "string",
          "additionalProp2": "string",
          "additionalProp3": "string"
        }
      },
      "data": {
        "self": {
          "vendor": "string",
          "name": "string",
          "format": "string",
          "version": "1-0-1"
        },
        "schema": "string"
      }
    }`)
	expected := DataStructure{
		ApiVersion: "v1",
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity", CustomData: map[string]string{
			"additionalProp1": "string",
			"additionalProp2": "string",
			"additionalProp3": "string",
		},
		},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-0-1",
			},
			"schema": "string"},
	}
	res := DataStructure{}
	err := json.Unmarshal([]byte(jsonString), &res)
	if !reflect.DeepEqual(expected, res) || err != nil {
		t.Fatalf("Cant' parse json %s\n parsed %#v\n expected %#v", err, res, expected)
	}

}

func TestDataStructureJsonParseFailureWrongFormat(t *testing.T) {
	jsonString := string(`{
	  "apiVersion": "v1",
      "meta": {
        "hidden": true,
        "schemaType": "entity",
        "customData": {
          "additionalProp1": "string",
          "additionalProp2": "string",
          "additionalProp3": "string"
        }
      },
      "data": {
        "self": {
          "vendor": "string",
          "name": "string",
          "format": "string",
          "version": "1-2-0"
        },
      }
    }`)
	res := DataStructure{}
	err := json.Unmarshal([]byte(jsonString), &res)
	if err == nil {
		t.Fatal("Parsed data structure without schema")
	}

}

func TestDataStructureYamlParseSuccess(t *testing.T) {
	yamlString := string(`
apiVersion: v1
meta:
  hidden: true
  schemaType: entity
  customData:
    additionalProp1: string
    additionalProp2: string
    additionalProp3: string
data:
  self:
    vendor: string
    name: string
    format: string
    version: 1-2-0
  schema: string`)
	expected := DataStructure{
		ApiVersion: "v1",
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity", CustomData: map[string]string{
			"additionalProp1": "string",
			"additionalProp2": "string",
			"additionalProp3": "string",
		},
		},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-2-0",
			},
			"schema": "string"},
	}
	res := DataStructure{}
	err := yaml.Unmarshal([]byte(yamlString), &res)
	if !reflect.DeepEqual(expected, res) || err != nil {
		t.Fatalf("Cant' parse yaml %s\n parsed %#v\n expected %#v", err, res, expected)
	}

}

func TestParseDataParses(t *testing.T) {
	ds := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity", CustomData: map[string]string{
			"additionalProp1": "string",
			"additionalProp2": "string",
			"additionalProp3": "string",
		},
		},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-2-0",
			},
			"$schema":               "string",
			"additionalPropperties": false},
	}
	expected := DataStructureData{
		Self: DataStructureSelf{
			Vendor:  "string",
			Name:    "string",
			Format:  "string",
			Version: "1-2-0",
		},
		Schema: "string",
		Other: map[string]any{
			"additionalPropperties": false,
		},
	}

	dsParsed, err := ds.ParseData()
	if !reflect.DeepEqual(dsParsed, expected) || err != nil {
		t.Fatalf("Cant' parse map %s\n parsed %#v\n expected %#v", err, dsParsed, expected)
	}
}

func TestDataStructureHash(t *testing.T) {
	jsonString := string(`
	{
      "meta": {
        "hidden": true,
        "schemaType": "entity",
        "customData": {
          "additionalProp1": "string",
          "additionalProp2": "string",
          "additionalProp3": "string"
        }
      },
      "data": {
		  "description": "Schema for an example event",
		  "properties": {
			"cursed": {
			  "type": "string",
			  "description": "zażółć gęślą jaźń ->",
			  "maxLength": 10
			}
		  },
		  "additionalProperties": false,
		  "type": "object",
		  "required": [
			"cursed"
		  ],
		  "self": {
			"vendor": "cursed",
			"name": "unicode_normal",
			"format": "jsonschema",
			"version": "1-0-10"
		  },
		  "$schema": "http://iglucentral.com/schemas/com.snowplowanalytics.self-desc/schema/jsonschema/1-0-0#"
		}
	}`)

	expectedHash := "3bbd73b8afe99e47d1b02d04750ba03704a95a511035910f718ac9fb6c401490"
	res := DataStructure{}
	err := json.Unmarshal([]byte(jsonString), &res)
	if err != nil {
		t.Fatalf("Cant' parse json %s\n parsed %#v\n ", err, res)
	}

	hash, err := res.GetContentHash()
	if err != nil {
		t.Fatalf("Can't calculate hash: %s", err)
	}
	if hash != expectedHash {
		t.Fatalf("Not expected hash, expected: %s, got: %s\n", expectedHash, hash)
	}
}

func TestDefaultApiVersion(t *testing.T) {
	jsonString := string(`{
      "meta": {
        "hidden": true,
        "schemaType": "entity",
        "customData": {
          "additionalProp1": "string",
          "additionalProp2": "string",
          "additionalProp3": "string"
        }
      },
      "data": {
        "self": {
          "vendor": "string",
          "name": "string",
          "format": "string",
          "version": "1-0-1"
        },
        "schema": "string"
      }
    }`)
	expected := DataStructure{
		ApiVersion: "v1",
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity", CustomData: map[string]string{
			"additionalProp1": "string",
			"additionalProp2": "string",
			"additionalProp3": "string",
		},
		},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-0-1",
			},
			"schema": "string"},
	}
	res := DataStructure{ApiVersion: "v1"}
	err := json.Unmarshal([]byte(jsonString), &res)
	if !reflect.DeepEqual(expected, res) || err != nil {
		t.Fatalf("Cant' parse json %s\n parsed %#v\n expected %#v", err, res, expected)
	}

}
