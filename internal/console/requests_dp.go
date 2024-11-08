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
	Id                   string               `json:"id"`
	Name                 string               `json:"name"`
	SourceApplicationIds []string             `json:"sourceApplications"`
	Domain               string               `json:"domain"`
	Owner                string               `json:"owner"`
	Description          string               `json:"description"`
	EventSpecifications  []EventSpecReference `json:"trackingScenarios"`
}

type EventSpecReference struct {
	Id string `json:"id"`
}

type RemoteEventSpec struct {
	Id                   string    `json:"id"`
	SourceApplicationIds []string  `json:"sourceApplications"`
	Name                 string    `json:"name"`
	Triggers             []Trigger `json:"triggers"`
	Event                Event     `json:"event"`
	Entities             Entities  `json:"entities"`
}

type Event struct {
	Source string         `json:"source"`
	Schema map[string]any `json:"schema"`
}

type Trigger struct {
	Description string `json:"description"`
}

type dataProductsResponse struct {
	Data     []RemoteDataProduct `json:"data"`
	Includes includes            `json:"includes"`
}

type includes struct {
	TrackingScenarios  []RemoteEventSpec         `json:"trackingScenarios"`
	SourceApplications []RemoteSourceApplication `json:"sourceApplications"`
}

type RemoteSourceApplication struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Owner       string   `json:"owner"`
	AppIds      []string `json:"appIds"`
	Entities    Entities `json:"entities"`
}

type Entities struct {
	Tracked  []Entity `json:"tracked"`
	Enriched []Entity `json:"enriched"`
}

type Entity struct {
	Source         string `json:"source"`
	MinCardinality *int   `json:"minCardinality"`
	MaxCardinality *int   `json:"maxCardinality"`
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

	var dpResponse dataProductsResponse
	err = json.Unmarshal(rbody, &dpResponse)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not expected response code %d", resp.StatusCode)
	}

	res := DataProductsAndRelatedResources{
		dpResponse.Data,
		dpResponse.Includes.TrackingScenarios,
		dpResponse.Includes.SourceApplications,
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
