/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package cmd

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

func Test_ValidateBaseRefNeedsNoCredentials(t *testing.T) {
	repo := t.TempDir()
	tracking := filepath.Join(repo, "tracking")
	if err := os.MkdirAll(tracking, 0755); err != nil {
		t.Fatal(err)
	}
	dp := `apiVersion: v1
resourceType: data-product
resourceName: dbc68917-3bcb-4640-a165-483de5fc033c
data:
  name: checkout
  sourceApplications: []
  eventSpecifications:
    - resourceName: b57f712e-ee43-4e10-8b36-48a0f6f2e628
      name: Checkout Started
      version: 0
`
	if err := os.WriteFile(filepath.Join(tracking, "checkout.yaml"), []byte(dp), 0644); err != nil {
		t.Fatal(err)
	}
	gitInTest(t, repo, "init", "-q", "-b", "main")
	gitInTest(t, repo, "commit", "-q", "--allow-empty", "-m", "base")

	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, name := range []string{"SNOWPLOW_CONSOLE_API_KEY_ID", "SNOWPLOW_CONSOLE_API_KEY", "SNOWPLOW_CONSOLE_ORG_ID", "SNOWPLOW_CONSOLE_HOST"} {
		t.Setenv(name, "")
	}
	t.Chdir(tracking)

	RootCmd.SetArgs([]string{"dp", "validate", "--base-ref", "main", "--host", "http://127.0.0.1:1", "."})
	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("expected validate --base-ref to run without credentials, got: %v", err)
	}
}
