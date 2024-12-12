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
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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

	imgToFile, err := createImages(esIdToRes, res.DataProducts, cnx, client, files)
	if err != nil {
		return err
	}
	slog.Info("download", "msg", "wrote images", "count", len(imgToFile), "img", imgToFile)

	dps := remoteDpsToLocalResources(res.DataProducts, saIdToRef, esIdToRes)

	_, err = files.CreateDataProducts(dps)
	if err != nil {
		return err
	}

	slog.Info("download", "msg", "wrote data products", "count", len(dps))
	return nil
}

func createImages(
	esLookupById map[string]console.RemoteEventSpec,
	rdp []console.RemoteDataProduct,
	cnx context.Context,
	client *console.ApiClient,
	files util.Files,
) (map[string]string, error) {

	imgToFile := map[string]string{}
	for _, d := range rdp {
		for _, eId := range d.EventSpecs {
			if e, ok := esLookupById[eId.Id]; ok {
				triggersWithImg := 0
				for _, t := range e.Triggers {
					if original, ok := t.VariantUrls["original"]; ok {
						img, err := console.GetImage(cnx, client, original)
						if err != nil {
							return nil, err
						}
						fname := util.ResourceNameToFileName(e.Name) + img.Ext
						if triggersWithImg > 0 {
							fname = util.ResourceNameToFileName(e.Name) + fmt.Sprintf("_%d", triggersWithImg) + img.Ext
						}
						cwd, err := os.Getwd()
						if err != nil {
							return nil, err
						}
						path := filepath.Join(cwd, files.DataProductsLocation, files.ImagesLocation)
						err = os.MkdirAll(path, os.ModePerm)

						if err != nil {
							return nil, err
						}

						fpath := filepath.Join(path, fname)
						err = os.WriteFile(fpath, img.Data, 0644)
						if err != nil {
							return nil, err
						}

						absp, err := filepath.Abs(fpath)
						if err != nil {
							return nil, err
						}
						imgToFile[original] = absp

						triggersWithImg++
					}
				}
			}
		}
	}
	return imgToFile, nil
}
