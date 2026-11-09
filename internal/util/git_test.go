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
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

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
	newFile := filepath.Join(repo, "tracking", "new.json")
	writeInTest(t, newFile, `{"resourceType": "data-product"}`)
	gitInTest(t, repo, "add", ".")
	gitInTest(t, repo, "commit", "-q", "-m", "feature work")

	res, err := MaybeResourcesAtMergeBase(filepath.Join(repo, "tracking"), "main", []string{dpFile, newFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := res[dpFile]["data"].(map[string]any)
	if data["name"] != "at base" {
		t.Errorf("expected the file at the merge base, got: %v", res[dpFile])
	}
	if _, ok := res[newFile]; ok {
		t.Errorf("a file absent at base should be left out, got: %v", res[newFile])
	}
}

func Test_MaybeResourcesAtMergeBase_InvalidRef(t *testing.T) {
	repo := t.TempDir()
	gitInTest(t, repo, "init", "-q", "-b", "main")
	writeInTest(t, filepath.Join(repo, "dp.yaml"), "resourceType: data-product\n")
	gitInTest(t, repo, "add", ".")
	gitInTest(t, repo, "commit", "-q", "-m", "base")

	for _, ref := range []string{"origin/does-not-exist", "--all"} {
		_, err := MaybeResourcesAtMergeBase(repo, ref, []string{filepath.Join(repo, "dp.yaml")})
		if err == nil {
			t.Errorf("expected an error for ref %s", ref)
		}
	}
}

func Test_MaybeResourcesAtMergeBase_NotARepo(t *testing.T) {
	dir := t.TempDir()
	writeInTest(t, filepath.Join(dir, "dp.yaml"), "resourceType: data-product\n")
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))

	_, err := MaybeResourcesAtMergeBase(dir, "main", []string{filepath.Join(dir, "dp.yaml")})
	if err == nil {
		t.Error("expected an error outside a git repository")
	}
}
