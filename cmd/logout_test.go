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
	"path/filepath"
	"testing"

	"github.com/snowplow/snowplow-cli/internal/auth"
)

func Test_LogoutCmd_RemovesSavedLogin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "login.json")
	old := auth.Path
	auth.Path = func() string { return path }
	t.Cleanup(func() { auth.Path = old })

	if err := auth.Save(&auth.Login{AccessToken: "token", Host: "https://console.example.com"}); err != nil {
		t.Fatal(err)
	}

	RootCmd.SetArgs([]string{"logout", "--silent"})
	if err := RootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("the login file was not removed")
	}

	RootCmd.SetArgs([]string{"logout", "--silent"})
	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("logging out twice must not fail: %v", err)
	}
}

func Test_LogoutCmd_RejectsArguments(t *testing.T) {
	RootCmd.SetArgs([]string{"logout", "extra"})
	RootCmd.SilenceErrors = true
	t.Cleanup(func() { RootCmd.SilenceErrors = false })
	if err := RootCmd.Execute(); err == nil {
		t.Error("expected an error for an unexpected argument")
	}
}
