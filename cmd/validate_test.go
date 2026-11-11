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
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

func gitInTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.email=test@example.com", "-c", "user.name=test", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func isolateCommandInTest(t *testing.T, args ...string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, name := range []string{"SNOWPLOW_CONSOLE_API_KEY_ID", "SNOWPLOW_CONSOLE_API_KEY", "SNOWPLOW_CONSOLE_ORG_ID", "SNOWPLOW_CONSOLE_HOST"} {
		t.Setenv(name, "")
	}

	t.Cleanup(func() {
		RootCmd.SetArgs(nil)
		found, _, err := RootCmd.Find(args)
		if err != nil {
			return
		}
		found.SilenceErrors = false
		found.SilenceUsage = false
		for c := found; c != nil; c = c.Parent() {
			for _, fs := range []*pflag.FlagSet{c.Flags(), c.PersistentFlags()} {
				fs.VisitAll(func(f *pflag.Flag) {
					_ = f.Value.Set(f.DefValue)
					f.Changed = false
				})
			}
		}
	})
}

func Test_ValidateBaseRefNeedsNoCredentials(t *testing.T) {
	isolateCommandInTest(t, "dp", "validate")
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

	t.Chdir(tracking)

	RootCmd.SetArgs([]string{"dp", "validate", "--base-ref", "main", "--host", "http://127.0.0.1:1", "."})
	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("expected validate --base-ref to run without credentials, got: %v", err)
	}
}

func Test_ValidateBaseRefReturnsValidationFailure(t *testing.T) {
	isolateCommandInTest(t, "dp", "validate")
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
      version: 5
`
	if err := os.WriteFile(filepath.Join(tracking, "checkout.yaml"), []byte(dp), 0644); err != nil {
		t.Fatal(err)
	}
	gitInTest(t, repo, "init", "-q", "-b", "main")
	gitInTest(t, repo, "commit", "-q", "--allow-empty", "-m", "base")

	t.Chdir(tracking)

	RootCmd.SetArgs([]string{"dp", "validate", "--base-ref", "main", "."})
	err := RootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("expected a validation failure, got: %v", err)
	}

	found, _, _ := RootCmd.Find([]string{"dp", "validate"})
	if !found.SilenceUsage || !found.SilenceErrors {
		t.Errorf("the validation failure should be reported once by the logger, without usage")
	}
}

func Test_ValidateWithoutBaseRefStillNeedsCredentials(t *testing.T) {
	isolateCommandInTest(t, "dp", "validate")
	t.Chdir(t.TempDir())

	RootCmd.SetArgs([]string{"dp", "validate", "."})
	err := RootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "not set") {
		t.Fatalf("expected a missing configuration error, got: %v", err)
	}

	found, _, _ := RootCmd.Find([]string{"dp", "validate"})
	if !found.SilenceUsage || !found.SilenceErrors {
		t.Errorf("the configuration failure should be reported once by the logger, without usage")
	}
}
