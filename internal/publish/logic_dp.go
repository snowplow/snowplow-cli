/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/
package publish

import (
	"log/slog"
	"path/filepath"

	"github.com/mitchellh/mapstructure"
	"github.com/snowplow-product/snowplow-cli/internal/console"
	"github.com/snowplow-product/snowplow-cli/internal/model"
	"golang.org/x/net/context"
)

type LocalFilesRefsResolved struct {
	DataProudcts []model.DataProduct
	SourceApps   []model.SourceApp
}

type DataProductChangeSet struct {
	saCreate []console.RemoteSourceApplication
	saUpdate []console.RemoteSourceApplication
	dpCreate []console.RemoteDataProduct
	dpUpdate []console.RemoteDataProduct
	esCreate []console.RemoteEventSpec
	esUpdate []console.RemoteEventSpec
}

func ReadLocalDataProducts(dp map[string]map[string]any) (*LocalFilesRefsResolved, error) {

	probablyDps := []model.DataProduct{}
	probablySas := []model.SourceApp{}
	filenameToSa := make(map[string]model.SourceApp)
	filenameToDp := make(map[string]model.DataProduct)

	for f, maybeDp := range dp {
		if resourceType, ok := maybeDp["resourceType"]; ok {
			switch resourceType {
			case "data-product":
				var dp model.DataProduct
				if err := mapstructure.Decode(maybeDp, &dp); err == nil {
					filenameToDp[f] = dp
				} else {
					return nil, err
				}
			case "source-application":
				var sa model.SourceApp
				if err := mapstructure.Decode(maybeDp, &sa); err == nil {
					filenameToSa[f] = sa
					probablySas = append(probablySas, sa)
				} else {
					return nil, err
				}
			}
		}
	}

	for dpFile, dp := range filenameToDp {
		var sourceApps []map[string]string
		for _, sa := range dp.Data.SourceApplications {
			ids := map[string]string{}
			for _, fileName := range sa {
				absPath, err := filepath.Abs(filepath.Join(filepath.Dir(dpFile), fileName))
				if err != nil {
					return nil, err
				}
				id := filenameToSa[absPath].ResourceName
				ids["id"] = id
			}
			sourceApps = append(sourceApps, ids)
		}
		dp.Data.SourceApplications = sourceApps
		for idx, es := range dp.Data.EventSpecifications {
			var excludedSourceApps []map[string]string
			for _, esSa := range es.ExcludedSourceApplications {
				ids := map[string]string{}
				fileName := esSa["$ref"]
				absPath, err := filepath.Abs(filepath.Join(filepath.Dir(dpFile), fileName))
				if err != nil {
					return nil, err
				}
				id := filenameToSa[absPath].ResourceName
				ids["id"] = id
				excludedSourceApps = append(excludedSourceApps, ids)
			}
			dp.Data.EventSpecifications[idx].ExcludedSourceApplications = excludedSourceApps
		}
		probablyDps = append(probablyDps, dp)
	}

	res := LocalFilesRefsResolved{
		DataProudcts: probablyDps,
		SourceApps:   probablySas,
	}
	return &res, nil
}

