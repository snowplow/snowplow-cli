package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Files struct {
	DataStructuresLocation string
	ExtentionPreference    string
}

func (f Files) createDataStructures(dss []DataStructure) error {
	dataStrucutresPath := filepath.Join(".", f.DataStructuresLocation)
	for _, ds := range dss {
		data, err := ds.parseData()
		if err != nil {
			return err
		}
		vendorPath := filepath.Join(dataStrucutresPath, data.Self.Vendor)
		err = os.MkdirAll(vendorPath, os.ModePerm)
		if err != nil {
			return err
		}

		var bytes []byte

		if f.ExtentionPreference == "yaml" {
			bytes, err = yaml.Marshal(ds)
			if err != nil {
				return err
			}
		} else {
			bytes = nil
		}

		filePath := filepath.Join(vendorPath, fmt.Sprintf("%s.%s", data.Self.Name, f.ExtentionPreference))
		err = os.WriteFile(filePath, bytes, 0644)
		if err != nil {
			return err
		}
	}
	return nil
}
