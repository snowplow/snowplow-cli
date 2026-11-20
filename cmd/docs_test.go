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
	"slices"
	"testing"
)

func TestDedent(t *testing.T) {
	got := dedent([]string{
		"",
		"\t$ snowplow-cli ds publish prod",
		"\t  --dry-run",
		"",
		"\t$ snowplow-cli ds validate",
		"\t",
	})
	want := []string{
		"$ snowplow-cli ds publish prod",
		"  --dry-run",
		"",
		"$ snowplow-cli ds validate",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDedentEmpty(t *testing.T) {
	if got := dedent([]string{"", "  "}); len(got) != 0 {
		t.Fatalf("got %q, want no lines", got)
	}
}
