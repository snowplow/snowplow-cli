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
	"testing"

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

	changes, err := findChanges(local, remote)
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

	changes, err := findChanges(local, remote)
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

	changes, err := findChanges(local, remote)
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
