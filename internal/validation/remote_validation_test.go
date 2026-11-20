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
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/changes"
	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/model"
)

func remoteValidationDS(name string, version string) model.DataStructure {
	return model.DataStructure{
		Meta: model.DataStructureMeta{SchemaType: "entity"},
		Data: map[string]any{
			"self":   map[string]any{"vendor": "com.acme", "name": name, "format": "jsonschema", "version": version},
			"schema": "string",
		},
	}
}

func remoteValidationClient(t *testing.T, changeType string) *console.ApiClient {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/validation-requests"):
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			var ds model.DataStructure
			if err := json.Unmarshal(body, &ds); err != nil {
				t.Error(err)
			}
			data, err := ds.ParseData()
			if err != nil {
				t.Error(err)
			}
			w.WriteHeader(http.StatusCreated)
			if data.Self.Name == "noisy" {
				_, _ = io.WriteString(w, `{"success":false,"warnings":["a warning"],"info":["an info"],"errors":["an error"]}`)
				return
			}
			_, _ = io.WriteString(w, `{"success":true}`)
		case strings.HasSuffix(r.URL.Path, "/destinations/v3"):
			_, _ = io.WriteString(w, `[{"destinationType":"snowflake"}]`)
		case strings.HasSuffix(r.URL.Path, "/schema-migrations"):
			_, _ = io.WriteString(w, `{"changeType":"`+changeType+`","migrations":[{"message":"field removed"}]}`)
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return &console.ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: server.URL}
}

func Test_ValidateChanges_CollectsIgluAndMigrationFailures(t *testing.T) {
	client := remoteValidationClient(t, "major")
	toValidate := changes.Changes{
		ToCreate:           []model.DSChangeContext{changes.NewDSChangeContext(remoteValidationDS("noisy", "1-0-0"), "noisy.yml")},
		ToUpdateNewVersion: []model.DSChangeContext{changes.NewDSChangeContextWithVersion(remoteValidationDS("bumped", "1-0-1"), "bumped.yml", "1-0-0")},
	}

	vr, err := ValidateChanges(context.Background(), client, toValidate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vr.Valid || vr.Message != "2 validation failures" {
		t.Errorf("valid = %v, message = %q, want invalid with 2 failures", vr.Valid, vr.Message)
	}
	wantIglu := []igluValidation{
		{"noisy.yml", []string{"a warning"}, igluValidationWarn},
		{"noisy.yml", []string{"an info"}, igluValidationInfo},
		{"noisy.yml", []string{"an error"}, igluValidationError},
	}
	if !reflect.DeepEqual(vr.Iglu, wantIglu) {
		t.Errorf("iglu = %+v, want %+v in this order", vr.Iglu, wantIglu)
	}
	wantMigration := []migrationValidation{{"bumped.yml", "2-0-0", "snowflake", []string{"field removed"}}}
	if !reflect.DeepEqual(vr.Migration, wantMigration) {
		t.Errorf("migration = %+v, want %+v", vr.Migration, wantMigration)
	}
}

func Test_ValidateChanges_Valid(t *testing.T) {
	client := remoteValidationClient(t, "no-change")
	toValidate := changes.Changes{
		ToCreate:      []model.DSChangeContext{changes.NewDSChangeContext(remoteValidationDS("created", "1-0-0"), "created.yml")},
		ToUpdatePatch: []model.DSChangeContext{changes.NewDSChangeContextWithVersion(remoteValidationDS("patched", "1-0-0"), "patched.yml", "1-0-0")},
	}

	vr, err := ValidateChanges(context.Background(), client, toValidate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !vr.Valid || vr.Message != "" || len(vr.Iglu) != 0 || len(vr.Migration) != 0 {
		t.Errorf("expected a clean valid result, got %+v", vr)
	}
}
