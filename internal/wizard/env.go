/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package wizard

import (
	"errors"
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"
)

// upsertDotenv sets values in a .env file, keeping its other lines.
func upsertDotenv(path string, values map[string]string) error {
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	var lines []string
	if s := strings.TrimRight(string(b), "\n"); s != "" {
		lines = strings.Split(s, "\n")
	}
	pending := maps.Clone(values)
	for i, line := range lines {
		key, _, _ := strings.Cut(line, "=")
		name := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(key), "export "))
		if v, ok := values[name]; ok {
			lines[i] = key + "=" + v
			delete(pending, name)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(pending)) {
		lines = append(lines, name+"="+pending[name])
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		return err
	}
	// WriteFile only sets the mode on a new file, and this one now holds a key.
	return os.Chmod(path, 0600)
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func gitDirty(dir string) bool {
	out, err := git(dir, "status", "--porcelain")
	return err == nil && out != ""
}

func gitTracked(dir, name string) bool {
	_, err := git(dir, "ls-files", "--error-unmatch", "--", name)
	return err == nil
}
