/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

type SchemaType string

const (
	Event  SchemaType = "event"
	Entity SchemaType = "entity"
)

type DataStructure struct {
	Meta struct {
		Hidden     bool              `yaml:"hidden" json:"hidden"`
		SchemaType SchemaType        `yaml:"schemaType" json:"schemaType"`
		CustomData map[string]string `yaml:"customData" json:"customData"`
	} `yaml:"meta" json:"meta"`
	Data struct {
		Self struct {
			Vendor  string `yaml:"vendor" json:"vendor"`
			Name    string `yaml:"name" json:"name"`
			Format  string `yaml:"format" json:"format"`
			Version struct {
				Model    int `yaml:"model" json:"model"`
				Revision int `yaml:"revision" json:"revision"`
				Addition int `yaml:"addition" json:"addition"`
			} `yaml:"version" json:"version"`
		} `yaml:"self" json:"self"`
		Schema string `yaml:"schema" json:"schema"`
	} `yaml:"data" json:"data"`
}
