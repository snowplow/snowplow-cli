/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package validation

import (
	"encoding/json"
	"errors"
	. "github.com/snowplow-product/snowplow-cli/internal/model"
	"strings"
	"testing"
)

func Test_Correct(t *testing.T) {
	jsonString := string(`{
      "apiVersion": "v1",
      "resourceType": "data-structure",
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
          "vendor": "example",
          "name": "example",
          "format": "jsonschema",
          "version": "1-0-1"
        },
        "$schema": "string"
      }
    }`)
	res := DataStructure{}
	err := json.Unmarshal([]byte(jsonString), &res)
	if err != nil {
		t.Fatalf("Cant' parse json %s\n parsed ", err)
	}
	errs := ValidateLocalDs(map[string]DataStructure{"test": res})
	if len(errs) > 0 {
		t.Fatalf("Errors raised for correct data structure %s", errors.Join(errs...).Error())
	}

}

func Test_FailWithoutMeta(t *testing.T) {
	jsonString := string(`{
	  "apiVersion": "v1",
      "resourceType": "data-structure",
      "meta": {},
      "data": {
        "self": {
          "vendor": "string",
          "name": "string",
          "format": "jsonschema",
          "version": "1-0-1"
        },
        "$schema": "string"
      }
    }`)
	res := DataStructure{}
	err := json.Unmarshal([]byte(jsonString), &res)
	if err != nil {
		t.Fatalf("Cant' parse json %s\n parsed ", err)
	}
	errs := ValidateLocalDs(map[string]DataStructure{"test": res})
	if len(errs) == 0 {
		t.Fatalf("No errors raised if metadata is missing")
	}
	e := errs[0].Error()
	if !strings.Contains(e, "dataStructure.meta") {
		t.Fatalf("Error message does not complain about metadata %s", e)
	}

}

func Test_FailWithoutFullMeta(t *testing.T) {
	jsonString := string(`{
      "apiVersion": "v1",
      "resourceType": "data-structure",
      "meta": {
        "hidden": true,
        "customData": {
          "additionalProp1": "string",
          "additionalProp2": "string",
          "additionalProp3": "string"
        }
      },
      "data": {
        "self": {
          "vendor": "example",
          "name": "example",
          "format": "jsonschema",
          "version": "1-0-1"
        },
        "$schema": "string"
      }
    }`)
	res := DataStructure{}
	err := json.Unmarshal([]byte(jsonString), &res)
	if err != nil {
		t.Fatalf("Cant' parse json %s\n parsed ", err)
	}
	errs := ValidateLocalDs(map[string]DataStructure{"test": res})
	if len(errs) == 0 {
		t.Fatalf("No errors raised if schemaType is missing")
	}

	e := errs[0].Error()
	if !strings.Contains(e, "dataStructure.meta.schemaType") {
		t.Fatalf("Error message does not complain about missing schemaType %s", e)
	}

}

func Test_FailWithWrongSchemaType(t *testing.T) {
	jsonString := string(`{
	  "apiVersion": "v1",
      "resourceType": "data-structure",
      "meta": {
        "hidden": true,
        "schemaType": "helloThere",
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
          "format": "jsonschema",
          "version": "1-0-1"
        },
        "$schema": "string"
      }
    }`)
	res := DataStructure{}
	err := json.Unmarshal([]byte(jsonString), &res)
	if err != nil {
		t.Fatalf("Cant' parse json %s\n parsed ", err)
	}
	errs := ValidateLocalDs(map[string]DataStructure{"test": res})
	if len(errs) == 0 {
		t.Fatalf("No errors raised if schemaType have incorrect value")
	}

	e := errs[0].Error()
	if !strings.Contains(e, "dataStructure.meta.schemaType") || !strings.Contains(e, "event, entity") {
		t.Fatalf("Error message does not complain about incorrect schemaType %s", e)
	}
}

func Test_FailWithWrongApiVersion(t *testing.T) {
	jsonString := string(`{
	  "apiVersion": "v2",
      "resourceType": "data-structure",
      "meta": {
        "hidden": true,
        "schemaType": "event",
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
          "format": "jsonschema",
          "version": "1-0-1"
        },
        "$schema": "string"
      }
    }`)
	res := DataStructure{}
	err := json.Unmarshal([]byte(jsonString), &res)
	if err != nil {
		t.Fatalf("Cant' parse json %s\n parsed ", err)
	}
	errs := ValidateLocalDs(map[string]DataStructure{"test": res})
	if len(errs) == 0 {
		t.Fatalf("No errors raised if api has wrong version")
	}

	e := errs[0].Error()
	if !strings.Contains(e, "dataStructure.apiVersion") || !strings.Contains(e, "v1") {
		t.Fatalf("Error message does not complain about incorrect apiVersion %s", e)
	}

}

func Test_WithoutFullSelf(t *testing.T) {
	jsonString := string(`{
      "apiVersion": "v1",
      "resourceType": "data-structure",
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
          "vendor": "example",
          "name": "example",
          "format": "jsonschema"
        },
        "$schema": "string"
      }
    }`)
	res := DataStructure{}
	err := json.Unmarshal([]byte(jsonString), &res)
	if err != nil {
		t.Fatalf("Cant' parse json %s\n parsed ", err)
	}
	errs := ValidateLocalDs(map[string]DataStructure{"test": res})
	if len(errs) == 0 {
		t.Fatalf("No errors raised if schemaType have incorrect value")
	}
	e := errs[0].Error()
	if !strings.Contains(e, "dataStructure.data.self.version") {
		t.Fatalf("Error message does not complain about missing version %s", e)
	}

}

func Test_DuplicateFile(t *testing.T) {
	jsonString := string(`{
      "apiVersion": "v1",
      "resourceType": "data-structure",
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
          "vendor": "example",
          "name": "example",
          "format": "jsonschema",
          "version": "1-0-1"
        },
        "$schema": "string"
      }
    }`)
	res := DataStructure{}
	err := json.Unmarshal([]byte(jsonString), &res)
	if err != nil {
		t.Fatalf("Cant' parse json %s\n parsed ", err)
	}
	errs := ValidateLocalDs(map[string]DataStructure{"test": res, "test2": res})
	if len(errs) == 0 {
		t.Fatalf("No errors raised when ds is duplicated in files")
	}
	e := errs[0].Error()
	if !strings.Contains(e, "unique") {
		t.Fatalf("Error message does not complain about duplicate files %s", e)
	}

}
