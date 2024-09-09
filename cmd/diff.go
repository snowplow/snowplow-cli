package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/r3labs/diff/v3"
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

func DiffDs(locals []DataStructure, remotes []DataStructure) ([]DataStructureWithDiff, error) {
	var res []DataStructureWithDiff
	remotesSet := make(map[DataStructureId]DataStructure)

	for _, remote := range remotes {
		dataRemote, err := remote.parseData()
		if err != nil {
			return nil, err
		}
		remotesSet[idFromSelf(dataRemote.Self)] = remote
	}

	for _, local := range locals {
		dataLocal, err := local.parseData()
		if err != nil {
			return nil, err
		}
		remote, remoteExists := remotesSet[idFromSelf(dataLocal.Self)]
		if !remoteExists {
			res = append(res, DataStructureWithDiff{DataStructure: local, Operation: "CREATE"})
		} else {
			difference, err := diff.Diff(remote, local)
			if err != nil {
				return nil, err
			}
			if len(difference) != 0 {
				res = append(res, DataStructureWithDiff{DataStructure: local, Operation: "UPDATE", Diff: difference})
			}

		}

	}

	return res, nil
}

type DSChangeContext struct {
	DS                DataStructure
	FileName          string
	RemoteVersion     string
	LocalContentHash  string
	RemoteContentHash string
}

func NewDSChangeContext(ds DataStructure, fileName string) DSChangeContext {
	return DSChangeContext{ds, fileName, "", "", ""}
}

func NewDSChangeContextWithVersion(ds DataStructure, fileName string, v string) DSChangeContext {
	return DSChangeContext{ds, fileName, v, "", ""}
}

func NewDSChangeContextWithHashes(ds DataStructure, fileName string, localHash string, remoteHash string) DSChangeContext {
	return DSChangeContext{ds, fileName, "", localHash, remoteHash}
}

type Changes struct {
	toCreate           []DSChangeContext
	toUpdateMeta       []DSChangeContext
	toUpdateNewVersion []DSChangeContext
	toUpdatePatch      []DSChangeContext
}

func getChanges(locals map[string]DataStructure, remoteListing []ListResponse, env dataStructureEnv) (Changes, error) {
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
		data, err := ds.parseData()
		if err != nil {
			return Changes{}, err
		}
		remotePair, exists := remotesSet[idFromSelf(data.Self)]
		// DS does not exists, we should create it
		if !exists {
			res.toCreate = append(res.toCreate, NewDSChangeContext(ds, f))
		} else {
			//Remote DS exists,
			if !reflect.DeepEqual(ds.Meta, remotePair.Meta) {
				// Meta is different, needs updating
				res.toUpdateMeta = append(res.toUpdateMeta, NewDSChangeContext(ds, f))
			}
			contentHash, err := ds.getContentHash()
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
							res.toUpdateNewVersion = append(res.toUpdateNewVersion, NewDSChangeContextWithVersion(ds, f, deployment.Version))
						} else {
							// Same version, but different hash, patch
							res.toUpdatePatch = append(res.toUpdatePatch, NewDSChangeContextWithHashes(ds, f, contentHash, deployment.ContentHash))
						}
					}
				}
			}
			if !foundDeployment {
				// DS exists, but we didn't find a version of it
				// We should deploy from dev to prod
				res.toUpdateNewVersion = append(res.toUpdateNewVersion, NewDSChangeContextWithVersion(ds, f, ""))
			}
		}
	}
	return res, nil
}

func validate(cnx context.Context, c *ApiClient, changes Changes) error {
	// Create and create new version both follow the same logic
	// Patch there will error out on validate, we'll implement it separately
	validate := append(append(changes.toCreate, changes.toUpdateNewVersion...), changes.toUpdatePatch...)
	failed := 0
	for _, ds := range validate {
		resp, err := Validate(cnx, c, ds.DS)
		if resp != nil {
			if len(resp.Warnings) > 0 {
				slog.Warn("validation", "file", ds.FileName, "messages", strings.Join(resp.Warnings, "\n"))
			}
			if len(resp.Info) > 0 {
				slog.Info("validation", "file", ds.FileName, "messages", strings.Join(resp.Info, "\n"))
			}
		}
		if err != nil {
			slog.Error("validation", "file", ds.FileName, "messages", err)
			failed++
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d validation failures", failed)
	}
	return nil
}

func performChangesDev(cnx context.Context, c *ApiClient, changes Changes) error {
	// Create and create new version both follow the same logic
	validatePublish := append(changes.toCreate, changes.toUpdateNewVersion...)
	for _, ds := range validatePublish {
		_, err := Validate(cnx, c, ds.DS)
		if err != nil {
			return err
		}
		_, err = PublishDev(cnx, c, ds.DS, false)
		if err != nil {
			return err
		}
	}
	for _, ds := range changes.toUpdatePatch {
		_, err := Validate(cnx, c, ds.DS)
		if err != nil {
			return err
		}
		_, err = PublishDev(cnx, c, ds.DS, true)
		if err != nil {
			return err
		}
	}
	for _, ds := range changes.toUpdateMeta {
		err := MetadateUpdate(cnx, c, &ds.DS)
		if err != nil {
			return err
		}
	}

	return nil
}

func performChangesProd(cnx context.Context, c *ApiClient, changes Changes) error {
	if len(changes.toUpdatePatch) != 0 {
		return errors.New("Patching is not availabe on prod. You must increment versions on dev before deploying.")
	}
	validatePublish := append(changes.toCreate, changes.toUpdateNewVersion...)
	for _, ds := range validatePublish {
		_, err := Validate(cnx, c, ds.DS)
		if err != nil {
			return err
		}
		_, err = PublishProd(cnx, c, ds.DS)
		if err != nil {
			return err
		}
	}
	for _, ds := range changes.toUpdateMeta {
		err := MetadateUpdate(cnx, c, &ds.DS)
		if err != nil {
			return err
		}
	}

	return nil
}

func printChangeset(changes Changes) error {
	if len(changes.toUpdateMeta) != 0 {
		for _, ds := range changes.toUpdateMeta {
			data, err := ds.DS.parseData()
			if err != nil {
				return err
			}
			slog.Info("will update metadata of", "file", ds.FileName, "vendor", data.Self.Vendor, "name", data.Self.Name)
		}
	}
	if len(changes.toCreate) != 0 {
		for _, ds := range changes.toCreate {
			data, err := ds.DS.parseData()
			if err != nil {
				return err
			}
			slog.Info("will create", "file", ds.FileName, "vendor", data.Self.Vendor, "name", data.Self.Name, "version", data.Self.Version)
		}
	}
	if len(changes.toUpdateNewVersion) != 0 {
		for _, ds := range changes.toUpdateNewVersion {
			data, err := ds.DS.parseData()
			if err != nil {
				return err
			}
			slog.Info("will update", "file", ds.FileName, "local", data.Self.Version, "remote", ds.RemoteVersion)
		}
	}
	if len(changes.toUpdatePatch) != 0 {
		for _, ds := range changes.toUpdatePatch {
			data, err := ds.DS.parseData()
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
