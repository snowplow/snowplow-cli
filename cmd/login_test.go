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
	"github.com/snowplow/snowplow-cli/internal/config"
)

// login reads its configuration before signing in, which needs every console
// flag to be registered on the command.
func Test_LoginCmd_ConfigInit(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	old := auth.Path
	auth.Path = func() string { return filepath.Join(dir, "login.json") }
	t.Cleanup(func() { auth.Path = old })

	if err := LoginCmd.ParseFlags([]string{}); err != nil {
		t.Fatal(err)
	}
	if err := config.InitConsoleConfigForSetup(LoginCmd); err != nil {
		t.Fatalf("not logged in: %v", err)
	}

	// Logging in again must work whatever state the saved login is in
	if err := os.WriteFile(auth.Path(), []byte("{not json"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := config.InitConsoleConfigForSetup(LoginCmd); err != nil {
		t.Fatalf("with an unreadable login file: %v", err)
	}
}
