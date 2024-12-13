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
	"crypto/md5"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/snowplow-product/snowplow-cli/internal/console"
	"github.com/snowplow-product/snowplow-cli/internal/model"
	"golang.org/x/net/context"
)

type LocalFilesRefsResolved struct {
	DataProudcts    []model.DataProduct
	SourceApps      []model.SourceApp
	HashToImageFile map[string]string
	IdToFileName    map[string]string
}

type ImageCreationEnv struct {
	fname string
	hash  string
}

type DataProductChangeSet struct {
	saCreate     []console.RemoteSourceApplication
	saUpdate     []console.RemoteSourceApplication
	dpCreate     []console.RemoteDataProduct
	dpUpdate     []console.RemoteDataProduct
	esCreate     []console.RemoteEventSpec
	esUpdate     []console.RemoteEventSpec
	esDelete     []console.RemoteEventSpec
	imageCreate  []ImageCreationEnv
	IdToFileName map[string]string
}

func (cs DataProductChangeSet) isEmpty() bool {
	return len(cs.saCreate) == 0 &&
		len(cs.saUpdate) == 0 &&
		len(cs.dpCreate) == 0 &&
		len(cs.dpUpdate) == 0 &&
		len(cs.esCreate) == 0 &&
		len(cs.esUpdate) == 0 &&
		len(cs.esDelete) == 0 &&
		len(cs.imageCreate) == 0
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

	localHashToImage := map[string]string{}
	for dpFile, dp := range filenameToDp {
		for _, es := range dp.Data.EventSpecifications {
			for _, t := range es.Triggers {
				if len(t.Image.Ref) != 0 {
					dppath, err := filepath.Abs(dpFile)
					if err != nil {
						return nil, err
					}
					fname := filepath.Clean(filepath.Join(filepath.Dir(dppath), t.Image.Ref))
					f, err := os.Open(fname)
					if err != nil {
						return nil, err
					}
					defer f.Close()

					h := md5.New()
					if _, err := io.Copy(h, f); err != nil {
						return nil, err
					}

					hash := fmt.Sprintf("%x", h.Sum(nil))

					localHashToImage[hash] = fname
				}
			}
		}
	}

	res := LocalFilesRefsResolved{
		DataProudcts:    probablyDps,
		SourceApps:      probablySas,
		IdToFileName:    idToFileName,
		HashToImageFile: localHashToImage,
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
	var esDelete []console.RemoteEventSpec

	esLocalIds := make(map[string]bool)

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

			esLocalIds[localEs.ResourceName] = true
		}

		for _, remoteEsId := range remoteDp.EventSpecs {
			_, localExists := esLocalIds[remoteEsId.Id]
			if !localExists {
				// Remote exists, but local is missing, delete event spec
				esDelete = append(esDelete, esRemoteIds[remoteEsId.Id])
				// Not a filename, DP name, since file does not exist anymore
				idToFileName[remoteEsId.Id] = remoteDp.Name
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
		esDelete:     esDelete,
		IdToFileName: idToFileName,
	}, nil
}

func ApplyDpChanges(changes DataProductChangeSet, cnx context.Context, client *console.ApiClient) error {
	slog.Info("publish", "msg", "applying changes")
	for _, img := range changes.imageCreate {
		variants, err := console.PublishImage(cnx, client, img.fname, img.hash)
		if err != nil {
			return err
		}
		slog.Info("publish", "msg", "what do we do with the images we just uploaded? they need feeding back in to give context to the event spec changes", "img", img.fname,"variants", variants)
	}
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
	for _, esD := range changes.esDelete {
		err := console.DeleteEventSpec(cnx, client, esD.Id)
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
				slog.Info("publish", "msg", "will create source apps", "file", idToFile[sa.Id], "name", sa.Name, "resource name", sa.Id)
			}
		}
		if len(changes.saUpdate) != 0 {
			for _, sa := range changes.saUpdate {
				slog.Info("publish", "msg", "will update source apps", "file", idToFile[sa.Id], "name", sa.Name, "resource name", sa.Id)
			}
		}
		if len(changes.dpCreate) != 0 {
			for _, dp := range changes.dpCreate {
				slog.Info("publish", "msg", "will create data product", "file", idToFile[dp.Id], "name", dp.Name, "resource name", dp.Id)
			}
		}
		if len(changes.dpUpdate) != 0 {
			for _, dp := range changes.dpUpdate {
				slog.Info("publish", "msg", "will update data product", "file", idToFile[dp.Id], "name", dp.Name, "resource name", dp.Id)
			}
		}
		if len(changes.esCreate) != 0 {
			for _, es := range changes.esCreate {
				slog.Info("publish", "msg", "will create event specifications", "file", idToFile[es.Id], "name", es.Name, "resource name", es.Id)
			}
		}
		if len(changes.esUpdate) != 0 {
			for _, es := range changes.esUpdate {
				slog.Info("publish", "msg", "will update event specifications", "file", idToFile[es.Id], "name", es.Name, "resource name", es.Id, "in data product", es.DataProductId)
			}
		}
		if len(changes.esDelete) != 0 {
			for _, es := range changes.esDelete {
				slog.Info("publish", "msg", "will delete event specifications", "name", es.Name, "resource name", es.Id, "in data product", es.DataProductId, "data product name", idToFile[es.Id])
			}
		}
		if len(changes.imageCreate) != 0 {
			if cwd, err := os.Getwd(); err == nil {
				for _, img := range changes.imageCreate {
					if relp, err := filepath.Rel(cwd, img.fname); err == nil {
						slog.Info("publish", "msg", "will publish image", "file", relp)
					}
				}
			}
		}
		slog.Info("publish", "msg", "total entities to update", "data products", len(changes.dpCreate)+len(changes.dpUpdate), "event specs", len(changes.esCreate)+len(changes.esUpdate)+len(changes.esDelete), "source apps", len(changes.saCreate)+len(changes.saUpdate), "images", len(changes.imageCreate))
	}
}

