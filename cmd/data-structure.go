/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import "github.com/go-viper/mapstructure/v2"

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

type DataStructure struct {
	Meta DataStructureMeta `yaml:"meta" json:"meta"`
	Data map[string]any    `yaml:"data" json:"data"`
}

func (d DataStructure) parseData() (DataStrucutreData, error) {
	var data DataStrucutreData
	err := mapstructure.Decode(d.Data, &data)
	return data, err
}

type DataStructureSelf struct {
	Vendor  string `mapstructure:"vendor"`
	Name    string `mapstructure:"name"`
	Format  string `mapstructure:"format"`
	Version string `mapstructure:"version"`
}

type DataStrucutreData struct {
	Self   DataStructureSelf `mapstructure:"self"`
	Schema string            `mapstructure:"schema"`
	Other  map[string]any    `mapstructure:",remain"`
}
