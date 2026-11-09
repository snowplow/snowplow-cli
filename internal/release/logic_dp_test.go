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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/model"
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

	result, err := ReadLocalDataProducts(context.Background(), input)
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

	_ = Purge(context.Background(), mockApi, dp, true)

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

	_ = Purge(context.Background(), mockApi, dp, false)

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

func Test_LockChanged(t *testing.T) {
	changeSet := &DataProductChangeSet{
		dpCreate: []console.RemoteDataProduct{{}},
		dpUpdate: []console.RemoteDataProduct{{}},
		saCreate: []console.RemoteSourceApplication{{}},
		saUpdate: []console.RemoteSourceApplication{{}},
		esCreate: []console.RemoteEventSpec{{}},
		esUpdate: []console.RemoteEventSpec{{}},
	}

	LockChanged(changeSet, "my repo")

	results := []bool{
		changeSet.dpCreate[0].LockStatus == "locked",
		changeSet.dpCreate[0].ManagedFrom == "my repo",
		changeSet.dpUpdate[0].LockStatus == "locked",
		changeSet.dpUpdate[0].ManagedFrom == "my repo",

		changeSet.saCreate[0].LockStatus == "locked",
		changeSet.saCreate[0].ManagedFrom == "my repo",
		changeSet.saUpdate[0].LockStatus == "locked",
		changeSet.saUpdate[0].ManagedFrom == "my repo",

		changeSet.esCreate[0].LockStatus == "locked",
		changeSet.esCreate[0].ManagedFrom == "my repo",
		changeSet.esUpdate[0].LockStatus == "locked",
		changeSet.esUpdate[0].ManagedFrom == "my repo",
	}

	for i, r := range results {
		if !r {
			t.Errorf("lockstatus or managedfrom failure at i:%d", i)
		}
	}
}

func newTestClientAndServer(t *testing.T, eventSpecsJSON string) (*console.ApiClient, *httptest.Server) {
	t.Helper()
	client, server, _ := newTestClientAndServerWithPublishCounter(t, eventSpecsJSON)
	return client, server
}

