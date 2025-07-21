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

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

func Test_SaveConfig(t *testing.T) {
	t.Run("SaveConfig with existing file - merge values", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "snowplow")
		configPath := filepath.Join(configDir, "snowplow.yml")

		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		existingConfig := map[string]any{
			"console": map[string]any{
				"existing-key": "existing-value",
				"api-key-id":   "old-key-id", // This should be overwritten
			},
			"other-section": map[string]any{
				"other-key": "other-value",
			},
		}

		data, err := yaml.Marshal(existingConfig)
		if err != nil {
			t.Fatal(err)
		}

		err = os.WriteFile(configPath, data, 0644)
		if err != nil {
			t.Fatal(err)
		}

		originalGetConfigPath := getConfigPath
		getConfigPath = func() string {
			return configPath
		}
		defer func() { getConfigPath = originalGetConfigPath }()

		// Test SaveConfig
		err = SaveConfig("test-org-id", "test-api-key-id", "test-api-key", "https://test.example.com")
		if err != nil {
			t.Fatal(err)
		}

		// Read the saved config file
		savedData, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		var savedConfig map[string]any
		err = yaml.Unmarshal(savedData, &savedConfig)
		if err != nil {
			t.Fatal(err)
		}

		consoleConfig, ok := savedConfig["console"].(map[string]any)
		if !ok {
			t.Fatal("console section not found or not a map")
		}

		if consoleConfig["api-key"] != "test-api-key" {
			t.Errorf("api-key got '%v' want 'test-api-key'", consoleConfig["api-key"])
		}
		if consoleConfig["api-key-id"] != "test-api-key-id" {
			t.Errorf("api-key-id got '%v' want 'test-api-key-id'", consoleConfig["api-key-id"])
		}
		if consoleConfig["org-id"] != "test-org-id" {
			t.Errorf("org-id got '%v' want 'test-org-id'", consoleConfig["org-id"])
		}
		if consoleConfig["host"] != "https://test.example.com" {
			t.Errorf("host got '%v' want 'https://test.example.com'", consoleConfig["host"])
		}

		if consoleConfig["existing-key"] != "existing-value" {
			t.Errorf("existing-key got '%v' want 'existing-value'", consoleConfig["existing-key"])
		}

		otherSection, ok := savedConfig["other-section"].(map[string]any)
		if !ok {
			t.Fatal("other-section not found or not a map")
		}
		if otherSection["other-key"] != "other-value" {
			t.Errorf("other-key got '%v' want 'other-value'", otherSection["other-key"])
		}
	})

	t.Run("SaveConfig with non-existent file - create new", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "snowplow")
		configPath := filepath.Join(configDir, "snowplow.yml")

		originalGetConfigPath := getConfigPath
		getConfigPath = func() string {
			return configPath
		}
		defer func() { getConfigPath = originalGetConfigPath }()

		err := SaveConfig("new-org-id", "new-api-key-id", "new-api-key", "https://new.example.com")
		if err != nil {
			t.Fatal(err)
		}

		if _, err := os.Stat(configDir); os.IsNotExist(err) {
			t.Error("config directory was not created")
		}

		savedData, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		var savedConfig map[string]any
		err = yaml.Unmarshal(savedData, &savedConfig)
		if err != nil {
			t.Fatal(err)
		}

		consoleConfig, ok := savedConfig["console"].(map[string]any)
		if !ok {
			t.Fatal("console section not found or not a map")
		}

		if consoleConfig["api-key"] != "new-api-key" {
			t.Errorf("api-key got '%v' want 'new-api-key'", consoleConfig["api-key"])
		}
		if consoleConfig["api-key-id"] != "new-api-key-id" {
			t.Errorf("api-key-id got '%v' want 'new-api-key-id'", consoleConfig["api-key-id"])
		}
		if consoleConfig["org-id"] != "new-org-id" {
			t.Errorf("org-id got '%v' want 'new-org-id'", consoleConfig["org-id"])
		}
		if consoleConfig["host"] != "https://new.example.com" {
			t.Errorf("host got '%v' want 'https://new.example.com'", consoleConfig["host"])
		}
	})

	t.Run("SaveConfig with default host - no host saved", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "snowplow")
		configPath := filepath.Join(configDir, "snowplow.yml")

		originalGetConfigPath := getConfigPath
		getConfigPath = func() string {
			return configPath
		}
		defer func() { getConfigPath = originalGetConfigPath }()

		err := SaveConfig("org-id", "api-key-id", "api-key", "https://console.snowplowanalytics.com")
		if err != nil {
			t.Fatal(err)
		}

		savedData, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		var savedConfig map[string]any
		err = yaml.Unmarshal(savedData, &savedConfig)
		if err != nil {
			t.Fatal(err)
		}

		consoleConfig, ok := savedConfig["console"].(map[string]any)
		if !ok {
			t.Fatal("console section not found or not a map")
		}

		if _, exists := consoleConfig["host"]; exists {
			t.Error("host should not be saved when it's the default value")
		}

		if consoleConfig["api-key"] != "api-key" {
			t.Errorf("api-key got '%v' want 'api-key'", consoleConfig["api-key"])
		}
		if consoleConfig["api-key-id"] != "api-key-id" {
			t.Errorf("api-key-id got '%v' want 'api-key-id'", consoleConfig["api-key-id"])
		}
		if consoleConfig["org-id"] != "org-id" {
			t.Errorf("org-id got '%v' want 'org-id'", consoleConfig["org-id"])
		}
	})

	t.Run("SaveConfig with corrupted existing file - create new", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := filepath.Join(tempDir, ".config", "snowplow")
		configPath := filepath.Join(configDir, "snowplow.yml")

		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		err = os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0644)
		if err != nil {
			t.Fatal(err)
		}

		originalGetConfigPath := getConfigPath
		getConfigPath = func() string {
			return configPath
		}
		defer func() { getConfigPath = originalGetConfigPath }()

		err = SaveConfig("recovery-org-id", "recovery-api-key-id", "recovery-api-key", "https://recovery.example.com")
		if err != nil {
			t.Fatal(err)
		}

		savedData, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		var savedConfig map[string]any
		err = yaml.Unmarshal(savedData, &savedConfig)
		if err != nil {
			t.Fatal(err)
		}

		consoleConfig, ok := savedConfig["console"].(map[string]any)
		if !ok {
			t.Fatal("console section not found or not a map")
		}

		if consoleConfig["api-key"] != "recovery-api-key" {
			t.Errorf("api-key got '%v' want 'recovery-api-key'", consoleConfig["api-key"])
		}
		if consoleConfig["api-key-id"] != "recovery-api-key-id" {
			t.Errorf("api-key-id got '%v' want 'recovery-api-key-id'", consoleConfig["api-key-id"])
		}
		if consoleConfig["org-id"] != "recovery-org-id" {
			t.Errorf("org-id got '%v' want 'recovery-org-id'", consoleConfig["org-id"])
		}
		if consoleConfig["host"] != "https://recovery.example.com" {
			t.Errorf("host got '%v' want 'https://recovery.example.com'", consoleConfig["host"])
		}
	})
}

