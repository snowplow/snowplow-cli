package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type pubResponse struct {
	Success bool
	Errors  []string
	Message string
}

type pubError struct {
	Messages []string
}

type PublishResponse = pubResponse
type ValidateResponse = pubResponse

type PublishError = pubError
type ValidationError = pubError

func (e *pubError) Error() string {
	return strings.Join(e.Messages, "\n")
}

type dataStructureEnv string

const (
	DEV       dataStructureEnv = "DEV"
	PROD      dataStructureEnv = "PROD"
	VALIDATED dataStructureEnv = "VALIDATED"
)

type publishRequest struct {
	Format  string           `json:"format"`
	Message string           `json:"message"`
	Name    string           `json:"name"`
	Source  dataStructureEnv `json:"source"`
	Target  dataStructureEnv `json:"target"`
	Vendor  string           `json:"vendor"`
	Version string           `json:"version"`
}

func Validate(cnx context.Context, client *ApiClient, ds DataStructure) (*ValidateResponse, error) {

	body, err := json.Marshal(ds)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(cnx, "POST", fmt.Sprintf("%s/data-structures/v1/validation-requests", client.BaseUrl), bytes.NewBuffer(body))
	auth := fmt.Sprintf("Bearer %s", client.Jwt)
	req.Header.Add("authorization", auth)

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

	var vresp ValidateResponse
	err = json.Unmarshal(rbody, &vresp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, errors.New(vresp.Message)
	}

	if !vresp.Success {
		return nil, &ValidationError{Messages: vresp.Errors}
	}

	return &vresp, nil
}

func PublishDev(cnx context.Context, client *ApiClient, ds DataStructure) (*PublishResponse, error) {
	return publish(cnx, client, VALIDATED, DEV, ds)
}

func PublishProd(cnx context.Context, client *ApiClient, ds DataStructure) (*PublishResponse, error) {
	return publish(cnx, client, DEV, PROD, ds)
}

func publish(cnx context.Context, client *ApiClient, from dataStructureEnv, to dataStructureEnv, ds DataStructure) (*PublishResponse, error) {

	dsData, err := ds.parseData()
	if err != nil {
		return nil, err
	}

	pr := &publishRequest{
		Message: "",
		Source:  from,
		Target:  to,
		Vendor:  dsData.Self.Vendor,
		Name:    dsData.Self.Name,
		Format:  dsData.Self.Format,
		Version: dsData.Self.Version,
	}

	body, err := json.Marshal(pr)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(cnx, "POST", fmt.Sprintf("%s/data-structures/v1/deployment-requests", client.BaseUrl), bytes.NewBuffer(body))
	auth := fmt.Sprintf("Bearer %s", client.Jwt)
	req.Header.Add("authorization", auth)

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

	var dresp PublishResponse
	err = json.Unmarshal(rbody, &dresp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, errors.New(dresp.Message)
	}

	if !dresp.Success {
		return nil, &PublishError{Messages: dresp.Errors}
	}

	return &dresp, nil
}

type deployment struct {
	Version string           `json:"version"`
	Env     dataStructureEnv `json:"env"`
}

type listRespone struct {
	Hash        string            `json:"hash"`
	Vendor      string            `json:"vendor"`
	Name        string            `json:"name"`
	Meta        DataStructureMeta `json:"meta"`
	Deployments []deployment      `json:"deployments"`
}

func GetAllDataStructures(cnx context.Context, client *ApiClient) ([]DataStructure, error) {

	req, err := http.NewRequestWithContext(cnx, "GET", fmt.Sprintf("%s/data-structures/v1", client.BaseUrl), nil)
	auth := fmt.Sprintf("Bearer %s", client.Jwt)
	req.Header.Add("authorization", auth)

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

	var listResp []listRespone
	err = json.Unmarshal(rbody, &listResp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("Not expected response code %d", resp.StatusCode))
	}

	var res []DataStructure

	for _, dsResp := range listResp {
		for _, deployment := range dsResp.Deployments {
			if deployment.Env == DEV {
				req, err := http.NewRequestWithContext(cnx, "GET", fmt.Sprintf("%s/data-structures/v1/%s/versions/%s", client.BaseUrl, dsResp.Hash, deployment.Version), nil)
				auth := fmt.Sprintf("Bearer %s", client.Jwt)
				req.Header.Add("authorization", auth)
				fmt.Printf("Fetching %s:%s version %s\n", dsResp.Vendor, dsResp.Name, deployment.Version)

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

				var ds map[string]any
				err = json.Unmarshal(rbody, &ds)
				if err != nil {
					return nil, err
				}

				if resp.StatusCode == http.StatusNotFound {
					continue
				}

				if resp.StatusCode != http.StatusOK {
					return nil, errors.New(fmt.Sprintf("Not expected response code %d", resp.StatusCode))
				}

				dataStructure := DataStructure{dsResp.Meta, ds}
				res = append(res, dataStructure)
			}
		}
	}

	return res, nil

}
