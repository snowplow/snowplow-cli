/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package util

import (
	"encoding/json"
	"fmt"
	. "github.com/snowplow-product/snowplow-cli/internal/model"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Files struct {
	DataStructuresLocation string
	DataProductsLocation   string
	SourceAppsLocation     string
	ExtentionPreference    string
}

func (f Files) CreateDataStructures(dss []DataStructure) error {
	dataStrucutresPath := filepath.Join(".", f.DataStructuresLocation)
	for _, ds := range dss {
		data, err := ds.ParseData()
		if err != nil {
			return err
		}
		vendorPath := filepath.Join(dataStrucutresPath, data.Self.Vendor)
		err = os.MkdirAll(vendorPath, os.ModePerm)
		if err != nil {
			return err
		}
		_, err = WriteSerializableToFile(ds, vendorPath, data.Self.Name, f.ExtentionPreference)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f Files) CreateSourceApps(sas []CliResource[SourceAppData]) (map[string]CliResource[SourceAppData], error) {
	sourceAppsPath := filepath.Join(".", f.DataProductsLocation, f.SourceAppsLocation)
	err := os.MkdirAll(sourceAppsPath, os.ModePerm)

	if err != nil {
		return nil, err
	}

	var res = make(map[string]CliResource[SourceAppData])

	for _, sa := range sas {
		abs, err := WriteSerializableToFile(sa, sourceAppsPath, sa.Data.Name, f.ExtentionPreference)
		if err != nil {
			return nil, err
		}
		res[abs] = sa
	}
	return res, nil
}

func (f Files) CreateDataProducts(dps []CliResource[DataProductCanonicalData]) (map[string]CliResource[DataProductCanonicalData], error) {
	dataProductsPath := filepath.Join(".", f.DataProductsLocation)
	err := os.MkdirAll(dataProductsPath, os.ModePerm)

	if err != nil {
		return nil, err
	}

	var res = make(map[string]CliResource[DataProductCanonicalData])

	for _, dp := range dps {
		abs, err := WriteSerializableToFile(dp, dataProductsPath, dp.Data.Name, f.ExtentionPreference)
		if err != nil {
			return nil, err
		}
		res[abs] = dp
	}
	return res, nil
}

func WriteSerializableToFile(body any, dir string, name string, ext string) (string, error) {
	var bytes []byte
	var err error

	if ext == "yaml" {
		bytes, err = yaml.Marshal(body)
		if err != nil {
			return "", err
		}
	} else {
		bytes, err = json.MarshalIndent(body, "", "  ")
		if err != nil {
			return "", err
		}
	}

	filePath := filepath.Join(dir, fmt.Sprintf("%s.%s", name, ext))
	err = os.WriteFile(filePath, bytes, 0644)
	if err != nil {
		return "", err
	}

	slog.Debug("wrote", "file", filePath)

	return filePath, err
}
