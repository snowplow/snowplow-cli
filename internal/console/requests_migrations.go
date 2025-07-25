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

	"github.com/snowplow/snowplow-cli/internal/model"
	"github.com/snowplow/snowplow-cli/internal/util"
	kjson "k8s.io/apimachinery/pkg/util/json"
)

type destination struct {
	Type string `json:"destinationType"`
}

type apiError struct {
	Message string `json:"message" yaml:"message"`
}

type migrationRequest struct {
	DestinationType string                  `json:"destinationType"`
	SourceSchemaKey model.DataStructureSelf `json:"sourceSchemaKey"`
	TargetSchema    map[string]any          `json:"targetSchema"`
}

type migrationResponse struct {
	ChangeType string      `json:"changeType" yaml:"changeType"`
	Migrations []migration `json:"migrations" yaml:"migrations"`
}

type migration struct {
	MigrationType string `json:"migrationType" yaml:"migrationType"`
	ChangeType    string `json:"changeType" yaml:"changeType"`
	Path          string `json:"path" yaml:"path"`
	Message       string `json:"message" yaml:"message"`
}

type MigrationReport struct {
	SuggestedVersion string   `json:"suggestedVersion" yaml:"suggestedVersion"`
	Messages         []string `json:"messages" yaml:"messages"`
}

func fetchMigration(cnx context.Context, client *ApiClient, destination string, from model.DataStructureSelf, to map[string]any) (*migrationResponse, error) {

	mreq := migrationRequest{destination, from, to}

	body, err := json.Marshal(mreq)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/data-structures/v1/schema-migrations", client.BaseUrl)
	req, err := http.NewRequestWithContext(cnx, "POST", url, bytes.NewBuffer(body))
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
		var errMessage apiError
		err = kjson.Unmarshal(rbody, &errMessage)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(errMessage.Message)
	}

	var migration migrationResponse
	err = kjson.Unmarshal(rbody, &migration)
	if err != nil {
		return nil, err
	}

	return &migration, nil
}

func fetchDestinations(cnx context.Context, client *ApiClient) ([]destination, error) {
	req, err := http.NewRequestWithContext(cnx, "GET", fmt.Sprintf("%s/destinations/v3", client.BaseUrl), nil)
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
		var errMessage apiError
		err = kjson.Unmarshal(rbody, &errMessage)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(errMessage.Message)
	}
	var destinations []destination
	err = kjson.Unmarshal(rbody, &destinations)
	if err != nil {
		return nil, err
	}

	return destinations, err

}

func ValidateMigrations(cnx context.Context, client *ApiClient, ds model.DSChangeContext) (map[string]MigrationReport, error) {

	destinations, err := fetchDestinations(cnx, client)
	if err != nil {
		return nil, err
	}

	result := map[string]MigrationReport{}

	f, err := ds.DS.ParseData()
	if err != nil {
		return nil, err
	}
	from := f.Self
	from.Version = ds.RemoteVersion

	for _, dest := range destinations {
		m, err := fetchMigration(cnx, client, dest.Type, from, ds.DS.Data)
		if err != nil {
			return nil, err
		}
		if m == nil {
			return nil, errors.New("migration failed to parse")
		}
		if m.ChangeType != "no-change" {
			var messages []string
			for _, migration := range m.Migrations {
				messages = append(messages, migration.Message)
			}

			remoteV, err := model.ParseSemVer(ds.RemoteVersion)
			if err != nil {
				return nil, err
			}
			localV, err := model.ParseSemVer(f.Self.Version)
			if err != nil {
				return nil, err
			}

			nextVer := model.SemNextVer(*remoteV, m.ChangeType)

			if model.SemVerCmp(nextVer, *localV) == 1 {
				result[dest.Type] = MigrationReport{
					Messages:         messages,
					SuggestedVersion: nextVer.String(),
				}
			}
		}
	}

	return result, nil
}
