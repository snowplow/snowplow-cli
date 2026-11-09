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
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	kjson "k8s.io/apimachinery/pkg/util/json"
)

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func MaybeResourcesAtMergeBase(ctx context.Context, dir string, ref string, searchPaths []string) (map[string]map[string]any, error) {
	files := map[string]map[string]any{}

	if strings.HasPrefix(ref, "-") {
		return nil, fmt.Errorf("invalid git ref %s", ref)
	}

	out, err := runGit(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	topLevel, err := filepath.EvalSymlinks(strings.TrimSpace(out))
	if err != nil {
		return nil, err
	}

	out, err = runGit(ctx, dir, "merge-base", "HEAD", ref)
	if err != nil {
		return nil, fmt.Errorf("could not find where the branch started from %s: fetch the full history (for example actions/checkout with fetch-depth: 0) and make sure %s is fetched: %w", ref, ref, err)
	}
	mergeBase := strings.TrimSpace(out)

	for _, searchPath := range searchPaths {
		absPath, err := filepath.Abs(searchPath)
		if err != nil {
			return nil, err
		}
		realPath, err := filepath.EvalSymlinks(absPath)
		if err != nil {
			return nil, err
		}
		relPath, err := filepath.Rel(topLevel, realPath)
		if err != nil {
			return nil, err
		}
		if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
			return nil, fmt.Errorf("path %s is outside the git repository %s", searchPath, topLevel)
		}

		listed, err := runGit(ctx, topLevel, "ls-tree", "-r", "-z", "--name-only", mergeBase, "--", filepath.ToSlash(relPath))
		if err != nil {
			return nil, err
		}

		for _, gitPath := range strings.Split(listed, "\x00") {
			ext := filepath.Ext(gitPath)
			if ext != ".json" && ext != ".yaml" && ext != ".yml" {
				continue
			}

			body, err := runGit(ctx, topLevel, "show", fmt.Sprintf("%s:%s", mergeBase, gitPath))
			if err != nil {
				return nil, err
			}

			ds := map[string]any{}
			switch ext {
			case ".json":
				err = kjson.Unmarshal([]byte(body), &ds)
			case ".yaml", ".yml":
				err = yaml.Unmarshal([]byte(body), &ds)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to read %s at %s: %w", gitPath, mergeBase, err)
			}

			files[filepath.Join(topLevel, filepath.FromSlash(gitPath))] = ds
		}
	}

	return files, nil
}
