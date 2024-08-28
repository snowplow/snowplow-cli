package cmd

import (

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
