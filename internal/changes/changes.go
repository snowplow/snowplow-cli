/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package changes

import (
	"context"
	"errors"
	"log/slog"
	"reflect"

	"github.com/r3labs/diff/v3"
	. "github.com/snowplow-product/snowplow-cli/internal/console"
	. "github.com/snowplow-product/snowplow-cli/internal/model"
)

type DataStructureWithDiff struct {
	DataStructure DataStructure
	Operation     string
	Diff          diff.Changelog
}

type DataStructureId struct {
	Vendor string
	Name   string
	Format string
}

func idFromSelf(self DataStructureSelf) DataStructureId {
	return DataStructureId{
		self.Vendor,
		self.Name,
		self.Format,
	}
}

func NewDSChangeContext(ds DataStructure, fileName string) DSChangeContext {
	return DSChangeContext{DS: ds, FileName: fileName, RemoteVersion: "", LocalContentHash: "", RemoteContentHash: ""}
}

func NewDSChangeContextWithVersion(ds DataStructure, fileName string, v string) DSChangeContext {
	return DSChangeContext{DS: ds, FileName: fileName, RemoteVersion: v, LocalContentHash: "", RemoteContentHash: ""}
}

func NewDSChangeContextWithVersionAndHashes(ds DataStructure, fileName string, v string, localHash string, remoteHash string) DSChangeContext {
	return DSChangeContext{DS: ds, FileName: fileName, RemoteVersion: v, LocalContentHash: localHash, RemoteContentHash: remoteHash}
}

type Changes struct {
	ToCreate           []DSChangeContext
	ToUpdateMeta       []DSChangeContext
	ToUpdateNewVersion []DSChangeContext
	ToUpdatePatch      []DSChangeContext
}

func GetChanges(locals map[string]DataStructure, remoteListing []ListResponse, env DataStructureEnv) (Changes, error) {
	res := Changes{
		make([]DSChangeContext, 0),
		make([]DSChangeContext, 0),
		make([]DSChangeContext, 0),
		make([]DSChangeContext, 0),
	}
	remotesSet := make(map[DataStructureId]ListResponse)
	for _, remote := range remoteListing {
		remotesSet[DataStructureId{remote.Vendor, remote.Name, remote.Format}] = remote
	}

	for f, ds := range locals {
		data, err := ds.ParseData()
		if err != nil {
			return Changes{}, err
		}
		remotePair, exists := remotesSet[idFromSelf(data.Self)]
		// DS does not exists, we should create it
		if !exists {
			res.ToCreate = append(res.ToCreate, NewDSChangeContext(ds, f))
		} else {
			//Remote DS exists,
			if !reflect.DeepEqual(ds.Meta, remotePair.Meta) {
				// Meta is different, needs updating
				res.ToUpdateMeta = append(res.ToUpdateMeta, NewDSChangeContext(ds, f))
			}
			contentHash, err := ds.GetContentHash()
			if err != nil {
				return Changes{}, err
			}
			var foundDeployment bool
			// find the correct deployment to compare to
			for _, deployment := range remotePair.Deployments {
				if deployment.Env == env {
					foundDeployment = true
					if deployment.ContentHash != contentHash {
						// data structure has changed
						if data.Self.Version != deployment.Version {
							// Different version, create new version
							res.ToUpdateNewVersion = append(res.ToUpdateNewVersion, NewDSChangeContextWithVersion(ds, f, deployment.Version))
						} else {
							// Same version, but different hash, patch
							res.ToUpdatePatch = append(res.ToUpdatePatch, NewDSChangeContextWithVersionAndHashes(ds, f, deployment.Version, contentHash, deployment.ContentHash))
						}
					}
				}
			}
			if !foundDeployment {
				// DS exists, but we didn't find a version of it
				// We should deploy from dev to prod
				res.ToUpdateNewVersion = append(res.ToUpdateNewVersion, NewDSChangeContextWithVersion(ds, f, ""))
			}
		}
	}
	return res, nil
}

func PerformChangesDev(cnx context.Context, c *ApiClient, changes Changes, managedFrom string) error {
	// Create and create new version both follow the same logic
	validatePublish := append(changes.ToCreate, changes.ToUpdateNewVersion...)
	for _, ds := range validatePublish {
		vr, err := Validate(cnx, c, ds.DS)
		if err != nil {
			return err
		}
		if !vr.Valid {
			return errors.New(vr.Message)
		}
		_, err = PublishDev(cnx, c, ds.DS, false, managedFrom)
		if err != nil {
			return err
		}
	}
	for _, ds := range changes.ToUpdatePatch {
		vr, err := Validate(cnx, c, ds.DS)
		if err != nil {
			return err
		}
		if !vr.Valid {
			return errors.New(vr.Message)
		}
		_, err = PublishDev(cnx, c, ds.DS, true, managedFrom)
		if err != nil {
			return err
		}
	}
	for _, ds := range changes.ToUpdateMeta {
		err := MetadateUpdate(cnx, c, &ds.DS, managedFrom)
		if err != nil {
			return err
		}
	}

	return nil
}

func PerformChangesProd(cnx context.Context, c *ApiClient, changes Changes, managedFrom string) error {
	if len(changes.ToUpdatePatch) != 0 {
		return errors.New("patching is not available on prod. You must increment versions on dev before deploying")
	}
	validatePublish := append(changes.ToCreate, changes.ToUpdateNewVersion...)
	for _, ds := range validatePublish {
		_, err := PublishProd(cnx, c, ds.DS, managedFrom)
		if err != nil {
			return err
		}
	}
	for _, ds := range changes.ToUpdateMeta {
		err := MetadateUpdate(cnx, c, &ds.DS, managedFrom)
		if err != nil {
			return err
		}
	}

	return nil
}

func PrintChangeset(changes Changes) error {
	if len(changes.ToUpdateMeta) != 0 {
		for _, ds := range changes.ToUpdateMeta {
			data, err := ds.DS.ParseData()
			if err != nil {
				return err
			}
			slog.Info("will update metadata of", "file", ds.FileName, "vendor", data.Self.Vendor, "name", data.Self.Name)
		}
	}
	if len(changes.ToCreate) != 0 {
		for _, ds := range changes.ToCreate {
			data, err := ds.DS.ParseData()
			if err != nil {
				return err
			}
			slog.Info("will create", "file", ds.FileName, "vendor", data.Self.Vendor, "name", data.Self.Name, "version", data.Self.Version)
		}
	}
	if len(changes.ToUpdateNewVersion) != 0 {
		for _, ds := range changes.ToUpdateNewVersion {
			data, err := ds.DS.ParseData()
			if err != nil {
				return err
			}
			slog.Info("will update", "file", ds.FileName, "local", data.Self.Version, "remote", ds.RemoteVersion)
		}
	}
	if len(changes.ToUpdatePatch) != 0 {
		for _, ds := range changes.ToUpdatePatch {
			data, err := ds.DS.ParseData()
			if err != nil {
				return err
			}
			slog.Info(
				"will patch", "file", ds.FileName, "vendor", data.Self.Vendor, "name", data.Self.Name,
				"version", data.Self.Version, "local", ds.LocalContentHash, "remote", ds.RemoteContentHash,
			)
		}
	}
	return nil
}
