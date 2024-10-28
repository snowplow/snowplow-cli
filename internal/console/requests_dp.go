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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/snowplow-product/snowplow-cli/internal/util"
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

type CompatStatus = string

const (
	CompatCompatible   CompatStatus = "compatible"
	CompatUndecidable  CompatStatus = "undecidable"
	CompatIncompatible CompatStatus = "incompatible"
)

type CompatSource struct {
	Source     string
	Status     CompatStatus
	Properties map[string]string
}

type CompatResult struct {
	Status  string
	Sources []CompatSource
	Message string
}

type CompatCheckable struct {
	Source string         `json:"source"`
	Schema map[string]any `json:"schema"`
}

type CompatChecker = func(event CompatCheckable, entities []CompatCheckable) (*CompatResult, error)

func CompatCheck(cnx context.Context, client *ApiClient, event CompatCheckable, entities []CompatCheckable) (*CompatResult, error) {
	realArgs := map[string]any{
		"spec": map[string]any{
			"event": event,
			"entities": map[string]any{
				"tracked": entities,
			},
			// this uuid does not reference any existing event spec, I made it up
			"id":      "312d3987-4874-498d-af6c-162ce0da39d7",
			"name":    "cli-compat-check",
			"status":  "draft",
			"version": "0",
		},
	}

	body, err := json.Marshal(realArgs)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(cnx, "POST", fmt.Sprintf("%s/event-specs/v1/compatibility", client.BaseUrl), bytes.NewBuffer(body))
	auth := fmt.Sprintf("Bearer %s", client.Jwt)
	req.Header.Add("authorization", auth)
	req.Header.Add("X-SNOWPLOW-CLI", util.VersionInfo)

	if err != nil {
		return nil, err
	}
	resp, err := client.Http.Do(req)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	var cresp CompatResult
	err = json.Unmarshal(rbody, &cresp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response got status: %d with message: %s", resp.StatusCode, cresp.Message)
	}

	return &cresp, nil
}
