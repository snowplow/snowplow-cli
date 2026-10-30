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
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestUpsertDotenvKeepsOtherLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	orig := "# app settings\nDATABASE_URL=postgres://x\nexport SNOWPLOW_SIGNALS_API_KEY=old\n"
	if err := os.WriteFile(path, []byte(orig), 0644); err != nil {
		t.Fatal(err)
	}

	err := UpsertDotenv(path, map[string]string{
		"SNOWPLOW_SIGNALS_API_KEY": "new",
		"SNOWPLOW_SIGNALS_ORG_ID":  "org",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(path)
	want := "# app settings\nDATABASE_URL=postgres://x\nSNOWPLOW_SIGNALS_API_KEY=new\nSNOWPLOW_SIGNALS_ORG_ID=org\n"
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestUpsertDotenvNewFileIsPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := UpsertDotenv(path, map[string]string{"A": "1"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected 0600, got %v", info.Mode().Perm())
	}
	got, _ := os.ReadFile(path)
	if string(got) != "A=1\n" {
		t.Fatalf("unexpected content %q", got)
	}
}

func TestEnsureGitignored(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	if err := exec.Command("git", "-C", dir, "init", "-q").Run(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("node_modules"), 0644); err != nil {
		t.Fatal(err)
	}

	changed, err := EnsureGitignored(dir, ".env")
	if err != nil || !changed {
		t.Fatalf("expected .gitignore to change, changed=%v err=%v", changed, err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if string(got) != "node_modules\n.env\n" {
		t.Fatalf("unexpected .gitignore %q", got)
	}

	changed, err = EnsureGitignored(dir, ".env")
	if err != nil || changed {
		t.Fatalf("expected no change the second time, changed=%v err=%v", changed, err)
	}
}

func TestGitTracked(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.email=t@t", "-c", "user.name=t"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	git("init", "-q")
	_ = os.WriteFile(filepath.Join(dir, ".env"), []byte("A=1\n"), 0600)
	if GitTracked(dir, ".env") {
		t.Fatal("an untracked .env reported as tracked")
	}
	git("add", ".env")
	git("commit", "-q", "-m", "oops")
	if !GitTracked(dir, ".env") {
		t.Fatal("a committed .env wasn't detected")
	}
	if GitTracked(t.TempDir(), ".env") {
		t.Fatal("outside a repo nothing is tracked")
	}
}

func TestEnsureGitignoredOutsideRepo(t *testing.T) {
	dir := t.TempDir()
	changed, err := EnsureGitignored(dir, ".env")
	if err != nil || changed {
		t.Fatalf("expected nothing outside a repo, changed=%v err=%v", changed, err)
	}
}
