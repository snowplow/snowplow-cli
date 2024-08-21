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
		{"api-key-secret", "00beb000-0b0c-00ed-b0ad-000b00a00000"},
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
	t.Setenv("SNOWPLOW_CONSOLE_API_KEY_SECRET", "but not a secret")

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
		{"api-key-secret", "but not a secret"},
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
