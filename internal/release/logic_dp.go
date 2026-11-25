/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package release

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"context"

	"github.com/go-viper/mapstructure/v2"
	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/model"
	"github.com/snowplow/snowplow-cli/internal/util"
)

type LocalFilesRefsResolved struct {
	DataProudcts         []model.DataProduct
	SourceApps           []model.SourceApp
	TriggerImageRefsById map[string]TriggerImageReference
	IdToFileName         map[string]string
}

type TriggerImageReference struct {
	eventSpecId string
	triggerId   string
	fname       string
	hash        string
}

type DataProductChangeSet struct {
	saCreate          []console.RemoteSourceApplication
	saUpdate          []console.RemoteSourceApplication
	dpCreate          []console.RemoteDataProduct
	dpUpdate          []console.RemoteDataProduct
	esCreate          []console.RemoteEventSpec
	esUpdate          []console.RemoteEventSpec
	esDelete          []console.RemoteEventSpec
	imageCreate       []TriggerImageReference
	IdToFileName      map[string]string
	localEventSpecIds []string
	declaredVersions  map[string]int
	VersionMismatches []model.EventSpecVersionMismatch
	MissingVersions   []model.EventSpecMissingVersion
}

func (cs DataProductChangeSet) declaredVersion(id string) *int {
	if v, ok := cs.declaredVersions[id]; ok {
		return &v
	}
	return nil
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

func ReadLocalDataProducts(ctx context.Context, dp map[string]map[string]any) (*LocalFilesRefsResolved, error) {
	probablyDps := []model.DataProduct{}
	probablySas := []model.SourceApp{}
	filenameToSa := make(map[string]model.SourceApp)
	filenameToDp := make(map[string]model.DataProduct)
	idToFileName := make(map[string]string)

	for f, maybeDp := range dp {
		switch maybeDp["resourceType"] {
		case "data-product":
			var dp model.DataProduct
			if err := mapstructure.Decode(maybeDp, &dp); err != nil {
				return nil, err
			}
			filenameToDp[f] = dp
			idToFileName[dp.ResourceName] = f
		case "source-application":
			var sa model.SourceApp
			if err := mapstructure.Decode(maybeDp, &sa); err != nil {
				return nil, err
			}
			filenameToSa[f] = sa
			idToFileName[sa.ResourceName] = f
			probablySas = append(probablySas, sa)
		}
	}

	for _, dpFile := range slices.Sorted(maps.Keys(filenameToDp)) {
		dp := filenameToDp[dpFile]
		if err := replaceSourceAppRefsWithIds(&dp, dpFile, filenameToSa); err != nil {
			return nil, err
		}
		for _, es := range dp.Data.EventSpecifications {
			idToFileName[es.ResourceName] = dpFile
		}
		probablyDps = append(probablyDps, dp)
	}

	triggerImageRefs, err := hashTriggerImages(ctx, filenameToDp)
	if err != nil {
		return nil, err
	}

	res := LocalFilesRefsResolved{
		DataProudcts:         probablyDps,
		SourceApps:           probablySas,
		IdToFileName:         idToFileName,
		TriggerImageRefsById: triggerImageRefs,
	}
	return &res, nil
}

func replaceSourceAppRefsWithIds(dp *model.DataProduct, dpFile string, filenameToSa map[string]model.SourceApp) error {
	var sourceApps []map[string]string
	for _, sa := range dp.Data.SourceApplications {
		ids := map[string]string{}
		for _, fileName := range sa {
			absPath, err := filepath.Abs(filepath.Join(filepath.Dir(dpFile), fileName))
			if err != nil {
				return err
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
				return err
			}
			id := filenameToSa[absPath].ResourceName
			ids["id"] = id
			excludedSourceApps = append(excludedSourceApps, ids)
		}
		dp.Data.EventSpecifications[idx].ExcludedSourceApplications = excludedSourceApps
	}
	return nil
}

func hashTriggerImages(ctx context.Context, filenameToDp map[string]model.DataProduct) (map[string]TriggerImageReference, error) {
	triggerImageRefs := make(map[string]TriggerImageReference)
	for dpFile, dp := range filenameToDp {
		for _, es := range dp.Data.EventSpecifications {
			for _, t := range es.Triggers {
				if t.Image == nil || len(t.Image.Ref) == 0 {
					continue
				}
				triggerImageRef, err := hashTriggerImage(ctx, dpFile, es.ResourceName, t)
				if err != nil {
					return nil, err
				}
				triggerImageRefs[t.Id] = triggerImageRef
			}
		}
	}
	return triggerImageRefs, nil
}

func hashTriggerImage(ctx context.Context, dpFile string, eventSpecId string, t model.Trigger) (TriggerImageReference, error) {
	dppath, err := filepath.Abs(dpFile)
	if err != nil {
		return TriggerImageReference{}, err
	}
	fname := filepath.Clean(filepath.Join(filepath.Dir(dppath), t.Image.Ref))
	f, err := os.Open(fname)
	if err != nil {
		return TriggerImageReference{}, err
	}
	defer util.LoggingCloser(ctx, f)

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return TriggerImageReference{}, err
	}

	hash := hex.EncodeToString(h.Sum(nil))

	return TriggerImageReference{
		eventSpecId: eventSpecId,
		triggerId:   t.Id,
		fname:       fname,
		hash:        hash,
	}, nil
}

