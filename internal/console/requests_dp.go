/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package console

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type DataProductsAndRelatedResources struct {
	DataProducts      []RemoteDataProduct
	TrackingScenarios []RemoteEventSpec
	SourceApplication []RemoteSourceApplication
}

type RemoteDataProduct struct {
	ResourceName        string               `yaml:"resourceName" json:"id" validate:"required"`
	Name                string               `yaml:"name" json:"name" validate:"required"`
	SourceApplications  []string             `yaml:"sourceApplications" json:"sourceApplications"`
	Domain              string               `yaml:"domain" json:"domain"`
	Owner               string               `yaml:"owner" json:"owner"`
	Description         string               `yaml:"description" json:"description"`
	EventSpecifications []eventSpecReference `yaml:"eventSpecifications" json:"trackingScenarios"`
}

type eventSpecReference struct {
	Id string `yaml:"id" json:"id" validate:"required"`
}

type RemoteEventSpec struct {
	ResourceName       string    `yaml:"resourceName" json:"id"`
	SourceApplications []string  `yaml:"sourceApplications" json:"sourceApplications"`
	Name               string    `yaml:"name" json:"name"`
	Triggers           []trigger `yaml:"triggers" json:"triggers"`
	Event              event     `yaml:"event" json:"event"`
	Entities           entities  `yaml:"entities" json:"entities"`
}

type event struct {
	Source string         `yaml:"source" json:"source"`
	Schema map[string]any `yaml:"schema" json:"schema"`
}

type trigger struct {
	Description string `yaml:"description" json:"description"`
}

type dataProductsResponse struct {
	Data     []RemoteDataProduct `yaml:"data" json:"data"`
	Includes includes            `yaml:"includes" json:"includes"`
}

type includes struct {
	TrackingScenarios  []RemoteEventSpec         `yaml:"trackingScenarios" json:"trackingScenarios"`
	SourceApplications []RemoteSourceApplication `yaml:"sourceApplications" json:"sourceApplications"`
}

type RemoteSourceApplication struct {
	ResourceName string   `yaml:"id" json:"id" validate:"required"`
	Name         string   `yaml:"name" json:"name" validate:"required"`
	Description  string   `yaml:"description" json:"description"`
	Owner        string   `yaml:"owner" json:"owner"`
	AppIds       []string `yaml:"appIds" json:"appIds"`
	Entities     entities `yaml:"entities" json:"entities"`
}

type entities struct {
	Tracked  []entity `yaml:"tracked" json:"tracked"`
	Enriched []entity `yaml:"enriched" json:"enriched"`
}

type entity struct {
	Source         string `yaml:"source" json:"source" validate:"required"`
	MinCardinality int    `yaml:"minCardinality" json:"minCardinality"`
	MaxCardinality int    `yaml:"maxCardinality" json:"maxCardinality"`
	Schema         map[string]any
}

func GetDataProductsAndRelatedResources(cnx context.Context, client *ApiClient) (*DataProductsAndRelatedResources, error) {

	resp, err := ConsoleRequest("GET", fmt.Sprintf("%s/data-products/v1", client.BaseUrl), client, cnx)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	var sourceApps dataProductsResponse
	err = json.Unmarshal(rbody, &sourceApps)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not expected response code %d", resp.StatusCode)
	}

	res := DataProductsAndRelatedResources{
		sourceApps.Data,
		sourceApps.Includes.TrackingScenarios,
		sourceApps.Includes.SourceApplications,
	}
	return &res, nil
}
