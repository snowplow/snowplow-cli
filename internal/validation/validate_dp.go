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

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/publish"
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

	c, err := console.NewApiClient(ctx, host, apiKeyId, apiKeySecret, org)
	if err != nil {
		return err
	}

	return ValidateDataProductsWithClient(ctx, c, paths, basePath, ghOut, full, concurrentReq)
}

func ValidateDataProductsWithClient(ctx context.Context, client *console.ApiClient, paths []string, basePath string, ghOut bool, full bool, concurrentReq int) error {
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

	changes, err := publish.FindChanges(ctx, client, files)
	if err != nil {
		return err
	}

	return Validate(ctx, client, files, searchPaths, basePath, ghOut, full, changes.IdToFileName, concurrentReq)
}
