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
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	kjson "k8s.io/apimachinery/pkg/util/json"

	"github.com/snowplow/snowplow-cli/internal/model"
	"github.com/snowplow/snowplow-cli/internal/util"
)

type msgResponse struct {
	Message string `json:"message" yaml:"message"`
}

type pubResponse struct {
	Success  bool     `json:"success" yaml:"success"`
	Errors   []string `json:"errors" yaml:"errors"`
	Warnings []string `json:"warnings" yaml:"warnings"`
	Info     []string `json:"info" yaml:"info"`
	msgResponse
}

type pubError struct {
	Messages []string `json:"messages" yaml:"messages"`
}

type PublishResponse = pubResponse
type ValidateResponse struct {
	pubResponse
	Valid bool `json:"valid" yaml:"valid"`
}

type PublishError = pubError

func (e *pubError) Error() string {
	return strings.Join(e.Messages, "\n")
}

type DataStructureEnv string

const (
	DEV       DataStructureEnv = "DEV"
	PROD      DataStructureEnv = "PROD"
	VALIDATED DataStructureEnv = "VALIDATED"
)

type publishRequest struct {
	Format  string           `json:"format"`
	Message string           `json:"message"`
	Name    string           `json:"name"`
	Source  DataStructureEnv `json:"source"`
	Target  DataStructureEnv `json:"target"`
	Vendor  string           `json:"vendor"`
	Version string           `json:"version"`
}

type fullMeta struct {
	Hidden      *bool              `json:"hidden,omitempty"`
	SchemaType  string             `json:"schemaType,omitempty"`
	CustomData  *map[string]string `json:"customData,omitempty"`
	LockStatus  string             `json:"lockStatus,omitempty"`
	ManagedFrom string             `json:"managedFrom,omitempty"`
}

