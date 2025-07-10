/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func build() *cobra.Command {
	var testCmd = &cobra.Command{
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := InitConsoleConfig(cmd); err != nil {
				return err
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {},
	}

	InitConsoleFlags(testCmd)
	testCmd.PersistentFlags().String("config", "", "")
	testCmd.PersistentFlags().String("env-file", "", "")

	return testCmd
}

func Test_ConfigFromFile(t *testing.T) {
	defer func(old []string) { os.Args = old }(os.Args)

	os.Args = []string{"xxx", "--config", "../testdata/config/config.yml"}

	testCmd := build()

	err := testCmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	table := []struct {
		flag string
		want string
	}{
		{"host", "totally a url"},
		{"api-key-id", "00000000-0c00-000b-aa00-000000a00000"},
		{"api-key", "00beb000-0b0c-00ed-b0ad-000b00a00000"},
		{"org-id", "0000a0aa-aaba-0fda-a00e-0e0ab0c00b00"},
	}

	for _, row := range table {
		value, _ := testCmd.Flags().GetString(row.flag)
		if value != row.want {
			t.Errorf("%s got %s want %s", row.flag, value, row.want)
		}
	}
}

func Test_ConfigEnvOveride(t *testing.T) {
	defer func(old []string) { os.Args = old }(os.Args)

	os.Args = []string{"xxx", "--config", "../testdata/config/config.yml"}

	t.Setenv("SNOWPLOW_CONSOLE_HOST", "a real url this time")
	t.Setenv("SNOWPLOW_CONSOLE_API_KEY", "but not a secret")

	testCmd := build()

	err := testCmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	table := []struct {
		flag string
		want string
	}{
		{"host", "a real url this time"},
		{"api-key-id", "00000000-0c00-000b-aa00-000000a00000"},
		{"api-key", "but not a secret"},
		{"org-id", "0000a0aa-aaba-0fda-a00e-0e0ab0c00b00"},
	}

	for _, row := range table {
		value, _ := testCmd.Flags().GetString(row.flag)
		if value != row.want {
			t.Errorf("%s got '%s' want '%s'", row.flag, value, row.want)
		}
	}
}

func Test_ConfigValidate(t *testing.T) {
	defer func(old []string) { os.Args = old }(os.Args)

	table := [][]string{
		{"xxx", "-a", "something", "-H", "something", "-S", "somethign", "-o", ""},
		{"xxx", "-a", "something", "-H", "something", "-S", "", "-o", "something"},
		{"xxx", "-a", "something", "-H", "", "-S", "somethign", "-o", "something"},
		{"xxx", "-a", "", "-H", "something", "-S", "somethign", "-o", "something"},
	}

	testCmd := build()

	for _, os.Args = range table {
		err := testCmd.Execute()
		if err == nil {
			t.Errorf("should have failed for %v", os.Args)
		}
	}
}

func Test_ConfigFromEnvFile(t *testing.T) {
	defer func(old []string) { os.Args = old }(os.Args)

	envFile := "../testdata/config/test.env"
	os.Args = []string{"xxx", "--env-file", envFile}

	testCmd := build()

	err := testCmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	table := []struct {
		flag string
		want string
	}{
		{"host", "env-file-host"},
		{"api-key-id", "env-file-api-key-id"},
		{"api-key", "env-file-api-key"},
		{"org-id", "env-file-org-id"},
	}

	for _, row := range table {
		value, _ := testCmd.Flags().GetString(row.flag)
		if value != row.want {
			t.Errorf("%s got %s want %s", row.flag, value, row.want)
		}
	}
}

func Test_ConfigEnvFilePrecedence(t *testing.T) {
	defer func(old []string) { os.Args = old }(os.Args)

	envFile := "../testdata/config/test.env"
	configFile := "../testdata/config/config.yml"
	os.Args = []string{"xxx", "--config", configFile, "--env-file", envFile}

	// Set environment variable to test precedence
	t.Setenv("SNOWPLOW_CONSOLE_HOST", "direct-env-var")

	testCmd := build()

	err := testCmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	table := []struct {
		flag string
		want string
		desc string
	}{
		{"host", "direct-env-var", "direct env var should override .env file"},
		{"api-key-id", "env-file-api-key-id", ".env file should override yaml config"},
		{"api-key", "env-file-api-key", ".env file should override yaml config"},
		{"org-id", "env-file-org-id", ".env file should override yaml config"},
	}

	for _, row := range table {
		value, _ := testCmd.Flags().GetString(row.flag)
		if value != row.want {
			t.Errorf("%s: got '%s' want '%s' (%s)", row.flag, value, row.want, row.desc)
		}
	}
}

func Test_ConfigEnvFileNotFound(t *testing.T) {
	defer func(old []string) { os.Args = old }(os.Args)

	// Test with non-existent explicit .env file
	os.Args = []string{"xxx", "--env-file", "/nonexistent/path/.env"}

	testCmd := build()

	err := testCmd.Execute()
	if err == nil {
		t.Error("should have failed with non-existent .env file")
	}
}

func Test_ConfigEnvFileDiscovery(t *testing.T) {
	defer func(old []string) { os.Args = old }(os.Args)

	// Clean up any existing environment variables that might interfere
	for _, env := range []string{"SNOWPLOW_CONSOLE_HOST", "SNOWPLOW_CONSOLE_API_KEY_ID", "SNOWPLOW_CONSOLE_API_KEY", "SNOWPLOW_CONSOLE_ORG_ID"} {
		_ = os.Unsetenv(env)
	}

	// Create a temporary .env file in current directory
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env")

	err := os.WriteFile(envFile, []byte("SNOWPLOW_CONSOLE_HOST=discovered-host\nSNOWPLOW_CONSOLE_API_KEY_ID=discovered-key-id\nSNOWPLOW_CONSOLE_API_KEY=discovered-key\nSNOWPLOW_CONSOLE_ORG_ID=discovered-org-id"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Change to temp directory to test discovery
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatal(err)
	}

	os.Args = []string{"xxx"}

	testCmd := build()

	err = testCmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	// Check that values were loaded from discovered .env file
	host, _ := testCmd.Flags().GetString("host")
	if host != "discovered-host" {
		t.Errorf("host got '%s' want 'discovered-host'", host)
	}
}
