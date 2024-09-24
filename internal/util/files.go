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
		err = WriteSerializableToFile(ds, vendorPath, data.Self.Name, f.ExtentionPreference)
		if err != nil {
			return err
		}
	}

	return nil
}

func WriteSerializableToFile(body any, dir string, name string, ext string) error {
	var bytes []byte
	var err error

	if ext == "yaml" {
		bytes, err = yaml.Marshal(body)
		if err != nil {
			return err
		}
	} else {
		bytes, err = json.MarshalIndent(body, "", "  ")
		if err != nil {
			return err
		}
	}

	filePath := filepath.Join(dir, fmt.Sprintf("%s.%s", name, ext))
	err = os.WriteFile(filePath, bytes, 0644)
	if err != nil {
		return err
	}

	slog.Debug("wrote", "file", filePath)

	return err
}
