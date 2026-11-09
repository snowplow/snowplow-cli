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
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func isolateGitInTest(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
}

func gitInTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.email=test@example.com", "-c", "user.name=test", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func writeInTest(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func Test_MaybeResourcesAtMergeBase(t *testing.T) {
	isolateGitInTest(t)
	repo := t.TempDir()
	gitInTest(t, repo, "init", "-q", "-b", "main")
	dpFile := filepath.Join(repo, "tracking", "dp.yaml")
	writeInTest(t, dpFile, "resourceType: data-product\ndata:\n  name: at base\n")
	gitInTest(t, repo, "add", ".")
	gitInTest(t, repo, "commit", "-q", "-m", "base")
	gitInTest(t, repo, "checkout", "-q", "-b", "feature")

	gitInTest(t, repo, "checkout", "-q", "main")
	writeInTest(t, dpFile, "resourceType: data-product\ndata:\n  name: on main tip\n")
	gitInTest(t, repo, "commit", "-q", "-am", "main moves on")
	gitInTest(t, repo, "checkout", "-q", "feature")

	writeInTest(t, dpFile, "resourceType: data-product\ndata:\n  name: local\n")
	writeInTest(t, filepath.Join(repo, "tracking", "new.json"), `{"resourceType": "data-product"}`)
	gitInTest(t, repo, "add", ".")
	gitInTest(t, repo, "commit", "-q", "-m", "feature work")

	t.Chdir(filepath.Join(repo, "tracking"))
	res, err := MaybeResourcesAtMergeBase(context.Background(), ".", "main", []string{"."})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	realRepo, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatal(err)
	}
	baseDpFile := filepath.Join(realRepo, "tracking", "dp.yaml")
	data, _ := res[baseDpFile]["data"].(map[string]any)
	if data["name"] != "at base" {
		t.Errorf("expected the file at the merge base, got: %v", res)
	}
	if len(res) != 1 {
		t.Errorf("a file absent at base should be left out, got: %v", res)
	}
}

func Test_MaybeResourcesAtMergeBase_InvalidRef(t *testing.T) {
	isolateGitInTest(t)
	repo := t.TempDir()
	gitInTest(t, repo, "init", "-q", "-b", "main")
	writeInTest(t, filepath.Join(repo, "dp.yaml"), "resourceType: data-product\n")
	gitInTest(t, repo, "add", ".")
	gitInTest(t, repo, "commit", "-q", "-m", "base")

	for _, ref := range []string{"origin/does-not-exist", "--all"} {
		_, err := MaybeResourcesAtMergeBase(context.Background(), repo, ref, []string{repo})
		if err == nil {
			t.Errorf("expected an error for ref %s", ref)
		}
	}
}

func Test_MaybeResourcesAtMergeBase_MissingGit(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := MaybeResourcesAtMergeBase(context.Background(), t.TempDir(), "main", []string{"."})
	if err == nil || !strings.Contains(err.Error(), "executable file not found") {
		t.Errorf("expected the missing git binary in the error, got: %v", err)
	}
}

func Test_MaybeResourcesAtMergeBase_NotARepo(t *testing.T) {
	isolateGitInTest(t)
	dir := t.TempDir()
	writeInTest(t, filepath.Join(dir, "dp.yaml"), "resourceType: data-product\n")
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))

	_, err := MaybeResourcesAtMergeBase(context.Background(), dir, "main", []string{dir})
	if err == nil {
		t.Error("expected an error outside a git repository")
	}
}
