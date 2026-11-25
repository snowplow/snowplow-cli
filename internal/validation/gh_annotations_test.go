/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package validation

import (
	"io"
	"os"
	"testing"
)

func stdoutInTest(t *testing.T, run func()) string {
	t.Helper()
	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	os.Stdout = w
	run()
	os.Stdout = stdout
	_ = w.Close()
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(got)
}

func Test_writeGhAnnotation(t *testing.T) {
	tests := []struct {
		name  string
		level ghAnnotationLevel
		lines []string
		want  string
	}{
		{name: "notice", level: ghAnnotationNotice, lines: []string{"one"}, want: "::notice file=dp.yml::one\n"},
		{name: "warning", level: ghAnnotationWarning, lines: []string{"one"}, want: "::warning file=dp.yml::one\n"},
		{name: "error", level: ghAnnotationError, lines: []string{"one"}, want: "::error file=dp.yml::one\n"},
		{name: "several lines", level: ghAnnotationError, lines: []string{"one", "two", "three"}, want: "::error file=dp.yml::one%0Atwo%0Athree\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stdoutInTest(t, func() { writeGhAnnotation(tt.level, "dp.yml", tt.lines...) })
			if got != tt.want {
				t.Errorf("got: %q want: %q", got, tt.want)
			}
		})
	}
}

func Test_GhAnnotateValidations_UsesGitHubLevels(t *testing.T) {
	tests := []struct {
		name        string
		validations DPValidations
		want        string
	}{
		{name: "nothing", validations: DPValidations{}, want: ""},
		{name: "info", validations: DPValidations{Info: []string{"i1", "i2"}}, want: "::notice file=dp.yml::i1%0Ai2\n"},
		{name: "warnings", validations: DPValidations{Warnings: []string{"w1", "w2"}}, want: "::warning file=dp.yml::w1%0Aw2\n"},
		{name: "errors", validations: DPValidations{Errors: []string{"e1", "e2"}}, want: "::error file=dp.yml::e1%0Ae2\n"},
		{name: "warnings with paths", validations: DPValidations{WarningsWithPaths: map[string][]string{"/a": {"w1", "w2"}}}, want: "::warning file=dp.yml::/a%0Aw1%0Aw2\n"},
		{name: "errors with paths", validations: DPValidations{ErrorsWithPaths: map[string][]string{"/a": {"e1", "e2"}}}, want: "::error file=dp.yml::/a%0Ae1%0Ae2\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := &DPLookup{Validations: map[string]DPValidations{"/base/dp.yml": tt.validations}}

			var annotateErr error
			got := stdoutInTest(t, func() { annotateErr = lookup.GhAnnotateValidations("/base") })

			if annotateErr != nil {
				t.Fatalf("unexpected error: %v", annotateErr)
			}
			if got != tt.want {
				t.Errorf("got: %q want: %q", got, tt.want)
			}
		})
	}
}

func Test_ValidationResults_GithubAnnotate(t *testing.T) {
	tests := []struct {
		name    string
		results ValidationResults
		want    string
	}{
		{name: "nothing", results: ValidationResults{}, want: ""},
		{
			name:    "iglu info",
			results: ValidationResults{Iglu: []igluValidation{{File: "ds.yml", Messages: []string{"i1", "i2"}, Level: igluValidationInfo}}},
			want:    "::notice file=ds.yml::i1%0Ai2\n",
		},
		{
			name:    "iglu warning",
			results: ValidationResults{Iglu: []igluValidation{{File: "ds.yml", Messages: []string{"w1", "w2"}, Level: igluValidationWarn}}},
			want:    "::warning file=ds.yml::w1%0Aw2\n",
		},
		{
			name:    "iglu error",
			results: ValidationResults{Iglu: []igluValidation{{File: "ds.yml", Messages: []string{"e1", "e2"}, Level: igluValidationError}}},
			want:    "::error file=ds.yml::e1%0Ae2\n",
		},
		{
			name: "migrations of one file",
			results: ValidationResults{Migration: []migrationValidation{
				{File: "ds.yml", Suggested: "2-0-0", Destination: "snowflake", Messages: []string{"m1", "m2"}},
				{File: "ds.yml", Suggested: "1-1-0", Destination: "bigquery", Messages: []string{"m3"}},
			}},
			want: "::error file=ds.yml::%0ASuggested version 2-0-0 for snowflake%0Am1%0Am2%0ASuggested version 1-1-0 for bigquery%0Am3\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stdoutInTest(t, func() { tt.results.GithubAnnotate() })
			if got != tt.want {
				t.Errorf("got: %q want: %q", got, tt.want)
			}
		})
	}
}
