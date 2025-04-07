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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"

	. "github.com/snowplow/snowplow-cli/internal/model"
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

func dataFromFileName(f string) (map[string]any, error) {
	file, err := os.Open(f)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	ds := map[string]any{}
	switch filepath.Ext(file.Name()) {
	case ".json":
		err = json.Unmarshal(body, &ds)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(body, &ds)
	}

	if err != nil {
		return nil, err
	}

	return ds, nil
}

func MaybeResourcesfromPaths(paths []string) (map[string]map[string]any, error) {

	files := map[string]map[string]any{}

	for _, path := range paths {
		err := filepath.WalkDir(path, func(path string, di fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !di.IsDir() {
				absPath, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				files[absPath], err = dataFromFileName(path)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

func ResourceNameToFileName(s string) string {
	reservedWindowsNamesRegex := regexp.MustCompile(`(?i)^(con|prn|aux|nul|com[0-9]|lpt[0-9])$`)
	allPrintableAsciiNegates := regexp.MustCompile("[^ -~]")
	invalidPathCharacters := regexp.MustCompile("[ /]")
	t := invalidPathCharacters.ReplaceAllLiteralString(reservedWindowsNamesRegex.ReplaceAllLiteralString(allPrintableAsciiNegates.ReplaceAllLiteralString(s, ""), ""), "-")
	res := strings.ToLower(strings.ReplaceAll(strings.Trim(t, " "), " ", "-"))
	if len(res) == 0 {
		return "unnamed"
	}
	return res
}

func SetMinus(s1, s2 []string) []string {
	diff := make(map[string]bool, len(s1))
	for _, v1 := range s1 {
		diff[v1] = true
	}
	for _, v2 := range s2 {
		if _, ok := diff[v2]; ok {
			diff[v2] = false
		}
	}

	var result []string
	for value, ok := range diff {
		if ok {
			result = append(result, value)
		}
	}
	return result
}

func StringSlicesEqual(one []string, other []string) bool {
	return len(one) == 0 && len(other) == 0 || reflect.DeepEqual(one, other)
}
