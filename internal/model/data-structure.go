/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package model

import (
	"bytes"
	"crypto"
	"encoding/json"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
)

type SchemaType string

const (
	Event  SchemaType = "event"
	Entity SchemaType = "entity"
)

type DataStructureMeta struct {
	Hidden     bool              `yaml:"hidden" json:"hidden"`
	SchemaType SchemaType        `yaml:"schemaType" json:"schemaType" validate:"required,oneof=event entity"`
	CustomData map[string]string `yaml:"customData" json:"customData" validate:"required"`
}

type DataStructure struct {
	ApiVersion   string            `yaml:"apiVersion" json:"apiVersion" validate:"required,oneof=v1"`
	ResourceType string            `yaml:"resourceType" json:"resourceType" validate:"required,oneof=data-structure"`
	Meta         DataStructureMeta `yaml:"meta" json:"meta" validate:"required"`
	Data         map[string]any    `yaml:"data" json:"data" validate:"required"`
}

func (ds DataStructure) GetContentHash() (string, error) {
	byteBuffer := new(bytes.Buffer)
	e := json.NewEncoder(byteBuffer)
	e.SetEscapeHTML(false)
	err := e.Encode(ds.Data)
	if err != nil {
		return "", err
	}
	// Encode adds a line feed at the end, scala does not
	b := byteBuffer.Bytes()[:len(byteBuffer.Bytes())-1]
	hasher := crypto.SHA256.New()
	hasher.Write(b)
	hash := hasher.Sum(nil)
	// render bytes as base-16
	return fmt.Sprintf("%x", hash), nil
}

func (d DataStructure) ParseData() (DataStrucutreData, error) {
	var data DataStrucutreData
	err := mapstructure.Decode(d.Data, &data)
	return data, err
}

type DataStructureSelf struct {
	Vendor  string `mapstructure:"vendor" json:"vendor" validate:"required"`
	Name    string `mapstructure:"name" json:"name" validate:"required"`
	Format  string `mapstructure:"format" json:"format" validate:"required,oneof=jsonschema"`
	Version string `mapstructure:"version" json:"version" validate:"required"`
}

type DataStrucutreData struct {
	Self   DataStructureSelf `mapstructure:"self" json:"self" validate:"required"`
	Schema string            `mapstructure:"$schema" json:"$schema" validate:"required"`
	Other  map[string]any    `mapstructure:",remain"`
}

type DSChangeContext struct {
	DS                DataStructure
	FileName          string
	RemoteVersion     string
	LocalContentHash  string
	RemoteContentHash string
}

