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
	"slices"
	"strings"

	"github.com/snowplow/snowplow-cli/internal/changes"
	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/logging"
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
			writeGhAnnotation(ghAnnotationError, iglu.File, iglu.Messages...)
		case igluValidationWarn:
			writeGhAnnotation(ghAnnotationWarning, iglu.File, iglu.Messages...)
		case igluValidationInfo:
			writeGhAnnotation(ghAnnotationNotice, iglu.File, iglu.Messages...)
		}
	}

	byFile := map[string][]migrationValidation{}
	for _, m := range vr.Migration {
		byFile[m.File] = append(byFile[m.File], m)
	}

	for f, ms := range byFile {
		lines := []string{""}
		for _, m := range ms {
			lines = append(lines, fmt.Sprintf("Suggested version %s for %s", m.Suggested, m.Destination))
			lines = append(lines, m.Messages...)
		}
		writeGhAnnotation(ghAnnotationError, f, lines...)
	}
}

func (vr *ValidationResults) Slog(ctx context.Context) {
	logger := logging.LoggerFromContext(ctx)
	for _, iglu := range vr.Iglu {
		switch iglu.Level {
		case igluValidationError:
			logger.Error("validation", "file", iglu.File, "messages", strings.Join(iglu.Messages, "\n"))
		case igluValidationWarn:
			logger.Warn("validation", "file", iglu.File, "messages", strings.Join(iglu.Messages, "\n"))
		case igluValidationInfo:
			logger.Warn("validation", "file", iglu.File, "messages", strings.Join(iglu.Messages, "\n"))
		}
	}

	for _, migration := range vr.Migration {
		logger.Error("validation", "file", migration.File, "destination", migration.Destination,
			"suggestedVersion", migration.Suggested, "messages", strings.Join(migration.Messages, "\n"))
	}
}

func (vr *ValidationResults) failures() int {
	failed := len(vr.Migration)
	for _, iglu := range vr.Iglu {
		if iglu.Level == igluValidationError {
			failed++
		}
	}
	return failed
}

func ValidateChanges(cnx context.Context, c *console.ApiClient, changes changes.Changes) (*ValidationResults, error) {
	var vr ValidationResults

	// Create and create new version both follow the same logic
	// Patch there will error out on validate, we'll implement it separately
	validate := append(append(changes.ToCreate, changes.ToUpdateNewVersion...), changes.ToUpdatePatch...)
	for _, ds := range validate {
		resp, err := console.Validate(cnx, c, ds.DS)
		if err != nil {
			return nil, err
		}
		vr.Iglu = append(vr.Iglu, igluValidations(ds.FileName, resp)...)
	}

	migrationsToCheck := slices.Concat(changes.ToUpdateNewVersion, changes.ToUpdatePatch)
	for _, ds := range migrationsToCheck {
		result, err := console.ValidateMigrations(cnx, c, ds)
		if err != nil {
			return nil, err
		}
		for dest, r := range result {
			vr.Migration = append(vr.Migration, migrationValidation{ds.FileName, r.SuggestedVersion, dest, r.Messages})
		}
	}

	if failed := vr.failures(); failed > 0 {
		vr.Valid = false
		vr.Message = fmt.Sprintf("%d validation failures", failed)
	} else {
		vr.Valid = true
	}

	return &vr, nil
}

func igluValidations(fileName string, resp *console.ValidateResponse) []igluValidation {
	var res []igluValidation
	if len(resp.Warnings) > 0 {
		res = append(res, igluValidation{fileName, resp.Warnings, igluValidationWarn})
	}
	if len(resp.Info) > 0 {
		res = append(res, igluValidation{fileName, resp.Info, igluValidationInfo})
	}
	if len(resp.Errors) > 0 {
		res = append(res, igluValidation{fileName, resp.Errors, igluValidationError})
	}
	return res
}
