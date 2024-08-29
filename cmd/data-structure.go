/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

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
	SchemaType SchemaType        `yaml:"schemaType" json:"schemaType"`
	CustomData map[string]string `yaml:"customData" json:"customData"`
}

type DataStructure struct {
	Meta DataStructureMeta `yaml:"meta" json:"meta"`
	Data map[string]any    `yaml:"data" json:"data"`
}

func (ds DataStructure) getContentHash() (string, error) {
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
	Schema map[string]any    `mapstructure:"schema"`
	Other  map[string]any    `mapstructure:",remain"`
}
