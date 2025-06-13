/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package console

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/snowplow/snowplow-cli/internal/model"
)

func Test_NewClient_Ok(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, idHeader := r.Header["X-Api-Key-Id"]
		_, secretHeader := r.Header["X-Api-Key"]
		if r.URL.Path == "/api/msc/v1/organizations/orgid/credentials/v3/token" && idHeader && secretHeader {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"accessToken":"token"}`))
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client, _ := NewApiClient(cnx, server.URL, "apiKeyId", "apiKeySecret", "orgid")

	if client.Jwt != "token" {
		t.Errorf("jwt not ok, got: %s", client.Jwt)
	}
}

func Test_Validate_Ok(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1/validation-requests" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			b, err := io.ReadAll(r.Body)
			defer func() { _ = r.Body.Close() }()
			if err != nil {
				t.Error(err)
			}
			var ds DataStructure
			_ = json.Unmarshal(b, &ds)

			if ds.Meta.SchemaType != "entity" {
				t.Errorf("ds meta not as expected, got: %s", ds.Meta.SchemaType)
			}

			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"success":true}`)
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	ds := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity", CustomData: map[string]string{}},
		Data: map[string]any{},
	}

	result, err := Validate(cnx, client, ds)
	if err != nil {
		t.Error(err)
	}

	if !result.Success {
		t.Errorf("expected success, got failure")
	}
}

func Test_Validate_Fail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1/validation-requests" {
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"success":false,"errors":["error1"]}`)
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	vr, err := Validate(cnx, client, DataStructure{})

	if vr.Valid || err != nil {
		t.Error("expected failure, got success")
	}
}

func Test_Validate_FailCompletely(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1/validation-requests" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"message":"bad"}`)
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	result, err := Validate(cnx, client, DataStructure{})

	if result != nil {
		t.Error(result)
	}

	if err == nil {
		t.Error("expected failure, got success")
	}
}

func Test_publish_Ok(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1/deployment-requests" {
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"success":true}`)
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	result, err := publish(cnx, client, VALIDATED, DEV, DataStructure{}, false)
	if err != nil {
		t.Error(err)
	}

	if !result.Success {
		t.Error("expected success, got failure")
	}
}

func Test_publish_Fail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1/deployment-requests" {
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"success":false, "errors": ["error1"]}`)
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	result, err := publish(cnx, client, VALIDATED, DEV, DataStructure{}, false)

	if result != nil {
		t.Error(result)
	}

	if err == nil || err.Error() != "error1" {
		t.Error("expected failure, got success")
	}
}

func Test_publish_FailCompletely(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1/deployment-requests" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"message":"very bad"}`)
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	result, err := publish(cnx, client, VALIDATED, DEV, DataStructure{}, false)

	if result != nil {
		t.Error(result)
	}

	if err == nil {
		t.Error("expected failure, got success")
	}
}

