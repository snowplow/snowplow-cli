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
	"reflect"

	"github.com/go-viper/mapstructure/v2"
	"github.com/snowplow-product/snowplow-cli/internal/console"
	"github.com/snowplow-product/snowplow-cli/internal/model"
	"golang.org/x/net/context"
)

type LocalFilesRefsResolved struct {
	DataProudcts []model.DataProduct
	SourceApps   []model.SourceApp
	IdToFileName map[string]string
}

type DataProductChangeSet struct {
	saCreate     []console.RemoteSourceApplication
	saUpdate     []console.RemoteSourceApplication
	dpCreate     []console.RemoteDataProduct
	dpUpdate     []console.RemoteDataProduct
	esCreate     []console.RemoteEventSpec
	esUpdate     []console.RemoteEventSpec
	IdToFileName map[string]string
}

func (cs DataProductChangeSet) isEmpty() bool {
	return len(cs.saCreate) == 0 &&
		len(cs.saUpdate) == 0 &&
		len(cs.dpCreate) == 0 &&
		len(cs.dpUpdate) == 0 &&
		len(cs.esCreate) == 0 &&
		len(cs.esUpdate) == 0
}

func ReadLocalDataProducts(dp map[string]map[string]any) (*LocalFilesRefsResolved, error) {

	probablyDps := []model.DataProduct{}
	probablySas := []model.SourceApp{}
	filenameToSa := make(map[string]model.SourceApp)
	filenameToDp := make(map[string]model.DataProduct)
	idToFileName := make(map[string]string)

	for f, maybeDp := range dp {
		if resourceType, ok := maybeDp["resourceType"]; ok {
			switch resourceType {
			case "data-product":
				var dp model.DataProduct
				if err := mapstructure.Decode(maybeDp, &dp); err == nil {
					filenameToDp[f] = dp
					idToFileName[dp.ResourceName] = f
				} else {
					return nil, err
				}
			case "source-application":
				var sa model.SourceApp
				if err := mapstructure.Decode(maybeDp, &sa); err == nil {
					filenameToSa[f] = sa
					idToFileName[sa.ResourceName] = f
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
			idToFileName[es.ResourceName] = dpFile
		}
		probablyDps = append(probablyDps, dp)
	}

	res := LocalFilesRefsResolved{
		DataProudcts: probablyDps,
		SourceApps:   probablySas,
		IdToFileName: idToFileName,
	}
	return &res, nil
}

func findChanges(local LocalFilesRefsResolved, remote console.DataProductsAndRelatedResources) (*DataProductChangeSet, error) {
	saRemoteIds := make(map[string]console.RemoteSourceApplication)
	idToFileName := make(map[string]string)
	for _, remoteSa := range remote.SourceApplication {
		saRemoteIds[remoteSa.Id] = remoteSa
	}
	var saCreate []console.RemoteSourceApplication
	var saUpdate []console.RemoteSourceApplication

	for _, localSa := range local.SourceApps {
		currentRemote, remoteExists := saRemoteIds[localSa.ResourceName]

		if remoteExists {
			possibleUpdate := localSaToRemote(localSa)
			if !reflect.DeepEqual(possibleUpdate, currentRemote) {
				saUpdate = append(saUpdate, possibleUpdate)
				idToFileName[localSa.ResourceName] = local.IdToFileName[localSa.ResourceName]
			}
		} else {
			saCreate = append(saCreate, localSaToRemote(localSa))
			idToFileName[localSa.ResourceName] = local.IdToFileName[localSa.ResourceName]
		}
	}

	dpRemoteIds := make(map[string]console.RemoteDataProduct)
	for _, remoteDp := range remote.DataProducts {
		dpRemoteIds[remoteDp.Id] = remoteDp
	}

	var dpCreate []console.RemoteDataProduct
	var dpUpdate []console.RemoteDataProduct

	esRemoteIds := make(map[string]console.RemoteEventSpec)
	for _, remoteEs := range remote.EventSpecs {
		esRemoteIds[remoteEs.Id] = remoteEs
	}

	var esCreate []console.RemoteEventSpec
	var esUpdate []console.RemoteEventSpec

	for _, localDp := range local.DataProudcts {
		remoteDp, remoteExists := dpRemoteIds[localDp.ResourceName]
		if remoteExists {
			possibleUpdate := LocalDpToRemote(localDp)
			if !reflect.DeepEqual(dpToDiff(possibleUpdate), dpToDiff(remoteDp)) {
				dpUpdate = append(dpUpdate, possibleUpdate)
				idToFileName[localDp.ResourceName] = local.IdToFileName[localDp.ResourceName]
			}
		} else {
			dpCreate = append(dpCreate, LocalDpToRemote(localDp))
			idToFileName[localDp.ResourceName] = local.IdToFileName[localDp.ResourceName]
		}
		var dpSaIds []string
		for _, sa := range localDp.Data.SourceApplications {
			dpSaIds = append(dpSaIds, sa["id"])
		}

		for _, localEs := range localDp.Data.EventSpecifications {
			remoteEs, remoteExists := esRemoteIds[localEs.ResourceName]
			if remoteExists {
				possibleUpdate := LocalEventSpecToRemote(localEs, dpSaIds, localDp.ResourceName)
				updateDiff, err := esToDiff(possibleUpdate)
				if err != nil {
					return nil, err
				}
				remoteDiff, err := esToDiff(remoteEs)
				if err != nil {
					return nil, err
				}
				if !reflect.DeepEqual(*remoteDiff, *updateDiff) {
					esUpdate = append(esUpdate, possibleUpdate)
					idToFileName[localEs.ResourceName] = local.IdToFileName[localEs.ResourceName]
				}
			} else {
				esCreate = append(esCreate, LocalEventSpecToRemote(localEs, dpSaIds, localDp.ResourceName))
				idToFileName[localEs.ResourceName] = local.IdToFileName[localEs.ResourceName]
			}
		}
	}

	return &DataProductChangeSet{
		saCreate:     saCreate,
		saUpdate:     saUpdate,
		dpCreate:     dpCreate,
		dpUpdate:     dpUpdate,
		esCreate:     esCreate,
		esUpdate:     esUpdate,
		IdToFileName: idToFileName,
	}, nil
}

func ApplyDpChanges(changes DataProductChangeSet, cnx context.Context, client *console.ApiClient) error {
	slog.Info("publish", "msg", "applying changes")
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

func PrintChangeset(changes DataProductChangeSet, idToFile map[string]string) {
	if changes.isEmpty() {
		slog.Info("publish", "msg", "no changes detected, nothing to apply")
	} else {
		if len(changes.saCreate) != 0 {
			for _, sa := range changes.saCreate {
				slog.Info("publish", "msg", "will create source apps", "file", idToFile[sa.Id], "name", sa.Name, "id", sa.Id)
			}
		}
		if len(changes.saUpdate) != 0 {
			for _, sa := range changes.saUpdate {
				slog.Info("publish", "msg", "will update source apps", "file", idToFile[sa.Id], "name", sa.Name, "id", sa.Id)
			}
		}
		if len(changes.dpCreate) != 0 {
			for _, dp := range changes.dpCreate {
				slog.Info("publish", "msg", "will create data product", "file", idToFile[dp.Id], "name", dp.Name, "id", dp.Id)
			}
		}
		if len(changes.dpUpdate) != 0 {
			for _, dp := range changes.dpUpdate {
				slog.Info("publish", "msg", "will update data product", "file", idToFile[dp.Id], "name", dp.Name, "id", dp.Id)
			}
		}
		if len(changes.esCreate) != 0 {
			for _, es := range changes.esCreate {
				slog.Info("publish", "msg", "will create event specifications", "file", idToFile[es.Id], "name", es.Name, "id", es.Id)
			}
		}
		if len(changes.esUpdate) != 0 {
			for _, es := range changes.esUpdate {
				slog.Info("publish", "msg", "will update event specifications", "file", idToFile[es.Id], "name", es.Name, "id", es.Id, "in data product id", es.DataProductId)
			}
		}
		slog.Info("publish", "msg", "total entities to update", "data products", len(changes.dpCreate)+len(changes.dpUpdate), "event specs", len(changes.esCreate)+len(changes.esUpdate), "source apps", len(changes.saCreate)+len(changes.saUpdate))
	}
}

func FindChanges(cnx context.Context, client *console.ApiClient, dp map[string]map[string]any) (*DataProductChangeSet, error) {
	localResolved, err := ReadLocalDataProducts(dp)
	if err != nil {
		return nil, err
	}
	remote, err := console.GetDataProductsAndRelatedResources(cnx, client)
	if err != nil {
		return nil, err
	}
	changeSet, err := findChanges(*localResolved, *remote)
	if err != nil {
		return nil, err
	}
	return changeSet, err
}

func Publish(cnx context.Context, client *console.ApiClient, changeSet *DataProductChangeSet, dryRun bool) error {
	PrintChangeset(*changeSet, changeSet.IdToFileName)
	var err error
	if !dryRun && !changeSet.isEmpty() {
		err = ApplyDpChanges(*changeSet, cnx, client)
	}
	return err
}
