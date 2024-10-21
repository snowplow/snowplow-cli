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
	ResourceName       string
	SourceApplications []map[string]string
}

type DataProductData struct {
	Name string
	SourceApplications  []map[string]string
	EventSpecifications []EventSpec
}

type DataProduct struct {
	ResourceType string
	ResourceName string
	ApiVersion   string
	Data         DataProductData
}

type SourceApp struct {
	ResourceType string
	ApiVersion   string
	ResourceName string
	Data SourceAppData
}

type SourceAppData struct {
	Name string
	Description string
	AppIds []string
	Entities *EntitiesDef
}

type EntitiesDef struct {
	Tracked []SchemaRef
	Enriched []SchemaRef
}

type SchemaRef struct {
	Source string
	MinCardinality *int
	MaxCardinality *int
	Schema map[string]any
}
