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
	"errors"

	"github.com/snowplow/snowplow-cli/internal/changes"
	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/util"
	"github.com/spf13/cobra"
)

func ValidateDataStructuresFromCmd(ctx context.Context, cmd *cobra.Command, paths []string) error {
	apiKeyId, _ := cmd.Flags().GetString("api-key-id")
	apiKeySecret, _ := cmd.Flags().GetString("api-key")
	host, _ := cmd.Flags().GetString("host")
	org, _ := cmd.Flags().GetString("org-id")
	ghOut, _ := cmd.Flags().GetBool("gh-annotate")

	c, err := console.NewApiClient(ctx, host, apiKeyId, apiKeySecret, org)
	if err != nil {
		return err
	}

	return ValidateDataStructuresWithClient(ctx, c, paths, ghOut)
}

func ValidateDataStructuresWithClient(ctx context.Context, client *console.ApiClient, paths []string, ghOut bool) error {
	logger := logging.LoggerFromContext(ctx)

	dataStructureFolders := []string{util.DataStructuresFolder}
	if len(paths) > 0 {
		dataStructureFolders = paths
	}

	dataStructuresLocal, err := util.DataStructuresFromPaths(dataStructureFolders)
	logger.Info("validating from", "paths", dataStructureFolders)
	if err != nil {
		return err
	}

	errs := ValidateLocalDs(dataStructuresLocal)
	if len(errs) > 0 {
		logger.Error("validation", "error", errs)
		return errors.Join(errs...)
	}

	remotesListing, err := console.GetDataStructureListing(ctx, client)
	if err != nil {
		return err
	}

	changed, err := changes.GetChanges(dataStructuresLocal, remotesListing, "DEV")
	if err != nil {
		return err
	}

	err = changes.PrintChangeset(ctx, changed)
	if err != nil {
		return err
	}

	vr, err := ValidateChanges(ctx, client, changed)
	if err != nil {
		return err
	}

	vr.Slog(ctx)

	if ghOut {
		vr.GithubAnnotate()
	}

	if !vr.Valid {
		return errors.New(vr.Message)
	}

	return nil
}
