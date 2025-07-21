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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

const envNamePrefix = "SNOWPLOW_CONSOLE_"

func InitConsoleFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("api-key-id", "a", "", "BDP console api key id")
	cmd.PersistentFlags().StringP("api-key", "S", "", "BDP console api key")
	cmd.PersistentFlags().StringP("host", "H", "https://console.snowplowanalytics.com", "BDP console host")
	cmd.PersistentFlags().StringP("org-id", "o", "", "Your organization id")
	cmd.PersistentFlags().StringP("managed-from", "m", "", "Link to a github repo where the data structure is managed")
}

type rawAppConfig struct {
	Console map[string]string
}

func loadEnvFiles(cmd *cobra.Command, baseDir string) error {
	var envFilePaths []string
	var isExplicitFile bool

	if envFile, _ := cmd.Flags().GetString("env-file"); envFile != "" {
		envFilePaths = append(envFilePaths, envFile)
		isExplicitFile = true
	} else {
		workingDir := baseDir
		if workingDir == "" {
			cwd, err := os.Getwd()
			if err == nil {
				workingDir = cwd
			}
		}

		if workingDir != "" {
			envFilePaths = append(envFilePaths, filepath.Join(workingDir, ".env"))
		}

		home, err := os.UserHomeDir()
		if err == nil {
			envFilePaths = append(envFilePaths, filepath.Join(home, ".config", "snowplow", ".env"))
		}

		userConfigDir, err := os.UserConfigDir()
		if err == nil {
			envFilePaths = append(envFilePaths, filepath.Join(userConfigDir, "snowplow", ".env"))
		}
	}

	slog.Debug("looking for .env files at", "paths", strings.Join(envFilePaths, "\n"))

	for _, path := range envFilePaths {
		if _, err := os.Stat(path); err == nil {
			slog.Debug(".env file found at", "file", path)
			if err := godotenv.Load(path); err != nil {
				return fmt.Errorf("failed to load .env file %s: %w", path, err)
			}
			slog.Debug(".env file loaded successfully", "file", path)
			return nil
		} else {
			slog.Debug(".env file not found at", "file", path, "err", err)
			if isExplicitFile {
				return fmt.Errorf("specified .env file not found: %s", path)
			}
		}
	}

	slog.Debug("no .env file found")
	return nil
}

func InitConsoleConfig(cmd *cobra.Command) error {
	return initConsoleConfigWithOptions(cmd, false, "")
}

func InitConsoleConfigForSetup(cmd *cobra.Command) error {
	return initConsoleConfigWithOptions(cmd, true, "")
}

func InitConsoleConfigWithBaseDir(cmd *cobra.Command, baseDir string) error {
	return initConsoleConfigWithOptions(cmd, false, baseDir)
}

func initConsoleConfigWithOptions(cmd *cobra.Command, skipMissingCheck bool, baseDir string) error {

	if err := loadEnvFiles(cmd, baseDir); err != nil {
		return fmt.Errorf("failed to load .env file: %w", err)
	}

	var configBytes []byte
	var err error
	var potentialConfigs []string

	if configFileName, _ := cmd.Flags().GetString("config"); configFileName != "" {
		potentialConfigs = append(potentialConfigs, configFileName)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(userConfigDir, "snowplow", "snowplow.yml")
	unixish := filepath.Join(home, ".config", "snowplow", "snowplow.yml")

	paths := []string{unixish, configDir}

	potentialConfigs = append(potentialConfigs, paths...)

	slog.Debug("looking for config at", "paths", strings.Join(potentialConfigs, "\n"))

	for _, p := range potentialConfigs {
		configBytes, err = os.ReadFile(p)
		if err != nil {
			slog.Debug("config not found at", "file", p, "err", err)
		} else {
			slog.Debug("config found at", "file", p)
			break
		}
	}

	var config rawAppConfig
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		return err
	}

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if value, ok := config.Console[f.Name]; ok && !f.Changed && err == nil {
			err = cmd.Flags().Set(f.Name, value)
			slog.Debug("config value found in file", "flag", f.Name)
		}
	})

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		envName := toEnvName(f.Name)
		if value, ok := os.LookupEnv(envName); err == nil && ok && value != "" {
			err = cmd.Flags().Set(f.Name, value)
			slog.Debug("config value found in env", "flag", f.Name, "env", envName)
		}
	})

	var missingVars []string
	for _, f := range []string{"api-key-id", "api-key", "host", "org-id"} {
		value, err := cmd.Flags().GetString(f)
		if err != nil {
			return err
		}
		if value == "" {
			missingVars = append(missingVars, f)
		}
	}

	if len(missingVars) > 0 && !skipMissingCheck {
		var errorMsg strings.Builder
		if len(missingVars) == 1 {
			errorMsg.WriteString(fmt.Sprintf(`config value "%s" not set`, missingVars[0]))
		} else {
			errorMsg.WriteString(fmt.Sprintf(`config values not set: %s`, strings.Join(missingVars, ", ")))
		}

		errorMsg.WriteString(`

Configuration can be provided via:
  1. Command-line flags: --<name> <value>
  2. Environment variables: SNOWPLOW_CONSOLE_<NAME>=<value>
  3. Environment file (.env): SNOWPLOW_CONSOLE_<NAME>=<value>
  4. Config file (snowplow.yml): console.<name>: <value>

Required variables:`)

		for _, v := range missingVars {
			errorMsg.WriteString(fmt.Sprintf(`
  %s: --%s or %s`, v, v, toEnvName(v)))
		}

		errorMsg.WriteString(`

See 'snowplow-cli --help' for config file and .env file locations.
Get API credentials at: https://docs.snowplow.io/docs/using-the-snowplow-console/managing-console-api-authentication/`)

		return errors.New(errorMsg.String())
	}

	if err != nil {
		return err
	}

	return nil
}

