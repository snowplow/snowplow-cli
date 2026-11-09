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

	_, err := MaybeResourcesAtMergeBase(context.Background(), repo, "origin/does-not-exist", []string{repo})
	if err == nil || !strings.Contains(err.Error(), "fetch-depth: 0") {
		t.Errorf("expected an actionable error for an unknown ref, got: %v", err)
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

func Test_MaybeResourcesAtMergeBase_ShallowClone(t *testing.T) {
	isolateGitInTest(t)
	origin := t.TempDir()
	gitInTest(t, origin, "init", "-q", "-b", "main")
	writeInTest(t, filepath.Join(origin, "dp.yaml"), "resourceType: data-product\n")
	gitInTest(t, origin, "add", ".")
	gitInTest(t, origin, "commit", "-q", "-m", "base")
	gitInTest(t, origin, "checkout", "-q", "-b", "feature")
	writeInTest(t, filepath.Join(origin, "dp.yaml"), "resourceType: data-product\ndata: {name: feature}\n")
	gitInTest(t, origin, "commit", "-q", "-am", "feature")
	gitInTest(t, origin, "checkout", "-q", "main")
	writeInTest(t, filepath.Join(origin, "other.yaml"), "resourceType: data-product\n")
	gitInTest(t, origin, "add", ".")
	gitInTest(t, origin, "commit", "-q", "-m", "main moves on")

	parent := t.TempDir()
	gitInTest(t, parent, "clone", "-q", "--depth", "1", "--no-single-branch", "--branch", "feature", "file://"+origin, "clone")
	clone := filepath.Join(parent, "clone")

	_, err := MaybeResourcesAtMergeBase(context.Background(), clone, "origin/main", []string{clone})
	if err == nil {
		t.Fatal("expected an error when the merge base is not in a shallow clone")
	}
	want := "could not find where the branch started from origin/main: fetch the full history (for example actions/checkout with fetch-depth: 0) and make sure origin/main is fetched"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("expected an actionable error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "git merge-base HEAD origin/main failed") {
		t.Errorf("expected the git error to stay wrapped, got: %v", err)
	}
}

func Test_MaybeResourcesAtMergeBase_DotDotPrefixedDirectory(t *testing.T) {
	isolateGitInTest(t)
	repo := t.TempDir()
	gitInTest(t, repo, "init", "-q", "-b", "main")
	writeInTest(t, filepath.Join(repo, "..config", "dp.yaml"), "resourceType: data-product\ndata: {name: at base}\n")
	gitInTest(t, repo, "add", ".")
	gitInTest(t, repo, "commit", "-q", "-m", "base")

	res, err := MaybeResourcesAtMergeBase(context.Background(), repo, "main", []string{filepath.Join(repo, "..config")})
	if err != nil {
		t.Fatalf("a directory named ..config inside the repo should be accepted, got: %v", err)
	}
	if len(res) != 1 {
		t.Errorf("expected the file under ..config, got: %v", res)
	}

	_, err = MaybeResourcesAtMergeBase(context.Background(), repo, "main", []string{filepath.Dir(repo)})
	if err == nil || !strings.Contains(err.Error(), "outside the git repository") {
		t.Errorf("expected a path outside the repository to be rejected, got: %v", err)
	}
}
