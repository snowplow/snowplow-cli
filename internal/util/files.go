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
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/snowplow/snowplow-cli/internal/model"
	. "github.com/snowplow/snowplow-cli/internal/model"

	"gopkg.in/yaml.v3"
)

type Files struct {
	DataStructuresLocation string
	DataProductsLocation   string
	SourceAppsLocation     string
	ImagesLocation         string
	ExtentionPreference    string
}

func (f Files) CreateDataStructures(dss []DataStructure) error {
	dataStructuresPath := filepath.Join(".", f.DataStructuresLocation)
	for _, ds := range dss {
		data, err := ds.ParseData()
		if err != nil {
			return err
		}
		vendorPath := filepath.Join(dataStructuresPath, data.Self.Vendor)
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

type idFileName struct {
	Id       string
	FileName string
}

func createUniqueNames(idsToFileNames []idFileName) []idFileName {
	//sort to map conflicting names to suffixes consistently between runs
	sort.Slice(idsToFileNames, func(i, j int) bool {
		return idsToFileNames[i].Id < idsToFileNames[j].Id
	})
	normalizedNameToIds := make(map[string][]string)
	for _, originalName := range idsToFileNames {
		normalizedName := ResourceNameToFileName(originalName.FileName)
		normalizedNameToIds[normalizedName] = append(normalizedNameToIds[normalizedName], originalName.Id)
	}
	var idToUniqueName []idFileName
	for name, ids := range normalizedNameToIds {
		if len(ids) > 1 {
			for idx, id := range ids {
				uniqueName := fmt.Sprintf("%s-%d", name, idx+1)
				idToUniqueName = append(idToUniqueName, idFileName{Id: id, FileName: uniqueName})
			}
		} else {
			idToUniqueName = append(idToUniqueName, idFileName{Id: ids[0], FileName: name})
		}
	}
	return idToUniqueName
}

func (f Files) CreateSourceApps(sas []CliResource[SourceAppData]) (map[string]CliResource[SourceAppData], error) {
	sourceAppsPath := filepath.Join(".", f.DataProductsLocation, f.SourceAppsLocation)
	err := os.MkdirAll(sourceAppsPath, os.ModePerm)

	if err != nil {
		return nil, err
	}

	var idToFileName []idFileName
	idToSa := make(map[string]CliResource[SourceAppData])
	for _, sa := range sas {
		idToSa[sa.ResourceName] = sa
		idToFileName = append(idToFileName, idFileName{Id: sa.ResourceName, FileName: sa.Data.Name})
	}

	uniqueNames := createUniqueNames(idToFileName)

	var res = make(map[string]CliResource[SourceAppData])

	for _, idToName := range uniqueNames {
		sa := idToSa[idToName.Id]
		abs, err := WriteSerializableToFile(sa, sourceAppsPath, idToName.FileName, f.ExtentionPreference)
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

	var idToFileName []idFileName
	idToDp := make(map[string]CliResource[DataProductCanonicalData])
	for _, dp := range dps {
		idToDp[dp.ResourceName] = dp
		idToFileName = append(idToFileName, idFileName{Id: dp.ResourceName, FileName: dp.Data.Name})
	}

	uniqueNames := createUniqueNames(idToFileName)

	var res = make(map[string]CliResource[DataProductCanonicalData])

	for _, idToName := range uniqueNames {
		dp := idToDp[idToName.Id]
		abs, err := WriteSerializableToFile(dp, dataProductsPath, idToName.FileName, f.ExtentionPreference)
		if err != nil {
			return nil, err
		}
		res[abs] = dp
	}

	return res, nil
}

func (f Files) CreateImageFolder() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	imagesPath := filepath.Join(cwd, f.DataProductsLocation, f.ImagesLocation)
	err = os.MkdirAll(imagesPath, os.ModePerm)

	if err != nil {
		return "", err
	}

	relativePath, err := filepath.Rel(cwd, imagesPath)
	if err != nil {
		return "", err
	}
	return relativePath, nil
}

func (f Files) WriteImage(name string, dir string, image *model.Image) (string, error) {
	filePath := filepath.Join(dir, fmt.Sprintf("%s%s", name, image.Ext))
	err := os.WriteFile(filePath, image.Data, 0644)
	if err != nil {
		return "", err
	}

	slog.Debug("wrote", "file", filePath)

	relativePath := fmt.Sprintf(".%s", strings.TrimPrefix(filePath, f.DataProductsLocation))

	return relativePath, err
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
