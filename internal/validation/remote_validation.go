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
	"context"
	"fmt"
	. "github.com/snowplow-product/snowplow-cli/internal/changes"
	"github.com/snowplow-product/snowplow-cli/internal/console"
	"log/slog"
	"strings"
)

type igluValidationLevel uint

const (
	igluValidationInfo igluValidationLevel = iota
	igluValidationWarn
	igluValidationError
)

type igluValidation struct {
	File     string
	Messages []string
	Level    igluValidationLevel
}

type migrationValidation struct {
	File        string
	Suggested   string
	Destination string
	Messages    []string
}

type ValidationResults struct {
	Valid     bool
	Message   string
	Migration []migrationValidation
	Iglu      []igluValidation
}

func (vr *ValidationResults) GithubAnnotate() {
	for _, iglu := range vr.Iglu {
		switch iglu.Level {
		case igluValidationError:
			fmt.Printf("::error file=%s::%s\n", iglu.File, strings.Join(iglu.Messages, "%0A"))
		case igluValidationWarn:
			fmt.Printf("::warning file=%s::%s\n", iglu.File, strings.Join(iglu.Messages, "%0A"))
		case igluValidationInfo:
			fmt.Printf("::notice file=%s::%s\n", iglu.File, strings.Join(iglu.Messages, "%0A"))
		}
	}

	byFile := map[string][]migrationValidation{}
	for _, m := range vr.Migration {
		byFile[m.File] = append(byFile[m.File], m)
	}

	for f, ms := range byFile {
		fmt.Printf("::error file=%s::", f)
		for _, m := range ms {
			fmt.Printf("%%0ASuggested version %s for %s%%0A%s", m.Suggested, m.Destination, strings.Join(m.Messages, "%0A"))
		}
		fmt.Println()
	}
}

func (vr *ValidationResults) Slog() {
	for _, iglu := range vr.Iglu {
		switch iglu.Level {
		case igluValidationError:
			slog.Error("validation", "file", iglu.File, "messages", strings.Join(iglu.Messages, "\n"))
		case igluValidationWarn:
			slog.Warn("validation", "file", iglu.File, "messages", strings.Join(iglu.Messages, "\n"))
		case igluValidationInfo:
			slog.Warn("validation", "file", iglu.File, "messages", strings.Join(iglu.Messages, "\n"))
		}
	}

	for _, migration := range vr.Migration {
		slog.Error("validation", "file", migration.File, "destination", migration.Destination,
			"suggestedVersion", migration.Suggested, "messages", strings.Join(migration.Messages, "\n"))
	}
}

func ValidateChanges(cnx context.Context, c *console.ApiClient, changes Changes) (*ValidationResults, error) {
	var vr ValidationResults

	// Create and create new version both follow the same logic
	// Patch there will error out on validate, we'll implement it separately
	validate := append(append(changes.ToCreate, changes.ToUpdateNewVersion...), changes.ToUpdatePatch...)
	failed := 0
	for _, ds := range validate {
		resp, err := console.Validate(cnx, c, ds.DS)
		if resp != nil {
			if len(resp.Warnings) > 0 {
				vr.Iglu = append(vr.Iglu, igluValidation{ds.FileName, resp.Warnings, igluValidationWarn})
			}
			if len(resp.Info) > 0 {
				vr.Iglu = append(vr.Iglu, igluValidation{ds.FileName, resp.Info, igluValidationInfo})
			}
			if len(resp.Errors) > 0 {
				vr.Iglu = append(vr.Iglu, igluValidation{ds.FileName, resp.Errors, igluValidationError})
				failed++
			}
		}
		if err != nil {
			return nil, err
		}
	}

	migrationsToCheck := append(changes.ToUpdateNewVersion, changes.ToUpdatePatch...)
	for _, ds := range migrationsToCheck {
		result, err := console.ValidateMigrations(cnx, c, ds)
		if err != nil {
			return nil, err
		}
		for dest, r := range result {
			vr.Migration = append(vr.Migration, migrationValidation{ds.FileName, r.SuggestedVersion, dest, r.Messages})
			failed++
		}
	}

	if failed > 0 {
		vr.Valid = false
		vr.Message = fmt.Sprintf("%d validation failures", failed)
	} else {
		vr.Valid = true
	}

	return &vr, nil
}
