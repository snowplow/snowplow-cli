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
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// UpsertDotenv sets values in a .env file, keeping every other line (and
// comment) as it was. New files are created readable by the owner only.
func UpsertDotenv(path string, values map[string]string) error {
	var lines []string
	b, err := os.ReadFile(path)
	switch {
	case err == nil:
		lines = strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	case errors.Is(err, os.ErrNotExist):
	default:
		return err
	}

	done := map[string]bool{}
	for i, line := range lines {
		name, _, ok := strings.Cut(strings.TrimPrefix(strings.TrimSpace(line), "export "), "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if v, ok := values[name]; ok {
			lines[i] = name + "=" + v
			done[name] = true
		}
	}

	var missing []string
	for name := range values {
		if !done[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	for _, name := range missing {
		lines = append(lines, name+"="+values[name])
	}

	if len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}

func InGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = dir
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func GitDirty(dir string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

func GitTracked(dir, name string) bool {
	cmd := exec.Command("git", "ls-files", "--error-unmatch", "--", name)
	cmd.Dir = dir
	return cmd.Run() == nil
}

func EnsureGitignored(dir, name string) (bool, error) {
	if !InGitRepo(dir) {
		return false, nil
	}
	check := exec.Command("git", "check-ignore", "-q", name)
	check.Dir = dir
	if check.Run() == nil {
		return false, nil
	}

	path := filepath.Join(dir, ".gitignore")
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	prefix := ""
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		prefix = "\n"
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()
	_, err = f.WriteString(prefix + name + "\n")
	return err == nil, err
}
