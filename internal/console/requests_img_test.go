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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func Test_uploadImage_SendsFileAsMultipartForm(t *testing.T) {
	fname := filepath.Join(t.TempDir(), "trigger.png")
	if err := os.WriteFile(fname, []byte("image bytes"), 0o600); err != nil {
		t.Fatal(err)
	}

	var gotName, gotContent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Errorf("request is not a readable multipart form: %v", err)
			return
		}
		content, _ := io.ReadAll(file)
		gotName, gotContent = header.Filename, string(content)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := &ApiClient{Http: server.Client(), OrgId: "orgid"}
	err := uploadImage(context.Background(), client, fname, &imageUploadLinkResponse{Id: "linkid", UploadURL: server.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotName != "orgid_linkid_trigger.png" || gotContent != "image bytes" {
		t.Errorf("uploaded %q with content %q", gotName, gotContent)
	}
}

func Test_uploadImage_MissingFileSendsNothing(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := &ApiClient{Http: server.Client(), OrgId: "orgid"}
	fname := filepath.Join(t.TempDir(), "missing.png")
	err := uploadImage(context.Background(), client, fname, &imageUploadLinkResponse{Id: "linkid", UploadURL: server.URL})
	if err == nil {
		t.Fatal("expected an error for a missing image file")
	}
	if requests.Load() != 0 {
		t.Errorf("expected no upload request, got %d", requests.Load())
	}
}
