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

// upsertDotenv sets values in a .env file, keeping every other line (and
// comment) as it was. The file is left readable by the owner only.
func upsertDotenv(path string, values map[string]string) error {
	var lines []string
	b, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := os.Chmod(path, 0600); err != nil {
			return err
		}
		lines = strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	case errors.Is(err, os.ErrNotExist):
	default:
		return err
	}

	done := map[string]bool{}
	for i, line := range lines {
		rest, exported := strings.CutPrefix(strings.TrimSpace(line), "export ")
		name, _, ok := strings.Cut(rest, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if v, ok := values[name]; ok {
			lines[i] = name + "=" + v
			if exported {
				lines[i] = "export " + lines[i]
			}
			done[name] = true
		}
	}

	for _, name := range slices.Sorted(maps.Keys(values)) {
		if !done[name] {
			lines = append(lines, name+"="+values[name])
		}
	}

	if len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600)
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
