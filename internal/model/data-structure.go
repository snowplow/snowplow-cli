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
	"encoding/hex"
	"encoding/json"

	"github.com/go-viper/mapstructure/v2"
)

type SchemaType string

type DataStructureMeta struct {
	Hidden     bool              `json:"hidden"     yaml:"hidden"`
	SchemaType string            `json:"schemaType" validate:"required,oneof=event entity" yaml:"schemaType"`
	CustomData map[string]string `json:"customData" validate:"required"                    yaml:"customData"`
}

type DataStructure struct {
	ApiVersion   string            `json:"apiVersion"   validate:"required,oneof=v1"             yaml:"apiVersion"`
	ResourceType string            `json:"resourceType" validate:"required,oneof=data-structure" yaml:"resourceType"`
	Meta         DataStructureMeta `json:"meta"         validate:"required"                      yaml:"meta"`
	Data         map[string]any    `json:"data"         validate:"required"                      yaml:"data"`
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
	return hex.EncodeToString(hash), nil
}

func (d DataStructure) ParseData() (DataStructureData, error) {
	var data DataStructureData
	err := mapstructure.Decode(d.Data, &data)
	return data, err
}

type DataStructureSelf struct {
	Vendor  string `json:"vendor"  mapstructure:"vendor"  validate:"required"`
	Name    string `json:"name"    mapstructure:"name"    validate:"required"`
	Format  string `json:"format"  mapstructure:"format"  validate:"required,oneof=jsonschema"`
	Version string `json:"version" mapstructure:"version" validate:"required"`
}

type DataStructureData struct {
	Self   DataStructureSelf `json:"self"            mapstructure:"self"    validate:"required"`
	Schema string            `json:"$schema"         mapstructure:"$schema" validate:"required"`
	Other  map[string]any    `mapstructure:",remain"`
}

type DSChangeContext struct {
	DS                DataStructure
	FileName          string
	RemoteVersion     string
	LocalContentHash  string
	RemoteContentHash string
}
