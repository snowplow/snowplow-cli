/**
 * Copyright (c) 2013-present Snowplow Analytics Ltd.
 * All rights reserved.
 * This software is made available by Snowplow Analytics, Ltd.,
 * under the terms of the Snowplow Limited Use License Agreement, Version 1.0
 * located at https://docs.snowplow.io/limited-use-license-1.0
 * BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
 * OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
 */

package console

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
)

var builtInSchema = []string{
	"iglu:com.snowplowanalytics.snowplow/page_ping/jsonschema/1-0-0",
	"iglu:com.snowplowanalytics.snowplow/page_view/jsonschema/1-0-0",
}

type SchemaDeployChecker interface {
	IsDSDeployed(uri string) (found bool, deployedVersions []string, err error)
}

type SchemaDeployCheckProvider struct {
	igluCentralList  []string
	dsList           []ListResponse
	getDsDeployments func(hash string) ([]Deployment, error)
}

func NewSchemaDeployChecker(cnx context.Context, c *ApiClient) (*SchemaDeployCheckProvider, error) {
	igluCentral, err := GetIgluCentralListing(cnx, c)
	if err != nil {
		return nil, err
	}

	dsList, err := GetDataStructureListing(cnx, c)
	if err != nil {
		return nil, err
	}

	lookup := func(hash string) ([]Deployment, error) {
		deploys, err := GetDataStructureDeployments(cnx, c, hash)
		if err != nil {
			return nil, err
		}
		return deploys, nil
	}

	return &SchemaDeployCheckProvider{igluCentral, dsList, lookup}, nil
}

func (sdc *SchemaDeployCheckProvider) IsDSDeployed(uri string) (bool, []string, error) {
	splut := strings.Split(strings.TrimPrefix(uri, "iglu:"), "/")

	if len(splut) != 4 {
		return false, nil, fmt.Errorf("invalid iglu uri got: %s", uri)
	}

	if slices.Contains(sdc.igluCentralList, uri) {
		slog.Debug("validation", "msg", "iglu central resolved "+uri)
		return true, nil, nil
	}

	if slices.Contains(builtInSchema, uri) {
		slog.Debug("validation", "msg", "built in resolved "+uri)
		return true, nil, nil
	}

	for _, ds := range sdc.dsList {
		if ds.Vendor == splut[0] && ds.Name == splut[1] && ds.Format == splut[2] {
			return sdc.isVersionDeployed(ds, uri, splut[3])
		}
	}

	return false, nil, nil
}

func (sdc *SchemaDeployCheckProvider) isVersionDeployed(ds ListResponse, uri string, version string) (bool, []string, error) {
	if resolvedInConsole(ds.Deployments, uri, version) {
		return true, nil, nil
	}

	deploys, err := sdc.getDsDeployments(ds.Hash)
	if err != nil {
		return false, nil, err
	}

	if resolvedInConsole(deploys, uri, version) {
		return true, nil, nil
	}

	alternativeVersions := map[string]bool{}
	for _, deploy := range deploys {
		if deploy.Version != version {
			alternativeVersions[deploy.Version] = true
		}
	}

	foundVersions := []string{}
	for v := range alternativeVersions {
		foundVersions = append(foundVersions, v)
	}

	return false, foundVersions, nil
}

func resolvedInConsole(deployments []Deployment, uri string, version string) bool {
	for _, deployment := range deployments {
		if deployment.Version == version && (deployment.Env == DEV || deployment.Env == PROD) {
			slog.Debug("validation", "msg", fmt.Sprintf("console resolved %s in %s", uri, deployment.Env))
			return true
		}
	}
	return false
}
