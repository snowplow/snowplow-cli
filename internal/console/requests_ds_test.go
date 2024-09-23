package console

import (
	"context"
	"encoding/json"
	"fmt"
	. "github.com/snowplow-product/snowplow-cli/internal/model"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
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
			defer r.Body.Close()
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
		if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp :=
				`[
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
		} else if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1/1d0e5aecd7b08c8dc0ee37e68a3a6cab9bb737ca7114f4ef67f16d415f23e6e8/versions/2-0-0" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp := `{
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
					  }`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		} else if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1/ea9631259272070c8a6f56aa3ec0c5d3fc41ee7390bf4830211e894128978733/versions/1-0-0" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp := `{
					"$schema": "http://iglucentral.com/schemas/com.snowplowanalytics.self-desc/schema/jsonschema/1-0-0#",
					"self": {
					  "format": "jsonschema",
					  "name": "ad_click",
					  "vendor": "com.acme",
					  "version": "1-0-0"
					},
					"type": "BodyOnly"
				  }`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	result, err := GetAllDataStructures(cnx, client)

	if err != nil {
		t.Error(err)
	}

	if len(result) != 2 {
		t.Errorf("Unexpected number of results, expected 2, got: %d", len(result))
	}

}

func Test_GetAllDataStructuresSkips404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp :=
				`[
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
		} else if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1/1d0e5aecd7b08c8dc0ee37e68a3a6cab9bb737ca7114f4ef67f16d415f23e6e8/versions/2-0-0" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"message":"Im lost"}`)
			return
		} else if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1/ea9631259272070c8a6f56aa3ec0c5d3fc41ee7390bf4830211e894128978733/versions/1-0-0" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp := `{
					"$schema": "http://iglucentral.com/schemas/com.snowplowanalytics.self-desc/schema/jsonschema/1-0-0#",
					"self": {
					  "format": "jsonschema",
					  "name": "ad_click",
					  "vendor": "com.acme",
					  "version": "1-0-0"
					},
					"type": "BodyOnly"
				  }`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	result, err := GetAllDataStructures(cnx, client)

	if err != nil {
		t.Error(err)
	}

	if len(result) != 1 {
		t.Errorf("Unexpected number of results, expected 1, got: %d", len(result))
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