type changeFinder struct {
	local               LocalFilesRefsResolved
	remoteDataProducts  map[string]console.RemoteDataProduct
	remoteEventSpecs    map[string]console.RemoteEventSpec
	remoteImageHashById map[string]string
	changes             *DataProductChangeSet
	seen                seenEventSpecs
	missingVersions     []model.EventSpecMissingVersion
	missingVersionIds   []string
}

type seenEventSpecs struct {
	files     map[string]string
	versioned map[string]bool
	count     map[string]int
}

func (seen seenEventSpecs) add(es model.EventSpec, dpFile string) {
	seen.files[es.ResourceName] = dpFile
	seen.count[es.ResourceName]++
	if es.Version != nil {
		seen.versioned[es.ResourceName] = true
	}
}

type expectedEventSpecVersion struct {
	version               int
	isNew                 bool
	changed               []string
	entityCommentsRemoved bool
}

func findChanges(local LocalFilesRefsResolved, remote console.DataProductsAndRelatedResources, remoteImageHashById map[string]string) (*DataProductChangeSet, error) {
	finder := changeFinder{
		local:               local,
		remoteDataProducts:  make(map[string]console.RemoteDataProduct),
		remoteEventSpecs:    make(map[string]console.RemoteEventSpec),
		remoteImageHashById: remoteImageHashById,
		changes: &DataProductChangeSet{
			IdToFileName:     make(map[string]string),
			declaredVersions: make(map[string]int),
		},
		seen: seenEventSpecs{
			files:     make(map[string]string),
			versioned: make(map[string]bool),
			count:     make(map[string]int),
		},
	}
	for _, remoteDp := range remote.DataProducts {
		finder.remoteDataProducts[remoteDp.Id] = remoteDp
	}
	for _, remoteEs := range remote.EventSpecs {
		finder.remoteEventSpecs[remoteEs.Id] = remoteEs
	}

	finder.addSourceAppChanges(remote.SourceApplication)
	for _, localDp := range local.DataProudcts {
		finder.addDataProductChange(localDp)
		if err := finder.addEventSpecChanges(localDp); err != nil {
			return nil, err
		}
		finder.addEventSpecDeletes(localDp)
	}

	finder.changes.localEventSpecIds = make([]string, 0, len(finder.seen.files))
	for id := range finder.seen.files {
		finder.changes.localEventSpecIds = append(finder.changes.localEventSpecIds, id)
	}
	finder.changes.MissingVersions = withoutDuplicatedEventSpecs(finder.missingVersions, finder.missingVersionIds, finder.seen.count)

	return finder.changes, nil
}

func (f *changeFinder) addFileName(id string) {
	f.changes.IdToFileName[id] = f.local.IdToFileName[id]
}

func (f *changeFinder) addSourceAppChanges(remote []console.RemoteSourceApplication) {
	saRemoteIds := make(map[string]console.RemoteSourceApplication)
	for _, remoteSa := range remote {
		saRemoteIds[remoteSa.Id] = remoteSa
	}

	for _, localSa := range f.local.SourceApps {
		possibleUpdate := localSaToRemote(localSa)
		currentRemote, remoteExists := saRemoteIds[localSa.ResourceName]
		if !remoteExists {
			f.changes.saCreate = append(f.changes.saCreate, possibleUpdate)
			f.addFileName(localSa.ResourceName)
			continue
		}
		if !reflect.DeepEqual(saToDiff(possibleUpdate), saToDiff(currentRemote)) {
			f.changes.saUpdate = append(f.changes.saUpdate, possibleUpdate)
			f.addFileName(localSa.ResourceName)
		}
	}
}

func (f *changeFinder) addDataProductChange(localDp model.DataProduct) {
	possibleUpdate := LocalDpToRemote(localDp)
	remoteDp, remoteExists := f.remoteDataProducts[localDp.ResourceName]
	if !remoteExists {
		f.changes.dpCreate = append(f.changes.dpCreate, possibleUpdate)
		f.addFileName(localDp.ResourceName)
		return
	}
	if !reflect.DeepEqual(dpToDiff(possibleUpdate), dpToDiff(remoteDp)) {
		f.changes.dpUpdate = append(f.changes.dpUpdate, possibleUpdate)
		f.addFileName(localDp.ResourceName)
	}
}

