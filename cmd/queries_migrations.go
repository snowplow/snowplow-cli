package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type destination struct {
	Type string `json:"destinationType"`
}

type apiError struct {
	Message string
}

type migrationRequest struct {
	DestinationType string            `json:"destinationType"`
	SourceSchemaKey DataStructureSelf `json:"sourceSchemaKey"`
	TargetSchema    map[string]any    `json:"targetSchema"`
}

type migrationResponse struct {
	ChangeType string
	Migrations []migration
}

type migration struct {
	MigrationType string
	ChangeType    string
	Path          string
	Message       string
}

type MigrationReport struct {
	SuggestedVersion string
	CombinedMessages string
}

func fetchMigration(cnx context.Context, client *ApiClient, destination string, from DataStructureSelf, to map[string]any) (*migrationResponse, error) {

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

	auth := fmt.Sprintf("Bearer %s", client.Jwt)
	req.Header.Add("authorization", auth)

	resp, err := client.Http.Do(req)
	if err != nil {
		return nil, err
	}
	rbody, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errMessage apiError
		err = json.Unmarshal(rbody, &errMessage)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(errMessage.Message)
	}

	var migration migrationResponse
	err = json.Unmarshal(rbody, &migration)
	if err != nil {
		return nil, err
	}

	return &migration, nil
}

func fetchDestinations(cnx context.Context, client *ApiClient) ([]destination, error) {
	req, err := http.NewRequestWithContext(cnx, "GET", fmt.Sprintf("%s/destinations/v3", client.BaseUrl), nil)
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

	if resp.StatusCode != http.StatusOK {
		var errMessage apiError
		err = json.Unmarshal(rbody, &errMessage)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(errMessage.Message)
	}
	var destinations []destination
	err = json.Unmarshal(rbody, &destinations)
	if err != nil {
		return nil, err
	}

	return destinations, err

}

func semNextVer(current string, upgradeType string) (string, error) {
	version := strings.Split(current, "-")

	intVersion := make([]int, 3)

	var err error

	intVersion[0], err = strconv.Atoi(version[0])
	if err != nil {
		return "", err
	}
	intVersion[1], err = strconv.Atoi(version[1])
	if err != nil {
		return "", err
	}
	intVersion[2], err = strconv.Atoi(version[2])
	if err != nil {
		return "", err
	}

	switch upgradeType {
	case "major":
		return fmt.Sprintf("%d-%d-%d", intVersion[0]+1, 0, 0), nil
	case "revision":
		return fmt.Sprintf("%d-%d-%d", intVersion[0], intVersion[1]+1, 0), nil
	case "minor":
		return fmt.Sprintf("%d-%d-%d", intVersion[0], intVersion[1], intVersion[2]+1), nil
	}

	return current, nil

}

func ValidateMigrations(cnx context.Context, client *ApiClient, ds DSChangeContext) (map[string]MigrationReport, error) {

	destinations, err := fetchDestinations(cnx, client)
	if err != nil {
		return nil, err
	}

	result := map[string]MigrationReport{}

	f, err := ds.DS.parseData()
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
			nextVer, err := semNextVer(from.Version, m.ChangeType)
			if err != nil {
				return nil, err
			}
			result[dest.Type] = MigrationReport{
				CombinedMessages: strings.Join(messages, "\n"),
				SuggestedVersion: nextVer,
			}
		}
	}

	return result, nil
}
