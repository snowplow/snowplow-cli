/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDataStructureJsonParseSuccess(t *testing.T) {
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
	yamlString := string(`meta:
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
			"schema":                "string",
			"additionalPropperties": false},
	}
	expected := DataStrucutreData{
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

	dsParsed, err := ds.parseData()
	if !reflect.DeepEqual(dsParsed, expected) || err != nil {
		t.Fatalf("Cant' parse map %s\n parsed %#v\n expected %#v", err, dsParsed, expected)
	}
}
