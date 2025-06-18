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
	"errors"
	"fmt"
	"io"
	"net/http"

	kjson "k8s.io/apimachinery/pkg/util/json"

	"github.com/snowplow/snowplow-cli/internal/util"
)

type DataProductsAndRelatedResources struct {
	DataProducts      []RemoteDataProduct       `json:"dataProducts" yaml:"dataProducts"`
	EventSpecs        []RemoteEventSpec         `json:"eventSpecs" yaml:"eventSpecs"`
	SourceApplication []RemoteSourceApplication `json:"sourceApplication" yaml:"sourceApplication"`
}

type RemoteDataProduct struct {
	Id                   string               `json:"id"`
	Name                 string               `json:"name"`
	Status               string               `json:"status"`
	SourceApplicationIds []string             `json:"sourceApplications"`
	Domain               string               `json:"domain,omitempty"`
	Owner                string               `json:"owner,omitempty"`
	Description          string               `json:"description,omitempty"`
	EventSpecs           []EventSpecReference `json:"eventSpecs"`
	LockStatus           string               `json:"lockStatus,omitempty"`
	ManagedFrom          string               `json:"managedFrom,omitempty"`
}

type EventSpecReference struct {
	Id string `json:"id"`
}

type RemoteTrigger struct {
	Id          string            `json:"id,omitempty"`
	Description string            `json:"description"`
	AppIds      []string          `json:"appIds,omitempty"`
	Url         string            `json:"url,omitempty"`
	VariantUrls map[string]string `json:"variantUrls,omitempty"`
}

type RemoteEventSpec struct {
	Id                   string          `json:"id"`
	SourceApplicationIds []string        `json:"sourceApplications"`
	Name                 string          `json:"name"`
	Description          string          `json:"description"`
	Triggers             []RemoteTrigger `json:"triggers,omitempty"`
	Status               string          `json:"status"`
	Version              int             `json:"version"`
	Event                *EventWrapper   `json:"event,omitempty"`
	Entities             Entities        `json:"entities"`
	DataProductId        string          `json:"dataProductId"`
	LockStatus           string          `json:"lockStatus,omitempty"`
	ManagedFrom          string          `json:"managedFrom,omitempty"`
}

type Event struct {
	Source string         `json:"source,omitempty"`
	Schema map[string]any `json:"schema,omitempty"`
}

type EventWrapper struct {
	Event
}

func (ew EventWrapper) MarshalJSON() ([]byte, error) {
	if ew.Source == "" && (len(ew.Schema) == 0) {
		return []byte("null"), nil
	} else {
		return json.Marshal(ew.Event)
	}
}

type dataProductsResponse struct {
	Data     []RemoteDataProduct `json:"data"`
	Includes includes            `json:"includes"`
}

type includes struct {
	EventSpecs []RemoteEventSpec `json:"eventSpecs"`
}

type RemoteSourceApplication struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Owner       string   `json:"owner,omitempty"`
	AppIds      []string `json:"appIds"`
	Entities    Entities `json:"entities"`
	LockStatus  string   `json:"lockStatus,omitempty"`
	ManagedFrom string   `json:"managedFrom,omitempty"`
}

type Entities struct {
	Tracked  []Entity `json:"tracked"`
	Enriched []Entity `json:"enriched"`
}

type Entity struct {
	Source         string         `json:"source"`
	MinCardinality *int           `json:"minCardinality"`
	MaxCardinality *int           `json:"maxCardinality"`
	Schema         map[string]any `json:"schema,omitempty"`
}

type remoteEventSpecPost struct {
	Spec    RemoteEventSpec `json:"spec"`
	Message string          `json:"message"`
}

type esData struct {
	Data []RemoteEventSpec `json:"data"`
}

type saData struct {
	Data []RemoteSourceApplication `json:"data"`
}

