package cmd

import (
	"fmt"
	"os"
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

		v.AddConfigPath(home)
		v.AddConfigPath("$XDG_CONFIG_HOME/snowplow")
		v.SetConfigName(".snowplow")
	}

	v.SetEnvPrefix("SNOWPLOW_CONSOLE")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", v.ConfigFileUsed())
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
		return fmt.Errorf("console config file key parse failure, got: %v", v.Get("console"))
	}

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if value, ok := m[f.Name]; !f.Changed && ok {
			cmd.Flags().Set(f.Name, fmt.Sprintf("%s", value))
		}
	})
	return nil
}
