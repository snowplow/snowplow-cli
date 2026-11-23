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
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/snowplow/snowplow-cli/internal/model"
)

type closeCountingBody struct {
	io.Reader

	closes int
}

func (b *closeCountingBody) Close() error {
	b.closes++
	return nil
}

type closeCountingTransport struct {
	status   int
	body     string
	failRead bool
	err      error
	requests int
	bodies   []*closeCountingBody
}

func (tr *closeCountingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tr.requests++
	if tr.err != nil {
		return nil, tr.err
	}
	content := tr.body
	if content == "" {
		content = `{}`
	}
	body := &closeCountingBody{Reader: strings.NewReader(content)}
	if tr.failRead {
		body.Reader = iotest.ErrReader(errors.New("read failed"))
	}
	tr.bodies = append(tr.bodies, body)
	return &http.Response{
		StatusCode: tr.status,
		Header:     http.Header{"Content-Type": []string{"image/png"}},
		Body:       body,
		Request:    req,
	}, nil
}

func closeCountingClient(tr *closeCountingTransport) *ApiClient {
	return &ApiClient{Http: &http.Client{Transport: tr}, Jwt: "token", BaseUrl: "http://console.test", OrgId: "orgid"}
}

type requestCall struct {
	name string
	body string
	call func(ctx context.Context, c *ApiClient) error
}

func dataProductRequestCalls() []requestCall {
	return []requestCall{
		{"GetDataProductsAndRelatedResources", "", func(ctx context.Context, c *ApiClient) error {
			_, err := GetDataProductsAndRelatedResources(ctx, c)
			return err
		}},
		{"CompatCheck", "", func(ctx context.Context, c *ApiClient) error {
			_, err := CompatCheck(ctx, c, CompatCheckable{}, nil)
			return err
		}},
		{"CreateSourceApp", "", func(ctx context.Context, c *ApiClient) error {
			return CreateSourceApp(ctx, c, RemoteSourceApplication{})
		}},
		{"UpdateSourceApp", "", func(ctx context.Context, c *ApiClient) error {
			return UpdateSourceApp(ctx, c, RemoteSourceApplication{})
		}},
		{"DeleteSourceApp", "", func(ctx context.Context, c *ApiClient) error {
			return DeleteSourceApp(ctx, c, RemoteSourceApplication{})
		}},
		{"CreateDataProduct", "", func(ctx context.Context, c *ApiClient) error {
			return CreateDataProduct(ctx, c, RemoteDataProduct{})
		}},
		{"UpdateDataProduct", "", func(ctx context.Context, c *ApiClient) error {
			return UpdateDataProduct(ctx, c, RemoteDataProduct{})
		}},
		{"DeleteDataProduct", "", func(ctx context.Context, c *ApiClient) error {
			return DeleteDataProduct(ctx, c, RemoteDataProduct{})
		}},
		{"CreateEventSpec", "", func(ctx context.Context, c *ApiClient) error {
			return CreateEventSpec(ctx, c, RemoteEventSpec{}, nil)
		}},
		{"UpdateEventSpec", "", func(ctx context.Context, c *ApiClient) error {
			return UpdateEventSpec(ctx, c, RemoteEventSpec{}, nil)
		}},
		{"DeleteEventSpec", "", func(ctx context.Context, c *ApiClient) error {
			return DeleteEventSpec(ctx, c, "id")
		}},
		{"BatchPublishEventSpecs", "", func(ctx context.Context, c *ApiClient) error {
			return BatchPublishEventSpecs(ctx, c, []string{"id"}, nil)
		}},
	}
}

func dataStructureRequestCalls() []requestCall {
	return []requestCall{
		{"Validate", "", func(ctx context.Context, c *ApiClient) error {
			_, err := Validate(ctx, c, model.DataStructure{})
			return err
		}},
		{"publish", `{"success":true}`, func(ctx context.Context, c *ApiClient) error {
			_, err := publish(ctx, c, VALIDATED, DEV, model.DataStructure{}, false)
			return err
		}},
		{"GetIgluCentralListing", `[]`, func(ctx context.Context, c *ApiClient) error {
			_, err := GetIgluCentralListing(ctx, c)
			return err
		}},
		{"GetDataStructureListing", `[]`, func(ctx context.Context, c *ApiClient) error {
			_, err := GetDataStructureListing(ctx, c)
			return err
		}},
		{"GetDataStructureDeployments", `[]`, func(ctx context.Context, c *ApiClient) error {
			_, err := GetDataStructureDeployments(ctx, c, "hash")
			return err
		}},
		{"getLatestSchemaVersions", `[]`, func(ctx context.Context, c *ApiClient) error {
			_, err := getLatestSchemaVersions(ctx, c)
			return err
		}},
		{"GetAllDataStructuresDrafts", `[]`, func(ctx context.Context, c *ApiClient) error {
			_, err := GetAllDataStructuresDrafts(ctx, c, nil)
			return err
		}},
		{"patchMeta", "", func(ctx context.Context, c *ApiClient) error {
			return patchMeta(ctx, c, &model.DataStructureSelf{}, fullMeta{})
		}},
		{"fetchMigration", "", func(ctx context.Context, c *ApiClient) error {
			_, err := fetchMigration(ctx, c, "snowflake", model.DataStructureSelf{}, nil)
			return err
		}},
		{"fetchDestinations", `[]`, func(ctx context.Context, c *ApiClient) error {
			_, err := fetchDestinations(ctx, c)
			return err
		}},
	}
}

