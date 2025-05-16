/**
 * Copyright (c) 2013-present Snowplow Analytics Ltd.
 * All rights reserved.
 * This software is made available by Snowplow Analytics, Ltd.,
 * under the terms of the Snowplow Limited Use License Agreement, Version 1.0
 * located at https://docs.snowplow.io/limited-use-license-1.0
 * BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
 * OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
 */

package model

type EventSpec struct {
	ResourceName               string
	ExcludedSourceApplications []map[string]string `yaml:"excludedSourceApplications,omitempty" json:"excludedSourceApplications,omitempty"`
	Name                       string
	Description                string
	Triggers                   []Trigger
	Event                      SchemaRef
	Entities                   EntitiesDef
}

type Trigger struct {
	Id          string   `yaml:"id,omitempty" json:"id,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	AppIds      []string `yaml:"appIds,omitempty" json:"appIds,omitempty"`
	Url         string   `yaml:"url,omitempty" json:"url,omitempty"`
	Image       *Ref     `yaml:"image,omitempty" json:"image,omitempty"`
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
	ResourceName string       `yaml:"-" json:"-"`
	Name         string       `yaml:"name" json:"name"`
	Description  string       `yaml:"description,omitempty" json:"description,omitempty"`
	Owner        string       `yaml:"owner,omitempty" json:"owner,omitempty"`
	AppIds       []string     `yaml:"appIds" json:"appIds"`
	Entities     *EntitiesDef `yaml:"entities" json:"entities"`
}

type EntitiesDef struct {
	Tracked  []SchemaRef `yaml:"tracked" json:"tracked"`
	Enriched []SchemaRef `yaml:"enriched" json:"enriched"`
}

type SchemaRef struct {
	Source         string         `yaml:"source,omitempty" json:"source,omitempty"`
	MinCardinality *int           `yaml:"minCardinality,omitempty" json:"minCardinality,omitempty"`
	MaxCardinality *int           `yaml:"maxCardinality,omitempty" json:"maxCardinality,omitempty"`
	Schema         map[string]any `yaml:"schema,omitempty" json:"schema,omitempty"`
}

type DataProductCanonicalData struct {
	ResourceName        string               `yaml:"-" json:"-"`
	Name                string               `yaml:"name" json:"name"`
	SourceApplications  []Ref                `yaml:"sourceApplications" json:"sourceApplications"`
	Domain              string               `yaml:"domain,omitempty" json:"domain,omitempty"`
	Owner               string               `yaml:"owner,omitempty" json:"owner,omitempty"`
	Description         string               `yaml:"description,omitempty" json:"description,omitempty"`
	EventSpecifications []EventSpecCanonical `yaml:"eventSpecifications" json:"eventSpecifications"`
}

type EventSpecCanonical struct {
	ResourceName               string      `yaml:"resourceName" json:"resourceName"`
	ExcludedSourceApplications []Ref       `yaml:"excludedSourceApplications,omitempty" json:"excludedSourceApplications,omitempty"`
	Name                       string      `yaml:"name" json:"name"`
	Description                string      `yaml:"description,omitempty" json:"description,omitempty"`
	Event                      SchemaRef   `yaml:"event,omitempty" json:"event,omitempty"`
	Entities                   EntitiesDef `yaml:"entities" json:"entities"`
	Triggers                   []Trigger   `yaml:"triggers,omitempty" json:"triggers,omitempty"`
}

type Ref struct {
	Ref string `yaml:"$ref,omitempty" json:"$ref,omitempty" mapstructure:"$ref"`
}

type Image struct {
	Ext  string
	Data []byte
}