func (f *changeFinder) addEventSpecChanges(localDp model.DataProduct) error {
	dpFile := f.local.IdToFileName[localDp.ResourceName]
	var dpSaIds []string
	for _, sa := range localDp.Data.SourceApplications {
		dpSaIds = append(dpSaIds, sa["id"])
	}

	for esIdx, es := range localDp.Data.EventSpecifications {
		if previousFile, seen := f.seen.files[es.ResourceName]; seen && (es.Version != nil || f.seen.versioned[es.ResourceName]) {
			return duplicateEventSpecError(es.ResourceName, previousFile, dpFile)
		}
		f.seen.add(es, dpFile)

		expected, err := f.addEventSpecChange(LocalEventSpecToRemote(es, dpSaIds, localDp.ResourceName))
		if err != nil {
			return err
		}
		f.recordEventSpecVersion(localEventSpec{fileName: dpFile, index: esIdx, spec: es}, expected)
	}

	return nil
}

func (f *changeFinder) addEventSpecChange(possibleUpdate console.RemoteEventSpec) (expectedEventSpecVersion, error) {
	remoteEs, remoteExists := f.remoteEventSpecs[possibleUpdate.Id]
	if remoteExists {
		return f.addEventSpecUpdate(possibleUpdate, remoteEs)
	}

	f.changes.esCreate = append(f.changes.esCreate, possibleUpdate)
	f.addFileName(possibleUpdate.Id)
	for _, t := range possibleUpdate.Triggers {
		triggerImage, exists := f.local.TriggerImageRefsById[t.Id]
		if exists {
			f.changes.imageCreate = append(f.changes.imageCreate, triggerImage)
		}
	}

	return expectedEventSpecVersion{isNew: true, changed: []string{}}, nil
}

func (f *changeFinder) addEventSpecUpdate(possibleUpdate console.RemoteEventSpec, remoteEs console.RemoteEventSpec) (expectedEventSpecVersion, error) {
	updateDiff, err := esToDiff(possibleUpdate)
	if err != nil {
		return expectedEventSpecVersion{}, err
	}
	remoteDiff, err := esToDiff(remoteEs)
	if err != nil {
		return expectedEventSpecVersion{}, err
	}
	// check triggers separately because of images
	triggerChangeset := findTriggerChanges(possibleUpdate.Triggers, remoteEs.Triggers, f.local.TriggerImageRefsById, f.remoteImageHashById)
	// assign remote variant urls to update where image hashes are the same
	possibleUpdate.Triggers = triggerChangeset.triggersWithOriginalVariantUrls
	if reflect.DeepEqual(*remoteDiff, *updateDiff) && !triggerChangeset.isChanged {
		return expectedEventSpecVersion{version: remoteEs.Version, changed: []string{}}, nil
	}

	expected, err := expectedVersionAfterUpdate(remoteEs, possibleUpdate)
	if err != nil {
		return expectedEventSpecVersion{}, err
	}
	f.changes.esUpdate = append(f.changes.esUpdate, possibleUpdate)
	f.addFileName(possibleUpdate.Id)
	f.changes.imageCreate = append(f.changes.imageCreate, triggerChangeset.imagesToUpload...)

	return expected, nil
}

func expectedVersionAfterUpdate(remoteEs console.RemoteEventSpec, update console.RemoteEventSpec) (expectedEventSpecVersion, error) {
	changedFields, err := changedStructuralFields(remoteEs, update)
	if err != nil {
		return expectedEventSpecVersion{}, err
	}
	expected := expectedEventSpecVersion{version: remoteEs.Version, changed: changedFields}
	if len(changedFields) > 0 {
		expected.version = remoteEs.Version + 1
	}
	if !slices.Contains(changedFields, "entities") {
		return expected, nil
	}

	remoteWithoutComments := remoteEs
	remoteWithoutComments.Entities = console.Entities{
		Tracked:  entitiesWithoutComments(remoteEs.Entities.Tracked),
		Enriched: entitiesWithoutComments(remoteEs.Entities.Enriched),
	}
	changedIgnoringComments, err := changedStructuralFields(remoteWithoutComments, update)
	if err != nil {
		return expectedEventSpecVersion{}, err
	}
	expected.entityCommentsRemoved = !slices.Contains(changedIgnoringComments, "entities")

	return expected, nil
}

