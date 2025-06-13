/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package validation

import _ "embed"

//go:embed schema/data-structure.json
var dataStructureSchema string

//go:embed schema/data-product.json
var dataProductSchema string

//go:embed schema/source-application.json
var sourceApplicationSchema string

// GetEmbeddedSchemas returns all embedded schema files as a map
func GetEmbeddedSchemas() map[string]string {
	return map[string]string{
		"data-structure":     dataStructureSchema,
		"data-product":       dataProductSchema,
		"source-application": sourceApplicationSchema,
	}
}

// GetEmbeddedSchema returns a specific embedded schema file
func GetEmbeddedSchema(schemaType string) (string, bool) {
	schemas := GetEmbeddedSchemas()
	schema, exists := schemas[schemaType]
	return schema, exists
}