func Validate(cnx context.Context, client *ApiClient, ds model.DataStructure) (*ValidateResponse, error) {

	body, err := json.Marshal(ds)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(cnx, "POST", fmt.Sprintf("%s/data-structures/v1/validation-requests", client.BaseUrl), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	addStandardHeaders(req, cnx, client)
	resp, err := client.Http.Do(req)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	defer util.LoggingCloser(cnx, resp.Body)
	if err != nil {
		return nil, err
	}

	var vresp ValidateResponse
	err = kjson.Unmarshal(rbody, &vresp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, errors.New(vresp.Message)
	}

	vresp.Valid = vresp.Success

	return &vresp, nil
}

func PublishDev(cnx context.Context, client *ApiClient, ds model.DataStructure, isPatch bool, managedFrom string) (*PublishResponse, error) {
	// during first creation we have to publish first, otherwise metatdata patch fails with 404
	res, err := publish(cnx, client, VALIDATED, DEV, ds, isPatch)
	if err != nil {
		return nil, err
	}
	err = metadataLock(cnx, client, &ds, managedFrom)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func PublishProd(cnx context.Context, client *ApiClient, ds model.DataStructure, managedFrom string) (*PublishResponse, error) {
	err := metadataLock(cnx, client, &ds, managedFrom)
	if err != nil {
		return nil, err
	}
	return publish(cnx, client, DEV, PROD, ds, false)
}

func publish(cnx context.Context, client *ApiClient, from DataStructureEnv, to DataStructureEnv, ds model.DataStructure, isPatch bool) (*PublishResponse, error) {

	dsData, err := ds.ParseData()
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
	if err != nil {
		return nil, err
	}

	addStandardHeaders(req, cnx, client)
	if isPatch {
		q := req.URL.Query()
		q.Add("patch", "true")
		req.URL.RawQuery = q.Encode()
	}
	resp, err := client.Http.Do(req)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	defer util.LoggingCloser(cnx, resp.Body)
	if err != nil {
		return nil, err
	}

	var dresp PublishResponse
	err = kjson.Unmarshal(rbody, &dresp)
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

type Deployment struct {
	Version     string           `json:"version"`
	Env         DataStructureEnv `json:"env"`
	ContentHash string           `json:"contentHash"`
}

type ListResponse struct {
	Hash        string                  `json:"hash"`
	Vendor      string                  `json:"vendor"`
	Format      string                  `json:"format"`
	Name        string                  `json:"name"`
	Meta        model.DataStructureMeta `json:"meta"`
	Deployments []Deployment            `json:"deployments"`
}

func GetIgluCentralListing(cnx context.Context, client *ApiClient) ([]string, error) {
	req, err := http.NewRequestWithContext(cnx, "GET", "https://com-iglucentral-eu1-prod.iglu.snplow.net/api/schemas", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Http.Do(req)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	defer util.LoggingCloser(cnx, resp.Body)
	if err != nil {
		return nil, err
	}

	var list []string
	err = kjson.Unmarshal(rbody, &list)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not expected response code %d", resp.StatusCode)
	}

	return list, err
}

func GetDataStructureListing(cnx context.Context, client *ApiClient) ([]ListResponse, error) {
	req, err := http.NewRequestWithContext(cnx, "GET", fmt.Sprintf("%s/data-structures/v1", client.BaseUrl), nil)
	if err != nil {
		return nil, err
	}

	addStandardHeaders(req, cnx, client)
	resp, err := client.Http.Do(req)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	defer util.LoggingCloser(cnx, resp.Body)
	if err != nil {
		return nil, err
	}

	var listResp []ListResponse
	err = kjson.Unmarshal(rbody, &listResp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not expected response code %d", resp.StatusCode)
	}
	return listResp, nil
}

func GetDataStructureDeployments(cnx context.Context, client *ApiClient, dsHash string) ([]Deployment, error) {
	req, err := http.NewRequestWithContext(cnx, "GET", fmt.Sprintf("%s/data-structures/v1/%s/deployments?from=0&size=1000000000", client.BaseUrl, dsHash), nil)
	if err != nil {
		return nil, err
	}

	addStandardHeaders(req, cnx, client)
	resp, err := client.Http.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not expected response code %d", resp.StatusCode)
	}

	rbody, err := io.ReadAll(resp.Body)
	defer util.LoggingCloser(cnx, resp.Body)
	if err != nil {
		return nil, err
	}

	var deploys []Deployment
	err = kjson.Unmarshal(rbody, &deploys)
	if err != nil {
		return nil, err
	}

	return deploys, nil
}

func GetAllDataStructures(cnx context.Context, client *ApiClient, match []string, includeLegacy bool) ([]model.DataStructure, error) {

	listResp, err := GetDataStructureListing(cnx, client)
	if err != nil {
		return nil, err
	}

	var res []model.DataStructure
	var dsData []map[string]any
	var skippedCount int
	var includedLegacyCount int

	req, err := http.NewRequestWithContext(cnx, "GET", fmt.Sprintf("%s/data-structures/v1/schemas/versions?latest=true", client.BaseUrl), nil)
	if err != nil {
		return nil, err
	}

	addStandardHeaders(req, cnx, client)
	resp, err := client.Http.Do(req)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	defer util.LoggingCloser(cnx, resp.Body)
	if err != nil {
		return nil, err
	}

	err = kjson.Unmarshal(rbody, &dsData)
	if err != nil {
		return nil, err
	}

	dsDataMap := map[string]map[string]any{}
	for _, ds := range dsData {
		if self, ok := ds["self"].(map[string]any); ok {
			dsDataMap[fmt.Sprintf("%s-%s-%s-%s", self["vendor"], self["name"], self["format"], self["version"])] = ds
		} else {
			return nil, fmt.Errorf("wrong data structure self section %s", ds["self"])
		}
	}

	for _, dsResp := range listResp {
		matched := false
		for _, m := range match {
			dsUri := fmt.Sprintf("%s/%s/%s", dsResp.Vendor, dsResp.Name, dsResp.Format)
			if strings.HasPrefix(dsUri, m) {
				matched = true
			}

			slog.Debug("fetching data structure", "match", m, "dsUri", dsUri, "result", matched)
		}

		if !matched && len(match) > 0 {
			continue
		}

		for _, deployment := range dsResp.Deployments {
			if deployment.Env == DEV {
				if dsResp.Meta.SchemaType == "" {
					if !includeLegacy {
						skippedCount++
						continue
					} else {
						includedLegacyCount++
						meta := dsResp.Meta
						meta.SchemaType = "entity"
						dataStructure := model.DataStructure{ApiVersion: "v1", ResourceType: "data-structure", Meta: meta, Data: dsDataMap[fmt.Sprintf("%s-%s-%s-%s", dsResp.Vendor, dsResp.Name, dsResp.Format, deployment.Version)]}
						res = append(res, dataStructure)
					}
				} else {
					dataStructure := model.DataStructure{ApiVersion: "v1", ResourceType: "data-structure", Meta: dsResp.Meta, Data: dsDataMap[fmt.Sprintf("%s-%s-%s-%s", dsResp.Vendor, dsResp.Name, dsResp.Format, deployment.Version)]}
					res = append(res, dataStructure)
				}
			}
		}
	}

	if skippedCount > 0 {
		slog.Info("skipped legacy data structures with empty schemaType", "count", skippedCount, "note", "use --include-legacy to include them")
	}
	if includedLegacyCount > 0 {
		slog.Warn("included legacy data structures with empty schemaType, converted to 'entity'", "count", includedLegacyCount)
	}

	return res, nil
}

func MetadateUpdate(cnx context.Context, client *ApiClient, ds *model.DataStructure, managedFrom string) error {

	data, err := ds.ParseData()
	if err != nil {
		return err
	}

	body := fullMeta{
		Hidden:      &ds.Meta.Hidden,
		SchemaType:  ds.Meta.SchemaType,
		CustomData:  &ds.Meta.CustomData,
		LockStatus:  "locked",
		ManagedFrom: managedFrom,
	}

	return patchMeta(cnx, client, &data.Self, body)
}

func metadataLock(cnx context.Context, client *ApiClient, ds *model.DataStructure, managedFrom string) error {

	data, err := ds.ParseData()
	if err != nil {
		return err
	}

	body := fullMeta{LockStatus: "locked", ManagedFrom: managedFrom}

	return patchMeta(cnx, client, &data.Self, body)
}

func patchMeta(cnx context.Context, client *ApiClient, ds *model.DataStructureSelf, fullMeta fullMeta) error {

	toHash := fmt.Sprintf("%s-%s-%s-%s", client.OrgId, ds.Vendor, ds.Name, ds.Format)
	dsHash := sha256.Sum256([]byte(toHash))

	body, err := json.Marshal(fullMeta)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/data-structures/v1/%x/meta", client.BaseUrl, dsHash)
	req, err := http.NewRequestWithContext(cnx, "PATCH", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	addStandardHeaders(req, cnx, client)

	resp, err := client.Http.Do(req)
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

// GenerateDataStructureHash generates a SHA-256 hash for a data structure
// based on organization ID, vendor, name, and format as per Snowplow API documentation
func GenerateDataStructureHash(orgId, vendor, name, format string) string {
	// Concatenate with dashes as separator: orgId-vendor-name-format
	concatenated := fmt.Sprintf("%s-%s-%s-%s", orgId, vendor, name, format)

	// Hash with SHA-256
	hasher := sha256.New()
	hasher.Write([]byte(concatenated))
	hash := hasher.Sum(nil)

	// Return as hex string
	return fmt.Sprintf("%x", hash)
}

// GetSpecificDataStructure retrieves a specific data structure by its hash
func GetSpecificDataStructure(cnx context.Context, client *ApiClient, dsHash string) (*model.DataStructure, error) {
	// First get the listing to get metadata and deployments
	req, err := http.NewRequestWithContext(cnx, "GET", fmt.Sprintf("%s/data-structures/v1/%s", client.BaseUrl, dsHash), nil)
	if err != nil {
		return nil, err
	}

	addStandardHeaders(req, cnx, client)
	resp, err := client.Http.Do(req)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	defer util.LoggingCloser(cnx, resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve data structure: status %d", resp.StatusCode)
	}

	// Parse the listing response
	var listingResp struct {
		Hash        string                  `json:"hash"`
		Vendor      string                  `json:"vendor"`
		Name        string                  `json:"name"`
		Format      string                  `json:"format"`
		Meta        model.DataStructureMeta `json:"meta"`
		Deployments []Deployment            `json:"deployments"`
	}

	err = kjson.Unmarshal(rbody, &listingResp)
	if err != nil {
		return nil, err
	}

	// Find the latest version in DEV environment
	var latestVersion string
	for _, deployment := range listingResp.Deployments {
		if deployment.Env == DEV {
			latestVersion = deployment.Version
			break
		}
	}

	if latestVersion == "" {
		return nil, fmt.Errorf("no deployment found in DEV environment")
	}

	// Now get the actual schema data using the same approach as bulk download
	return GetSpecificDataStructureVersion(cnx, client, dsHash, latestVersion)
}

// GetSpecificDataStructureVersion retrieves a specific version of a data structure
func GetSpecificDataStructureVersion(cnx context.Context, client *ApiClient, dsHash, version string) (*model.DataStructure, error) {
	// First get the listing to get metadata
	req, err := http.NewRequestWithContext(cnx, "GET", fmt.Sprintf("%s/data-structures/v1/%s", client.BaseUrl, dsHash), nil)
	if err != nil {
		return nil, err
	}

	addStandardHeaders(req, cnx, client)
	resp, err := client.Http.Do(req)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	defer util.LoggingCloser(cnx, resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve data structure: status %d", resp.StatusCode)
	}

	// Parse the listing response
	var listingResp struct {
		Hash        string                  `json:"hash"`
		Vendor      string                  `json:"vendor"`
		Name        string                  `json:"name"`
		Format      string                  `json:"format"`
		Meta        model.DataStructureMeta `json:"meta"`
		Deployments []Deployment            `json:"deployments"`
	}

	err = kjson.Unmarshal(rbody, &listingResp)
	if err != nil {
		return nil, err
	}

	// Now get the actual schema data using the bulk download approach
	// We need to get all schema versions and find the one we want
	req2, err := http.NewRequestWithContext(cnx, "GET", fmt.Sprintf("%s/data-structures/v1/schemas/versions", client.BaseUrl), nil)
	if err != nil {
		return nil, err
	}

	addStandardHeaders(req2, cnx, client)
	resp2, err := client.Http.Do(req2)
	if err != nil {
		return nil, err
	}
	rbody2, err := io.ReadAll(resp2.Body)
	defer util.LoggingCloser(cnx, resp2.Body)
	if err != nil {
		return nil, err
	}

	if resp2.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve schema versions: status %d", resp2.StatusCode)
	}

	var dsData []map[string]any
	err = kjson.Unmarshal(rbody2, &dsData)
	if err != nil {
		return nil, err
	}

	// Find the specific version we want
	key := fmt.Sprintf("%s-%s-%s-%s", listingResp.Vendor, listingResp.Name, listingResp.Format, version)
	var schemaData map[string]any
	for _, ds := range dsData {
		if self, ok := ds["self"].(map[string]any); ok {
			dsKey := fmt.Sprintf("%s-%s-%s-%s", self["vendor"], self["name"], self["format"], self["version"])
			if dsKey == key {
				schemaData = ds
				break
			}
		}
	}

	if schemaData == nil {
		return nil, fmt.Errorf("schema data not found for version %s", version)
	}

	// Construct the data structure
	ds := model.DataStructure{
		ApiVersion:   "v1",
		ResourceType: "data-structure",
		Meta:         listingResp.Meta,
		Data:         schemaData,
	}

	return &ds, nil
}

// GetAllDataStructureVersions downloads all versions of a specific data structure
func GetAllDataStructureVersions(cnx context.Context, client *ApiClient, dsHash string, envFilter string) ([]model.DataStructure, error) {
	// First get all deployments for this data structure
	deployments, err := GetDataStructureDeployments(cnx, client, dsHash)
	if err != nil {
		return nil, err
	}

	var dataStructures []model.DataStructure

	// Download each version, filtering by environment if specified
	for _, deployment := range deployments {
		// Filter by environment if specified
		if envFilter != "" && string(deployment.Env) != envFilter {
			continue
		}

		ds, err := GetSpecificDataStructureVersion(cnx, client, dsHash, deployment.Version)
		if err != nil {
			// Log error but continue with other versions
			slog.Warn("failed to download version", "version", deployment.Version, "env", deployment.Env, "error", err)
			continue
		}
		dataStructures = append(dataStructures, *ds)
	}

	return dataStructures, nil
}
