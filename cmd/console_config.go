package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func InitConsoleFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP("api-key", "a", "", "BDP console api key")
	cmd.PersistentFlags().StringP("host", "H", "https://console.snowplowanalytics.com", "BDP console host")
	cmd.PersistentFlags().StringP("org-id", "o", "", "Your organization id")
}

func InitConsoleConfig(cmd *cobra.Command) error {
	v := viper.New()
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		configDir := filepath.Join(userConfigDir, "snowplow")
		unixish := filepath.Join(home, ".config", "snowplow")

		slog.Debug(
			"looking for '.snowplow.(toml|yaml|json)'",
			"paths", strings.Join([]string{configDir, unixish, home}, "\n"),
		)

		v.AddConfigPath(home)
		v.AddConfigPath(unixish)
		v.AddConfigPath(configDir)
		v.SetConfigName(".snowplow")
	}

	v.SetEnvPrefix("SNOWPLOW_CONSOLE")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err == nil {
		slog.Debug("found config", "file", v.ConfigFileUsed())
	}

	err := populateCmdConsoleConfigFlags(cmd, v)
	if err != nil {
		return err
	}

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed && v.IsSet(f.Name) {
			cmd.Flags().Set(f.Name, fmt.Sprintf("%s", v.Get(f.Name)))
		}
	})

	return nil
}

func populateCmdConsoleConfigFlags(cmd *cobra.Command, v *viper.Viper) error {
	if !v.IsSet("console") {
		return nil
	}

	m, ok := v.Get("console").(map[string]interface{})
	if !ok {
		return errors.New("console config file parse failure")
	}

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if value, ok := m[f.Name]; !f.Changed && ok {
			cmd.Flags().Set(f.Name, fmt.Sprintf("%s", value))
		}
	})
	return nil
}
