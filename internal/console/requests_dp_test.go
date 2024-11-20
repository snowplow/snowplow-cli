/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package console

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func intPtr(i int) *int {
	return &i
}

var sampleEntity1 = Entity{
	Source:         "iglu:com.snplow.msc.aws/data-product-auto/jsonschema/1-0-0",
	MinCardinality: intPtr(0),
	MaxCardinality: intPtr(5),
	Schema:         map[string]any{},
}

var sampleEntity2 = Entity{
	Source:         "iglu:com.snplow.msc.aws/spo_zod_2/jsonschema/1-0-0",
	MinCardinality: intPtr(0),
	MaxCardinality: nil,
	Schema:         map[string]any{},
}

var sampleSa1 = RemoteSourceApplication{
	Id:          "6b1146d6-7b23-4dbb-b069-f568458dda8f",
	Name:        "cli test update 2",
	Description: "cli test update 2",
	Owner:       "me2@me.com",
	AppIds:      []string{"ios2", "android3"},
	Entities:    Entities{Tracked: []Entity{sampleEntity1, sampleEntity2}, Enriched: []Entity{}},
}

var sampleRemoteEs = RemoteEventSpec{
	Id:                   "84614b3b-6039-458e-8ce2-615eaf2113e3",
	SourceApplicationIds: []string{},
	Name:                 "test ES 3",
	DataProductId:        "46d47289-f3d5-4ef8-a82c-b19597e6e503",
	Event: &EventWrapper{Event{
		Source: "iglu:com.snplow.msc.aws/spo__ds_test_bug/jsonschema/4-0-0",
		Schema: nil,
	},
	},
	Entities: Entities{Tracked: []Entity{
		{
			Source:         sampleEntity1.Source,
			MinCardinality: sampleEntity1.MinCardinality,
			MaxCardinality: sampleEntity1.MaxCardinality,
			Schema:         nil,
		},
		{
			Source:         sampleEntity2.Source,
			MinCardinality: sampleEntity2.MinCardinality,
			MaxCardinality: sampleEntity2.MaxCardinality,
			Schema:         nil,
		},
	},
		Enriched: []Entity{},
	},
}

var sampleRemoteDp = RemoteDataProduct{
	Id:                   "46d47289-f3d5-4ef8-a82c-b19597e6e503",
	Name:                 "test form cli 2",
	SourceApplicationIds: []string{},
	Domain:               "testing cli",
	Owner:                "me2@me.me",
	Description:          "this is a test 2",
}

