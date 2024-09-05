package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"gopkg.in/yaml.v3"
)

func DataStructuresFromPaths(paths []string) (map[string]DataStructure, error) {

	files := map[string]bool{}

	for _, path := range paths {
		err := filepath.WalkDir(path, func(path string, di fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !di.IsDir() {
				files[path] = true
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	ds := make(map[string]DataStructure)

	exts := []string{".yaml", ".yml", ".json"}

	for k := range files {
		if slices.Index(exts, filepath.Ext(k)) != -1 {
			d, err := dataStructureFromFileName(k)
			if err != nil {
				return nil, errors.Join(err, fmt.Errorf("file: %s", k))
			} else {
				ds[k] = *d
			}
		}
	}

	return ds, nil
}

func dataStructureFromFileName(f string) (*DataStructure, error) {
	file, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	ds := DataStructure{ApiVersion: "v1"}
	switch filepath.Ext(file.Name()) {
	case ".json":
		err = json.Unmarshal(body, &ds)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(body, &ds)
	}

	if err != nil {
		return nil, err
	}

	return &ds, nil
}
