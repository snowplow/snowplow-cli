/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDataStructuJsonParseSuccess(t *testing.T) {
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
		  "version": {
			"model": 1073741824,
			"revision": 1073741824,
			"addition": 1073741824
		  }
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
		Data: DataStrucutreData{
			Self: DataStructureSelf{
				Vendor: "string",
				Name:   "string",
				Format: "string",
				Version: DataStructureVersion{
					Model:    1073741824,
					Revision: 1073741824,
					Addition: 1073741824,
				},
			},
			Schema: "string"},
	}
	res := DataStructure{}
	err := json.Unmarshal([]byte(jsonString), &res)
	if !reflect.DeepEqual(expected, res) || err != nil {
		t.Fatalf("Cant' parse json %s\n parsed %#v\n expected %#v", err, res, expected)
	}

}

func TestDataStructuJsonParseFailureWrongFormat(t *testing.T) {
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
		  "version": {
			"model": 1073741824,
			"revision": 1073741824,
			"addition": 1073741824
		  }
		},
	  }
	}`)
	res := DataStructure{}
	err := json.Unmarshal([]byte(jsonString), &res)
	if err == nil {
		t.Fatal("Parsed data structure without schema")
	}

}
