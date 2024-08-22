package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_NewClient_Ok(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/credentials/v2/token" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"accessToken":"token"}`))
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client, _ := NewApiClient(cnx, server.URL, "apikey", "orgid")

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
			json.Unmarshal(b, &ds)

			if ds.Meta.SchemaType != "entity" {
				t.Errorf("ds meta not as expected, got: %s", ds.Meta.SchemaType)
			}

			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, `{"success":true}`)
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	ds := &DataStructure{
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
			io.WriteString(w, `{"success":false,"errors":["error1"]}`)
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	result, err := Validate(cnx, client, &DataStructure{})

	if result != nil {
		t.Error(result)
	}

	if err == nil || err.Error() != "error1" {
		t.Error("expected failure, got success")
	}
}
