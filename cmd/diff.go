package cmd

import (
	"fmt"

	"github.com/r3labs/diff/v3"
)

type DataStructureWithDiff struct {
	DataStructure DataStructure
	Operation     string
	Diff          diff.Changelog
}

func DiffDs(locals []DataStructure, remotes []DataStructure) ([]DataStructureWithDiff, error) {
	fmt.Printf("locals: %v\n\n", locals)
	fmt.Printf("remotes: %v\n\n", remotes)
	var res []DataStructureWithDiff
	remotesSet := make(map[DataStructureSelf]DataStructure)

	for _, remote := range remotes {
		dataRemote, err := remote.parseData()
		if err != nil {
			return nil, err
		}
		remotesSet[dataRemote.Self] = remote
	}

	for _, local := range locals {
		dataLocal, err := local.parseData()
		if err != nil {
			return nil, err
		}
		remote, remoteExists := remotesSet[dataLocal.Self]
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

	fmt.Printf("Differences %+v", res)

	return res, nil
}
