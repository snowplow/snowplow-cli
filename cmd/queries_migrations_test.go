package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_ValidateMigrationsDestinations_Fail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/destinations/v2" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"message":"bad"}`)
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	result, err := ValidateMigrations(cnx, client, DSChangeContext{})

	if result != nil {
		t.Error(result)
	}

	if err == nil {
		t.Error("expected failure, got success")
	}
}

func Test_ValidateMigrations_Fail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/destinations/v2" {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `[{"destinationType":"something"}]`)
			return
		}

		if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1/schema-migrations" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"message":"bad"}`)
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	result, err := ValidateMigrations(cnx, client, DSChangeContext{
		DS: DataStructure{
			Data: map[string]any{
				"self": map[string]any{
					"vendor":  "test.test",
					"name":    "test",
					"format":  "jsonschema",
					"version": "1-0-1",
				},
			},
		},
	})

	if result != nil {
		t.Error(result)
	}

	if err != nil && err.Error() != "bad" {
		t.Error("expected failure, got success")
	}
}

func migrationServer(changeType string, t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/destinations/v2" {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `[{"destinationType":"something"},{"destinationType":"otherone"}]`)
			return
		}

		if r.URL.Path == "/api/msc/v1/organizations/orgid/data-structures/v1/schema-migrations" {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, fmt.Sprintf(`{"changeType": "%s", "migrations": [
				{"changeType": "major", "message": "major just because", "path": "", "migrationType": ""}
			]}`, changeType))
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
}

func Test_ValidateMigrationsMajor_Ok(t *testing.T) {
	server := migrationServer("major", t)
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	result, err := ValidateMigrations(cnx, client, DSChangeContext{
		RemoteVersion: "1-0-1",
		DS: DataStructure{
			Data: map[string]any{
				"self": map[string]any{
					"vendor":  "test.test",
					"name":    "test",
					"format":  "jsonschema",
					"version": "1-0-1",
				},
			},
		},
	})

	if result == nil {
		t.Error(err)
	}

	if r, ok := result["something"]; ok {
		if r.SuggestedVersion != "2-0-0" {
			t.Error(result)
		}
	}

	if r, ok := result["otherone"]; ok {
		if r.SuggestedVersion != "2-0-0" {
			t.Error(result)
		}
	}
}

func Test_ValidateMigrationsMinor_Ok(t *testing.T) {
	server := migrationServer("minor", t)
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	result, err := ValidateMigrations(cnx, client, DSChangeContext{
		RemoteVersion: "1-0-1",
		DS: DataStructure{
			Data: map[string]any{
				"self": map[string]any{
					"vendor":  "test.test",
					"name":    "test",
					"format":  "jsonschema",
					"version": "1-0-1",
				},
			},
		},
	})

	if result == nil {
		t.Error(err)
	}

	if r, ok := result["something"]; ok {
		if r.SuggestedVersion != "1-0-2" {
			t.Error(result)
		}
	}

	if r, ok := result["otherone"]; ok {
		if r.SuggestedVersion != "1-0-2" {
			t.Error(result)
		}
	}
}
