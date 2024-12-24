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
	"reflect"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/snowplow-product/snowplow-cli/internal/console"
	"github.com/snowplow-product/snowplow-cli/internal/model"
)

func ReadLocalDataProducts_Test(t *testing.T) {
	input := map[string]map[string]any{
		"/base/source-apps/a/b/file1.yml": {
			"apiVersion":   "v1",
			"resourceType": "source-application",
			"resourceName": "c3152a3f-4817-4300-9538-aaf3c9d92e0b",
		},
		"/base/source-apps/a/file1.yml": {
			"apiVersion":   "v1",
			"resourceType": "source-application",
			"resourceName": "92017fd4-a87d-4cc4-9268-5f945fb5601d",
		},
		"/base/somedir/file2.yml": {
			"apiVersion":   "v1",
			"resourceType": "data-product",
			"resourceName": "dcdfd5cd-2a87-49c6-b291-62f904ac2f01",
			"data": map[string]any{
				"sourceApplications": []map[string]string{
					{"$ref": "../source-apps/a/../a/file1.yml"},
					{"$ref": "../source-apps/a/../a/b/../b/file1.yml"},
				},
			},
		},
	}

	expected := LocalFilesRefsResolved{
		DataProudcts: []model.DataProduct{{
			ApiVersion:   "v1",
			ResourceType: "data-product",
			ResourceName: "dcdfd5cd-2a87-49c6-b291-62f904ac2f01",
			Data: model.DataProductData{
				ResourceName:        "dcdfd5cd-2a87-49c6-b291-62f904ac2f01",
				Name:                "",
				SourceApplications:  []map[string]string{{"id": "92017fd4-a87d-4cc4-9268-5f945fb5601d"}, {"id": "c3152a3f-4817-4300-9538-aaf3c9d92e0b"}},
				Domain:              "",
				Owner:               "",
				Description:         "",
				EventSpecifications: []model.EventSpec{},
			},
		}},
		SourceApps: []model.SourceApp{{
			ApiVersion:   "v1",
			ResourceType: "source-application",
			ResourceName: "c3152a3f-4817-4300-9538-aaf3c9d92e0b",
			Data:         model.SourceAppData{},
		},
			{
				ApiVersion:   "v1",
				ResourceType: "source-application",
				ResourceName: "92017fd4-a87d-4cc4-9268-5f945fb5601d",
				Data:         model.SourceAppData{},
			}},
	}

	result, err := ReadLocalDataProducts(input)
	if err != nil {
		t.Errorf("ReadLocalDataProducts_Test error %v", err)
	}

	if !reflect.DeepEqual(expected, *result) {
		t.Errorf("localSaToRemote() = %v, want %v", result, expected)
	}
}

