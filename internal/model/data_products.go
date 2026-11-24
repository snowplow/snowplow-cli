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
	"encoding/json"
)

type EventSpec struct {
	ResourceName               string
	ExcludedSourceApplications []map[string]string `json:"excludedSourceApplications,omitempty" yaml:"excludedSourceApplications,omitempty"`
	Name                       string
	Version                    *int
	Description                string
	Triggers                   []Trigger
	Event                      SchemaRef
	Entities                   EntitiesDef
}

type Trigger struct {
	Id          string   `json:"id,omitempty"          yaml:"id,omitempty"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	AppIds      []string `json:"appIds,omitempty"      yaml:"appIds,omitempty"`
	Url         string   `json:"url,omitempty"         yaml:"url,omitempty"`
	Image       *Ref     `json:"image,omitempty"       yaml:"image,omitempty"`
}

type DataProductData struct {
	ResourceName        string
	Name                string
	SourceApplications  []map[string]string
	Domain              string
	Owner               string
	Description         string
	EventSpecifications []EventSpec
}

type DataProduct struct {
	ApiVersion   string
	ResourceType string
	ResourceName string
	Data         DataProductData
}

type SourceApp struct {
	ApiVersion   string
	ResourceType string
	ResourceName string
	Data         SourceAppData
}

type SourceAppData struct {
	ResourceName string       `json:"-"                     yaml:"-"`
	Name         string       `json:"name"                  yaml:"name"`
	Description  string       `json:"description,omitempty" yaml:"description,omitempty"`
	Owner        string       `json:"owner,omitempty"       yaml:"owner,omitempty"`
	AppIds       []string     `json:"appIds"                yaml:"appIds"`
	Entities     *EntitiesDef `json:"entities,omitempty"    yaml:"entities"`
}

type EntitiesDef struct {
	Tracked  []SchemaRef `json:"tracked"  yaml:"tracked"`
	Enriched []SchemaRef `json:"enriched" yaml:"enriched"`
}

func (e EntitiesDef) MarshalJSON() ([]byte, error) {
	type Alias EntitiesDef

	if e.Tracked == nil {
		e.Tracked = []SchemaRef{}
	}
	if e.Enriched == nil {
		e.Enriched = []SchemaRef{}
	}

	return json.Marshal((Alias)(e))
}

type SchemaRef struct {
	Source         string         `json:"source,omitempty"         yaml:"source,omitempty"`
	MinCardinality *int           `json:"minCardinality,omitempty" yaml:"minCardinality,omitempty"`
	MaxCardinality *int           `json:"maxCardinality,omitempty" yaml:"maxCardinality,omitempty"`
	Schema         map[string]any `json:"schema,omitempty"         yaml:"schema,omitempty"`
}

type DataProductCanonicalData struct {
	ResourceName        string               `json:"-"                             yaml:"-"`
	Name                string               `json:"name"                          yaml:"name"`
	SourceApplications  []Ref                `json:"sourceApplications,omitempty"  yaml:"sourceApplications"`
	Domain              string               `json:"domain,omitempty"              yaml:"domain,omitempty"`
	Owner               string               `json:"owner,omitempty"               yaml:"owner,omitempty"`
	Description         string               `json:"description,omitempty"         yaml:"description,omitempty"`
	EventSpecifications []EventSpecCanonical `json:"eventSpecifications,omitempty" yaml:"eventSpecifications"`
}

type EventSpecCanonical struct {
	ResourceName               string      `json:"resourceName"                         yaml:"resourceName"`
	ExcludedSourceApplications []Ref       `json:"excludedSourceApplications,omitempty" yaml:"excludedSourceApplications,omitempty"`
	Name                       string      `json:"name"                                 yaml:"name"`
	Version                    *int        `json:"version,omitempty"                    yaml:"version,omitempty"`
	Description                string      `json:"description,omitempty"                yaml:"description,omitempty"`
	Event                      *SchemaRef  `json:"event,omitempty"                      yaml:"event,omitempty"`
	Entities                   EntitiesDef `json:"entities"                             yaml:"entities"`
	Triggers                   []Trigger   `json:"triggers,omitempty"                   yaml:"triggers,omitempty"`
}

func (e EventSpecCanonical) MarshalJSON() ([]byte, error) {
	type Alias EventSpecCanonical

	temp := Alias(e)
	if temp.Event != nil && temp.Event.isEmpty() {
		temp.Event = nil
	}

	return json.Marshal(temp)
}

func (e EventSpecCanonical) MarshalYAML() (any, error) {
	type Alias EventSpecCanonical

	temp := Alias(e)
	if temp.Event != nil && temp.Event.isEmpty() {
		temp.Event = nil
	}

	return temp, nil
}

func (s *SchemaRef) isEmpty() bool {
	if s == nil {
		return true
	}
	return s.Source == "" &&
		s.MinCardinality == nil &&
		s.MaxCardinality == nil &&
		len(s.Schema) == 0
}

type Ref struct {
	Ref string `json:"$ref,omitempty" mapstructure:"$ref" yaml:"$ref,omitempty"`
}

type Image struct {
	Ext  string
	Data []byte
}

type EventSpecVersionMismatch struct {
	FileName              string
	Index                 int
	Name                  string
	Declared              int
	Expected              int
	IsNew                 bool
	Changed               []string
	EntityCommentsRemoved bool
}

type EventSpecMissingVersion struct {
	FileName string
	Index    int
	Name     string
	Expected *int
	Reason   string
}