func Test_GetAllDataStructuresOk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/msc/v1/organizations/orgid/data-structures/v1":
			{
				if r.Header.Get("authorization") != "Bearer token" {
					t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
				}

				resp := `[
					{
						"hash": "1d0e5aecd7b08c8dc0ee37e68a3a6cab9bb737ca7114f4ef67f16d415f23e6e8",
						"organizationId": "177234df-d425-412e-ad8d-8b97515b2807",
						"vendor": "com.snplow.msc.aws",
						"name": "d_entity",
						"format": "jsonschema",
						"description": "",
						"meta": {
							"hidden": true,
							"schemaType": "entity",
							"customData": {}
						},
						"deployments": [
							{
								"version": "2-0-0",
								"patchLevel": 0,
								"contentHash": "cf9d2e8b2c1849d36611c0fb258698dbc48e268fcdeb0850a693d75f68c449fd",
								"env": "DEV",
								"ts": "2024-02-21T08:39:42Z",
								"message": null,
								"initiator": "Registry bootstrapping"
							}
						]
					},
					{
						"hash": "ea9631259272070c8a6f56aa3ec0c5d3fc41ee7390bf4830211e894128978733",
						"organizationId": "177234df-d425-412e-ad8d-8b97515b2807",
						"vendor": "com.acme",
						"name": "ad_click",
						"format": "jsonschema",
						"description": null,
						"meta": {
							"hidden": false,
							"schemaType": "event",
							"customData": {}
						},
						"deployments": [
							{
								"version": "1-0-0",
								"patchLevel": 0,
								"contentHash": "2725be307976bdcb14f826b2061d4011fb02b05e60c7ba4ebe02bd9a611f43d5",
								"env": "DEV",
								"ts": "2023-03-29T18:19:31Z",
								"message": "Inserted via registry bootstrapping",
								"initiator": "Registry bootstrapping"
							}
						]
					}
				]`

				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, resp)
				return
			}
		case "/api/msc/v1/organizations/orgid/data-structures/v1/schemas/versions":
			{
				if r.Header.Get("authorization") != "Bearer token" {
					t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
				}

				resp := `
			[
				{
						"$schema": "http://iglucentral.com/schemas/com.snowplowanalytics.self-desc/schema/jsonschema/1-0-0#",
						"additionalProperties": false,
						"description": "Schema for an example event",
						"properties": {
						  "a_mixed_boolean_type": {
							"description": "desc",
							"type": [
							  "boolean",
							  "null"
							]
						  },
						  "a_mixed_boolean_type_enum": {
							"description": "desc",
							"enum": [
							  false,
							  null
							],
							"type": [
							  "boolean",
							  "null"
							]
						  },
						  "a_mixed_integer_type": {
							"description": "desc",
							"maximum": 2147483647,
							"minimum": 0,
							"type": [
							  "integer",
							  "null"
							]
						  },
						  "a_mixed_integer_type_enum": {
							"description": "desc",
							"enum": [
							  11,
							  72,
							  null
							],
							"maximum": 120,
							"minimum": 0,
							"type": [
							  "integer",
							  "null"
							]
						  },
						  "a_mixed_number_type": {
							"description": "desc",
							"maximum": 123456,
							"minimum": 0,
							"type": [
							  "number",
							  "null"
							]
						  },
						  "a_mixed_number_type_enum": {
							"description": "desc",
							"enum": [
							  11,
							  72,
							  null
							],
							"maximum": 123456,
							"minimum": 0,
							"type": [
							  "number",
							  "null"
							]
						  },
						  "a_mixed_string_type": {
							"description": "desc",
							"maxLength": 200,
							"type": [
							  "string",
							  "null"
							]
						  },
						  "a_mixed_string_type_enum": {
							"description": "desc",
							"enum": [
							  "something",
							  "another",
							  null
							],
							"maxLength": 200,
							"type": [
							  "string",
							  "null"
							]
						  },
						  "a_null_type": {
							"type": "null"
						  },
						  "a_null_untyped_enum": {
							"enum": [
							  null
							]
						  },
						  "a_timestamp": {
							"format": "date-time",
							"type": "string"
						  },
						  "aboolean": {
							"type": "boolean"
						  },
						  "anarray": {
							"items": {
							  "maxLength": 500,
							  "type": "string"
							},
							"type": "array"
						  },
						  "aninteger": {
							"maximum": 2147483647,
							"minimum": 0,
							"type": "integer"
						  },
						  "aninteger_enum": {
							"description": "a number",
							"enum": [
							  17,
							  35,
							  48452
							],
							"maximum": 123456789,
							"minimum": 0,
							"type": "integer"
						  },
						  "anumber": {
							"maximum": 180,
							"minimum": -180,
							"type": "number"
						  },
						  "anumber_enum": {
							"description": "a number",
							"enum": [
							  -17,
							  35,
							  48452
							],
							"maximum": 123456789,
							"minimum": 0,
							"type": "number"
						  },
						  "astring": {
							"description": "a description",
							"maxLength": 200,
							"type": "string"
						  },
						  "astring_enum": {
							"description": "a description",
							"enum": [
							  "one",
							  "two"
							],
							"maxLength": 200,
							"type": "string"
						  },
						  "untyped_enum_component": {
							"enum": [
							  "something",
							  11,
							  null,
							  false
							]
						  }
						},
						"required": [
						  "astring"
						],
						"self": {
						  "format": "jsonschema",
						  "name": "d_test",
						  "vendor": "com.snplow.msc.aws",
						  "version": "2-0-0"
						},
						"type": "object"
				},
				{
						"$schema": "http://iglucentral.com/schemas/com.snowplowanalytics.self-desc/schema/jsonschema/1-0-0#",
						"additionalProperties": false,
						"description": "Schema for an example event",
						"properties": {
						  "astring": {
							"description": "a description",
							"maxLength": 200,
							"type": "string"
						  }
						},
						"required": [
						  "astring"
						],
						"self": {
						  "format": "jsonschema",
						  "name": "d_test",
						  "vendor": "com.snplow.msc.aws",
						  "version": "1-0-0"
						},
						"type": "object"
				}
			]`

				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, resp)
				return
			}
		default:
			t.Errorf("Unexpected request, got: %s", r.URL.Path)
			return
		}
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	result, err := GetAllDataStructures(cnx, client, []string{}, false)
	if err != nil {
		t.Error(err)
	}

	if len(result) != 2 {
		t.Errorf("Unexpected number of results, expected 2, got: %d", len(result))
	}
}

