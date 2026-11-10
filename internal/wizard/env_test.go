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
	orig := "# app settings\nDATABASE_URL=postgres://x\nSNOWPLOW_SIGNALS_API_KEY=old\n"
	if err := os.WriteFile(path, []byte(orig), 0644); err != nil {
		t.Fatal(err)
	}

	err := upsertDotenv(path, map[string]string{
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
	if info, _ := os.Stat(path); info.Mode().Perm() != 0600 {
		t.Fatalf("expected an existing file to be tightened to 0600, got %v", info.Mode().Perm())
	}
}

func TestUpsertDotenvReplacesEveryOccurrence(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	orig := "SNOWPLOW_SIGNALS_API_KEY=first\nexport SNOWPLOW_SIGNALS_API_KEY=second\n"
	if err := os.WriteFile(path, []byte(orig), 0600); err != nil {
		t.Fatal(err)
	}

	if err := upsertDotenv(path, map[string]string{"SNOWPLOW_SIGNALS_API_KEY": "new"}); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(path)
	want := "SNOWPLOW_SIGNALS_API_KEY=new\nexport SNOWPLOW_SIGNALS_API_KEY=new\n"
	if string(got) != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestUpsertDotenvNewFileIsPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := upsertDotenv(path, map[string]string{"A": "1"}); err != nil {
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
	if gitTracked(dir, ".env") {
		t.Fatal("an untracked .env reported as tracked")
	}
	git("add", ".env")
	git("commit", "-q", "-m", "oops")
	if !gitTracked(dir, ".env") {
		t.Fatal("a committed .env wasn't detected")
	}
	if gitTracked(t.TempDir(), ".env") {
		t.Fatal("outside a repo nothing is tracked")
	}
}
