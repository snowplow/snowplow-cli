package cmd

import (
	"context"
	"fmt"
	"reflect"

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

type Changes struct {
	toCreate           []DataStructure
	toUpdateMeta       []DataStructure
	toUpdateNewVersion []DataStructure
	toUpdatePatch      []DataStructure
}

func getChanges(locals []DataStructure, remoteListing []ListResponse, env dataStructureEnv) (Changes, error) {
	res := Changes{
		make([]DataStructure, 0),
		make([]DataStructure, 0),
		make([]DataStructure, 0),
		make([]DataStructure, 0),
	}
	remotesSet := make(map[DataStructureId]ListResponse)
	for _, remote := range remoteListing {
		remotesSet[DataStructureId{remote.Vendor, remote.Name, remote.Format}] = remote
	}

	for _, ds := range locals {
		data, err := ds.parseData()
		if err != nil {
			return Changes{}, err
		}
		remotePair, exists := remotesSet[idFromSelf(data.Self)]
		// DS does not exists, we should create it
		if !exists {
			res.toCreate = append(res.toCreate, ds)
		} else {
			//Remote DS exists,
			if !reflect.DeepEqual(ds.Meta, remotePair.Meta) {
				// Meta is different, needs updating
				res.toUpdateMeta = append(res.toUpdateMeta, ds)
			}
			contentHash, err := ds.getContentHash()
			if err != nil {
				return Changes{}, err
			}
			// find the correct deployment to compare to
			for _, deployment := range remotePair.Deployments {
				if deployment.Env == env {
					if deployment.ContentHash != contentHash {
						// data structure has changed
						if data.Self.Version != deployment.Version {
							// Different version, create new version
							res.toUpdateNewVersion = append(res.toUpdateNewVersion, ds)
						} else {
							// Same version, but different hash, patch
							res.toUpdatePatch = append(res.toUpdatePatch, ds)
						}
					}
				}
			}
		}
	}
	return res, nil
}

func validate(cnx context.Context, c *ApiClient, changes Changes) error {
	// Create and create new version both follow the same logic
	// Patch there will error out on validate, we'll implement it separately
	validate := append(append(changes.toCreate, changes.toUpdateNewVersion...), changes.toUpdatePatch...)
	for _, ds := range validate {
		_, err := Validate(cnx, c, ds)
		if err != nil {
			return err
		}
	}
	return nil
}

func performChangesDev(cnx context.Context, c *ApiClient, changes Changes) error {
	// Create and create new version both follow the same logic
	// Patch there will error out on validate, we'll implement it separately
	validatePublish := append(append(changes.toCreate, changes.toUpdateNewVersion...), changes.toUpdatePatch...)
	for _, ds := range validatePublish {
		_, err := Validate(cnx, c, ds)
		if err != nil {
			return err
		}
		_, err = PublishDev(cnx, c, ds)
		if err != nil {
			return err
		}
	}
	for _, ds := range changes.toUpdateMeta {
		err := MetadateUpdate(cnx, c, &ds)
		if err != nil {
			return err
		}
	}

	return nil
}

func printChangeset(changes Changes) error {
	fmt.Println("Planned changes:")
	if len(changes.toUpdateMeta) != 0 {
		fmt.Println("Going to update metadata for following data structures:")
		for _, ds := range changes.toUpdateMeta {
			data, err := ds.parseData()
			if err != nil {
				return err
			}
			fmt.Printf("	%s.%s\n", data.Self.Vendor, data.Self.Name)
		}
	}
	if len(changes.toCreate) != 0 {
		fmt.Println("Going to create new data strucutres")
		for _, ds := range changes.toCreate {
			data, err := ds.parseData()
			if err != nil {
				return err
			}
			fmt.Printf("	%s.%s\n", data.Self.Vendor, data.Self.Name)
		}
	}
	if len(changes.toUpdateNewVersion) != 0 {
		fmt.Println("Going to create a new version of a data stucture")
		for _, ds := range changes.toUpdateNewVersion {
			data, err := ds.parseData()
			if err != nil {
				return err
			}
			fmt.Printf("	%s.%s.%s\n", data.Self.Vendor, data.Self.Name, data.Self.Version)
		}
	}
	if len(changes.toUpdatePatch) != 0 {
		fmt.Println("PATCHING NOT SUPPORTED YET, BUMP VERSION, but in future Going to patch an existing version of a data stucture")
		for _, ds := range changes.toUpdatePatch {
			data, err := ds.parseData()
			if err != nil {
				return err
			}
			fmt.Printf("	%s.%s.%s\n", data.Self.Vendor, data.Self.Name, data.Self.Version)
		}
	}
	return nil
}