func Test_SaveDotenvFile(t *testing.T) {
	t.Run("SaveDotenvFile with existing file - merge values", func(t *testing.T) {
		tempDir := t.TempDir()
		dotenvPath := filepath.Join(tempDir, ".env")

		existingContent := `EXISTING_VAR=existing_value
SNOWPLOW_CONSOLE_API_KEY_ID=old_key_id
OTHER_VAR=other_value`
		err := os.WriteFile(dotenvPath, []byte(existingContent), 0644)
		if err != nil {
			t.Fatal(err)
		}

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

		err = SaveDotenvFile("test-org-id", "test-api-key-id", "test-api-key", "https://test.example.com")
		if err != nil {
			t.Fatal(err)
		}

		dotenvContent, err := godotenv.Read(dotenvPath)
		if err != nil {
			t.Fatal(err)
		}

		if dotenvContent["SNOWPLOW_CONSOLE_API_KEY"] != "test-api-key" {
			t.Errorf("SNOWPLOW_CONSOLE_API_KEY got '%v' want 'test-api-key'", dotenvContent["SNOWPLOW_CONSOLE_API_KEY"])
		}
		if dotenvContent["SNOWPLOW_CONSOLE_API_KEY_ID"] != "test-api-key-id" {
			t.Errorf("SNOWPLOW_CONSOLE_API_KEY_ID got '%v' want 'test-api-key-id'", dotenvContent["SNOWPLOW_CONSOLE_API_KEY_ID"])
		}
		if dotenvContent["SNOWPLOW_CONSOLE_ORG_ID"] != "test-org-id" {
			t.Errorf("SNOWPLOW_CONSOLE_ORG_ID got '%v' want 'test-org-id'", dotenvContent["SNOWPLOW_CONSOLE_ORG_ID"])
		}
		if dotenvContent["SNOWPLOW_CONSOLE_HOST"] != "https://test.example.com" {
			t.Errorf("SNOWPLOW_CONSOLE_HOST got '%v' want 'https://test.example.com'", dotenvContent["SNOWPLOW_CONSOLE_HOST"])
		}

		if dotenvContent["EXISTING_VAR"] != "existing_value" {
			t.Errorf("EXISTING_VAR got '%v' want 'existing_value'", dotenvContent["EXISTING_VAR"])
		}
		if dotenvContent["OTHER_VAR"] != "other_value" {
			t.Errorf("OTHER_VAR got '%v' want 'other_value'", dotenvContent["OTHER_VAR"])
		}
	})

	t.Run("SaveDotenvFile with non-existent file - create new", func(t *testing.T) {
		tempDir := t.TempDir()
		dotenvPath := filepath.Join(tempDir, ".env")

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

		err = SaveDotenvFile("new-org-id", "new-api-key-id", "new-api-key", "https://new.example.com")
		if err != nil {
			t.Fatal(err)
		}

		if _, err := os.Stat(dotenvPath); os.IsNotExist(err) {
			t.Error(".env file was not created")
		}

		dotenvContent, err := godotenv.Read(dotenvPath)
		if err != nil {
			t.Fatal(err)
		}

		if dotenvContent["SNOWPLOW_CONSOLE_API_KEY"] != "new-api-key" {
			t.Errorf("SNOWPLOW_CONSOLE_API_KEY got '%v' want 'new-api-key'", dotenvContent["SNOWPLOW_CONSOLE_API_KEY"])
		}
		if dotenvContent["SNOWPLOW_CONSOLE_API_KEY_ID"] != "new-api-key-id" {
			t.Errorf("SNOWPLOW_CONSOLE_API_KEY_ID got '%v' want 'new-api-key-id'", dotenvContent["SNOWPLOW_CONSOLE_API_KEY_ID"])
		}
		if dotenvContent["SNOWPLOW_CONSOLE_ORG_ID"] != "new-org-id" {
			t.Errorf("SNOWPLOW_CONSOLE_ORG_ID got '%v' want 'new-org-id'", dotenvContent["SNOWPLOW_CONSOLE_ORG_ID"])
		}
		if dotenvContent["SNOWPLOW_CONSOLE_HOST"] != "https://new.example.com" {
			t.Errorf("SNOWPLOW_CONSOLE_HOST got '%v' want 'https://new.example.com'", dotenvContent["SNOWPLOW_CONSOLE_HOST"])
		}
	})

	t.Run("SaveDotenvFile with default host - no host saved", func(t *testing.T) {
		tempDir := t.TempDir()
		dotenvPath := filepath.Join(tempDir, ".env")

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

		err = SaveDotenvFile("org-id", "api-key-id", "api-key", "https://console.snowplowanalytics.com")
		if err != nil {
			t.Fatal(err)
		}

		dotenvContent, err := godotenv.Read(dotenvPath)
		if err != nil {
			t.Fatal(err)
		}

		if _, exists := dotenvContent["SNOWPLOW_CONSOLE_HOST"]; exists {
			t.Error("SNOWPLOW_CONSOLE_HOST should not be saved when it's the default value")
		}

		if dotenvContent["SNOWPLOW_CONSOLE_API_KEY"] != "api-key" {
			t.Errorf("SNOWPLOW_CONSOLE_API_KEY got '%v' want 'api-key'", dotenvContent["SNOWPLOW_CONSOLE_API_KEY"])
		}
		if dotenvContent["SNOWPLOW_CONSOLE_API_KEY_ID"] != "api-key-id" {
			t.Errorf("SNOWPLOW_CONSOLE_API_KEY_ID got '%v' want 'api-key-id'", dotenvContent["SNOWPLOW_CONSOLE_API_KEY_ID"])
		}
		if dotenvContent["SNOWPLOW_CONSOLE_ORG_ID"] != "org-id" {
			t.Errorf("SNOWPLOW_CONSOLE_ORG_ID got '%v' want 'org-id'", dotenvContent["SNOWPLOW_CONSOLE_ORG_ID"])
		}
	})

	t.Run("SaveDotenvFile with empty values", func(t *testing.T) {
		tempDir := t.TempDir()
		dotenvPath := filepath.Join(tempDir, ".env")

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

		err = SaveDotenvFile("", "", "", "https://console.snowplowanalytics.com")
		if err != nil {
			t.Fatal(err)
		}

		dotenvContent, err := godotenv.Read(dotenvPath)
		if err != nil {
			t.Fatal(err)
		}

		if dotenvContent["SNOWPLOW_CONSOLE_API_KEY"] != "" {
			t.Errorf("SNOWPLOW_CONSOLE_API_KEY got '%v' want ''(empty)", dotenvContent["SNOWPLOW_CONSOLE_API_KEY"])
		}
		if dotenvContent["SNOWPLOW_CONSOLE_API_KEY_ID"] != "" {
			t.Errorf("SNOWPLOW_CONSOLE_API_KEY_ID got '%v' want ''(empty)", dotenvContent["SNOWPLOW_CONSOLE_API_KEY_ID"])
		}
		if dotenvContent["SNOWPLOW_CONSOLE_ORG_ID"] != "" {
			t.Errorf("SNOWPLOW_CONSOLE_ORG_ID got '%v' want ''(empty)", dotenvContent["SNOWPLOW_CONSOLE_ORG_ID"])
		}

		if _, exists := dotenvContent["SNOWPLOW_CONSOLE_HOST"]; exists {
			t.Error("SNOWPLOW_CONSOLE_HOST should not be saved when it's the default value")
		}
	})
}