func Test_GetDataProductsEventSpecs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/data-products/v2" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp := `{
    "data": [
        {
            "id": "2555c639-2844-4130-b7dc-566c0ef93e93",
            "name": "Base Web",
            "organizationId": "af3b1e5a-4396-4b0f-9be9-86171d94c478",
            "domain": "Tracking",
            "owner": "test@example.com",
            "description": "This Data Product contains all the tracking of Standard Events. Note: the event volume counts are calculated differently for this Data Product. They are counts of any page_ping/page_view events sent for App Id's of this data product",
            "status": "draft",
            "eventSpecs": [
                {
                    "id": "3a867d9e-59c5-4707-a1fb-fa988c713aaa",
                    "url": "https://test.example.com/api/msc/v1/organizations/af3b1e5a-4396-4b0f-9be9-86171d94c478/tracking-scenarios/v2/3a867d9e-59c5-4707-a1fb-fa988c713aaa"
                },
                {
                    "id": "75c611aa-02f4-4f6d-9f90-8aa225d42a82",
                    "url": "https://test.example.com/api/msc/v1/organizations/af3b1e5a-4396-4b0f-9be9-86171d94c478/tracking-scenarios/v2/75c611aa-02f4-4f6d-9f90-8aa225d42a82"
                }
            ],
            "templateReference": "base-web-1",
            "sourceApplications": [
                "18183ea7-0d6e-4698-b2f9-7164dc7f1be5"
            ],
            "type": "base",
            "createdAt": "2024-10-02T12:21:37.005594Z",
            "updatedAt": "2024-10-10T05:54:42.674962Z"
        }
    ],
    "includes": {
        "owners": [],
        "eventSpecs": [
            {
                "id": "3a867d9e-59c5-4707-a1fb-fa988c713aaa",
                "version": 0,
                "status": "draft",
                "name": "Button click",
                "dataProductId": "2555c639-2844-4130-b7dc-566c0ef93e93",
                "description": "Produced by the Button click plugin. Documentation can be found [here](https://docs.snowplow.io/docs/collecting-data/collecting-from-own-applications/javascript-trackers/web-tracker/tracking-events/button-click/)",
                "triggers": [],
                "event": {
                    "source": "iglu:com.snowplowanalytics.snowplow/button_click/jsonschema/1-0-0"
                },
                "entities": {
                    "tracked": [],
                    "enriched": []
                },
                "sourceApplications": [
                  "18183ea7-0d6e-4698-b2f9-7164dc7f1be5"
                ],
                "author": "712080b8-1735-4438-9f17-a42428dceb56",
                "message": "",
                "date": "2024-10-02T12:21:37.358262Z"
            },
            {
                "id": "75c611aa-02f4-4f6d-9f90-8aa225d42a82",
                "version": 0,
                "status": "draft",
                "name": "Custom event",
                "dataProductId": "2555c639-2844-4130-b7dc-566c0ef93e93",
                "description": "A custom event to model basic tracking",
                "triggers": [],
                "event": {
                    "source": "iglu:com.snowplowanalytics.snowplow/custom_event/jsonschema/1-0-0"
                },
                "entities": {
                    "tracked": [],
                    "enriched": []
                },
                "sourceApplications": [
                  "18183ea7-0d6e-4698-b2f9-7164dc7f1be5"
                ],
                "author": "712080b8-1735-4438-9f17-a42428dceb56",
                "message": "",
                "date": "2024-10-02T12:21:37.333870Z"
            }
        ],
        "sourceApplications": [
            {
                "id": "18183ea7-0d6e-4698-b2f9-7164dc7f1be5",
                "name": "Test Source Application",
                "description": "Source application for www.snowplow.io website",
                "owner": "example@example.com",
                "appIds": [
                    "website",
                    "website-qa",
                    "website-dev"
                ],
                "entities": {
                    "tracked": [
                        {
                            "source": "iglu:com.snplow.msc.aws/data_product/jsonschema/3-0-0",
                            "minCardinality": 0,
                            "comment": "When Data Products are available"
                        }
                    ],
                    "enriched": []
                }
            }
        ]
    },
    "errors": [],
    "metadata": null
}`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		}
		if r.URL.Path == "/api/msc/v1/organizations/orgid/source-apps/v1" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp := `{
    "data": [
            {
                "id": "18183ea7-0d6e-4698-b2f9-7164dc7f1be5",
                "name": "Test Source Application",
                "description": "Source application for www.snowplow.io website",
                "owner": "example@example.com",
                "appIds": [
                    "website",
                    "website-qa",
                    "website-dev"
                ],
                "entities": {
                    "tracked": [
                        {
                            "source": "iglu:com.snplow.msc.aws/data_product/jsonschema/3-0-0",
                            "minCardinality": 0,
                            "comment": "When Data Products are available"
                        }
                    ],
                    "enriched": []
                }
            }
        ]
}`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		}

		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	res, err := GetDataProductsAndRelatedResources(cnx, client)

	if err != nil {
		t.Error(err)
	}

	if len(res.DataProducts) != 1 {
		t.Errorf("Unexpected number of data products, expected 1, got: %d", len(res.DataProducts))
	}

	if len(res.EventSpecs) != 2 {
		t.Errorf("Unexpected number of event specs, expected 2, got: %d", len(res.EventSpecs))
	}

	if len(res.SourceApplication) != 1 {
		t.Errorf("Unexpected number of source apps, expected 1, got: %d", len(res.SourceApplication))
	}

}

func Test_CreateSourceApp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/source-apps/v1" && r.Method == "POST" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp := `{}`

			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, resp)
			return
		}
		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	err := CreateSourceApp(cnx, client, sampleSa1)

	if err != nil {
		t.Error(err)
	}

}

func Test_UpdateSourceApp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("/api/msc/v1/organizations/orgid/source-apps/v1/%s", sampleSa1.Id) && r.Method == "PUT" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp := `{}`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		}
		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	err := UpdateSourceApp(cnx, client, sampleSa1)

	if err != nil {
		t.Error(err)
	}

}

func Test_CreateDataProduct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/data-products/v2" && r.Method == "POST" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp := `{}`

			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, resp)
			return
		}
		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	err := CreateDataProduct(cnx, client, sampleRemoteDp)

	if err != nil {
		t.Error(err)
	}

}

func Test_UpdateDataProduct(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("/api/msc/v1/organizations/orgid/data-products/v2/%s", sampleRemoteDp.Id) && r.Method == "PUT" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp := `{}`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		}
		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	err := UpdateDataProduct(cnx, client, sampleRemoteDp)

	if err != nil {
		t.Error(err)
	}

}

func Test_CreateEventSpec(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/msc/v1/organizations/orgid/event-specs/v1" && r.Method == "POST" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp := `{}`

			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, resp)
			return
		}
		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	err := CreateEventSpec(cnx, client, sampleRemoteEs)

	if err != nil {
		t.Error(err)
	}

}

func Test_UpdateEventSpec(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("/api/msc/v1/organizations/orgid/event-specs/v1/%s", sampleRemoteEs.Id) && r.Method == "PUT" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			resp := `{}`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		}
		if r.URL.Path == fmt.Sprintf("/api/msc/v1/organizations/orgid/event-specs/v1/%s", sampleRemoteEs.Id) && r.Method == "GET" {
			if r.Header.Get("authorization") != "Bearer token" {
				t.Errorf("bad auth token, got: %s", r.Header.Get("authorization"))
			}

			// we only need this for version
			resp := `{"data": [{"version": 2}]}`

			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, resp)
			return
		}
		t.Errorf("Unexpected request, got: %s", r.URL.Path)
	}))
	defer server.Close()

	cnx := context.Background()
	client := &ApiClient{Http: &http.Client{}, Jwt: "token", BaseUrl: fmt.Sprintf("%s/api/msc/v1/organizations/orgid", server.URL)}

	err := UpdateEventSpec(cnx, client, sampleRemoteEs)

	if err != nil {
		t.Error(err)
	}

}
