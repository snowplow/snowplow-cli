package cmd

import (
	"reflect"
	"testing"
)

func Test_ShowsDifferenceInMetadata(t *testing.T) {
	remoteDs := DataStructure{
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
	localDs := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "event"},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-0-1",
			},
			"schema": "string"},
	}

	diff, err := DiffDs([]DataStructure{localDs}, []DataStructure{remoteDs})

	if err != nil {
		t.Fatalf("Can't calcuate diff %s", err)
	}

	if len(diff) != 1 {
		t.Fatalf("Not expected amount of changes, expected: 1, got: %d", len(diff))
	}

	if !reflect.DeepEqual(diff[0].Diff[0].Path, []string{"Meta", "SchemaType"}) {
		t.Fatalf("Not expected change path, %v", diff[0].Diff[0].Path)
	}
}

func Test_ShowsDifferenceInMetadataKnownField(t *testing.T) {
	remoteDs := DataStructure{
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
	localDs := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity", CustomData: map[string]string{
			"hidden": "true",
		}},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-0-1",
			},
			"schema": "string"},
	}

	diff, err := DiffDs([]DataStructure{localDs}, []DataStructure{remoteDs})

	if err != nil {
		t.Fatalf("Can't calcuate diff %s", err)
	}

	if len(diff) != 1 {
		t.Fatalf("Not expected amount of changes, expected: 1, got: %d", len(diff))
	}

	if !reflect.DeepEqual(diff[0].Diff[0].Path, []string{"Meta", "CustomData", "hidden"}) {
		t.Fatalf("Not expected change path, %v", diff[0].Diff[0].Path)
	}

}

func Test_ShowDifferenceInMetadataUnknownField(t *testing.T) {
	remoteDs := DataStructure{
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
	localDs := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity", CustomData: map[string]string{
			"foo": "bar",
		}},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-0-1",
			},
			"schema": "string"},
	}

	diff, err := DiffDs([]DataStructure{localDs}, []DataStructure{remoteDs})

	if err != nil {
		t.Fatalf("Can't calcuate diff %s", err)
	}

	if len(diff) != 1 {
		t.Fatalf("Not expected amount of changes, expected: 1, got: %d", len(diff))
	}

	if !reflect.DeepEqual(diff[0].Diff[0].Path, []string{"Meta", "CustomData", "foo"}) {
		t.Fatalf("Not expected change path, %v", diff[0].Diff[0].Path)
	}

}

func Test_ShowDifferenceInSchemaSelf(t *testing.T) {
	remoteDs := DataStructure{
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
	localDs := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity"},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-0-2",
			},
			"schema": "string"},
	}

	diff, err := DiffDs([]DataStructure{localDs}, []DataStructure{remoteDs})

	if err != nil {
		t.Fatalf("Can't calcuate diff %s", err)
	}

	if len(diff) != 1 {
		t.Fatalf("Not expected amount of changes, expected: 1, got: %d", len(diff))
	}

	if !reflect.DeepEqual(diff[0].Diff[0].Path, []string{"Data", "self", "version"}) {
		t.Fatalf("Not expected change path, %v", diff[0].Diff[0].Path)
	}

}

func Test_ShowDifferenceInSchema(t *testing.T) {
	remoteDs := DataStructure{
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
	localDs := DataStructure{
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity"},
		Data: map[string]any{
			"self": map[string]any{
				"vendor":  "string",
				"name":    "string",
				"format":  "string",
				"version": "1-0-1",
			},
			"schema": `string2`},
	}

	diff, err := DiffDs([]DataStructure{localDs}, []DataStructure{remoteDs})

	if err != nil {
		t.Fatalf("Can't calcuate diff %s", err)
	}

	if len(diff) != 1 {
		t.Fatalf("Not expected amount of changes, expected: 1, got: %d", len(diff))
	}

	if !reflect.DeepEqual(diff[0].Diff[0].Path, []string{"Data", "schema"}) {
		t.Fatalf("Not expected change path, %v", diff[0].Diff[0].Path)
	}

}

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

	res, err := getChanges(map[string]DataStructure{"file": local}, []ListResponse{}, "DEV")

	if err != nil {
		t.Fatalf("Can't calcuate changes %s", err)
	}

	if len(res.toCreate) != 1 || len(res.toUpdateMeta) != 0 || len(res.toUpdatePatch) != 0 || len(res.toUpdateNewVersion) != 0 {
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

	res, err := getChanges(map[string]DataStructure{"file":local}, []ListResponse{remote}, "DEV")

	if err != nil {
		t.Fatalf("Can't calcuate changes %s", err)
	}

	if len(res.toCreate) != 0 || len(res.toUpdateMeta) != 1 || len(res.toUpdatePatch) != 0 || len(res.toUpdateNewVersion) != 1 {
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

	res, err := getChanges(map[string]DataStructure{"file":local}, []ListResponse{remote}, "DEV")

	if err != nil {
		t.Fatalf("Can't calcuate changes %s", err)
	}

	if len(res.toCreate) != 0 || len(res.toUpdateMeta) != 0 || len(res.toUpdatePatch) != 1 || len(res.toUpdateNewVersion) != 0 {
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

	res, err := getChanges(map[string]DataStructure{"file":local}, []ListResponse{remote}, "DEV")

	if err != nil {
		t.Fatalf("Can't calcuate changes %s", err)
	}

	if len(res.toCreate) != 0 || len(res.toUpdateMeta) != 0 || len(res.toUpdatePatch) != 0 || len(res.toUpdateNewVersion) != 0 {
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

	res, err := getChanges(map[string]DataStructure{"file":local}, []ListResponse{remote}, "PROD")

	if err != nil {
		t.Fatalf("Can't calcuate changes %s", err)
	}

	if len(res.toCreate) != 0 || len(res.toUpdateMeta) != 0 || len(res.toUpdatePatch) != 0 || len(res.toUpdateNewVersion) != 1 {
		t.Fatalf("Unexpected result, expecting one data structre to update, got %+v", res)
	}

}
