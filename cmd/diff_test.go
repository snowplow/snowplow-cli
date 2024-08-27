package cmd

import (
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
		Meta: DataStructureMeta{Hidden: true, SchemaType: "entity" },
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

}