func imageAndAPIKeyRequestCalls(t *testing.T) []requestCall {
	t.Helper()
	image := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(image, []byte("image"), 0600); err != nil {
		t.Fatal(err)
	}
	return []requestCall{
		{"GetImage", "", func(ctx context.Context, c *ApiClient) error {
			_, _, err := GetImage(ctx, c, c.BaseUrl+"/image")
			return err
		}},
		{"GetImageHashLookup", "", func(ctx context.Context, c *ApiClient) error {
			_, err := GetImageHashLookup(ctx, c)
			return err
		}},
		{"createImageUploadLink", "", func(ctx context.Context, c *ApiClient) error {
			_, err := createImageUploadLink(ctx, c)
			return err
		}},
		{"uploadImage", `{"success":true}`, func(ctx context.Context, c *ApiClient) error {
			return uploadImage(ctx, c, image, &imageUploadLinkResponse{UploadURL: "http://upload.test", Id: "id"})
		}},
		{"confirmImage", "", func(ctx context.Context, c *ApiClient) error {
			_, err := confirmImage(ctx, c, "id", "hash")
			return err
		}},
		{"postAPIKey", "", func(ctx context.Context, c *ApiClient) error {
			_, err := postAPIKey(ctx, c, "description", nil)
			return err
		}},
	}
}

func requestCalls(t *testing.T) []requestCall {
	t.Helper()
	return slices.Concat(dataProductRequestCalls(), dataStructureRequestCalls(), imageAndAPIKeyRequestCalls(t))
}

func Test_RequestFunctionsCloseEveryResponseBodyOnce(t *testing.T) {
	responses := []struct {
		name     string
		status   int
		failRead bool
	}{
		{"200", http.StatusOK, false},
		{"201", http.StatusCreated, false},
		{"202", http.StatusAccepted, false},
		{"204", http.StatusNoContent, false},
		{"500", http.StatusInternalServerError, false},
		{"200 with a failing body", http.StatusOK, true},
		{"500 with a failing body", http.StatusInternalServerError, true},
	}

	for _, call := range requestCalls(t) {
		for _, response := range responses {
			t.Run(call.name+"/"+response.name, func(t *testing.T) {
				tr := &closeCountingTransport{status: response.status, body: call.body, failRead: response.failRead}

				err := call.call(t.Context(), closeCountingClient(tr))

				if len(tr.bodies) == 0 {
					t.Fatalf("no request was made, call returned: %v", err)
				}
				for i, body := range tr.bodies {
					if body.closes != 1 {
						t.Errorf("body of request %d closed %d times, want 1, call returned: %v", i+1, body.closes, err)
					}
				}
			})
		}
	}
}

func Test_RequestFunctionsCloseTheBodyOnceWhenTheySucceed(t *testing.T) {
	for _, call := range requestCalls(t) {
		t.Run(call.name, func(t *testing.T) {
			var errs []error
			for _, status := range []int{http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
				tr := &closeCountingTransport{status: status, body: call.body}

				err := call.call(t.Context(), closeCountingClient(tr))

				if err != nil {
					errs = append(errs, err)
					continue
				}
				for i, body := range tr.bodies {
					if body.closes != 1 {
						t.Errorf("body of request %d closed %d times after a success with status %d, want 1", i+1, body.closes, status)
					}
				}
				return
			}
			t.Errorf("call never succeeded, so its success return is not covered: %v", errs)
		})
	}
}

func Test_RequestFunctionsReturnAnErrorWhenTheRequestFails(t *testing.T) {
	for _, call := range requestCalls(t) {
		t.Run(call.name, func(t *testing.T) {
			tr := &closeCountingTransport{err: errors.New("request failed")}
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panicked on a failed request: %v", r)
				}
			}()

			err := call.call(t.Context(), closeCountingClient(tr))

			if err == nil {
				t.Errorf("got no error, want the failed request reported")
			}
			if tr.requests == 0 {
				t.Errorf("no request was made, call returned: %v", err)
			}
		})
	}
}

func Test_GetDataProductsAndRelatedResources_ClosesBothResponseBodies(t *testing.T) {
	tr := &closeCountingTransport{status: http.StatusOK}

	if _, err := GetDataProductsAndRelatedResources(t.Context(), closeCountingClient(tr)); err != nil {
		t.Fatal(err)
	}

	if len(tr.bodies) != 2 {
		t.Fatalf("got %d requests, want 2", len(tr.bodies))
	}
	for i, body := range tr.bodies {
		if body.closes != 1 {
			t.Errorf("body of request %d closed %d times, want 1", i+1, body.closes)
		}
	}
}
