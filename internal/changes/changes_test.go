/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package changes

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	. "github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/logging"
	. "github.com/snowplow/snowplow-cli/internal/model"
)

func Test_GetChangesCreate(t *testing.T) {
	local := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity"},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-0-1",
			},
			"schema": "string"},
	}

	res, err := GetChanges(map[string]DataStructure{"file": local}, []ListResponse{}, "DEV")

	if err != nil {
		t.Fatalf("Can't calcuate changes %s", err)
	}

	if len(res.ToCreate) != 1 || len(res.ToUpdateMeta) != 0 || len(res.ToUpdatePatch) != 0 || len(res.ToUpdateNewVersion) != 0 {
		t.Fatalf("Unexpected result, expecting one data structre to be created, got %+v", res)
	}
}

func Test_GetChangesUpdateAndMeta(t *testing.T) {
	local := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity"},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-0-1",
			},
			"schema": "string"},
	}
	remote := ListResponse{
		Hash:   "different",
		Vendor: "string",
		Name:   "string",
		Meta:   DataStructureMeta{Hidden: false, SchemaType: "entity"},
		Format: "string",
		Deployments: []Deployment{
			{
				Version:     "1-0-0",
				Env:         "DEV",
				ContentHash: "different",
			},
		},
	}

	res, err := GetChanges(map[string]DataStructure{"file": local}, []ListResponse{remote}, "DEV")

	if err != nil {
		t.Fatalf("Can't calcuate changes %s", err)
	}

	if len(res.ToCreate) != 0 || len(res.ToUpdateMeta) != 1 || len(res.ToUpdatePatch) != 0 || len(res.ToUpdateNewVersion) != 1 {
		t.Fatalf("Unexpected result, expecting one data structre to update metadata and new version, got %+v", res)
	}
}

func Test_GetChangesPatch(t *testing.T) {
	local := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity"},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-0-0",
			},
			"schema": "string"},
	}
	remote := ListResponse{
		Hash:   "different",
		Vendor: "string",
		Name:   "string",
		Meta:   DataStructureMeta{Hidden: true, SchemaType: "entity"},
		Format: "string",
		Deployments: []Deployment{
			{
				Version:     "1-0-0",
				Env:         "DEV",
				ContentHash: "different",
			},
		},
	}

	res, err := GetChanges(map[string]DataStructure{"file": local}, []ListResponse{remote}, "DEV")

	if err != nil {
		t.Fatalf("Can't calcuate changes %s", err)
	}

	if len(res.ToCreate) != 0 || len(res.ToUpdateMeta) != 0 || len(res.ToUpdatePatch) != 1 || len(res.ToUpdateNewVersion) != 0 {
		t.Fatalf("Unexpected result, expecting one data structre to update patch, got %+v", res)
	}
}

func Test_GetChangesNoChange(t *testing.T) {
	local := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity"},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-0-0",
			},
			"schema": "string"},
	}
	remote := ListResponse{
		Hash:   "different",
		Vendor: "string",
		Name:   "string",
		Meta:   DataStructureMeta{Hidden: true, SchemaType: "entity"},
		Format: "string",
		Deployments: []Deployment{
			{
				Version:     "1-0-0",
				Env:         "DEV",
				ContentHash: "d11f7d148af26aa88594bfe37e94b34dd5d933bbd7a19be50a0cde5feb532e2a",
			},
		},
	}

	res, err := GetChanges(map[string]DataStructure{"file": local}, []ListResponse{remote}, "DEV")

	if err != nil {
		t.Fatalf("Can't calcuate changes %s", err)
	}

	if len(res.ToCreate) != 0 || len(res.ToUpdateMeta) != 0 || len(res.ToUpdatePatch) != 0 || len(res.ToUpdateNewVersion) != 0 {
		t.Fatalf("Unexpected result, expecting no changes, got %+v", res)
	}
}

