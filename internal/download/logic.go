/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/
package download

import (
	"github.com/snowplow-product/snowplow-cli/internal/console"
	"github.com/snowplow-product/snowplow-cli/internal/util"
	"golang.org/x/net/context"
	"log/slog"
)

func DownloadDataProductsAndRelatedResources(files util.Files, cnx context.Context, client *console.ApiClient) error {
	res, err := console.GetDataProductsAndRelatedResources(cnx, client)
	if err != nil {
		return err
	}

	sas := RemoteSasToLocalResources(res.SourceApplication)

	fileNameToSa, err := files.CreateSourceApps(sas)
	if err != nil {
		return err
	}

	slog.Info("wrote source applications", "count", len(sas))

	saIdToRef := LocalSasToRefs(fileNameToSa, files.DataProductsLocation)

	esIdToRes := GroupRemoteEsById(res.TrackingScenarios)

	dps := RemoteDpsToLocalResources(res.DataProducts, saIdToRef, esIdToRes)

	_, err = files.CreateDataProducts(dps)
	if err != nil {
		return err
	}

	slog.Info("wrote data products", "count", len(dps))
	return nil
}