func Test_findChanges_NoChanges(t *testing.T) {
	sa := model.SourceApp{
		ResourceName: "9b6e4e4c-c34a-483c-a5e7-f728d66a53b3",
		Data: model.SourceAppData{
			Name: "Source App 1",
		},
	}
	dp := model.DataProduct{
		ResourceName: "d9967fb4-6233-49c1-94d2-6cc417abd7ed",
		Data: model.DataProductData{
			Name:               "Data Product 1",
			SourceApplications: []map[string]string{{"id": "9b6e4e4c-c34a-483c-a5e7-f728d66a53b3"}},
		},
	}
	local := LocalFilesRefsResolved{
		SourceApps:   []model.SourceApp{sa},
		DataProudcts: []model.DataProduct{dp},
	}

	remoteSa := console.RemoteSourceApplication{
		Id:   "9b6e4e4c-c34a-483c-a5e7-f728d66a53b3",
		Name: "Source App 1",
	}
	remoteDp := console.RemoteDataProduct{
		Id:                   "d9967fb4-6233-49c1-94d2-6cc417abd7ed",
		Name:                 "Data Product 1",
		SourceApplicationIds: []string{"9b6e4e4c-c34a-483c-a5e7-f728d66a53b3"},
	}
	remote := console.DataProductsAndRelatedResources{
		SourceApplication: []console.RemoteSourceApplication{remoteSa},
		DataProducts:      []console.RemoteDataProduct{remoteDp},
		EventSpecs:        []console.RemoteEventSpec{},
	}

	changes, err := findChanges(local, remote, map[string]string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !changes.isEmpty() {
		t.Errorf("expected changes to be empty, got: %+v", changes)
	}
}

func Test_findChanges_CreateAll(t *testing.T) {
	sa := model.SourceApp{
		ResourceName: "9b6e4e4c-c34a-483c-a5e7-f728d66a53b3",
		Data: model.SourceAppData{
			Name: "Source App 1",
		},
	}
	dp := model.DataProduct{
		ResourceName: "d9967fb4-6233-49c1-94d2-6cc417abd7ed",
		Data: model.DataProductData{
			Name:               "Data Product 1",
			SourceApplications: []map[string]string{{"id": "9b6e4e4c-c34a-483c-a5e7-f728d66a53b3"}},
			EventSpecifications: []model.EventSpec{{
				ResourceName: "6557ceee-a1c7-4ea9-910a-2a97194fb3f6",
				Name:         "Event Spec 1",
			}},
		},
	}
	local := LocalFilesRefsResolved{
		SourceApps:   []model.SourceApp{sa},
		DataProudcts: []model.DataProduct{dp},
	}

	// Empty remote state
	remote := console.DataProductsAndRelatedResources{
		SourceApplication: []console.RemoteSourceApplication{},
		DataProducts:      []console.RemoteDataProduct{},
		EventSpecs:        []console.RemoteEventSpec{},
	}

	changes, err := findChanges(local, remote, map[string]string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if changes.isEmpty() {
		t.Error("expected changes to not be empty")
	}
	if len(changes.saCreate) != 1 {
		t.Errorf("expected 1 source app creation, got %d", len(changes.saCreate))
	}
	if len(changes.dpCreate) != 1 {
		t.Errorf("expected 1 data product creation, got %d", len(changes.dpCreate))
	}
	if len(changes.esCreate) != 1 {
		t.Errorf("expected 1 event spec creation, got %d", len(changes.esCreate))
	}
}

func Test_findChanges_UpdateAll(t *testing.T) {
	sa := model.SourceApp{
		ResourceName: "9b6e4e4c-c34a-483c-a5e7-f728d66a53b3",
		Data: model.SourceAppData{
			Name: "Source App 1 Updated",
		},
	}
	dp := model.DataProduct{
		ResourceName: "d9967fb4-6233-49c1-94d2-6cc417abd7ed",
		Data: model.DataProductData{
			Name:               "Data Product 1 Updated",
			SourceApplications: []map[string]string{{"id": "9b6e4e4c-c34a-483c-a5e7-f728d66a53b3"}},
			EventSpecifications: []model.EventSpec{{
				ResourceName: "6557ceee-a1c7-4ea9-910a-2a97194fb3f6",
				Name:         "Event Spec 1 Updated",
			}},
		},
	}
	local := LocalFilesRefsResolved{
		SourceApps:   []model.SourceApp{sa},
		DataProudcts: []model.DataProduct{dp},
	}

	remoteSa := console.RemoteSourceApplication{
		Id:   "9b6e4e4c-c34a-483c-a5e7-f728d66a53b3",
		Name: "Source App 1",
	}
	remoteDp := console.RemoteDataProduct{
		Id:                   "d9967fb4-6233-49c1-94d2-6cc417abd7ed",
		Name:                 "Data Product 1",
		Status:               "",
		SourceApplicationIds: []string{"9b6e4e4c-c34a-483c-a5e7-f728d66a53b3"},
		Domain:               "",
		Owner:                "",
		Description:          "",
		EventSpecs: []console.EventSpecReference{{
			Id: "6557ceee-a1c7-4ea9-910a-2a97194fb3f6",
		}},
	}

	remoteEs := console.RemoteEventSpec{
		Id:   "6557ceee-a1c7-4ea9-910a-2a97194fb3f6",
		Name: "Event Spec 1",
	}

	remote := console.DataProductsAndRelatedResources{
		SourceApplication: []console.RemoteSourceApplication{remoteSa},
		DataProducts:      []console.RemoteDataProduct{remoteDp},
		EventSpecs:        []console.RemoteEventSpec{remoteEs},
	}

	changes, err := findChanges(local, remote, map[string]string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if changes.isEmpty() {
		t.Error("expected changes to not be empty")
	}
	if len(changes.saUpdate) != 1 {
		t.Errorf("expected 1 source app update, got %d", len(changes.saUpdate))
	}
	if len(changes.dpUpdate) != 1 {
		t.Errorf("expected 1 data product update, got %d", len(changes.dpUpdate))
	}
	if len(changes.esUpdate) != 1 {
		t.Errorf("expected 1 event spec update, got %d", len(changes.esUpdate))
	}
}

type mockPurgeApi struct {
	remoteResources     *console.DataProductsAndRelatedResources
	deletedSourceApps   []string
	deletedDataProducts []string
}

func (a *mockPurgeApi) DeleteSourceApp(sa console.RemoteSourceApplication) error {
	a.deletedSourceApps = append(a.deletedSourceApps, sa.Id)
	return nil
}

func (a *mockPurgeApi) DeleteDataProduct(dp console.RemoteDataProduct) error {
	a.deletedDataProducts = append(a.deletedDataProducts, dp.Id)
	return nil
}

func (a *mockPurgeApi) FetchDataProduct() (*console.DataProductsAndRelatedResources, error) {
	return a.remoteResources, nil
}

func Test_Purge(t *testing.T) {
	dp := map[string]map[string]any{
		"file1.yml": {
			"apiVersion":   "v1",
			"resourceType": "source-application",
			"resourceName": "do not purgeme",
		},
		"file2.yml": {
			"apiVersion":   "v1",
			"resourceType": "data-product",
			"resourceName": "dp no purge",
		},
	}

	remote := &console.DataProductsAndRelatedResources{
		SourceApplication: []console.RemoteSourceApplication{
			{Id: "purgeme"},
			{Id: "do not purgeme"},
		},
		DataProducts: []console.RemoteDataProduct{
			{Id: "dp no purge"},
			{Id: "dp purge"},
		},
	}

	mockApi := &mockPurgeApi{remoteResources: remote}

	_ = Purge(mockApi, dp, true)

	if !slices.Equal(mockApi.deletedSourceApps, []string{"purgeme"}) {
		t.Fatal("deleted the wrong source apps")
	}

	if !slices.Equal(mockApi.deletedDataProducts, []string{"dp purge"}) {
		t.Fatal("deleted the wrong data products")
	}
}

func Test_PurgeNoCommit(t *testing.T) {
	dp := map[string]map[string]any{}

	remote := &console.DataProductsAndRelatedResources{
		SourceApplication: []console.RemoteSourceApplication{
			{Id: "do not purgeme"},
		},
		DataProducts: []console.RemoteDataProduct{
			{Id: "dp no purge"},
		},
	}

	mockApi := &mockPurgeApi{remoteResources: remote}

	_ = Purge(mockApi, dp, false)

	if len(mockApi.deletedDataProducts) > 0 {
		t.Fatal("deleted source apps")
	}

	if len(mockApi.deletedDataProducts) > 0 {
		t.Fatal("deleted data products")
	}
}

func Test_findChanges_DeleteEventSpec(t *testing.T) {
	sa := model.SourceApp{
		ResourceName: "9b6e4e4c-c34a-483c-a5e7-f728d66a53b3",
		Data: model.SourceAppData{
			Name: "Source App 1",
		},
	}
	dp := model.DataProduct{
		ResourceName: "d9967fb4-6233-49c1-94d2-6cc417abd7ed",
		Data: model.DataProductData{
			Name:                "Data Product 1",
			SourceApplications:  []map[string]string{{"id": "9b6e4e4c-c34a-483c-a5e7-f728d66a53b3"}},
			EventSpecifications: []model.EventSpec{},
		},
	}
	local := LocalFilesRefsResolved{
		SourceApps:   []model.SourceApp{sa},
		DataProudcts: []model.DataProduct{dp},
	}

	remoteSa := console.RemoteSourceApplication{
		Id:   "9b6e4e4c-c34a-483c-a5e7-f728d66a53b3",
		Name: "Source App 1",
	}
	remoteDp := console.RemoteDataProduct{
		Id:                   "d9967fb4-6233-49c1-94d2-6cc417abd7ed",
		Name:                 "Data Product 1",
		Status:               "",
		SourceApplicationIds: []string{"9b6e4e4c-c34a-483c-a5e7-f728d66a53b3"},
		Domain:               "",
		Owner:                "",
		Description:          "",
		EventSpecs: []console.EventSpecReference{{
			Id: "6557ceee-a1c7-4ea9-910a-2a97194fb3f6",
		}},
	}

	remoteEs := console.RemoteEventSpec{
		Id:   "6557ceee-a1c7-4ea9-910a-2a97194fb3f6",
		Name: "Event Spec 1",
	}

	remote := console.DataProductsAndRelatedResources{
		SourceApplication: []console.RemoteSourceApplication{remoteSa},
		DataProducts:      []console.RemoteDataProduct{remoteDp},
		EventSpecs:        []console.RemoteEventSpec{remoteEs},
	}

	changes, err := findChanges(local, remote, map[string]string{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if changes.isEmpty() {
		t.Error("expected changes to not be empty")
	}
	if len(changes.esDelete) != 1 {
		t.Errorf("expected 1 event spec delete, got %d", len(changes.esDelete))
	}
}

func Test_findTriggerChanges_OK(t *testing.T) {
	sharedTrigger := console.RemoteTrigger{
		Id:          "f8500e68-4d1d-491c-b413-61f643e310aa",
		Description: "trigger 1",
		AppIds:      []string{"OK"},
		Url:         "",
		VariantUrls: map[string]string{},
	}
	deletedTrigger := console.RemoteTrigger{
		Id:          "a191e2ab-b3e8-4d7d-8218-962c19469675",
		Description: "trigger 2",
		AppIds:      []string{"OK"},
		Url:         "",
		VariantUrls: map[string]string{"original": "test.com/dcddfa7a-67ca-4a48-9d8a-0aaf69594e65/original"},
	}
	addedTrigger := console.RemoteTrigger{
		Id:          "74a9496c-dc39-4885-bf1e-56aec455f50e",
		Description: "trigger 3",
		AppIds:      []string{"OK"},
		Url:         "",
		VariantUrls: map[string]string{},
	}
	changedTrigger1_local := console.RemoteTrigger{
		Id:          "f15fa249-3a25-4479-a47d-571f1d12eafc",
		Description: "trigger 4 descrtiption change",
		AppIds:      []string{"OK"},
		Url:         "",
		VariantUrls: map[string]string{},
	}
	changedTrigger1_remote := console.RemoteTrigger{
		Id:          "f15fa249-3a25-4479-a47d-571f1d12eafc",
		Description: "trigger 4",
		AppIds:      []string{"OK"},
		Url:         "",
		VariantUrls: map[string]string{"original": "test.com/dcddfa7a-67ca-4a48-9d8a-0aaf69594e65/original"},
	}
	changedTrigger2_local := console.RemoteTrigger{
		Id:          "aa0e98d2-7f0e-4461-bd56-b3d2a7bcb113",
		Description: "trigger 5",
		AppIds:      []string{"OK", "app id change"},
		Url:         "",
		VariantUrls: map[string]string{},
	}
	changedTrigger2_remote := console.RemoteTrigger{
		Id:          "aa0e98d2-7f0e-4461-bd56-b3d2a7bcb113",
		Description: "trigger 5",
		AppIds:      []string{"OK"},
		Url:         "",
		VariantUrls: map[string]string{"original": "test.com/dcddfa7a-67ca-4a48-9d8a-0aaf69594e65/original"},
	}
	changedTrigger3_local := console.RemoteTrigger{
		Id:          "54b87f2b-c13c-459f-9fa2-20c6abdb85b3",
		Description: "trigger 6",
		AppIds:      []string{"OK"},
		Url:         "url.change",
		VariantUrls: map[string]string{},
	}
	changedTrigger3_remote := console.RemoteTrigger{
		Id:          "54b87f2b-c13c-459f-9fa2-20c6abdb85b3",
		Description: "trigger 6",
		AppIds:      []string{"OK"},
		Url:         "",
		VariantUrls: map[string]string{"original": "test.com/dcddfa7a-67ca-4a48-9d8a-0aaf69594e65/original"},
	}
	localTriggers := []console.RemoteTrigger{sharedTrigger, addedTrigger, changedTrigger1_local, changedTrigger2_local, changedTrigger3_local}
	remoteTriggers := []console.RemoteTrigger{sharedTrigger, deletedTrigger, changedTrigger1_remote, changedTrigger2_remote, changedTrigger3_remote}
	res := findTriggerChanges(localTriggers, remoteTriggers, map[string]TriggerImageReference{}, map[string]string{})
	expected := triggerChangeset{
		isChanged:      true,
		imagesToUpload: []TriggerImageReference{},
		triggersWithOriginalVariantUrls: []console.RemoteTrigger{
			sharedTrigger,
			{
				Id:          "74a9496c-dc39-4885-bf1e-56aec455f50e",
				Description: "trigger 3",
				AppIds:      []string{"OK"},
				Url:         "",
				VariantUrls: map[string]string{},
			},
			{
				Id:          "f15fa249-3a25-4479-a47d-571f1d12eafc",
				Description: "trigger 4 descrtiption change",
				AppIds:      []string{"OK"},
				Url:         "",
				VariantUrls: map[string]string{"original": "test.com/dcddfa7a-67ca-4a48-9d8a-0aaf69594e65/original"},
			},
			{
				Id:          "aa0e98d2-7f0e-4461-bd56-b3d2a7bcb113",
				Description: "trigger 5",
				AppIds:      []string{"OK", "app id change"},
				Url:         "",
				VariantUrls: map[string]string{"original": "test.com/dcddfa7a-67ca-4a48-9d8a-0aaf69594e65/original"},
			},
			{
				Id:          "54b87f2b-c13c-459f-9fa2-20c6abdb85b3",
				Description: "trigger 6",
				AppIds:      []string{"OK"},
				Url:         "url.change",
				VariantUrls: map[string]string{"original": "test.com/dcddfa7a-67ca-4a48-9d8a-0aaf69594e65/original"},
			},
		},
	}
	if !res.isChanged {
		t.Errorf("findTriggerChanges().isChanged = %t, want %t", res.isChanged, true)
	}
	if diff := cmp.Diff(expected.triggersWithOriginalVariantUrls, res.triggersWithOriginalVariantUrls, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("findTriggerChanges().triggersWithOriginalVariantUrls mismatch (-want +got):\n%s", diff)
	}
}

func Test_findTriggerChanges_ImageDiff(t *testing.T) {
	localTrigger := console.RemoteTrigger{
		Id:          "f8500e68-4d1d-491c-b413-61f643e310aa",
		Description: "trigger 1",
		AppIds:      []string{"OK"},
		Url:         "",
		VariantUrls: map[string]string{},
	}
	remoteTrigger := console.RemoteTrigger{
		Id:          localTrigger.Id,
		Description: "trigger 1",
		AppIds:      []string{"OK"},
		Url:         "",
		VariantUrls: map[string]string{"original": "test.com/dcddfa7a-67ca-4a48-9d8a-0aaf69594e65/original"},
	}
	triggerToCreateWithImage := console.RemoteTrigger{
		Id:          "09ae9c08-4bfa-42cf-a256-81fbedfc5c09",
		Description: "trigger 2",
		AppIds:      []string{"OK"},
		Url:         "",
		VariantUrls: map[string]string{},
	}
	notChangedTriggerLocal := console.RemoteTrigger{
		Id:          "41cdcdae-9a32-48e8-8a3b-875e6f284cbe",
		Description: "trigger 3",
		AppIds:      []string{"OK"},
		Url:         "",
		VariantUrls: map[string]string{},
	}
	notChangedTriggerRemote := console.RemoteTrigger{
		Id:          notChangedTriggerLocal.Id,
		Description: "trigger 3",
		AppIds:      []string{"OK"},
		Url:         "",
		VariantUrls: map[string]string{"original": "test.com/97c5fa2c-7159-4c69-921f-112acdd58b50/original"},
	}

	localImagesByTriggerId := map[string]TriggerImageReference{
		localTrigger.Id: TriggerImageReference{
			eventSpecId: "2fc5caf4-aa7a-477e-b200-aeb7edfda51d",
			triggerId:   localTrigger.Id,
			fname:       ".images/1.png",
			hash:        "326f4188c4850f06064996d2f0120eec7ccdaefa6080a9bb19f6b012c46fef69",
		},
		triggerToCreateWithImage.Id: TriggerImageReference{
			eventSpecId: "8cfc1547-d8d2-4498-bab7-066c7a23a821",
			triggerId:   triggerToCreateWithImage.Id,
			fname:       ".images/2.png",
			hash:        "0d7baa385d770797c54d69437473fdd378c3cf646a7469a1a7b54770fd53d24b",
		},
		notChangedTriggerLocal.Id: TriggerImageReference{
			eventSpecId: "8cfc1547-d8d2-4498-bab7-066c7a23a821",
			triggerId:   notChangedTriggerLocal.Id,
			fname:       ".images/3.png",
			hash:        "b5cbf7b50e2559f609e94f26a9b7d268f87caeff12f6982558cd68e4a877da4e",
		},
	}
	remoteImageHashById := map[string]string{
		"dcddfa7a-67ca-4a48-9d8a-0aaf69594e65": "57970a73313c4f20ea526c23d17ee53a9110c17331ce298f2201c895132c4110",
		"97c5fa2c-7159-4c69-921f-112acdd58b50": "b5cbf7b50e2559f609e94f26a9b7d268f87caeff12f6982558cd68e4a877da4e",
	}
	localTriggers := []console.RemoteTrigger{localTrigger, triggerToCreateWithImage, notChangedTriggerLocal}
	remoteTriggers := []console.RemoteTrigger{remoteTrigger, notChangedTriggerRemote}

	res := findTriggerChanges(localTriggers, remoteTriggers, localImagesByTriggerId, remoteImageHashById)
	expected := []TriggerImageReference{
		{
			eventSpecId: "2fc5caf4-aa7a-477e-b200-aeb7edfda51d",
			triggerId:   localTrigger.Id,
			fname:       ".images/1.png",
			hash:        "326f4188c4850f06064996d2f0120eec7ccdaefa6080a9bb19f6b012c46fef69",
		},
		{
			eventSpecId: "8cfc1547-d8d2-4498-bab7-066c7a23a821",
			triggerId:   triggerToCreateWithImage.Id,
			fname:       ".images/2.png",
			hash:        "0d7baa385d770797c54d69437473fdd378c3cf646a7469a1a7b54770fd53d24b",
		},
	}

	if diff := cmp.Diff(expected, res.imagesToUpload, cmpopts.EquateEmpty(), cmp.AllowUnexported(TriggerImageReference{})); diff != "" {
		t.Errorf("findTriggerChanges().imagesToUpload mismatch (-want +got):\n%s", diff)
	}

}

func Test_findTriggerChanges_MissingOriginal(t *testing.T) {
	localTrigger := console.RemoteTrigger{
		Id:          "f8500e68-4d1d-491c-b413-61f643e310aa",
		Description: "trigger 1",
		AppIds:      []string{"OK"},
		Url:         "",
		VariantUrls: map[string]string{},
	}
	remoteTrigger := console.RemoteTrigger{
		Id:          localTrigger.Id,
		Description: "trigger 1",
		AppIds:      []string{"OK"},
		Url:         "",
		VariantUrls: map[string]string{"no_original": "test.com/dcddfa7a-67ca-4a48-9d8a-0aaf69594e65/original"},
	}
	localImagesByTriggerId := map[string]TriggerImageReference{
		localTrigger.Id: TriggerImageReference{
			eventSpecId: "2fc5caf4-aa7a-477e-b200-aeb7edfda51d",
			triggerId:   localTrigger.Id,
			fname:       ".images/1.png",
			hash:        "326f4188c4850f06064996d2f0120eec7ccdaefa6080a9bb19f6b012c46fef69",
		},
	}
	remoteImageHashById := map[string]string{
		"dcddfa7a-67ca-4a48-9d8a-0aaf69594e65": "57970a73313c4f20ea526c23d17ee53a9110c17331ce298f2201c895132c4110",
	}
	localTriggers := []console.RemoteTrigger{localTrigger}
	remoteTriggers := []console.RemoteTrigger{remoteTrigger}

	res := findTriggerChanges(localTriggers, remoteTriggers, localImagesByTriggerId, remoteImageHashById)

	expected := []TriggerImageReference{
		{
			eventSpecId: "2fc5caf4-aa7a-477e-b200-aeb7edfda51d",
			triggerId:   localTrigger.Id,
			fname:       ".images/1.png",
			hash:        "326f4188c4850f06064996d2f0120eec7ccdaefa6080a9bb19f6b012c46fef69",
		},
	}

	if diff := cmp.Diff(expected, res.imagesToUpload, cmpopts.EquateEmpty(), cmp.AllowUnexported(TriggerImageReference{})); diff != "" {
		t.Errorf("findTriggerChanges().imagesToUpload mismatch (-want +got):\n%s", diff)
	}

}