func Test_GetChangesProdDeploy(t *testing.T) {
	local := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity"},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-0-0",
			},
			"schema": "string"},
	}
	remote := ListResponse{
		Hash:   "different",
		Vendor: "string",
		Name:   "string",
		Meta:   DataStructureMeta{Hidden: true, SchemaType: "entity"},
		Format: "string",
		Deployments: []Deployment{
			{
				Version:     "1-0-0",
				Env:         "DEV",
				ContentHash: "different",
			},
		},
	}

	res, err := GetChanges(map[string]DataStructure{"file": local}, []ListResponse{remote}, "PROD")

	if err != nil {
		t.Fatalf("Can't calcuate changes %s", err)
	}

	if len(res.ToCreate) != 0 || len(res.ToUpdateMeta) != 0 || len(res.ToUpdatePatch) != 0 || len(res.ToUpdateNewVersion) != 1 {
		t.Fatalf("Unexpected result, expecting one data structre to update, got %+v", res)
	}
}

func Test_GetChangesParseDataError(t *testing.T) {
	local := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity"},
		Data: map[string]any{
			"self":   "not an object",
			"schema": "string"},
	}

	_, err := GetChanges(map[string]DataStructure{"file": local}, []ListResponse{}, "DEV")

	if err == nil {
		t.Fatal("Expected an error for data that can't be parsed")
	}
}

func Test_GetChangesContentHashError(t *testing.T) {
	local := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity"},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-0-0",
			},
			"schema": math.Inf(1)},
	}
	remote := ListResponse{
		Hash:   "different",
		Vendor: "string",
		Name:   "string",
		Meta:   DataStructureMeta{Hidden: true, SchemaType: "entity"},
		Format: "string",
		Deployments: []Deployment{
			{
				Version:     "1-0-0",
				Env:         "DEV",
				ContentHash: "different",
			},
		},
	}

	_, err := GetChanges(map[string]DataStructure{"file": local}, []ListResponse{remote}, "DEV")

	if err == nil {
		t.Fatal("Expected an error for data that can't be hashed")
	}
}

func namedDS(name string, version string, schemaType string) DataStructure {
	return DataStructure{
		Meta: DataStructureMeta{SchemaType: schemaType},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "com.acme",
				"name":    name,
				"format":  "jsonschema",
				"version": version,
			},
			"schema": "string",
		},
	}
}

