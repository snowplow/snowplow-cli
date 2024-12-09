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

type schemaDeployCheckProvider struct {
	igluCentralList  []string
	dsList           []ListResponse
	getDsDeployments func(hash string) ([]Deployment, error)
}

func (sdc *schemaDeployCheckProvider) IsDSDeployed(uri string) (found bool, deployedVersions []string, err error) {
	splut := strings.Split(strings.TrimPrefix(uri, "iglu:"), "/")

	if len(splut) != 4 {
		return false, nil, fmt.Errorf("invalid iglu uri got: %s", uri)
	}

	if slices.Contains(sdc.igluCentralList, uri) {
		slog.Debug("validation", "msg", fmt.Sprintf("iglu central resolved %s", uri))
		return true, nil, nil
	}

	if slices.Contains(builtInSchema, uri) {
		slog.Debug("validation", "msg", fmt.Sprintf("built in resolved %s", uri))
		return true, nil, nil
	}

	for _, ds := range sdc.dsList {
		if ds.Vendor == splut[0] && ds.Name == splut[1] && ds.Format == splut[2] {

			for _, deployment := range ds.Deployments {
				if deployment.Version == splut[3] {
					switch deployment.Env {
					case DEV:
						slog.Debug("validation", "msg", fmt.Sprintf("console resolved %s in %s", uri, deployment.Env))
						return true, nil, nil
					case PROD:
						slog.Debug("validation", "msg", fmt.Sprintf("console resolved %s in %s", uri, deployment.Env))
						return true, nil, nil
					}
				}
			}

			deploys, err := sdc.getDsDeployments(ds.Hash)
			if err != nil {
				return false, nil, err
			}

			alternativeVersions := map[string]bool{}
			for _, deploy := range deploys {
				if deploy.Version == splut[3] {
					switch deploy.Env {
					case DEV:
						slog.Debug("validation", "msg", fmt.Sprintf("console resolved %s in %s", uri, deploy.Env))
						return true, nil, nil
					case PROD:
						slog.Debug("validation", "msg", fmt.Sprintf("console resolved %s in %s", uri, deploy.Env))
						return true, nil, nil
					}
				} else {
					alternativeVersions[deploy.Version] = true
				}
			}

			foundVersions := []string{}
			for v := range alternativeVersions {
				foundVersions = append(foundVersions, v)
			}

			return false, foundVersions, nil
		}
	}

	return false, nil, nil
}

func NewSchemaDeployChecker(cnx context.Context, c *ApiClient) (SchemaDeployChecker, error) {
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

	return &schemaDeployCheckProvider{igluCentral, dsList, lookup}, nil
}
