/**
 * Copyright (c) 2013-present Snowplow Analytics Ltd.
 * All rights reserved.
 * This software is made available by Snowplow Analytics, Ltd.,
 * under the terms of the Snowplow Limited Use License Agreement, Version 1.0
 * located at https://docs.snowplow.io/limited-use-license-1.0
 * BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
 * OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
 */

package validation

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/snowplow/snowplow-cli/internal/console"
	snplog "github.com/snowplow/snowplow-cli/internal/logging"
)

func Validate(cnx context.Context, c *console.ApiClient, files map[string]map[string]any, searchPaths []string, basePath string, ghOut bool, validateAll bool, changedIdToFile map[string]string, concurrency int) {
	possibleFiles := []string{}
	for n := range files {
		possibleFiles = append(possibleFiles, n)
	}

	schemaResolver, err := console.NewSchemaDeployChecker(cnx, c)
	if err != nil {
		snplog.LogFatal(err)
	}

	compatChecker := func(event console.CompatCheckable, entities []console.CompatCheckable) (*console.CompatResult, error) {
		return console.CompatCheck(cnx, c, event, entities)
	}

	lookup, err := NewDPLookup(compatChecker, schemaResolver, files, changedIdToFile, validateAll, concurrency)
	if err != nil {
		snplog.LogFatal(err)
	}

	slog.Debug("validation", "msg", "from", "paths", searchPaths, "files", possibleFiles)

	err = lookup.SlogValidations(basePath)
	if err != nil {
		snplog.LogFatal(err)
	}

	if ghOut {
		err := lookup.GhAnnotateValidations(basePath)
		if err != nil {
			snplog.LogFatal(err)
		}
	}

	numErrors := lookup.ValidationErrorCount()

	if numErrors > 0 {
		snplog.LogFatal(fmt.Errorf("validation failed %d errors", numErrors))
	} else {
		dpCount := 0
		for range lookup.DataProducts {
			dpCount++
		}
		saCount := 0
		for range lookup.SourceApps {
			saCount++
		}
		slog.Info("validation", "msg", "success", "data products found", dpCount, "source applications found", saCount)
	}
}
