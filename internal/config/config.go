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

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

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
	return InitConsoleConfigWithBaseDir(cmd, "")
}

func InitConsoleConfigWithBaseDir(cmd *cobra.Command, baseDir string) error {

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
		name := strings.ReplaceAll(strings.ToUpper(f.Name), "-", "_")
		envName := "SNOWPLOW_CONSOLE_" + name
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

	if len(missingVars) > 0 {
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
			envName := strings.ReplaceAll(strings.ToUpper(v), "-", "_")
			errorMsg.WriteString(fmt.Sprintf(`
  %s: --%s or SNOWPLOW_CONSOLE_%s`, v, v, envName))
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
