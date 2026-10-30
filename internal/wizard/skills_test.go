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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func skillsArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		_, _ = tw.Write([]byte(body))
	}
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

func TestFetchSkills(t *testing.T) {
	archive := skillsArchive(t, map[string]string{
		"skills-main/README.md":                                    "readme",
		"skills-main/plugins/snowplow/skills/signals/SKILL.md":     "signals",
		"skills-main/plugins/snowplow/skills/signals/refs/a.md":    "a",
		"skills-main/plugins/snowplow/skills/no-skill-md/notes.md": "x",
		"skills-main/plugins/snowplow/skills/evil/../../escape.md": "x",
	})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tar.gz/main" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(archive)
	}))
	defer ts.Close()
	orig := skillsArchiveURL
	skillsArchiveURL = ts.URL + "/tar.gz/%s"
	defer func() { skillsArchiveURL = orig }()

	skills, err := FetchSkills(context.Background(), "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].Name != "signals" || len(skills[0].Files) != 2 {
		t.Fatalf("unexpected skills %+v", skills)
	}

	if _, err := FetchSkills(context.Background(), "missing"); err == nil {
		t.Fatal("expected an error for a missing ref")
	}
}

func TestInstallSkills(t *testing.T) {
	dir := t.TempDir()
	userOwned := filepath.Join(dir, "troubleshooting")
	if err := os.MkdirAll(userOwned, 0755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(userOwned, "SKILL.md"), []byte("mine"), 0644)

	skills := []Skill{
		{Name: "signals", Files: map[string][]byte{"SKILL.md": []byte("v1"), "refs/old.md": []byte("old")}},
		{Name: "troubleshooting", Files: map[string][]byte{"SKILL.md": []byte("ours")}},
	}
	skipped, err := InstallSkills(dir, skills)
	if err != nil {
		t.Fatal(err)
	}
	if len(skipped) != 1 || skipped[0] != "troubleshooting" {
		t.Fatalf("expected the user's folder to be skipped, got %v", skipped)
	}
	if b, _ := os.ReadFile(filepath.Join(userOwned, "SKILL.md")); string(b) != "mine" {
		t.Fatal("user's skill was overwritten")
	}

	// A second run replaces our own folder, removing files no longer shipped.
	skills[0].Files = map[string][]byte{"SKILL.md": []byte("v2")}
	if _, err := InstallSkills(dir, skills[:1]); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "signals", "SKILL.md")); string(b) != "v2" {
		t.Fatalf("skill not updated: %q", b)
	}
	if _, err := os.Stat(filepath.Join(dir, "signals", "refs", "old.md")); !os.IsNotExist(err) {
		t.Fatal("stale file was kept")
	}
}