func (f *changeFinder) recordEventSpecVersion(localEs localEventSpec, expected expectedEventSpecVersion) {
	es := localEs.spec
	if es.Version == nil {
		f.missingVersions = append(f.missingVersions, model.EventSpecMissingVersion{
			FileName: localEs.fileName,
			Index:    localEs.index,
			Name:     es.Name,
			Expected: &expected.version,
		})
		f.missingVersionIds = append(f.missingVersionIds, es.ResourceName)
		return
	}

	f.changes.declaredVersions[es.ResourceName] = *es.Version
	if *es.Version != expected.version {
		f.changes.VersionMismatches = append(f.changes.VersionMismatches, model.EventSpecVersionMismatch{
			FileName:              f.local.IdToFileName[es.ResourceName],
			Index:                 localEs.index,
			Name:                  es.Name,
			Declared:              *es.Version,
			Expected:              expected.version,
			IsNew:                 expected.isNew,
			Changed:               expected.changed,
			EntityCommentsRemoved: expected.entityCommentsRemoved,
		})
	}
}

func (f *changeFinder) addEventSpecDeletes(localDp model.DataProduct) {
	remoteDp := f.remoteDataProducts[localDp.ResourceName]
	for _, remoteEsId := range remoteDp.EventSpecs {
		_, localExists := f.seen.files[remoteEsId.Id]
		if !localExists {
			// Remote exists, but local is missing, delete event spec
			f.changes.esDelete = append(f.changes.esDelete, f.remoteEventSpecs[remoteEsId.Id])
			// Not a filename, DP name, since file does not exist anymore
			f.changes.IdToFileName[remoteEsId.Id] = remoteDp.Name
		}
	}
}

type triggerChangeset struct {
	isChanged                       bool
	imagesToUpload                  []TriggerImageReference
	triggersWithOriginalVariantUrls []console.RemoteTrigger
}

func findTriggerChanges(local []console.RemoteTrigger, remote []console.RemoteTrigger, localImagesByTriggerId map[string]TriggerImageReference, remoteImageHashById map[string]string) triggerChangeset {
	differentTriggerSet := len(local) != len(remote)
	var differentTextData bool
	var differentImages bool
	imagesToUpload := []TriggerImageReference{}
	remoteById := map[string]console.RemoteTrigger{}
	for _, rt := range remote {
		remoteById[rt.Id] = rt
	}

	var triggersWithOriginalVariantUrls []console.RemoteTrigger

	for _, lt := range local {
		rt, remoteExists := remoteById[lt.Id]
		if !remoteExists {
			differentTriggerSet = true
		} else if lt.Url != rt.Url || lt.Description != rt.Description || !util.StringSlicesEqual(lt.AppIds, rt.AppIds) {
			differentTextData = true
		}
		localImage := localImagesByTriggerId[lt.Id]
		remoteImageHash := getRemoteImageHashFromTriggerAndHashes(rt, remoteImageHashById)
		if localImage.hash != remoteImageHash {
			differentImages = true
			imagesToUpload = append(imagesToUpload, localImage)
			// this one won't have variant urls until the images will get uploaded
			triggersWithOriginalVariantUrls = append(triggersWithOriginalVariantUrls, lt)
		} else {
			// populate triggers with links from remote - we dont' have them locally anymore, but we know that they are the same
			triggersWithOriginalVariantUrls = append(triggersWithOriginalVariantUrls,
				console.RemoteTrigger{
					Id:          lt.Id,
					Description: lt.Description,
					AppIds:      lt.AppIds,
					Url:         lt.Url,
					VariantUrls: rt.VariantUrls,
				})
		}
	}

	return triggerChangeset{
		isChanged:                       differentTriggerSet || differentTextData || differentImages,
		imagesToUpload:                  imagesToUpload,
		triggersWithOriginalVariantUrls: triggersWithOriginalVariantUrls,
	}
}

func getRemoteImageHashFromTriggerAndHashes(remote console.RemoteTrigger, remoteHashesById map[string]string) string {
	variant := "original"
	uri := strings.TrimSuffix(remote.VariantUrls[variant], "/"+variant)
	if uri != "" {
		imageId := uri[len(uri)-36:]
		return remoteHashesById[imageId]
	} else {
		return ""
	}
}

func ApplyDpChanges(changes DataProductChangeSet, cnx context.Context, client *console.ApiClient) error {
	slog.Info("sync", "msg", "applying changes")
	if err := createAndUpdateSourceApps(cnx, client, changes); err != nil {
		return err
	}
	if err := createAndUpdateDataProducts(cnx, client, changes); err != nil {
		return err
	}
	triggerIdToVariantUrl, err := publishTriggerImages(cnx, client, changes.imageCreate)
	if err != nil {
		return err
	}
	setUploadedVariantUrls(changes.esCreate, triggerIdToVariantUrl)
	if err := createEventSpecs(cnx, client, changes); err != nil {
		return err
	}
	setUploadedVariantUrls(changes.esUpdate, triggerIdToVariantUrl)
	if err := updateEventSpecs(cnx, client, changes); err != nil {
		return err
	}
	return deleteEventSpecs(cnx, client, changes.esDelete)
}

