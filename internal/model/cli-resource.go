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

type CliResource[A any] struct {
	ApiVersion   string `json:"apiVersion"   yaml:"apiVersion"`
	ResourceType string `json:"resourceType" yaml:"resourceType"`
	ResourceName string `json:"resourceName" yaml:"resourceName"`
	Data         A      `json:"data"         yaml:"data"`
}