func recordingConsole(t *testing.T, calls *[]string) *ApiClient {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/validation-requests"):
			var ds DataStructure
			if err := json.Unmarshal(body, &ds); err != nil {
				t.Error(err)
			}
			data, err := ds.ParseData()
			if err != nil {
				t.Error(err)
			}
			*calls = append(*calls, "validate "+data.Self.Name)
			w.WriteHeader(http.StatusCreated)
			if data.Self.Name == "invalid" {
				_, _ = io.WriteString(w, `{"success":false,"message":"schema is broken"}`)
				return
			}
			_, _ = io.WriteString(w, `{"success":true}`)
		case strings.HasSuffix(r.URL.Path, "/deployment-requests"):
			var req struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(body, &req); err != nil {
				t.Error(err)
			}
			*calls = append(*calls, strings.TrimSpace("deploy "+req.Name+" "+r.URL.RawQuery))
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"success":true}`)
		case strings.HasSuffix(r.URL.Path, "/meta"):
			var meta struct {
				SchemaType string `json:"schemaType"`
			}
			if err := json.Unmarshal(body, &meta); err != nil {
				t.Error(err)
			}
			if meta.SchemaType != "" {
				*calls = append(*calls, "update meta "+meta.SchemaType)
			} else {
				*calls = append(*calls, "lock")
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	return &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: server.URL}
}

func Test_PerformChangesDev_ValidatesAndPublishesInOrder(t *testing.T) {
	var calls []string
	client := recordingConsole(t, &calls)
	changes := Changes{
		ToCreate:           []DSChangeContext{NewDSChangeContext(namedDS("created", "1-0-0", "entity"), "created.yml")},
		ToUpdateNewVersion: []DSChangeContext{NewDSChangeContextWithVersion(namedDS("bumped", "2-0-0", "entity"), "bumped.yml", "1-0-0")},
		ToUpdatePatch:      []DSChangeContext{NewDSChangeContextWithVersion(namedDS("patched", "1-0-0", "entity"), "patched.yml", "1-0-0")},
		ToUpdateMeta:       []DSChangeContext{NewDSChangeContext(namedDS("meta", "1-0-0", "event"), "meta.yml")},
	}

	if err := PerformChangesDev(context.Background(), client, changes, "cli"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{
		"validate created", "deploy created", "lock",
		"validate bumped", "deploy bumped", "lock",
		"validate patched", "deploy patched patch=true", "lock",
		"update meta event",
	}
	if !slices.Equal(calls, want) {
		t.Errorf("calls = %q, want %q", calls, want)
	}
}

func Test_PerformChangesDev_StopsAtInvalidDataStructure(t *testing.T) {
	var calls []string
	client := recordingConsole(t, &calls)
	changes := Changes{
		ToCreate:           []DSChangeContext{NewDSChangeContext(namedDS("created", "1-0-0", "entity"), "created.yml")},
		ToUpdateNewVersion: []DSChangeContext{NewDSChangeContextWithVersion(namedDS("invalid", "2-0-0", "entity"), "invalid.yml", "1-0-0")},
		ToUpdatePatch:      []DSChangeContext{NewDSChangeContextWithVersion(namedDS("patched", "1-0-0", "entity"), "patched.yml", "1-0-0")},
		ToUpdateMeta:       []DSChangeContext{NewDSChangeContext(namedDS("meta", "1-0-0", "event"), "meta.yml")},
	}

	err := PerformChangesDev(context.Background(), client, changes, "cli")
	if err == nil || !strings.Contains(err.Error(), "schema is broken") {
		t.Fatalf("expected the validation message as the error, got %v", err)
	}

	want := []string{"validate created", "deploy created", "lock", "validate invalid"}
	if !slices.Equal(calls, want) {
		t.Errorf("calls = %q, want %q", calls, want)
	}
}

func Test_PerformChangesDev_StopsAtInvalidPatch(t *testing.T) {
	var calls []string
	client := recordingConsole(t, &calls)
	changes := Changes{
		ToUpdatePatch: []DSChangeContext{NewDSChangeContextWithVersion(namedDS("invalid", "1-0-0", "entity"), "invalid.yml", "1-0-0")},
		ToUpdateMeta:  []DSChangeContext{NewDSChangeContext(namedDS("meta", "1-0-0", "event"), "meta.yml")},
	}

	if err := PerformChangesDev(context.Background(), client, changes, "cli"); err == nil {
		t.Fatal("expected an error for an invalid patch")
	}

	want := []string{"validate invalid"}
	if !slices.Equal(calls, want) {
		t.Errorf("calls = %q, want %q", calls, want)
	}
}

func captureChangeset(t *testing.T, changes Changes) (string, error) {
	t.Helper()
	var out bytes.Buffer
	handler := slog.NewTextHandler(&out, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && len(groups) == 0 {
				return slog.Attr{}
			}
			return a
		},
	})
	ctx := logging.ContextWithLogger(context.Background(), slog.New(handler))
	err := PrintChangeset(ctx, changes)
	return out.String(), err
}

func Test_PrintChangeset(t *testing.T) {
	changes := Changes{
		ToCreate:           []DSChangeContext{NewDSChangeContext(namedDS("created", "1-0-0", "entity"), "created.yml")},
		ToUpdateNewVersion: []DSChangeContext{NewDSChangeContextWithVersion(namedDS("bumped", "2-0-0", "entity"), "bumped.yml", "1-0-0")},
		ToUpdatePatch:      []DSChangeContext{NewDSChangeContextWithVersionAndHashes(namedDS("patched", "1-0-0", "entity"), "patched.yml", "1-0-0", "localhash", "remotehash")},
		ToUpdateMeta:       []DSChangeContext{NewDSChangeContext(namedDS("meta", "1-0-0", "event"), "meta.yml")},
	}

	got, err := captureChangeset(t, changes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := `level=INFO msg="will update metadata of" file=meta.yml vendor=com.acme name=meta