func Test_MetadataUpdate_Ok(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1/20308fa345d397de04f26a34a6083744d06ae1aeb673e1658b0b50a7a86ea395/meta" {
			w.WriteHeader(http.StatusOK)
			return
		}

		t.Fatalf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	orgId := "00000000-0000-0000-0000-000000000000"
	var ds DataStructure
	_ = json.Unmarshal([]byte(`{
		"meta": { "hidden": false, "schemaType": "event", "customMetadata": {} },
		"data": { "self": { "name": "example", "vendor": "io.snowplow", "version": "1-0-0", "format": "jsonschema" } }
	}`), &ds)

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL), OrgId: orgId}

	err := MetadateUpdate(cnx, client, &ds, "")
	if err != nil {
		t.Fatal("expected failure, got success")
	}
}

func Test_MetadataUpdate_Fail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"message":"very bad"}`)
	}))
	defer server.Close()

	orgId := "00000000-0000-0000-0000-000000000000"
	var ds DataStructure
	_ = json.Unmarshal([]byte(`{
		"meta": { "hidden": false, "schemaType": "event", "customMetadata": {} },
		"data": { "self": { "name": "example", "vendor": "io.snowplow", "version": "1-0-0", "format": "jsonschema" } }
	}`), &ds)

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL), OrgId: orgId}

	err := MetadateUpdate(cnx, client, &ds, "")

	if err == nil {
		t.Fatal("expected failure, got success")
	}
}

func Test_Patch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1/deployment-requests" && r.URL.Query().Get("patch") == "true" {
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"success":true}`)
			return
		}
		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	result, err := publish(cnx, client, VALIDATED, DEV, DataStructure{}, true)
	if err != nil {
		t.Error(err)
	}

	if !result.Success {
		t.Error("expected success, got failure")
	}
}

