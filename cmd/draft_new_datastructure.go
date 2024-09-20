package cmd

func createAndValidate(name string) DataStructure {
	return DataStructure{
		ApiVersion:   "v1",
		ResourceType: "data-structure",
		Meta:         DataStructureMeta{SchemaType: "event"},
		Data: map[string]any{
			"description": "Schema for an example event",
			"properties": map[string]any{
				"my_field": map[string]any{
					"type":        "string",
					"description": "my field",
					"maxLength":   4096,
				},
			},
			"additionalProperties": false,
			"type":                 "object",
			"required":             []string{"my_field"},
			"self": map[string]any{
				"name": name,
				"vendor":  "com.example",
				"format":  "jsonschema",
				"version": "1-0-0",
			},
			"$schema": "http://iglucentral.com/schemas/com.snowplowanalytics.self-desc/schema/jsonschema/1-0-0#",
		},
	}
}

func CreateNewDataStructureFile(name string, directory string, format string) error {
	ds := createAndValidate(name)
	err := writeSerializableToFile(ds, directory, name, format)

	return err
}