func GetDataProductsAndRelatedResources(cnx context.Context, client *ApiClient) (*DataProductsAndRelatedResources, error) {

	resp, err := DoConsoleRequest("GET", fmt.Sprintf("%s/data-products/v2", client.BaseUrl), client, cnx, nil)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	defer util.LoggingCloser(cnx, resp.Body)
	if err != nil {
		return nil, err
	}

	var dpResponse dataProductsResponse
	err = kjson.Unmarshal(rbody, &dpResponse)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not expected response code %d", resp.StatusCode)
	}

	saResp, err := DoConsoleRequest("GET", fmt.Sprintf("%s/source-apps/v1", client.BaseUrl), client, cnx, nil)
	if err != nil {
		return nil, err
	}
	sarbody, err := io.ReadAll(saResp.Body)
	defer util.LoggingCloser(cnx, resp.Body)
	if err != nil {
		return nil, err
	}

	var saResponse saData
	err = kjson.Unmarshal(sarbody, &saResponse)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not expected response code %d", resp.StatusCode)
	}

	res := DataProductsAndRelatedResources{
		dpResponse.Data,
		dpResponse.Includes.EventSpecs,
		saResponse.Data,
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
	Source     string            `json:"source" yaml:"source"`
	Status     CompatStatus      `json:"status" yaml:"status"`
	Properties map[string]string `json:"properties" yaml:"properties"`
}

