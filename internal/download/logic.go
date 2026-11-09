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
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/model"
	"github.com/snowplow/snowplow-cli/internal/util"
)

func DownloadDataProductsAndRelatedResources(files util.Files, cnx context.Context, client *console.ApiClient, isPlain bool, withVersions bool) error {
	res, err := console.GetDataProductsAndRelatedResources(cnx, client)
	if err != nil {
		return err
	}

	versionedDpIds := map[string]bool{}
	if withVersions {
		for _, dp := range res.DataProducts {
			versionedDpIds[dp.Id] = true
		}
	} else if _, err := os.Stat(files.DataProductsLocation); err == nil {
		existing := map[string]map[string]any{}
		_ = filepath.WalkDir(files.DataProductsLocation, func(path string, di fs.DirEntry, err error) error {
			if err != nil {
				slog.Debug("download", "msg", "could not read existing file, its versions are not kept", "file", path, "error", err)
				return nil
			}
			ext := filepath.Ext(path)
			if di.IsDir() || (ext != ".json" && ext != ".yaml" && ext != ".yml") {
				return nil
			}
			resources, err := util.MaybeResourcesfromPaths([]string{path})
			if err != nil {
				slog.Debug("download", "msg", "could not read existing file, its versions are not kept", "file", path, "error", err)
				return nil
			}
			for f, resource := range resources {
				existing[f] = resource
			}
			return nil
		})
		versionedDpIds = dataProductsDeclaringVersions(existing)
	}

	sas := remoteSasToLocalResources(res.SourceApplication)

	fileNameToSa, err := files.CreateSourceApps(sas, isPlain)
	if err != nil {
		return err
	}

	slog.Info("download", "msg", "wrote source applications", "count", len(sas))

	saIdToRef, err := localSasToRefs(fileNameToSa, files.DataProductsLocation)
	if err != nil {
		return err
	}

	esIdToRes := groupRemoteEsById(res.EventSpecs)

	triggerIdToFilePath, err := downloadTriggerImages(res.EventSpecs, cnx, client, files)

	dps := remoteDpsToLocalResources(res.DataProducts, saIdToRef, esIdToRes, triggerIdToFilePath, versionedDpIds)

	if err != nil {
		return err
	}

	_, err = files.CreateDataProducts(dps, isPlain)
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

func GetDataStructuresWithOptions(ctx context.Context, client *console.ApiClient, includeDrafts bool, match []string, includeLegacy bool) ([]model.DataStructure, error) {
	dss, err := console.GetAllDataStructures(ctx, client, match, includeLegacy)
	if err != nil {
		return nil, err
	}

	allDss := dss

	if includeDrafts {
		dssDrafts, err := console.GetAllDataStructuresDrafts(ctx, client, match)
		if err != nil {
			return nil, err
		}
		allDss = append(allDss, dssDrafts...)

		slog.Info("fetched data structures drafts", "count", len(dssDrafts))
	}

	slog.Info("fetched data structures", "count", len(dss))

	return allDss, nil
}
