package util

import (
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/snowplow-product/snowplow-cli/internal/model"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

	wrongVersions := []string{}

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

	if len(wrongVersions) > 0 {
		return nil, errors.New(strings.Join(wrongVersions, "\n"))
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

	ds := DataStructure{}
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