func createAndUpdateSourceApps(cnx context.Context, client *console.ApiClient, changes DataProductChangeSet) error {
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
	return nil
}

func createAndUpdateDataProducts(cnx context.Context, client *console.ApiClient, changes DataProductChangeSet) error {
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
	return nil
}

func publishTriggerImages(cnx context.Context, client *console.ApiClient, images []TriggerImageReference) (map[string]console.VariantUrls, error) {
	triggerIdToVariantUrl := make(map[string]console.VariantUrls)
	for _, img := range images {
		variants, err := console.PublishImage(cnx, client, img.fname, img.hash)
		if err != nil {
			return nil, err
		}
		triggerIdToVariantUrl[img.triggerId] = variants
	}
	return triggerIdToVariantUrl, nil
}

func setUploadedVariantUrls(eventSpecs []console.RemoteEventSpec, triggerIdToVariantUrl map[string]console.VariantUrls) {
	for esI, es := range eventSpecs {
		for tI, t := range es.Triggers {
			uploadedVariants, exists := triggerIdToVariantUrl[t.Id]
			if exists {
				eventSpecs[esI].Triggers[tI].VariantUrls = uploadedVariants
			}
		}
	}
}

func createEventSpecs(cnx context.Context, client *console.ApiClient, changes DataProductChangeSet) error {
	for _, esC := range changes.esCreate {
		err := console.CreateEventSpec(cnx, client, esC, changes.declaredVersion(esC.Id))
		if err != nil {
			return err
		}
	}
	return nil
}

