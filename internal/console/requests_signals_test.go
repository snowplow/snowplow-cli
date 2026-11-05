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
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_GetSignalsEnvironments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/msc/v1/organizations/org/signals/v1" || r.Method != "GET" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("bad auth header: %s", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`[
			{"id":"a","status":"ready","pipeline":{"id":"p1","name":"Prod"},"config":{"personalizationApiHost":"abc.signals.snowplowanalytics.com"}},
			{"id":"b","status":"creating","pipeline":{"id":"p2","name":"Dev"}}
		]`))
	}))
	defer server.Close()

	envs, err := GetSignalsEnvironments(context.Background(), "token", server.URL, "org")
	if err != nil {
		t.Fatal(err)
	}
	if len(envs) != 2 {
		t.Fatalf("expected 2 environments, got %d", len(envs))
	}
	if envs[0].APIURL() != "https://abc.signals.snowplowanalytics.com" || envs[0].Pipeline.Name != "Prod" {
		t.Fatalf("unexpected first environment %+v", envs[0])
	}
	if envs[1].APIURL() != "" {
		t.Fatalf("expected no API URL while creating, got %q", envs[1].APIURL())
	}
}