func newTestClientAndServerWithPublishCounter(t *testing.T, eventSpecsJSON string) (*console.ApiClient, *httptest.Server, *atomic.Int32) {
	t.Helper()
	var publishCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/data-products/v2" && r.Method == "GET":
			resp := fmt.Sprintf(`{"data": [], "includes": {"eventSpecs": %s}}`, eventSpecsJSON)
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
		case r.URL.Path == "/source-apps/v1" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"data": []}`)
		case r.URL.Path == "/event-specs/v1/publish" && r.Method == "POST":
			publishCalls.Add(1)
			w.WriteHeader(http.StatusAccepted)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	client := &console.ApiClient{
		Http:    &http.Client{},
		Jwt:     "token",
		BaseUrl: server.URL,
	}
	return client, server, &publishCalls
}

func Test_GetEventSpecIdsToRelease(t *testing.T) {
	tests := []struct {
		name           string
		eventSpecsJSON string
		localIds       []string
		wantIds        []string
		wantSkipped    int
	}{
		{
			name: "draft with event is included",
			eventSpecsJSON: `[{
				"id": "es-1",
				"status": "draft",
				"name": "ES 1",
				"event": {"source": "iglu:com.example/event/jsonschema/1-0-0"},
				"entities": {"tracked": [], "enriched": []},
				"sourceApplications": []
			}]`,
			localIds:    []string{"es-1"},
			wantIds:     []string{"es-1"},
			wantSkipped: 0,
		},
		{
			name: "draft with nil event is skipped",
			eventSpecsJSON: `[{
				"id": "es-2",
				"status": "draft",
				"name": "ES 2",
				"entities": {"tracked": [], "enriched": []},
				"sourceApplications": []
			}]`,
			localIds:    []string{"es-2"},
			wantIds:     nil,
			wantSkipped: 1,
		},
		{
			name: "published event spec is excluded",
			eventSpecsJSON: `[{
				"id": "es-3",
				"status": "published",
				"name": "ES 3",
				"event": {"source": "iglu:com.example/event/jsonschema/1-0-0"},
				"entities": {"tracked": [], "enriched": []},
				"sourceApplications": []
			}]`,
			localIds:    []string{"es-3"},
			wantIds:     nil,
			wantSkipped: 0,
		},
		{
			name:           "empty localIds returns nothing",
			eventSpecsJSON: `[{"id": "es-4", "status": "draft", "name": "ES 4", "event": {"source": "iglu:com.example/event/jsonschema/1-0-0"}, "entities": {"tracked": [], "enriched": []}, "sourceApplications": []}]`,
			localIds:       []string{},
			wantIds:        nil,
			wantSkipped:    0,
		},
		{
			name: "no matching event specs",
			eventSpecsJSON: `[{
				"id": "es-5",
				"status": "draft",
				"name": "ES 5",
				"event": {"source": "iglu:com.example/event/jsonschema/1-0-0"},
				"entities": {"tracked": [], "enriched": []},
				"sourceApplications": []
			}]`,
			localIds:    []string{"es-other"},
			wantIds:     nil,
			wantSkipped: 0,
		},
		{
			name: "failed with event is included for retry",
			eventSpecsJSON: `[{
				"id": "es-failed",
				"status": "failed",
				"name": "ES Failed",
				"event": {"source": "iglu:com.example/event/jsonschema/1-0-0"},
				"entities": {"tracked": [], "enriched": []},
				"sourceApplications": []
			}]`,
			localIds:    []string{"es-failed"},
			wantIds:     []string{"es-failed"},
			wantSkipped: 0,
		},
		{
			name: "failed with nil event is skipped",
			eventSpecsJSON: `[{
				"id": "es-failed-no-event",
				"status": "failed",
				"name": "ES Failed No Event",
				"entities": {"tracked": [], "enriched": []},
				"sourceApplications": []
			}]`,
			localIds:    []string{"es-failed-no-event"},
			wantIds:     nil,
			wantSkipped: 1,
		},
		{
			name: "mixed statuses and events",
			eventSpecsJSON: `[
				{"id": "es-draft-with-event", "status": "draft", "name": "A", "event": {"source": "iglu:com.example/a/jsonschema/1-0-0"}, "entities": {"tracked": [], "enriched": []}, "sourceApplications": []},
				{"id": "es-draft-no-event", "status": "draft", "name": "B", "entities": {"tracked": [], "enriched": []}, "sourceApplications": []},
				{"id": "es-published", "status": "published", "name": "C", "event": {"source": "iglu:com.example/c/jsonschema/1-0-0"}, "entities": {"tracked": [], "enriched": []}, "sourceApplications": []},
				{"id": "es-not-local", "status": "draft", "name": "D", "event": {"source": "iglu:com.example/d/jsonschema/1-0-0"}, "entities": {"tracked": [], "enriched": []}, "sourceApplications": []},
				{"id": "es-failed-retry", "status": "failed", "name": "E", "event": {"source": "iglu:com.example/e/jsonschema/1-0-0"}, "entities": {"tracked": [], "enriched": []}, "sourceApplications": []}
			]`,
			localIds:    []string{"es-draft-with-event", "es-draft-no-event", "es-published", "es-failed-retry"},
			wantIds:     []string{"es-draft-with-event", "es-failed-retry"},
			wantSkipped: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := newTestClientAndServer(t, tt.eventSpecsJSON)
			defer server.Close()

			got, gotSkipped, err := GetEventSpecIdsToRelease(context.Background(), client, tt.localIds)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var gotIds []string
			for _, es := range got {
				gotIds = append(gotIds, es.Id)
			}
			if diff := cmp.Diff(tt.wantIds, gotIds); diff != "" {
				t.Errorf("unpublished ids mismatch (-want +got):\n%s", diff)
			}
			if gotSkipped != tt.wantSkipped {
				t.Errorf("skipped = %d, want %d", gotSkipped, tt.wantSkipped)
			}
		})
	}
}

func Test_Release_DryRunDoesNotPublish(t *testing.T) {
	eventSpecsJSON := `[{
		"id": "es-1",
		"status": "draft",
		"name": "ES 1",
		"event": {"source": "iglu:com.example/event/jsonschema/1-0-0"},
		"entities": {"tracked": [], "enriched": []},
		"sourceApplications": []
	}]`
	client, server, publishCalls := newTestClientAndServerWithPublishCounter(t, eventSpecsJSON)
	defer server.Close()

	changeSet := &DataProductChangeSet{
		IdToFileName:      map[string]string{},
		localEventSpecIds: []string{"es-1"},
	}

	err := Release(context.Background(), client, changeSet, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if publishCalls.Load() != 0 {
		t.Errorf("expected no publish calls during dry-run, got %d", publishCalls.Load())
	}
}

func Test_Release_PublishesWhenNotDryRun(t *testing.T) {
	eventSpecsJSON := `[{
		"id": "es-1",
		"status": "draft",
		"name": "ES 1",
		"event": {"source": "iglu:com.example/event/jsonschema/1-0-0"},
		"entities": {"tracked": [], "enriched": []},
		"sourceApplications": []
	}]`
	client, server, publishCalls := newTestClientAndServerWithPublishCounter(t, eventSpecsJSON)
	defer server.Close()

	changeSet := &DataProductChangeSet{
		IdToFileName:      map[string]string{},
		localEventSpecIds: []string{"es-1"},
	}

	err := Release(context.Background(), client, changeSet, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if publishCalls.Load() != 1 {
		t.Errorf("expected 1 publish call, got %d", publishCalls.Load())
	}
}

func Test_Release_DryRunNothingToRelease(t *testing.T) {
	eventSpecsJSON := `[{
		"id": "es-1",
		"status": "published",
		"name": "ES 1",
		"event": {"source": "iglu:com.example/event/jsonschema/1-0-0"},
		"entities": {"tracked": [], "enriched": []},
		"sourceApplications": []
	}]`
	client, server, publishCalls := newTestClientAndServerWithPublishCounter(t, eventSpecsJSON)
	defer server.Close()

	changeSet := &DataProductChangeSet{
		IdToFileName:      map[string]string{},
		localEventSpecIds: []string{"es-1"},
	}

	err := Release(context.Background(), client, changeSet, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if publishCalls.Load() != 0 {
		t.Errorf("expected no publish calls, got %d", publishCalls.Load())
	}
}

func strPtr(s string) *string {
	return &s
}

func versionRuleLocalAndRemote(localEs model.EventSpec, remoteEs console.RemoteEventSpec) (LocalFilesRefsResolved, console.DataProductsAndRelatedResources) {
	dp := model.DataProduct{
		ResourceName: "d9967fb4-6233-49c1-94d2-6cc417abd7ed",
		Data: model.DataProductData{
			Name:                "Data Product 1",
			EventSpecifications: []model.EventSpec{localEs},
		},
	}
	local := LocalFilesRefsResolved{
		DataProudcts: []model.DataProduct{dp},
		IdToFileName: map[string]string{localEs.ResourceName: "/base/dp.yaml"},
	}

	remoteDp := console.RemoteDataProduct{
		Id:                   "d9967fb4-6233-49c1-94d2-6cc417abd7ed",
		Name:                 "Data Product 1",
		SourceApplicationIds: []string{},
		EventSpecs:           []console.EventSpecReference{{Id: remoteEs.Id}},
	}
	remote := console.DataProductsAndRelatedResources{
		DataProducts: []console.RemoteDataProduct{remoteDp},
		EventSpecs:   []console.RemoteEventSpec{remoteEs},
	}
	return local, remote
}

func versionRuleRemoteEs() console.RemoteEventSpec {
	return console.RemoteEventSpec{
		Id:                   "6557ceee-a1c7-4ea9-910a-2a97194fb3f6",
		Name:                 "Checkout Started",
		Description:          "description",
		SourceApplicationIds: []string{},
		Version:              3,
		DataProductId:        "d9967fb4-6233-49c1-94d2-6cc417abd7ed",
		Event: &console.EventWrapper{Event: console.Event{
			Source: "iglu:com.example/checkout/jsonschema/1-0-0",
			Schema: map[string]any{"type": "object", "properties": map[string]any{"a": map[string]any{"maxLength": int64(10)}}},
		}},
		Entities: console.Entities{
			Tracked: []console.Entity{{
				Source:         "iglu:com.example/cart/jsonschema/1-0-0",
				MinCardinality: intPtr(0),
				MaxCardinality: intPtr(1),
			}},
			Enriched: []console.Entity{},
		},
	}
}

func versionRuleLocalEs() model.EventSpec {
	return model.EventSpec{
		ResourceName: "6557ceee-a1c7-4ea9-910a-2a97194fb3f6",
		Name:         "Checkout Started",
		Description:  "description",
		Event: model.SchemaRef{
			Source: "iglu:com.example/checkout/jsonschema/1-0-0",
			Schema: map[string]any{"properties": map[string]any{"a": map[string]any{"maxLength": 10}}, "type": "object"},
		},
		Entities: model.EntitiesDef{
			Tracked: []model.SchemaRef{{
				Source:         "iglu:com.example/cart/jsonschema/1-0-0",
				MinCardinality: intPtr(0),
				MaxCardinality: intPtr(1),
			}},
		},
	}
}

func Test_findChanges_VersionRule(t *testing.T) {
	tests := []struct {
		name         string
		changeLocal  func(es *model.EventSpec)
		changeRemote func(es *console.RemoteEventSpec)
		wantUpdate   bool
		wantExpected int
		wantChanged  []string

		wantCommentsRemoved bool
	}{
		{
			name:         "unchanged keeps the version",
			wantUpdate:   false,
			wantExpected: 3,
			wantChanged:  []string{},
		},
		{
			name:         "description change keeps the version",
			changeLocal:  func(es *model.EventSpec) { es.Description = "other" },
			wantUpdate:   true,
			wantExpected: 3,
			wantChanged:  []string{},
		},
		{
			name: "trigger change keeps the version",
			changeLocal: func(es *model.EventSpec) {
				es.Triggers = []model.Trigger{{Id: "a29e6ba7-769a-4919-b9a3-e6bb7bd97d67", Description: "new"}}
			},
			wantUpdate:   true,
			wantExpected: 3,
			wantChanged:  []string{},
		},
		{
			name:         "name change adds one",
			changeLocal:  func(es *model.EventSpec) { es.Name = "Checkout Begun" },
			wantUpdate:   true,
			wantExpected: 4,
			wantChanged:  []string{"name"},
		},
		{
			name:         "event source change adds one",
			changeLocal:  func(es *model.EventSpec) { es.Event.Source = "iglu:com.example/checkout/jsonschema/1-0-1" },
			wantUpdate:   true,
			wantExpected: 4,
			wantChanged:  []string{"event"},
		},
		{
			name: "event schema change adds one",
			changeLocal: func(es *model.EventSpec) {
				es.Event.Schema = map[string]any{"type": "object", "properties": map[string]any{"a": map[string]any{"maxLength": 11}}}
			},
			wantUpdate:   true,
			wantExpected: 4,
			wantChanged:  []string{"event"},
		},
		{
			name:         "entity cardinality change adds one",
			changeLocal:  func(es *model.EventSpec) { es.Entities.Tracked[0].MaxCardinality = intPtr(2) },
			wantUpdate:   true,
			wantExpected: 4,
			wantChanged:  []string{"entities"},
		},
		{
			name: "name, event and entities change adds one",
			changeLocal: func(es *model.EventSpec) {
				es.Name = "Checkout Begun"
				es.Event.Source = "iglu:com.example/checkout/jsonschema/1-0-1"
				es.Entities.Enriched = []model.SchemaRef{{Source: "iglu:com.example/geo/jsonschema/1-0-0"}}
			},
			wantUpdate:   true,
			wantExpected: 4,
			wantChanged:  []string{"name", "event", "entities"},
		},
		{
			name:         "remote entity comment alone sends nothing and keeps the version",
			changeRemote: func(es *console.RemoteEventSpec) { es.Entities.Tracked[0].Comment = strPtr("a note") },
			wantUpdate:   false,
			wantExpected: 3,
			wantChanged:  []string{},
		},
		{
			name:         "remote entity comment is dropped by an update and adds one",
			changeLocal:  func(es *model.EventSpec) { es.Description = "other" },
			changeRemote: func(es *console.RemoteEventSpec) { es.Entities.Tracked[0].Comment = strPtr("a note") },
			wantUpdate:   true,
			wantExpected: 4,
			wantChanged:  []string{"entities"},

			wantCommentsRemoved: true,
		},
		{
			name:         "remote entity comment alongside a real entity change is not the only cause",
			changeLocal:  func(es *model.EventSpec) { es.Entities.Tracked[0].MaxCardinality = intPtr(2) },
			changeRemote: func(es *console.RemoteEventSpec) { es.Entities.Tracked[0].Comment = strPtr("a note") },
			wantUpdate:   true,
			wantExpected: 4,
			wantChanged:  []string{"entities"},
		},
		{
			name:         "missing min cardinality is sent as null and treated as 0",
			changeLocal:  func(es *model.EventSpec) { es.Entities.Tracked[0].MinCardinality = nil },
			wantUpdate:   true,
			wantExpected: 3,
			wantChanged:  []string{},
		},
		{
			name:         "absent remote entities list differs from the empty list the cli sends",
			changeLocal:  func(es *model.EventSpec) { es.Description = "other" },
			changeRemote: func(es *console.RemoteEventSpec) { es.Entities.Enriched = nil },
			wantUpdate:   true,
			wantExpected: 4,
			wantChanged:  []string{"entities"},
		},
		{
			name:         "remote empty event schema differs from the absent schema the cli sends",
			changeLocal:  func(es *model.EventSpec) { es.Event.Schema = nil; es.Description = "other" },
			changeRemote: func(es *console.RemoteEventSpec) { es.Event.Schema = map[string]any{} },
			wantUpdate:   true,
			wantExpected: 4,
			wantChanged:  []string{"event"},
		},
		{
			name:         "removing the event adds one",
			changeLocal:  func(es *model.EventSpec) { es.Event = model.SchemaRef{} },
			wantUpdate:   true,
			wantExpected: 4,
			wantChanged:  []string{"event"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localEs := versionRuleLocalEs()
			if tt.changeLocal != nil {
				tt.changeLocal(&localEs)
			}
			localEs.Version = intPtr(99)
			remoteEs := versionRuleRemoteEs()
			if tt.changeRemote != nil {
				tt.changeRemote(&remoteEs)
			}
			local, remote := versionRuleLocalAndRemote(localEs, remoteEs)

			changes, err := findChanges(local, remote, map[string]string{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotUpdate := len(changes.esUpdate) == 1; gotUpdate != tt.wantUpdate {
				t.Errorf("update sent = %v, want %v", gotUpdate, tt.wantUpdate)
			}
			if len(changes.VersionMismatches) != 1 {
				t.Fatalf("expected 1 version mismatch, got %+v", changes.VersionMismatches)
			}
			want := model.EventSpecVersionMismatch{
				FileName: "/base/dp.yaml",
				Index:    0,
				Name:     localEs.Name,
				Declared: 99,
				Expected: tt.wantExpected,
				IsNew:    false,
				Changed:  tt.wantChanged,

				EntityCommentsRemoved: tt.wantCommentsRemoved,
			}
			if diff := cmp.Diff(want, changes.VersionMismatches[0]); diff != "" {
				t.Errorf("version mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_findChanges_VersionMatchesNoMismatch(t *testing.T) {
	localEs := versionRuleLocalEs()
	localEs.Name = "Checkout Begun"
	localEs.Version = intPtr(4)
	local, remote := versionRuleLocalAndRemote(localEs, versionRuleRemoteEs())

	changes, err := findChanges(local, remote, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes.VersionMismatches) != 0 {
		t.Errorf("expected no version mismatch, got %+v", changes.VersionMismatches)
	}
	if v := changes.declaredVersion(localEs.ResourceName); v == nil || *v != 4 {
		t.Errorf("expected declared version 4, got %v", v)
	}
}

func Test_findChanges_NewEventSpecExpectsZero(t *testing.T) {
	localEs := versionRuleLocalEs()
	localEs.Version = intPtr(0)
	local, remote := versionRuleLocalAndRemote(localEs, versionRuleRemoteEs())
	remote.EventSpecs = []console.RemoteEventSpec{}
	remote.DataProducts[0].EventSpecs = nil

	changes, err := findChanges(local, remote, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes.esCreate) != 1 {
		t.Fatalf("expected 1 event spec create, got %d", len(changes.esCreate))
	}
	if len(changes.VersionMismatches) != 0 {
		t.Errorf("expected no version mismatch, got %+v", changes.VersionMismatches)
	}

	localEs.Version = intPtr(1)
	local, _ = versionRuleLocalAndRemote(localEs, versionRuleRemoteEs())
	changes, err = findChanges(local, remote, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []model.EventSpecVersionMismatch{{
		FileName: "/base/dp.yaml",
		Index:    0,
		Name:     localEs.Name,
		Declared: 1,
		Expected: 0,
		IsNew:    true,
		Changed:  []string{},
	}}
	if diff := cmp.Diff(want, changes.VersionMismatches); diff != "" {
		t.Errorf("version mismatch (-want +got):\n%s", diff)
	}
}

func Test_findChanges_NoVersionDeclared(t *testing.T) {
	localEs := versionRuleLocalEs()
	localEs.Name = "Checkout Begun"
	local, remote := versionRuleLocalAndRemote(localEs, versionRuleRemoteEs())

	changes, err := findChanges(local, remote, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes.esUpdate) != 1 {
		t.Errorf("expected 1 event spec update, got %d", len(changes.esUpdate))
	}
	if len(changes.VersionMismatches) != 0 {
		t.Errorf("expected no version mismatch, got %+v", changes.VersionMismatches)
	}
	if v := changes.declaredVersion(localEs.ResourceName); v != nil {
		t.Errorf("expected no declared version, got %d", *v)
	}
}

func newEventSpecWriteServer(t *testing.T, bodies map[string]map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("unexpected body: %v", err)
		}
		bodies[r.Method+" "+r.URL.Path] = body
		switch {
		case r.URL.Path == "/event-specs/v1" && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{}`)
		case strings.HasPrefix(r.URL.Path, "/event-specs/v1/") && r.Method == "PUT":
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{}`)
		default:
			t.Errorf("Unexpected request, got: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func Test_ApplyDpChanges_UpdateExpectedVersionOnlyWhenDeclared(t *testing.T) {
	bodies := map[string]map[string]any{}
	server := newEventSpecWriteServer(t, bodies)
	defer server.Close()
	client := &console.ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: server.URL}

	changes := DataProductChangeSet{
		esUpdate:         []console.RemoteEventSpec{{Id: "es-update-declared", Name: "c"}, {Id: "es-update", Name: "d"}},
		declaredVersions: map[string]int{"es-update-declared": 4},
	}

	err := ApplyDpChanges(changes, context.Background(), client)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v, ok := bodies["PUT /event-specs/v1/es-update-declared"]["expectedVersion"]; !ok || v != float64(4) {
		t.Errorf("update with a declared version should send expectedVersion 4, got %v", bodies["PUT /event-specs/v1/es-update-declared"])
	}
	if _, ok := bodies["PUT /event-specs/v1/es-update"]["expectedVersion"]; ok {
		t.Errorf("update without a declared version should not send expectedVersion, got %v", bodies["PUT /event-specs/v1/es-update"])
	}
}

func Test_ApplyDpChanges_CreateExpectedVersionOnlyWhenDeclared(t *testing.T) {
	for _, tt := range []struct {
		id       string
		declared map[string]int
		want     any
		wantSent bool
	}{
		{id: "es-create-declared", declared: map[string]int{"es-create-declared": 0}, want: float64(0), wantSent: true},
		{id: "es-create", declared: map[string]int{}, wantSent: false},
	} {
		t.Run(tt.id, func(t *testing.T) {
			bodies := map[string]map[string]any{}
			server := newEventSpecWriteServer(t, bodies)
			defer server.Close()
			client := &console.ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: server.URL}

			changes := DataProductChangeSet{
				esCreate:         []console.RemoteEventSpec{{Id: tt.id, Name: "a"}},
				declaredVersions: tt.declared,
			}
			err := ApplyDpChanges(changes, context.Background(), client)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			v, ok := bodies["POST /event-specs/v1"]["expectedVersion"]
			if ok != tt.wantSent || (tt.wantSent && v != tt.want) {
				t.Errorf("expectedVersion sent = %v (%v), want %v (%v)", ok, v, tt.wantSent, tt.want)
			}
		})
	}
}

func Test_Release_ExpectedVersionsOnlyForDeclared(t *testing.T) {
	eventSpecsJSON := `[
		{"id": "es-1", "status": "draft", "name": "ES 1", "version": 2, "event": {"source": "iglu:com.example/event/jsonschema/1-0-0"}, "entities": {"tracked": [], "enriched": []}, "sourceApplications": []},
		{"id": "es-2", "status": "draft", "name": "ES 2", "version": 5, "event": {"source": "iglu:com.example/event/jsonschema/1-0-0"}, "entities": {"tracked": [], "enriched": []}, "sourceApplications": []}
	]`

	for _, tt := range []struct {
		name     string
		declared map[string]int
		wantBody string
	}{
		{
			name:     "no declared versions",
			declared: nil,
			wantBody: `{"eventSpecIds":["es-1","es-2"]}`,
		},
		{
			name:     "one declared version",
			declared: map[string]int{"es-2": 5, "es-not-released": 1},
			wantBody: `{"eventSpecIds":["es-1","es-2"],"expectedVersions":[{"id":"es-2","version":5}]}`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var publishBody string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/data-products/v2" && r.Method == "GET":
					w.WriteHeader(http.StatusOK)
					_, _ = io.WriteString(w, fmt.Sprintf(`{"data": [], "includes": {"eventSpecs": %s}}`, eventSpecsJSON))
				case r.URL.Path == "/source-apps/v1" && r.Method == "GET":
					w.WriteHeader(http.StatusOK)
					_, _ = io.WriteString(w, `{"data": []}`)
				case r.URL.Path == "/event-specs/v1/publish" && r.Method == "POST":
					b, _ := io.ReadAll(r.Body)
					publishBody = string(b)
					w.WriteHeader(http.StatusAccepted)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			client := &console.ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: server.URL}

			changeSet := &DataProductChangeSet{
				IdToFileName:      map[string]string{},
				localEventSpecIds: []string{"es-1", "es-2"},
				declaredVersions:  tt.declared,
			}

			err := Release(context.Background(), client, changeSet, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if publishBody != tt.wantBody {
				t.Errorf("publish body = %s, want %s", publishBody, tt.wantBody)
			}
		})
	}
}

func baseRuleFile(spec map[string]any) map[string]map[string]any {
	return map[string]map[string]any{
		"/base/dp.yaml": {
			"apiVersion":   "v1",
			"resourceType": "data-product",
			"resourceName": "d9967fb4-6233-49c1-94d2-6cc417abd7ed",
			"data": map[string]any{
				"name":                "Data Product 1",
				"eventSpecifications": []any{spec},
			},
		},
	}
}

func baseRuleSpec(version any) map[string]any {
	spec := map[string]any{
		"resourceName": "6557ceee-a1c7-4ea9-910a-2a97194fb3f6",
		"name":         "Checkout Started",
		"description":  "description",
		"event": map[string]any{
			"source": "iglu:com.example/checkout/jsonschema/1-0-0",
			"schema": map[string]any{"type": "object", "properties": map[string]any{"a": map[string]any{"maxLength": 10}}},
		},
		"entities": map[string]any{
			"tracked": []any{map[string]any{"source": "iglu:com.example/cart/jsonschema/1-0-0", "minCardinality": 0}},
		},
	}
	if version != nil {
		spec["version"] = version
	}
	return spec
}

func Test_FindVersionMismatchesAgainstBase(t *testing.T) {
	tests := []struct {
		name         string
		changeLocal  func(spec map[string]any)
		wantExpected int
		wantChanged  []string
	}{
		{
			name:         "unchanged keeps the base version",
			wantExpected: 3,
			wantChanged:  []string{},
		},
		{
			name:         "description change keeps the base version",
			changeLocal:  func(spec map[string]any) { spec["description"] = "other" },
			wantExpected: 3,
			wantChanged:  []string{},
		},
		{
			name:         "name change adds one",
			changeLocal:  func(spec map[string]any) { spec["name"] = "Checkout Begun" },
			wantExpected: 4,
			wantChanged:  []string{"name"},
		},
		{
			name: "event change adds one",
			changeLocal: func(spec map[string]any) {
				spec["event"] = map[string]any{"source": "iglu:com.example/checkout/jsonschema/1-0-1"}
			},
			wantExpected: 4,
			wantChanged:  []string{"event"},
		},
		{
			name: "entities change adds one",
			changeLocal: func(spec map[string]any) {
				spec["entities"] = map[string]any{"tracked": []any{map[string]any{"source": "iglu:com.example/cart/jsonschema/1-0-0", "minCardinality": 1}}}
			},
			wantExpected: 4,
			wantChanged:  []string{"entities"},
		},
		{
			name: "missing min cardinality equals 0",
			changeLocal: func(spec map[string]any) {
				spec["entities"] = map[string]any{"tracked": []any{map[string]any{"source": "iglu:com.example/cart/jsonschema/1-0-0"}}}
			},
			wantExpected: 3,
			wantChanged:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localSpec := baseRuleSpec(99)
			if tt.changeLocal != nil {
				tt.changeLocal(localSpec)
			}

			mismatches, skipped, err := FindVersionMismatchesAgainstBase(baseRuleFile(localSpec), baseRuleFile(baseRuleSpec(3)))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(skipped) != 0 {
				t.Errorf("expected nothing skipped, got %v", skipped)
			}
			want := []model.EventSpecVersionMismatch{{
				FileName: "/base/dp.yaml",
				Index:    0,
				Name:     localSpec["name"].(string),
				Declared: 99,
				Expected: tt.wantExpected,
				Changed:  tt.wantChanged,
			}}
			if diff := cmp.Diff(want, mismatches); diff != "" {
				t.Errorf("version mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_FindVersionMismatchesAgainstBase_MatchingVersion(t *testing.T) {
	localSpec := baseRuleSpec(4)
	localSpec["name"] = "Checkout Begun"

	mismatches, _, err := FindVersionMismatchesAgainstBase(baseRuleFile(localSpec), baseRuleFile(baseRuleSpec(float64(3))))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mismatches) != 0 {
		t.Errorf("expected no mismatch, got %+v", mismatches)
	}
}

func Test_FindVersionMismatchesAgainstBase_NewSpec(t *testing.T) {
	for _, tt := range []struct {
		name string
		base map[string]map[string]any
	}{
		{name: "spec absent at base", base: baseRuleFile(map[string]any{"resourceName": "aaa70141-c223-4172-ab39-86a0194e44e8", "name": "other", "version": 5})},
		{name: "file absent at base", base: map[string]map[string]any{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mismatches, _, err := FindVersionMismatchesAgainstBase(baseRuleFile(baseRuleSpec(0)), tt.base)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(mismatches) != 0 {
				t.Errorf("expected version 0 to match a new spec, got %+v", mismatches)
			}

			mismatches, _, err = FindVersionMismatchesAgainstBase(baseRuleFile(baseRuleSpec(2)), tt.base)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := []model.EventSpecVersionMismatch{{FileName: "/base/dp.yaml", Index: 0, Name: "Checkout Started", Declared: 2, Expected: 0, IsNew: true, Changed: []string{}}}
			if diff := cmp.Diff(want, mismatches); diff != "" {
				t.Errorf("version mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_FindVersionMismatchesAgainstBase_SkipsUndeclared(t *testing.T) {
	localSpec := baseRuleSpec(7)
	localSpec["name"] = "Checkout Begun"

	mismatches, skipped, err := FindVersionMismatchesAgainstBase(baseRuleFile(localSpec), baseRuleFile(baseRuleSpec(nil)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mismatches) != 0 {
		t.Errorf("expected no mismatch when the base declares no version, got %+v", mismatches)
	}
	if diff := cmp.Diff([]string{"Checkout Begun"}, skipped); diff != "" {
		t.Errorf("skipped (-want +got):\n%s", diff)
	}

	mismatches, skipped, err = FindVersionMismatchesAgainstBase(baseRuleFile(baseRuleSpec(nil)), baseRuleFile(baseRuleSpec(3)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mismatches) != 0 || len(skipped) != 0 {
		t.Errorf("a local spec without a version is not checked, got %+v %v", mismatches, skipped)
	}
}

func duplicateSpecLocal(firstVersion *int, secondVersion *int) LocalFilesRefsResolved {
	first := versionRuleLocalEs()
	first.Version = firstVersion
	second := versionRuleLocalEs()
	second.Version = secondVersion
	return LocalFilesRefsResolved{
		DataProudcts: []model.DataProduct{
			{ResourceName: "d9967fb4-6233-49c1-94d2-6cc417abd7ed", Data: model.DataProductData{Name: "one", EventSpecifications: []model.EventSpec{first}}},
			{ResourceName: "e0067fb4-6233-49c1-94d2-6cc417abd7ee", Data: model.DataProductData{Name: "two", EventSpecifications: []model.EventSpec{second}}},
		},
		IdToFileName: map[string]string{
			"d9967fb4-6233-49c1-94d2-6cc417abd7ed": "/base/one.yaml",
			"e0067fb4-6233-49c1-94d2-6cc417abd7ee": "/base/two.yaml",
		},
	}
}

func Test_findChanges_DuplicateEventSpecId(t *testing.T) {
	remote := console.DataProductsAndRelatedResources{}

	for _, tt := range []struct {
		name    string
		local   LocalFilesRefsResolved
		wantErr bool
	}{
		{name: "no version declared keeps the old behaviour", local: duplicateSpecLocal(nil, nil), wantErr: false},
		{name: "first declares a version", local: duplicateSpecLocal(intPtr(1), nil), wantErr: true},
		{name: "second declares a version", local: duplicateSpecLocal(nil, intPtr(1)), wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := findChanges(tt.local, remote, map[string]string{})
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, want error %v", err, tt.wantErr)
			}
			if tt.wantErr {
				want := "event specification 6557ceee-a1c7-4ea9-910a-2a97194fb3f6 is defined in both /base/one.yaml and /base/two.yaml"
				if err.Error() != want {
					t.Errorf("error = %q, want %q", err.Error(), want)
				}
			}
		})
	}
}

func Test_FindVersionMismatchesAgainstBase_DuplicateEventSpecId(t *testing.T) {
	local := baseRuleFile(baseRuleSpec(3))
	local["/base/other.yaml"] = baseRuleFile(baseRuleSpec(nil))["/base/dp.yaml"]

	_, _, err := FindVersionMismatchesAgainstBase(local, baseRuleFile(baseRuleSpec(3)))
	want := "event specification 6557ceee-a1c7-4ea9-910a-2a97194fb3f6 is defined in both /base/dp.yaml and /base/other.yaml"
	if err == nil || err.Error() != want {
		t.Errorf("error = %v, want %q", err, want)
	}

	base := baseRuleFile(baseRuleSpec(3))
	base["/base/other.yaml"] = baseRuleFile(baseRuleSpec(3))["/base/dp.yaml"]
	_, _, err = FindVersionMismatchesAgainstBase(baseRuleFile(baseRuleSpec(3)), base)
	if err != nil {
		t.Errorf("a duplicate at base should not fail the check, got: %v", err)
	}
}

func Test_findChanges_EntityCommentOnlyDifferenceWithoutVersion(t *testing.T) {
	localEs := versionRuleLocalEs()
	remoteEs := versionRuleRemoteEs()
	remoteEs.Entities.Tracked[0].Comment = strPtr("a note")
	local, remote := versionRuleLocalAndRemote(localEs, remoteEs)

	changes, err := findChanges(local, remote, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes.esUpdate) != 0 {
		t.Errorf("a comment-only difference sent no update before versions existed, got %d updates", len(changes.esUpdate))
	}
}

func Test_findChanges_SourceAppEntityCommentOnlyDifference(t *testing.T) {
	sa := model.SourceApp{
		ResourceName: "9b6e4e4c-c34a-483c-a5e7-f728d66a53b3",
		Data: model.SourceAppData{
			Name:     "Source App 1",
			Entities: &model.EntitiesDef{Tracked: []model.SchemaRef{{Source: "iglu:com.example/a/jsonschema/1-0-0", MinCardinality: intPtr(0)}}},
		},
	}
	remoteSa := console.RemoteSourceApplication{
		Id:   "9b6e4e4c-c34a-483c-a5e7-f728d66a53b3",
		Name: "Source App 1",
		Entities: console.Entities{
			Tracked:  []console.Entity{{Source: "iglu:com.example/a/jsonschema/1-0-0", MinCardinality: intPtr(0), Comment: strPtr("a note")}},
			Enriched: []console.Entity{},
		},
	}
	local := LocalFilesRefsResolved{SourceApps: []model.SourceApp{sa}}
	remote := console.DataProductsAndRelatedResources{SourceApplication: []console.RemoteSourceApplication{remoteSa}}

	changes, err := findChanges(local, remote, map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(changes.saUpdate) != 0 {
		t.Errorf("a comment-only difference sent no update before versions existed, got %d updates", len(changes.saUpdate))
	}
}
