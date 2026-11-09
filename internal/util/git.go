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
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	kjson "k8s.io/apimachinery/pkg/util/json"
)

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func MaybeResourcesAtMergeBase(dir string, ref string, paths []string) (map[string]map[string]any, error) {
	files := map[string]map[string]any{}

	if strings.HasPrefix(ref, "-") {
		return nil, fmt.Errorf("invalid git ref %s", ref)
	}

	out, err := runGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	topLevel, err := filepath.EvalSymlinks(strings.TrimSpace(out))
	if err != nil {
		return nil, err
	}

	out, err = runGit(dir, "merge-base", "HEAD", ref)
	if err != nil {
		return nil, err
	}
	mergeBase := strings.TrimSpace(out)

	for _, path := range paths {
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return nil, err
		}
		relPath, err := filepath.Rel(topLevel, realPath)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(relPath, "..") {
			return nil, fmt.Errorf("file %s is outside the git repository %s", path, topLevel)
		}
		gitPath := filepath.ToSlash(relPath)

		listed, err := runGit(topLevel, "ls-tree", "--name-only", mergeBase, "--", gitPath)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(listed) == "" {
			continue
		}

		body, err := runGit(topLevel, "show", fmt.Sprintf("%s:%s", mergeBase, gitPath))
		if err != nil {
			return nil, err
		}

		ds := map[string]any{}
		switch filepath.Ext(path) {
		case ".json":
			err = kjson.Unmarshal([]byte(body), &ds)
		case ".yaml", ".yml":
			err = yaml.Unmarshal([]byte(body), &ds)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read %s at %s: %w", gitPath, mergeBase, err)
		}

		files[path] = ds
	}

	return files, nil
}