func findChanges(local LocalFilesRefsResolved, remote console.DataProductsAndRelatedResources) DataProductChangeSet {
	saRemoteIds := make(map[string]bool)
	for _, remoteSa := range remote.SourceApplication {
		saRemoteIds[remoteSa.Id] = true
	}
	var saCreate []console.RemoteSourceApplication
	var saUpdate []console.RemoteSourceApplication

	for _, localSa := range local.SourceApps {
		_, remoteExists := saRemoteIds[localSa.ResourceName]

		if remoteExists {
			saUpdate = append(saUpdate, localSaToRemote(localSa))
		} else {
			saCreate = append(saCreate, localSaToRemote(localSa))
		}
	}

	dpRemoteIds := make(map[string]bool)
	for _, remoteDp := range remote.DataProducts {
		dpRemoteIds[remoteDp.Id] = true
	}

	var dpCreate []console.RemoteDataProduct
	var dpUpdate []console.RemoteDataProduct

	esRemoteIds := make(map[string]bool)
	for _, remoteEs := range remote.EventSpecs {
		esRemoteIds[remoteEs.Id] = true
	}

	var esCreate []console.RemoteEventSpec
	var esUpdate []console.RemoteEventSpec

	for _, localDp := range local.DataProudcts {
		_, remoteExists := dpRemoteIds[localDp.ResourceName]
		if remoteExists {
			dpUpdate = append(dpUpdate, LocalDpToRemote(localDp))
		} else {
			dpCreate = append(dpCreate, LocalDpToRemote(localDp))
		}
		var dpSaIds []string
		for _, sa := range localDp.Data.SourceApplications {
			dpSaIds = append(dpSaIds, sa["id"])
		}

		for _, localEs := range localDp.Data.EventSpecifications {
			_, remoteExists := esRemoteIds[localEs.ResourceName]
			if remoteExists {
				esUpdate = append(esUpdate, LocalEventSpecToRemote(localEs, dpSaIds, localDp.ResourceName))
			} else {
				esCreate = append(esCreate, LocalEventSpecToRemote(localEs, dpSaIds, localDp.ResourceName))
			}

		}
	}

	return DataProductChangeSet{
		saCreate: saCreate,
		saUpdate: saUpdate,
		dpCreate: dpCreate,
		dpUpdate: dpUpdate,
		esCreate: esCreate,
		esUpdate: esUpdate,
	}
}

func ApplyDpChanges(changes DataProductChangeSet, cnx context.Context, client *console.ApiClient) error {
	for _, saC := range changes.saCreate {
		err := console.CreateSourceApp(cnx, client, saC)
		if err != nil {
			return err
		}
	}
	for _, saU := range changes.saUpdate {
		err := console.UpdateSourceApp(cnx, client, saU)
		if err != nil {
			return err
		}
	}
	for _, dpC := range changes.dpCreate {
		err := console.CreateDataProduct(cnx, client, dpC)
		if err != nil {
			return err
		}
	}
	for _, dpU := range changes.dpUpdate {
		err := console.UpdateDataProduct(cnx, client, dpU)
		if err != nil {
			return err
		}
	}
	for _, esC := range changes.esCreate {
		err := console.CreateEventSpec(cnx, client, esC)
		if err != nil {
			return err
		}
	}
	for _, esU := range changes.esUpdate {
		err := console.UpdateEventSpec(cnx, client, esU)
		if err != nil {
			return err
		}
	}
	return nil
}

func PrintChangeset(changes DataProductChangeSet) {
	if len(changes.saCreate) != 0 {
		for _, sa := range changes.saCreate {
			slog.Info("will create source apps", "name", sa.Name, "id", sa.Id)
		}
	}
	if len(changes.saUpdate) != 0 {
		for _, sa := range changes.saUpdate {
			slog.Info("will update source apps", "name", sa.Name, "id", sa.Id)
		}
	}
	if len(changes.dpCreate) != 0 {
		for _, dp := range changes.dpCreate {
			slog.Info("will create data product", "name", dp.Name, "id", dp.Id)
		}
	}
	if len(changes.dpUpdate) != 0 {
		for _, dp := range changes.dpUpdate {
			slog.Info("will update data product", "name", dp.Name, "id", dp.Id)
		}
	}
	if len(changes.esCreate) != 0 {
		for _, es := range changes.esCreate {
			slog.Info("will create event specifications", "name", es.Name, "id", es.Id)
		}
	}
	if len(changes.esUpdate) != 0 {
		for _, es := range changes.esUpdate {
			slog.Info("will update event specifications", "name", es.Name, "id", es.Id)
		}
	}
}

func Publish(cnx context.Context, client *console.ApiClient, dp map[string]map[string]any) error {
	localResolved, err := ReadLocalDataProducts(dp)
	if err != nil {
		return err
	}
	remote, err := console.GetDataProductsAndRelatedResources(cnx, client)
	if err != nil {
		return err
	}
	changeSet := findChanges(*localResolved, *remote)
	PrintChangeset(changeSet)
	err = ApplyDpChanges(changeSet, cnx, client)
	return err
}
