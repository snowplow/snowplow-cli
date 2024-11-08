/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
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

type DataStructureMeta struct {
	Hidden     bool              `yaml:"hidden" json:"hidden"`
	SchemaType string            `yaml:"schemaType" json:"schemaType" validate:"required,oneof=event entity"`
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

func (d DataStructure) ParseData() (DataStructureData, error) {
	var data DataStructureData
	err := mapstructure.Decode(d.Data, &data)
	return data, err
}

type DataStructureSelf struct {
	Vendor  string `mapstructure:"vendor" json:"vendor" validate:"required"`
	Name    string `mapstructure:"name" json:"name" validate:"required"`
	Format  string `mapstructure:"format" json:"format" validate:"required,oneof=jsonschema"`
	Version string `mapstructure:"version" json:"version" validate:"required"`
}

type DataStructureData struct {
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
