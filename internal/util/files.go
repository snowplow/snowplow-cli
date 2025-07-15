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
	"slices"
	"sort"

	"github.com/snowplow/snowplow-cli/internal/model"

	"gopkg.in/yaml.v3"
)

type Files struct {
	DataStructuresLocation string
	DataProductsLocation   string
	SourceAppsLocation     string
	ImagesLocation         string
	ExtentionPreference    string
}

func (f Files) CreateDataStructures(dss []model.DataStructure, isPlain bool) error {
	var dataStructuresPath string
	if filepath.IsAbs(f.DataStructuresLocation) {
		dataStructuresPath = f.DataStructuresLocation
	} else {
		dataStructuresPath = filepath.Join(".", f.DataStructuresLocation)
	}

	vendorToSchemas := make(map[string][]model.DataStructure)
	var vendorIds []idFileName

	for _, ds := range dss {
		data, err := ds.ParseData()
		if err != nil {
			return err
		}

		vendor := data.Self.Vendor
		vendorToSchemas[vendor] = append(vendorToSchemas[vendor], ds)

		if len(vendorToSchemas[vendor]) == 1 {
			vendorIds = append(vendorIds, idFileName{
				Id:       vendor,
				FileName: vendor,
			})
		}
	}

	uniqueVendors := createUniqueNames(vendorIds)
	vendorMapping := make(map[string]string)
	for _, uv := range uniqueVendors {
		vendorMapping[uv.Id] = uv.FileName
	}

	for originalVendor, schemas := range vendorToSchemas {
		uniqueVendorName := vendorMapping[originalVendor]
		vendorPath := filepath.Join(dataStructuresPath, uniqueVendorName)

		err := os.MkdirAll(vendorPath, os.ModePerm)
		if err != nil {
			return err
		}

		var schemaIds []idFileName
		idToDs := map[string]model.DataStructure{}
		for _, ds := range schemas {
			data, _ := ds.ParseData()
			id := fmt.Sprintf("%s/%s", originalVendor, data.Self.Name)
			schemaIds = append(schemaIds, idFileName{
				Id:       id,
				FileName: data.Self.Name,
			})
			idToDs[id] = ds
		}

		uniqueSchemas := createUniqueNames(schemaIds)

		for _, schemaFile := range uniqueSchemas {
			_, err = WriteResourceToFile(idToDs[schemaFile.Id], vendorPath, schemaFile.FileName, f.ExtentionPreference, isPlain, DataStructureResourceType)
			if err != nil {
				return err
			}
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

func (f Files) CreateSourceApps(sas []model.CliResource[model.SourceAppData], isPlain bool) (map[string]model.CliResource[model.SourceAppData], error) {
	sourceAppsPath := filepath.Join(".", f.DataProductsLocation, f.SourceAppsLocation)
	err := os.MkdirAll(sourceAppsPath, os.ModePerm)

	if err != nil {
		return nil, err
	}

	var idToFileName []idFileName
	idToSa := make(map[string]model.CliResource[model.SourceAppData])
	for _, sa := range sas {
		idToSa[sa.ResourceName] = sa
		idToFileName = append(idToFileName, idFileName{Id: sa.ResourceName, FileName: sa.Data.Name})
	}

	uniqueNames := createUniqueNames(idToFileName)

	var res = make(map[string]model.CliResource[model.SourceAppData])

	for _, idToName := range uniqueNames {
		sa := idToSa[idToName.Id]
		abs, err := WriteResourceToFile(sa, sourceAppsPath, idToName.FileName, f.ExtentionPreference, isPlain, sa.ResourceType)
		if err != nil {
			return nil, err
		}
		res[abs] = sa
	}

	return res, nil
}

func (f Files) CreateDataProducts(dps []model.CliResource[model.DataProductCanonicalData], isPlain bool) (map[string]model.CliResource[model.DataProductCanonicalData], error) {
	dataProductsPath := filepath.Join(".", f.DataProductsLocation)
	err := os.MkdirAll(dataProductsPath, os.ModePerm)

	if err != nil {
		return nil, err
	}

	var idToFileName []idFileName
	idToDp := make(map[string]model.CliResource[model.DataProductCanonicalData])
	for _, dp := range dps {
		idToDp[dp.ResourceName] = dp
		idToFileName = append(idToFileName, idFileName{Id: dp.ResourceName, FileName: dp.Data.Name})
	}

	uniqueNames := createUniqueNames(idToFileName)

	var res = make(map[string]model.CliResource[model.DataProductCanonicalData])

	for _, idToName := range uniqueNames {
		dp := idToDp[idToName.Id]
		abs, err := WriteResourceToFile(dp, dataProductsPath, idToName.FileName, f.ExtentionPreference, isPlain, dp.ResourceType)
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

	rel, err := filepath.Rel(f.DataProductsLocation, filePath)
	if err != nil {
		return "", err
	}

	relativePath := fmt.Sprintf("./%s", rel)

	return relativePath, err
}

func WriteSerializableToFile(body any, dir string, name string, ext string, yamlPrefix string) (string, error) {
	var bytes []byte
	var err error

	if ext == "yaml" {
		bytes, err = yaml.Marshal(body)
		if err != nil {
			return "", err
		}
		if yamlPrefix != "" {
			bytes = append([]byte(yamlPrefix+"\n"), bytes...)
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

func WriteResourceToFile(body any, dir string, name string, ext string, isPlain bool, resourceType string) (string, error) {
	if isPlain {
		return WriteSerializableToFile(body, dir, name, ext, "")
	} else {
		prefix, err := getLspComment(resourceType)
		if err != nil {
			return "", err
		}
		return WriteSerializableToFile(body, dir, name, ext, prefix)
	}
}

func getLspComment(resourceType string) (string, error) {
	if slices.Contains([]string{DataStructureResourceType, DataProductResourceType, SourceApplicationResourceType}, resourceType) {
		template := "# yaml-language-server: $schema=%s%s.json\n"
		return fmt.Sprintf(template, RepoRawFileURL, resourceType), nil
	} else {
		return "", fmt.Errorf("value %s is not a valid resource type", resourceType)
	}

}
