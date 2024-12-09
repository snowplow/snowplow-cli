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
	Event                      SchemaRef
	Entities                   EntitiesDef
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
	ResourceName string `yaml:"-" json:"-"`
	Name         string
	Description  string   `yaml:"description,omitempty" json:"description,omitempty"`
	Owner        string   `yaml:"owner,omitempty" json:"owner,omitempty"`
	AppIds       []string `yaml:"appIds" json:"appIds"`
	Entities     *EntitiesDef
}

type EntitiesDef struct {
	Tracked  []SchemaRef
	Enriched []SchemaRef
}

type SchemaRef struct {
	Source         string         `yaml:"source,omitempty" json:"source,omitempty"`
	MinCardinality *int           `yaml:"minCardinality,omitempty" json:"minCardinality,omitempty"`
	MaxCardinality *int           `yaml:"maxCardinality,omitempty" json:"maxCardinality,omitempty"`
	Schema         map[string]any `yaml:"schema,omitempty" json:"schema,omitempty"`
}

type DataProductCanonicalData struct {
	ResourceName        string `yaml:"-" json:"-"`
	Name                string
	SourceApplications  []Ref                `yaml:"sourceApplications" json:"sourceApplications"`
	Domain              string               `yaml:"domain,omitempty" json:"domain,omitempty"`
	Owner               string               `yaml:"owner,omitempty" json:"owner,omitempty"`
	Description         string               `yaml:"description,omitempty" json:"description,omitempty"`
	EventSpecifications []EventSpecCanonical `yaml:"eventSpecifications" json:"eventSpecifications"`
}

type EventSpecCanonical struct {
	ResourceName               string `yaml:"resourceName" json:"resourceName"`
	ExcludedSourceApplications []Ref  `yaml:"excludedSourceApplications,omitempty" json:"excludedSourceApplications,omitempty"`
	Name                       string
	Event                      SchemaRef `yaml:"event,omitempty" json:"event,omitempty"`
	Entities                   EntitiesDef
}

type Ref struct {
	Ref string `yaml:"$ref" json:"$ref"`
}
