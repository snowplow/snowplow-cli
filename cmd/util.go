package cmd

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

func DataStructuresFromFileNames(files []string) ([]*DataStructure, error) {

	var dataStructures []*DataStructure

	for _, f := range files {
		err := func() error {
			file, err := os.Open(f)
			if err != nil {
				return err
			}
			defer file.Close()
			body, err := io.ReadAll(file)
			if err != nil {
				return err
			}
			ds := DataStructure{}
			err = yaml.Unmarshal(body, &ds)
			if err != nil {
				return err
			}
			dataStructures = append(dataStructures, &ds)
			return nil
		}()

		if err != nil {
			return nil, err
		}
	}

	return dataStructures, nil

}