func PersistConfig(orgID, apiKeyID, apiKeySecret, consoleHost string, isDotEnv bool) error {
	if isDotEnv {
		return SaveDotenvFile(orgID, apiKeyID, apiKeySecret, consoleHost)
	} else {
		return SaveConfig(orgID, apiKeyID, apiKeySecret, consoleHost)
	}
}

func SaveConfig(orgID, apiKeyID, apiKeySecret, consoleHost string) error {
	green := color.New(color.FgGreen)
	configPath := getConfigPath()
	slog.Debug("Saving configuration to file", "config-path", configPath)

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	var existingConfig map[string]any
	if existingData, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(existingData, &existingConfig); err != nil {
			slog.Warn("Failed to parse existing config, creating new one", "error", err)
			existingConfig = make(map[string]any)
		}
	} else {
		existingConfig = make(map[string]any)
	}

	if existingConfig["console"] == nil {
		existingConfig["console"] = make(map[string]any)
	}

	consoleConfig, ok := existingConfig["console"].(map[string]any)
	if !ok {
		consoleConfig = make(map[string]any)
		existingConfig["console"] = consoleConfig
	}

	consoleConfig["api-key"] = apiKeySecret
	consoleConfig["api-key-id"] = apiKeyID
	consoleConfig["org-id"] = orgID

	// Only save host if it's not the default production value
	if consoleHost != "https://console.snowplowanalytics.com" {
		consoleConfig["host"] = consoleHost
	}

	// Note: We don't save auth0-domain or client-id to config file
	// These should be provided via environment variables or command line flags

	data, err := yaml.Marshal(existingConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	green.Printf("✓ Configuration saved to %s\n", configPath)

	return nil
}

func SaveDotenvFile(orgID, apiKeyID, apiKeySecret, consoleHost string) error {
	green := color.New(color.FgGreen)
	dotenvPath := ".env"
	slog.Debug("Saving configuration to dot env file", "config-path", dotenvPath)

	var dotenvContent map[string]string
	if existingContent, err := godotenv.Read(); err == nil {
		dotenvContent = existingContent
	} else {
		dotenvContent = make(map[string]string)
	}

	dotenvContent[toEnvName("api-key")] = apiKeySecret
	dotenvContent[toEnvName("api-key-id")] = apiKeyID
	dotenvContent[toEnvName("org-id")] = orgID

	// Only save host if it's not the default production value
	if consoleHost != "https://console.snowplowanalytics.com" {
		dotenvContent[toEnvName("host")] = consoleHost
	}

	err := godotenv.Write(dotenvContent, dotenvPath)
	if err == nil {
		green.Printf("✓ Configuration saved to %s\n", dotenvPath)
		return nil
	} else {
		return err
	}
}

// getConfigPath is a variable holding the function to get config path (for testability)
var getConfigPath = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "snowplow", "snowplow.yml")
}

func toEnvName(s string) string {
	return envNamePrefix + strings.ReplaceAll(strings.ToUpper(s), "-", "_")
}
