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
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

func InitConsoleFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("api-key-id", "a", "", "BDP console api key id")
	cmd.PersistentFlags().StringP("api-key-secret", "S", "", "BDP console api key secret")
	cmd.PersistentFlags().StringP("host", "H", "https://console.snowplowanalytics.com", "BDP console host")
	cmd.PersistentFlags().StringP("org-id", "o", "", "Your organization id")
	cmd.PersistentFlags().StringP("managed-from", "m", "", "Link to a github repo where the data structure is managed")
}

type rawAppConfig struct {
	Console map[string]string
}

func InitConsoleConfig(cmd *cobra.Command) error {

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

	for _, f := range []string{"api-key-id", "api-key-secret", "host", "org-id"} {
		value, err := cmd.Flags().GetString(f)
		if err != nil {
			return err
		}
		if value == "" {
			return fmt.Errorf(`config value "%s" not set`, f)
		}
	}

	if err != nil {
		return err
	}

	return nil
}