func TestGetAllDataStructures_Matching(t *testing.T) {
	mockListings := []ListResponse{
		{
			Hash:   "abc123",
			Vendor: "com.acme",
			Name:   "event",
			Format: "jsonschema",
			Meta:   DataStructureMeta{SchemaType: "event"},
			Deployments: []Deployment{
				{Env: DEV, Version: "1-0-0"},
			},
		},
		{
			Hash:   "def456",
			Vendor: "org.example",
			Name:   "purchase",
			Format: "jsonschema",
			Meta:   DataStructureMeta{SchemaType: "event"},
			Deployments: []Deployment{
				{Env: DEV, Version: "2-0-0"},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/data-structures/v1") && len(r.URL.Path) == len("/data-structures/v1"):
			data, _ := json.Marshal(mockListings)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)

		case r.URL.Path == "/data-structures/v1/schemas/versions":
			_, _ = io.WriteString(w, `
				[
					{
						"self": { "name": "event", "vendor": "com.acme", "version": "1-0-0", "format": "jsonschema" }
					},
					{
						"self": { "name": "purchase", "vendor": "org.example", "version": "2-0-0", "format": "jsonschema" }
					}
				]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &ApiClient{
		BaseUrl: server.URL,
		Jwt:     "fake-jwt",
		Http:    server.Client(),
	}

	ctx := context.Background()
	match := []string{"com.acme/event"} // only match one of the two

	res, err := GetAllDataStructures(ctx, client, match, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(res) != 1 {
		t.Fatalf("expected 1 matching data structure, got %d", len(res))
	}

	if data, _ := res[0].ParseData(); data.Self.Name != "event" {
		t.Errorf("unexpected data structure: %+v", data.Self)
	}
}

func TestGetAllDataStructures_EmptySchemaType(t *testing.T) {
	mockListings := []ListResponse{
		{
			Hash:   "abc123",
			Vendor: "com.legacy",
			Name:   "old_event",
			Format: "jsonschema",
			Meta:   DataStructureMeta{SchemaType: ""}, // Empty schemaType
			Deployments: []Deployment{
				{Env: DEV, Version: "1-0-0"},
			},
		},
		{
			Hash:   "def456",
			Vendor: "com.modern",
			Name:   "new_event",
			Format: "jsonschema",
			Meta:   DataStructureMeta{SchemaType: "event"}, // Valid schemaType
			Deployments: []Deployment{
				{Env: DEV, Version: "1-0-0"},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/data-structures/v1") && len(r.URL.Path) == len("/data-structures/v1"):
			data, _ := json.Marshal(mockListings)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)

		case r.URL.Path == "/data-structures/v1/schemas/versions":
			_, _ = io.WriteString(w, `
				[
					{
						"self": { "name": "old_event", "vendor": "com.legacy", "version": "1-0-0", "format": "jsonschema" }
					},
					{
						"self": { "name": "new_event", "vendor": "com.modern", "version": "1-0-0", "format": "jsonschema" }
					}
				]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &ApiClient{
		BaseUrl: server.URL,
		Jwt:     "fake-jwt",
		Http:    server.Client(),
	}

	ctx := context.Background()

	res, err := GetAllDataStructures(ctx, client, []string{}, false)
	if err != nil {
		t.Fatalf("GetAllDataStructures with includeLegacy=false failed: %v", err)
	}

	if len(res) != 1 {
		t.Fatalf("default behavior should skip data structures with empty schemaType: expected 1 result, got %d", len(res))
	}

	if data, _ := res[0].ParseData(); data.Self.Name != "new_event" {
		t.Errorf("default behavior should only return modern data structure: expected new_event, got %s", data.Self.Name)
	}

	res, err = GetAllDataStructures(ctx, client, []string{}, true)
	if err != nil {
		t.Fatalf("GetAllDataStructures with includeLegacy=true failed: %v", err)
	}

	if len(res) != 2 {
		t.Fatalf("includeLegacy=true should include data structures with empty schemaType: expected 2 results, got %d", len(res))
	}

	var legacyDS *DataStructure
	for i := range res {
		if data, _ := res[i].ParseData(); data.Self.Name == "old_event" {
			legacyDS = &res[i]
			break
		}
	}

	if legacyDS == nil {
		t.Fatal("includeLegacy=true should include legacy data structure with name 'old_event'")
	}

	if legacyDS.Meta.SchemaType != "entity" {
		t.Errorf("legacy data structure with empty schemaType should be converted to 'entity': got '%s'", legacyDS.Meta.SchemaType)
	}
}