func updateEventSpecs(cnx context.Context, client *console.ApiClient, changes DataProductChangeSet) error {
	for _, esU := range changes.esUpdate {
		err := console.UpdateEventSpec(cnx, client, esU, changes.declaredVersion(esU.Id))
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteEventSpecs(cnx context.Context, client *console.ApiClient, eventSpecs []console.RemoteEventSpec) error {
	for _, esD := range eventSpecs {
		err := console.DeleteEventSpec(cnx, client, esD.Id)
		if err != nil {
			return err
		}
	}
	return nil
}

func PrintChangeset(changes DataProductChangeSet, idToFile map[string]string, isRelease bool) {
	if changes.isEmpty() {
		slog.Info("sync", "msg", "no changes detected, nothing to apply")
		return
	}
	for _, sa := range changes.saCreate {
		slog.Info("sync", "msg", "will create source apps", "file", idToFile[sa.Id], "name", sa.Name, "resource name", sa.Id)
	}
	for _, sa := range changes.saUpdate {
		slog.Info("sync", "msg", "will update source apps", "file", idToFile[sa.Id], "name", sa.Name, "resource name", sa.Id)
	}
	for _, dp := range changes.dpCreate {
		slog.Info("sync", "msg", "will create data product", "file", idToFile[dp.Id], "name", dp.Name, "resource name", dp.Id)
	}
	for _, dp := range changes.dpUpdate {
		slog.Info("sync", "msg", "will update data product", "file", idToFile[dp.Id], "name", dp.Name, "resource name", dp.Id)
	}
	for _, es := range changes.esCreate {
		slog.Info("sync", "msg", "will create event specifications", "file", idToFile[es.Id], "name", es.Name, "resource name", es.Id)
	}
	for _, es := range changes.esUpdate {
		slog.Info("sync", "msg", "will update event specifications", "file", idToFile[es.Id], "name", es.Name, "resource name", es.Id, "in data product", es.DataProductId)
	}
	for _, es := range changes.esDelete {
		slog.Info("sync", "msg", "will delete event specifications", "name", es.Name, "resource name", es.Id, "in data product", es.DataProductId, "data product name", idToFile[es.Id])
	}
	logImagesToPublish(changes.imageCreate)
	slog.Info("sync", "msg", "total entities to update", "data products", len(changes.dpCreate)+len(changes.dpUpdate), "event specs", len(changes.esCreate)+len(changes.esUpdate)+len(changes.esDelete), "source apps", len(changes.saCreate)+len(changes.saUpdate), "images", len(changes.imageCreate))
	if isRelease {
		logEventSpecsToRelease(changes)
	}
}

func logImagesToPublish(images []TriggerImageReference) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	for _, img := range images {
		if relp, err := filepath.Rel(cwd, img.fname); err == nil {
			slog.Info("sync", "msg", "will publish image", "file", relp)
		}
	}
}

func logEventSpecsToRelease(changes DataProductChangeSet) {
	for _, es := range slices.Concat(changes.esCreate, changes.esDelete) {
		slog.Info("release", "msg", "will release event spec", "name", es.Name, "resource name", es.Id)
	}
	for _, es := range changes.esUpdate {
		slog.Info("release", "msg", "might release event spec if the changes are structural(name, data structure, rules)", "name", es.Name, "resource name", es.Id)
	}
}

type DataProductPurger interface {
	DeleteSourceApp(ctx context.Context, sa console.RemoteSourceApplication) error
	DeleteDataProduct(ctx context.Context, dp console.RemoteDataProduct) error
	FetchDataProduct(ctx context.Context) (*console.DataProductsAndRelatedResources, error)
}

func Purge(ctx context.Context, api DataProductPurger, dp map[string]map[string]any, commit bool) error {
	localResolved, err := ReadLocalDataProducts(ctx, dp)
	if err != nil {
		return err
	}
	remote, err := api.FetchDataProduct(ctx)
	if err != nil {
		return err
	}

	purgeApps, saNames := remoteSourceAppsMissingLocally(remote.SourceApplication, localResolved.SourceApps)
	purgeProds, dpNames := remoteDataProductsMissingLocally(remote.DataProducts, localResolved.DataProudcts)

	slog.Info("purge",
		"msg", fmt.Sprintf("%d source apps and %d data products", len(saNames), len(dpNames)),
		"source apps", strings.Join(saNames, "\n"),
		"data products", strings.Join(dpNames, "\n"),
	)

	if !commit {
		slog.Info("purge", "msg", "re-run command with -y/--yes to commit changes")
		return nil
	}

	if err := deleteSourceAppsAndLog(ctx, api, purgeApps); err != nil {
		return err
	}
	if err := deleteDataProductsAndLog(ctx, api, purgeProds); err != nil {
		return err
	}

	slog.Info("purge", "msg", "complete")

	return nil
}

func remoteSourceAppsMissingLocally(remote []console.RemoteSourceApplication, local []model.SourceApp) (map[string]console.RemoteSourceApplication, []string) {
	missing := map[string]console.RemoteSourceApplication{}
	for _, r := range remote {
		missing[r.Id] = r
	}
	for _, r := range local {
		delete(missing, r.ResourceName)
	}
	names := []string{}
	for _, r := range missing {
		names = append(names, r.Name)
	}
	return missing, names
}

func remoteDataProductsMissingLocally(remote []console.RemoteDataProduct, local []model.DataProduct) (map[string]console.RemoteDataProduct, []string) {
	missing := map[string]console.RemoteDataProduct{}
	for _, r := range remote {
		missing[r.Id] = r
	}
	for _, r := range local {
		delete(missing, r.ResourceName)
	}
	names := []string{}
	for _, r := range missing {
		names = append(names, r.Name)
	}
	return missing, names
}

func deleteSourceAppsAndLog(ctx context.Context, api DataProductPurger, apps map[string]console.RemoteSourceApplication) error {
	for n, r := range apps {
		slog.Debug("purge", "msg", "deleting remote", "source app", n)
		if err := api.DeleteSourceApp(ctx, r); err != nil {
			slog.Error("purge", "msg", "failed", "source app", n)
			return fmt.Errorf("purge failed: %w", err)
		}

		slog.Debug("purge", "msg", "deleted", "source app", n)
	}
	return nil
}

func deleteDataProductsAndLog(ctx context.Context, api DataProductPurger, products map[string]console.RemoteDataProduct) error {
	for n, r := range products {
		slog.Debug("purge", "msg", "deleting remote", "data product", n)
		if err := api.DeleteDataProduct(ctx, r); err != nil {
			slog.Error("purge", "msg", "failed", "data product", n)
			return fmt.Errorf("purge failed: %w", err)
		}

		slog.Debug("purge", "msg", "deleted", "data product", n)
	}
	return nil
}

func FindChanges(cnx context.Context, client *console.ApiClient, dp map[string]map[string]any) (*DataProductChangeSet, error) {
	localResolved, err := ReadLocalDataProducts(cnx, dp)
	if err != nil {
		return nil, err
	}
	remote, err := console.GetDataProductsAndRelatedResources(cnx, client)
	if err != nil {
		return nil, err
	}
	hashLookup, err := console.GetImageHashLookup(cnx, client)
	if err != nil {
		return nil, err
	}
	changeSet, err := findChanges(*localResolved, *remote, hashLookup)
	if err != nil {
		return nil, err
	}

	return changeSet, err
}

func Sync(cnx context.Context, client *console.ApiClient, changeSet *DataProductChangeSet, dryRun bool, isRelease bool) error {
	PrintChangeset(*changeSet, changeSet.IdToFileName, isRelease)
	var err error
	if !dryRun && !changeSet.isEmpty() {
		err = ApplyDpChanges(*changeSet, cnx, client)
	}
	return err
}

func Release(cnx context.Context, client *console.ApiClient, changeSet *DataProductChangeSet, dryRun bool) error {
	err := Sync(cnx, client, changeSet, dryRun, true)
	if err != nil {
		return err
	}

	toRelease, skipped, err := GetEventSpecIdsToRelease(cnx, client, changeSet.localEventSpecIds)
	if err != nil {
		return err
	}

	if skipped > 0 {
		slog.Warn("release", "msg", "some event specs were not released due to absent event", "count", skipped)
	}

	if len(toRelease) == 0 {
		slog.Info("release", "msg", "no event specs to release")
		return nil
	}

	for _, es := range toRelease {
		slog.Info("release", "msg", "will release event spec", "name", es.Name, "resource name", es.Id)
	}

	if dryRun {
		return nil
	}

	ids := make([]string, len(toRelease))
	var expectedVersions []console.EventSpecVersionRef
	for i, es := range toRelease {
		ids[i] = es.Id
		if v := changeSet.declaredVersion(es.Id); v != nil {
			expectedVersions = append(expectedVersions, console.EventSpecVersionRef{Id: es.Id, Version: *v})
		}
	}
	err = console.BatchPublishEventSpecs(cnx, client, ids, expectedVersions)
	if err != nil {
		return err
	}
	slog.Info("release", "msg", "successfully released event specs", "count", len(toRelease))

	return nil
}

func GetEventSpecIdsToRelease(cnx context.Context, client *console.ApiClient, localIds []string) ([]console.RemoteEventSpec, int, error) {
	skipped := 0
	remote, err := console.GetDataProductsAndRelatedResources(cnx, client)
	if err != nil {
		return nil, skipped, err
	}

	localIdSet := make(map[string]bool, len(localIds))
	for _, id := range localIds {
		localIdSet[id] = true
	}

	var unpublished []console.RemoteEventSpec
	for _, es := range remote.EventSpecs {
		if (es.Status == "draft" || es.Status == "failed") && localIdSet[es.Id] {
			if es.Event != nil {
				unpublished = append(unpublished, es)
			} else {
				skipped++
			}
		}
	}
	return unpublished, skipped, nil
}

func LockChanged(changeSet *DataProductChangeSet, managedFrom string) {
	lockDataProducts(changeSet.dpCreate, managedFrom)
	lockDataProducts(changeSet.dpUpdate, managedFrom)
	lockSourceApps(changeSet.saCreate, managedFrom)
	lockSourceApps(changeSet.saUpdate, managedFrom)
	lockEventSpecs(changeSet.esCreate, managedFrom)
	lockEventSpecs(changeSet.esUpdate, managedFrom)
}

func lockDataProducts(dps []console.RemoteDataProduct, managedFrom string) {
	for i := range dps {
		dps[i].LockStatus = "locked"
		if managedFrom != "" {
			dps[i].ManagedFrom = managedFrom
		}
	}
}

func lockSourceApps(sas []console.RemoteSourceApplication, managedFrom string) {
	for i := range sas {
		sas[i].LockStatus = "locked"
		if managedFrom != "" {
			sas[i].ManagedFrom = managedFrom
		}
	}
}

func lockEventSpecs(ess []console.RemoteEventSpec, managedFrom string) {
	for i := range ess {
		ess[i].LockStatus = "locked"
		if managedFrom != "" {
			ess[i].ManagedFrom = managedFrom
		}
	}
}

func withoutDuplicatedEventSpecs(missing []model.EventSpecMissingVersion, ids []string, count map[string]int) []model.EventSpecMissingVersion {
	var res []model.EventSpecMissingVersion
	last := make(map[string]int)
	for i, id := range ids {
		last[id] = i
	}
	for i, m := range missing {
		if count[ids[i]] > 1 {
			if last[ids[i]] != i {
				continue
			}
			m.Expected = nil
			m.Reason = missingVersionDuplicateReason
		}
		res = append(res, m)
	}
	return res
}

const missingVersionDuplicateReason = "defined in more than one file"

func duplicateEventSpecError(id string, firstFile string, secondFile string) error {
	files := []string{firstFile, secondFile}
	slices.Sort(files)
	return fmt.Errorf("event specification %s is defined in both %s and %s", id, files[0], files[1])
}

type localEventSpec struct {
	fileName string
	index    int
	spec     model.EventSpec
}

func localEventSpecsById(files map[string]map[string]any, rejectDuplicates bool) (map[string]localEventSpec, []string, error) {
	specs := make(map[string]localEventSpec)
	versioned := make(map[string]bool)
	var ids []string

	for _, f := range slices.Sorted(maps.Keys(files)) {
		eventSpecs, err := eventSpecsOfDataProductFile(files[f])
		if err != nil {
			return nil, nil, err
		}
		for idx, es := range eventSpecs {
			if previous, seen := specs[es.ResourceName]; seen && rejectDuplicates && (es.Version != nil || versioned[es.ResourceName]) {
				return nil, nil, duplicateEventSpecError(es.ResourceName, previous.fileName, f)
			}
			specs[es.ResourceName] = localEventSpec{fileName: f, index: idx, spec: es}
			if es.Version != nil {
				versioned[es.ResourceName] = true
			}
			ids = append(ids, es.ResourceName)
		}
	}

	return specs, ids, nil
}

func eventSpecsOfDataProductFile(file map[string]any) ([]model.EventSpec, error) {
	if resourceType, ok := file["resourceType"]; !ok || resourceType != "data-product" {
		return nil, nil
	}
	var dp model.DataProduct
	if err := mapstructure.Decode(file, &dp); err != nil {
		return nil, err
	}
	return dp.Data.EventSpecifications, nil
}

func FindVersionMismatchesAgainstBase(local map[string]map[string]any, base map[string]map[string]any) ([]model.EventSpecVersionMismatch, []string, error) {
	localSpecs, localIds, err := localEventSpecsById(local, true)
	if err != nil {
		return nil, nil, err
	}
	baseSpecs, _, err := localEventSpecsById(base, false)
	if err != nil {
		return nil, nil, err
	}

	var versionMismatches []model.EventSpecVersionMismatch
	var skipped []string

	for _, id := range localIds {
		localEs := localSpecs[id]
		if localEs.spec.Version == nil {
			continue
		}

		expectedVersion := 0
		changedFields := []string{}
		baseEs, baseExists := baseSpecs[id]
		if baseExists {
			if baseEs.spec.Version == nil {
				skipped = append(skipped, localEs.spec.Name)
				continue
			}
			expectedVersion, changedFields, err = expectedVersionAgainstBase(baseEs.spec, localEs.spec)
			if err != nil {
				return nil, nil, err
			}
		}

		if *localEs.spec.Version != expectedVersion {
			versionMismatches = append(versionMismatches, model.EventSpecVersionMismatch{
				FileName: localEs.fileName,
				Index:    localEs.index,
				Name:     localEs.spec.Name,
				Declared: *localEs.spec.Version,
				Expected: expectedVersion,
				IsNew:    !baseExists,
				Changed:  changedFields,
			})
		}
	}

	return versionMismatches, skipped, nil
}

func expectedVersionAgainstBase(baseEs model.EventSpec, localEs model.EventSpec) (int, []string, error) {
	changedFields, err := changedSentStructuralFields(
		LocalEventSpecToRemote(baseEs, nil, ""),
		LocalEventSpecToRemote(localEs, nil, ""),
	)
	if err != nil {
		return 0, nil, err
	}
	if len(changedFields) > 0 {
		return *baseEs.Version + 1, changedFields, nil
	}
	return *baseEs.Version, changedFields, nil
}

func FindMissingVersionsAgainstBase(local map[string]map[string]any, base map[string]map[string]any) ([]model.EventSpecMissingVersion, error) {
	localSpecs, localIds, err := localEventSpecsById(local, true)
	if err != nil {
		return nil, err
	}
	baseSpecs, _, err := localEventSpecsById(base, false)
	if err != nil {
		return nil, err
	}

	count := make(map[string]int)
	for _, id := range localIds {
		count[id]++
	}

	var missing []model.EventSpecMissingVersion
	reported := make(map[string]bool)

	for _, id := range localIds {
		localEs := localSpecs[id]
		if localEs.spec.Version != nil || reported[id] {
			continue
		}
		reported[id] = true

		baseEs, baseExists := baseSpecs[id]
		m, err := missingVersionAgainstBase(localEs, baseEs, baseExists, count[id] > 1)
		if err != nil {
			return nil, err
		}

		missing = append(missing, m)
	}

	return missing, nil
}

func missingVersionAgainstBase(localEs localEventSpec, baseEs localEventSpec, baseExists bool, duplicated bool) (model.EventSpecMissingVersion, error) {
	m := model.EventSpecMissingVersion{
		FileName: localEs.fileName,
		Index:    localEs.index,
		Name:     localEs.spec.Name,
	}

	switch {
	case duplicated:
		m.Reason = missingVersionDuplicateReason
	case !baseExists:
		expected := 0
		m.Expected = &expected
	case baseEs.spec.Version == nil:
		m.Reason = "no version declared at the base, run dp validate --fix against Snowplow Console to add it"
	default:
		expected, _, err := expectedVersionAgainstBase(baseEs.spec, localEs.spec)
		if err != nil {
			return model.EventSpecMissingVersion{}, err
		}
		m.Expected = &expected
	}

	return m, nil
}
