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
		vendorPath := filepath.Join(dataStrucutresPath, ds.Data.Self.Vendor)
		err := os.MkdirAll(vendorPath, os.ModePerm)
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

		filePath := filepath.Join(vendorPath, fmt.Sprintf("%s.%s", ds.Data.Self.Name, f.ExtentionPreference))
		err2 := os.WriteFile(filePath, bytes, 0644)
		if err2 != nil {
			return err
		}
	}
	return nil
}