level=INFO msg="will create" file=created.yml vendor=com.acme name=created version=1-0-0
level=INFO msg="will update" file=bumped.yml local=2-0-0 remote=1-0-0
level=INFO msg="will patch" file=patched.yml vendor=com.acme name=patched version=1-0-0 local=localhash remote=remotehash
`
	if got != want {
		t.Errorf("output:\n%s\nwant:\n%s", got, want)
	}
}

func Test_PrintChangeset_ParseDataError(t *testing.T) {
	broken := []DSChangeContext{NewDSChangeContext(DataStructure{Data: map[string]any{"self": "not an object"}}, "broken.yml")}
	tests := map[string]Changes{
		"metadata update": {ToUpdateMeta: broken},
		"create":          {ToCreate: broken},
		"new version":     {ToUpdateNewVersion: broken},
		"patch":           {ToUpdatePatch: broken},
	}
	for name, changes := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := captureChangeset(t, changes)
			if err == nil {
				t.Error("expected an error for data that can't be parsed")
			}
			if got != "" {
				t.Errorf("expected no output, got %s", got)
			}
		})
	}
}

func Test_GetChangesFindsDeploymentAfterOtherEnv(t *testing.T) {
	local := namedDS("string", "1-0-0", "entity")
	remote := ListResponse{
		Hash:   "different",
		Vendor: "com.acme",
		Name:   "string",
		Meta:   DataStructureMeta{SchemaType: "entity"},
		Format: "jsonschema",
		Deployments: []Deployment{
			{Version: "1-0-0", Env: "PROD", ContentHash: "prod"},
			{Version: "1-0-0", Env: "DEV", ContentHash: "different"},
		},
	}

	res, err := GetChanges(map[string]DataStructure{"file": local}, []ListResponse{remote}, "DEV")
	if err != nil {
		t.Fatalf("Can't calcuate changes %s", err)
	}

	if len(res.ToCreate) != 0 || len(res.ToUpdateMeta) != 0 || len(res.ToUpdatePatch) != 1 || len(res.ToUpdateNewVersion) != 0 {
		t.Fatalf("Unexpected result, expecting one data structre to update patch, got %+v", res)
	}
}

func Test_PerformChangesDev_UpdatesEveryMeta(t *testing.T) {
	var calls []string
	client := recordingConsole(t, &calls)
	changes := Changes{
		ToUpdateMeta: []DSChangeContext{
			NewDSChangeContext(namedDS("first", "1-0-0", "event"), "first.yml"),
			NewDSChangeContext(namedDS("second", "1-0-0", "entity"), "second.yml"),
		},
	}

	if err := PerformChangesDev(context.Background(), client, changes, "cli"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"update meta event", "update meta entity"}
	if !slices.Equal(calls, want) {
		t.Errorf("calls = %q, want %q", calls, want)
	}
}

func Test_PerformChangesDev_StopsAtFailedMetaUpdate(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: server.URL}
	changes := Changes{
		ToUpdateMeta: []DSChangeContext{
			NewDSChangeContext(namedDS("first", "1-0-0", "event"), "first.yml"),
			NewDSChangeContext(namedDS("second", "1-0-0", "entity"), "second.yml"),
		},
	}

	if err := PerformChangesDev(context.Background(), client, changes, "cli"); err == nil {
		t.Fatal("expected an error for a failed metadata update")
	}
	if requests != 1 {
		t.Errorf("requests = %d, want 1", requests)
	}
}