type CompatResult struct {
	Status  string         `json:"status" yaml:"status"`
	Sources []CompatSource `json:"sources" yaml:"sources"`
	Message string         `json:"message" yaml:"message"`
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
	resp, err := DoConsoleRequest("POST", fmt.Sprintf("%s/event-specs/v1/compatibility", client.BaseUrl), client, cnx, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	defer util.LoggingCloser(cnx, resp.Body)
	if err != nil {
		return nil, err
	}

	var cresp CompatResult
	err = kjson.Unmarshal(rbody, &cresp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response got status: %d with message: %s", resp.StatusCode, cresp.Message)
	}

	return &cresp, nil
}

func CreateSourceApp(cnx context.Context, client *ApiClient, sa RemoteSourceApplication) error {
	body, err := json.Marshal(sa)
	if err != nil {
		return err
	}
	resp, err := DoConsoleRequest("POST", fmt.Sprintf("%s/source-apps/v1", client.BaseUrl), client, cnx, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		rbody, err := io.ReadAll(resp.Body)
		defer util.LoggingCloser(cnx, resp.Body)
		if err != nil {
			return err
		}

		var dresp msgResponse
		err = kjson.Unmarshal(rbody, &dresp)
		if err != nil {
			return errors.Join(err, errors.New("bad response with no message"))
		}

		return fmt.Errorf("bad response: %s", dresp.Message)

	}
	return nil
}

func UpdateSourceApp(cnx context.Context, client *ApiClient, sa RemoteSourceApplication) error {
	body, err := json.Marshal(sa)
	if err != nil {
		return err
	}
	resp, err := DoConsoleRequest("PUT", fmt.Sprintf("%s/source-apps/v1/%s", client.BaseUrl, sa.Id), client, cnx, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		rbody, err := io.ReadAll(resp.Body)
		defer util.LoggingCloser(cnx, resp.Body)
		if err != nil {
			return err
		}

		var dresp msgResponse
		err = kjson.Unmarshal(rbody, &dresp)
		if err != nil {
			return errors.Join(err, errors.New("bad response with no message"))
		}

		return fmt.Errorf("bad response: %s", dresp.Message)

	}
	return nil
}

func DeleteSourceApp(cnx context.Context, client *ApiClient, sa RemoteSourceApplication) error {
	resp, err := DoConsoleRequest("DELETE", fmt.Sprintf("%s/source-apps/v1/%s", client.BaseUrl, sa.Id), client, cnx, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		rbody, err := io.ReadAll(resp.Body)
		defer util.LoggingCloser(cnx, resp.Body)
		if err != nil {
			return err
		}

		var dresp msgResponse
		err = kjson.Unmarshal(rbody, &dresp)
		if err != nil {
			return errors.Join(err, errors.New("bad response with no message"))
		}

		return fmt.Errorf("bad response: %s", dresp.Message)

	}
	return nil
}

func CreateDataProduct(cnx context.Context, client *ApiClient, dp RemoteDataProduct) error {
	body, err := json.Marshal(dp)
	if err != nil {
		return err
	}
	resp, err := DoConsoleRequest("POST", fmt.Sprintf("%s/data-products/v2", client.BaseUrl), client, cnx, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		rbody, err := io.ReadAll(resp.Body)
		defer util.LoggingCloser(cnx, resp.Body)
		if err != nil {
			return err
		}

		var dresp msgResponse
		err = kjson.Unmarshal(rbody, &dresp)
		if err != nil {
			return errors.Join(err, errors.New("bad response with no message"))
		}

		return fmt.Errorf("bad response: %s", dresp.Message)

	}
	return nil
}

func UpdateDataProduct(cnx context.Context, client *ApiClient, dp RemoteDataProduct) error {
	dp.Status = "draft"
	body, err := json.Marshal(dp)
	if err != nil {
		return err
	}
	resp, err := DoConsoleRequest("PUT", fmt.Sprintf("%s/data-products/v2/%s", client.BaseUrl, dp.Id), client, cnx, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		rbody, err := io.ReadAll(resp.Body)
		defer util.LoggingCloser(cnx, resp.Body)
		if err != nil {
			return err
		}

		var dresp msgResponse
		err = kjson.Unmarshal(rbody, &dresp)
		if err != nil {
			return errors.Join(err, errors.New("bad response with no message"))
		}

		return fmt.Errorf("bad response: %s", dresp)

	}
	return nil
}

func DeleteDataProduct(cnx context.Context, client *ApiClient, dp RemoteDataProduct) error {
	resp, err := DoConsoleRequest("DELETE", fmt.Sprintf("%s/data-products/v2/%s", client.BaseUrl, dp.Id), client, cnx, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		rbody, err := io.ReadAll(resp.Body)
		defer util.LoggingCloser(cnx, resp.Body)
		if err != nil {
			return err
		}

		var dresp msgResponse
		err = kjson.Unmarshal(rbody, &dresp)
		if err != nil {
			return errors.Join(err, errors.New("bad response with no message"))
		}

		return fmt.Errorf("bad response: %s", dresp)

	}
	return nil
}

func CreateEventSpec(cnx context.Context, client *ApiClient, es RemoteEventSpec) error {

	esForm := remoteEventSpecPost{Spec: es, Message: ""}

	body, err := json.Marshal(esForm)
	if err != nil {
		return err
	}
	resp, err := DoConsoleRequest("POST", fmt.Sprintf("%s/event-specs/v1", client.BaseUrl), client, cnx, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusCreated {
		rbody, err := io.ReadAll(resp.Body)
		defer util.LoggingCloser(cnx, resp.Body)
		if err != nil {
			return err
		}

		var dresp msgResponse
		err = kjson.Unmarshal(rbody, &dresp)
		if err != nil {
			return errors.Join(err, errors.New("bad response with no message"))
		}

		return fmt.Errorf("bad response: %s", dresp)

	}
	return nil
}

func getEventSpec(cnx context.Context, client *ApiClient, id string) (*RemoteEventSpec, error) {
	resp, err := DoConsoleRequest("GET", fmt.Sprintf("%s/event-specs/v1/%s", client.BaseUrl, id), client, cnx, nil)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	defer util.LoggingCloser(cnx, resp.Body)
	if err != nil {
		return nil, err
	}

	var dpResponse esData
	err = kjson.Unmarshal(rbody, &dpResponse)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not expected response code %d", resp.StatusCode)
	}

	return &dpResponse.Data[0], nil

}

func UpdateEventSpec(cnx context.Context, client *ApiClient, es RemoteEventSpec) error {
	existingEs, err := getEventSpec(cnx, client, es.Id)
	if err != nil {
		return err
	}
	newVersion := existingEs.Version + 1

	es.Version = newVersion
	es.Status = "draft"

	esForm := remoteEventSpecPost{Spec: es, Message: ""}

	body, err := json.Marshal(esForm)
	if err != nil {
		return err
	}
	resp, err := DoConsoleRequest("PUT", fmt.Sprintf("%s/event-specs/v1/%s", client.BaseUrl, es.Id), client, cnx, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		rbody, err := io.ReadAll(resp.Body)
		defer util.LoggingCloser(cnx, resp.Body)
		if err != nil {
			return err
		}

		var dresp msgResponse
		err = kjson.Unmarshal(rbody, &dresp)
		if err != nil {
			return errors.Join(err, errors.New("bad response with no message"))
		}

		return fmt.Errorf("bad response: %s", dresp)

	}
	return nil
}

func DeleteEventSpec(cnx context.Context, client *ApiClient, id string) error {
	resp, err := DoConsoleRequest("DELETE", fmt.Sprintf("%s/event-specs/v1/%s", client.BaseUrl, id), client, cnx, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		rbody, err := io.ReadAll(resp.Body)
		defer util.LoggingCloser(cnx, resp.Body)
		if err != nil {
			return err
		}

		var dresp msgResponse
		err = kjson.Unmarshal(rbody, &dresp)
		if err != nil {
			return errors.Join(err, errors.New("bad response with no message"))
		}

		return fmt.Errorf("bad response: %s", dresp)

	}
	return nil
}