type DataProductPurger interface {
	DeleteSourceApp(sa console.RemoteSourceApplication) error
	DeleteDataProduct(dp console.RemoteDataProduct) error
	FetchDataProduct() (*console.DataProductsAndRelatedResources, error)
}

func Purge(api DataProductPurger, dp map[string]map[string]any, commit bool) error {
	localResolved, err := ReadLocalDataProducts(dp)
	if err != nil {
		return err
	}
	remote, err := api.FetchDataProduct()
	if err != nil {
		return err
	}

	purgeApps := map[string]console.RemoteSourceApplication{}
	for _, r := range remote.SourceApplication {
		purgeApps[r.Id] = r
	}
	for _, r := range localResolved.SourceApps {
		delete(purgeApps, r.ResourceName)
	}
	saNames := []string{}
	for _, r := range purgeApps {
		saNames = append(saNames, r.Name)
	}

	purgeProds := map[string]console.RemoteDataProduct{}
	for _, r := range remote.DataProducts {
		purgeProds[r.Id] = r
	}
	for _, r := range localResolved.DataProudcts {
		delete(purgeProds, r.ResourceName)
	}
	dpNames := []string{}
	for _, r := range purgeProds {
		dpNames = append(dpNames, r.Name)
	}

	slog.Info("purge",
		"msg", fmt.Sprintf("%d source apps and %d data products", len(saNames), len(dpNames)),
		"source apps", strings.Join(saNames, "\n"),
		"data products", strings.Join(dpNames, "\n"),
	)

	if !commit {
		slog.Info("purge", "msg", "re-run command with -y/--yes to commit changes")
		return nil
	}

	for n, r := range purgeApps {
		slog.Debug("purge", "msg", "deleting remote", "source app", n)
		if err := api.DeleteSourceApp(r); err != nil {
			slog.Error("purge", "msg", "failed", "source app", n)
			return fmt.Errorf("purge failed: %w", err)
		}

		slog.Debug("purge", "msg", "deleted", "source app", n)
	}

	for n, r := range purgeProds {
		slog.Debug("purge", "msg", "deleting remote", "data product", n)
		if err := api.DeleteDataProduct(r); err != nil {
			slog.Error("purge", "msg", "failed", "data product", n)
			return fmt.Errorf("purge failed: %w", err)
		}

		slog.Debug("purge", "msg", "deleted", "data product", n)
	}

	slog.Info("purge", "msg", "complete")

	return nil
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
	hashLookup, err := console.GetImageHashLookup(cnx, client)
	if err != nil {
		return nil, err
	}

	imagesToCreate := []ImageCreationEnv{}
	for h, f := range localResolved.HashToImageFile {
		if !slices.Contains(hashLookup, h) {
			imagesToCreate = append(imagesToCreate, ImageCreationEnv{f, h})
		}
	}

	changeSet.imageCreate = imagesToCreate

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
