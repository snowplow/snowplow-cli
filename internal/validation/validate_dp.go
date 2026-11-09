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

	"github.com/snowplow/snowplow-cli/internal/amend"
	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/model"
	"github.com/snowplow/snowplow-cli/internal/release"
	"github.com/snowplow/snowplow-cli/internal/util"
	"github.com/spf13/cobra"
)

func ValidateDataProductsFromCmd(ctx context.Context, cmd *cobra.Command, paths []string, basePath string) error {
	apiKeyId, _ := cmd.Flags().GetString("api-key-id")
	apiKeySecret, _ := cmd.Flags().GetString("api-key")
	host, _ := cmd.Flags().GetString("host")
	org, _ := cmd.Flags().GetString("org-id")
	ghOut, _ := cmd.Flags().GetBool("gh-annotate")
	full, _ := cmd.Flags().GetBool("full")
	concurrentReq, _ := cmd.Flags().GetInt("concurrency")
	fix, _ := cmd.Flags().GetBool("fix")

	c, err := console.NewApiClient(ctx, host, apiKeyId, apiKeySecret, org)
	if err != nil {
		return err
	}

	return ValidateDataProductsWithClient(ctx, c, paths, basePath, ghOut, full, concurrentReq, fix)
}

func ValidateDataProductsWithClient(ctx context.Context, client *console.ApiClient, paths []string, basePath string, ghOut bool, full bool, concurrentReq int, fix bool) error {
	logger := logging.LoggerFromContext(ctx)

	searchPaths := paths

	if concurrentReq > 10 {
		concurrentReq = 10
		logger.Debug("validation", "msg", "concurrency set to > 10, limited to 10")
	}

	if concurrentReq < 1 {
		concurrentReq = 1
		logger.Debug("validation", "msg", "concurrency set to < 1, increased to 1")
	}

	files, err := util.MaybeResourcesfromPaths(searchPaths)
	if err != nil {
		return err
	}

	changes, err := release.FindChanges(ctx, client, files)
	if err != nil {
		return err
	}

	versionMismatches := changes.VersionMismatches
	if fix {
		err = fixEventSpecVersions(ctx, versionMismatches)
		if err != nil {
			return err
		}
		versionMismatches = nil

		files, err = util.MaybeResourcesfromPaths(searchPaths)
		if err != nil {
			return err
		}
	}

	return Validate(ctx, client, files, searchPaths, basePath, ghOut, full, changes.IdToFileName, versionMismatches, concurrentReq)
}

func ValidateDataProductsAgainstBaseFromCmd(ctx context.Context, cmd *cobra.Command, paths []string, basePath string) error {
	ghOut, _ := cmd.Flags().GetBool("gh-annotate")
	baseRef, _ := cmd.Flags().GetString("base-ref")
	fix, _ := cmd.Flags().GetBool("fix")

	return ValidateDataProductsAgainstBase(ctx, paths, basePath, ghOut, baseRef, fix)
}

func ValidateDataProductsAgainstBase(ctx context.Context, paths []string, basePath string, ghOut bool, baseRef string, fix bool) error {
	logger := logging.LoggerFromContext(ctx)

	files, err := util.MaybeResourcesfromPaths(paths)
	if err != nil {
		return err
	}

	baseFiles, err := util.MaybeResourcesAtMergeBase(ctx, basePath, baseRef, paths)
	if err != nil {
		return err
	}

	versionMismatches, skipped, err := release.FindVersionMismatchesAgainstBase(files, baseFiles)
	if err != nil {
		return err
	}

	for _, name := range skipped {
		logger.Info("validation", "msg", "skipping version check, no version declared at base", "name", name, "base", baseRef)
	}

	if fix {
		err = fixEventSpecVersions(ctx, versionMismatches)
		if err != nil {
			return err
		}
		versionMismatches = nil

		files, err = util.MaybeResourcesfromPaths(paths)
		if err != nil {
			return err
		}
	}

	return ValidateLocal(ctx, files, paths, basePath, ghOut, versionMismatches)
}

func fixEventSpecVersions(ctx context.Context, versionMismatches []model.EventSpecVersionMismatch) error {
	logger := logging.LoggerFromContext(ctx)

	mismatchesByFile := map[string][]model.EventSpecVersionMismatch{}
	for _, m := range versionMismatches {
		mismatchesByFile[m.FileName] = append(mismatchesByFile[m.FileName], m)
	}

	for f, mismatches := range mismatchesByFile {
		versions := map[int]int{}
		for _, m := range mismatches {
			versions[m.Index] = m.Expected
		}
		err := amend.SetEventSpecVersionsInFile(versions, f)
		if err != nil {
			return err
		}
		for _, m := range mismatches {
			logger.Info("validation", "msg", "updated event specification version", "file", f, "name", m.Name, "from", m.Declared, "to", m.Expected)
		}
	}

	return nil
}
