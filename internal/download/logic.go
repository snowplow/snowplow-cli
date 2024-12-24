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
	"log/slog"

	"github.com/snowplow-product/snowplow-cli/internal/console"
	"github.com/snowplow-product/snowplow-cli/internal/util"
	"golang.org/x/net/context"
)

func DownloadDataProductsAndRelatedResources(files util.Files, cnx context.Context, client *console.ApiClient) error {
	res, err := console.GetDataProductsAndRelatedResources(cnx, client)
	if err != nil {
		return err
	}

	sas := remoteSasToLocalResources(res.SourceApplication)

	fileNameToSa, err := files.CreateSourceApps(sas)
	if err != nil {
		return err
	}

	slog.Info("download", "msg", "wrote source applications", "count", len(sas))

	saIdToRef := localSasToRefs(fileNameToSa, files.DataProductsLocation)

	esIdToRes := groupRemoteEsById(res.EventSpecs)

	triggerIdToFilePath, err := downloadTriggerImages(res.EventSpecs, cnx, client, files)

	dps := remoteDpsToLocalResources(res.DataProducts, saIdToRef, esIdToRes, triggerIdToFilePath)

	if err != nil {
		return err
	}

	_, err = files.CreateDataProducts(dps)
	if err != nil {
		return err
	}

	slog.Info("download", "msg", "wrote data products", "count", len(dps))
	return nil
}

func downloadTriggerImages(remoteEss []console.RemoteEventSpec, cnx context.Context, client *console.ApiClient, files util.Files) (map[string]string, error) {
	triggerIdToUrl := remoteEsToTriggerIdToUrlAndFilename(remoteEss)
	triggerIdToFilePath := make(map[string]string)
	if len(triggerIdToUrl) != 0 {
		slog.Debug("download", "msg", "will attempt to download trigger images")
		dir, err := files.CreateImageFolder()
		if err != nil {
			return nil, err
		}
		for id, urlAndFilename := range triggerIdToUrl {
			image, ok, err := console.GetImage(cnx, client, urlAndFilename.url)
			if err != nil {
				return nil, err
			}
			// handle 404s
			if ok {
				path, err := files.WriteImage(urlAndFilename.filename, dir, image)
				if err != nil {
					return nil, err
				}
				triggerIdToFilePath[id] = path

			}
		}
		slog.Debug("download", "msg", "wrote trigger images", "count", len(triggerIdToFilePath))
	}
	return triggerIdToFilePath, nil
}
