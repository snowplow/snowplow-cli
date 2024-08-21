/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

type SchemaType string

const (
	Event  SchemaType = "event"
	Entity SchemaType = "entity"
)

type DataStructureMeta struct {
	Hidden     bool              `yaml:"hidden" json:"hidden"`
	SchemaType SchemaType        `yaml:"schemaType" json:"schemaType"`
	CustomData map[string]string `yaml:"customData" json:"customData"`
}

type DataStructureVersion struct {
	Model    int `yaml:"model" json:"model"`
	Revision int `yaml:"revision" json:"revision"`
	Addition int `yaml:"addition" json:"addition"`
}

type DataStructureSelf struct {
	Vendor  string               `yaml:"vendor" json:"vendor"`
	Name    string               `yaml:"name" json:"name"`
	Format  string               `yaml:"format" json:"format"`
	Version DataStructureVersion `yaml:"version" json:"version"`
}

type DataStrucutreData struct {
	Self   DataStructureSelf `yaml:"self" json:"self"`
	Schema string            `yaml:"schema" json:"schema"`
}

type DataStructure struct {
	Meta DataStructureMeta `yaml:"meta" json:"meta"`
	Data DataStrucutreData `yaml:"data" json:"data"`
}
