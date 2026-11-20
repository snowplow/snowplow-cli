/**
 * Copyright (c) 2013-present Snowplow Analytics Ltd.
 * All rights reserved.
 * This software is made available by Snowplow Analytics, Ltd.,
 * under the terms of the Snowplow Limited Use License Agreement, Version 1.0
 * located at https://docs.snowplow.io/limited-use-license-1.0
 * BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
 * OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
 */

package dp

import (
	"bytes"
	"log/slog"
	"slices"
	"strings"
	"testing"
)

func captureDebugLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

func Test_limitConcurrencyAndLogChange(t *testing.T) {
	for _, tt := range []struct {
		name        string
		input       int
		expected    int
		expectedLog string
	}{
		{name: "above the maximum", input: 11, expected: 10, expectedLog: "concurrency set to > 10, limited to 10"},
		{name: "the maximum", input: 10, expected: 10},
		{name: "the default", input: 3, expected: 3},
		{name: "the minimum", input: 1, expected: 1},
		{name: "zero", input: 0, expected: 1, expectedLog: "concurrency set to < 1, increased to 1"},
		{name: "negative", input: -5, expected: 1, expectedLog: "concurrency set to < 1, increased to 1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			log := captureDebugLog(t)

			got := limitConcurrencyAndLogChange(tt.input)

			if got != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, got)
			}
			if tt.expectedLog == "" && log.Len() != 0 {
				t.Errorf("expected no log, got %q", log.String())
			}
			if !strings.Contains(log.String(), tt.expectedLog) {
				t.Errorf("expected log to contain %q, got %q", tt.expectedLog, log.String())
			}
		})
	}
}

func Test_searchPathsOrDefaultAndLogIt_NoArgs(t *testing.T) {
	log := captureDebugLog(t)

	got := searchPathsOrDefaultAndLogIt([]string{})

	if !slices.Equal(got, []string{"data-products"}) {
		t.Errorf("expected the default folder, got %v", got)
	}
	if !strings.Contains(log.String(), "no path provided, using default (./data-products)") {
		t.Errorf("unexpected log: %q", log.String())
	}
}

func Test_searchPathsOrDefaultAndLogIt_Args(t *testing.T) {
	for _, args := range [][]string{{"./one"}, {"./one", "./two"}} {
		log := captureDebugLog(t)

		got := searchPathsOrDefaultAndLogIt(args)

		if !slices.Equal(got, args) {
			t.Errorf("expected the given paths in order, got %v", got)
		}
		if log.Len() != 0 {
			t.Errorf("expected no log, got %q", log.String())
		}
	}
}
