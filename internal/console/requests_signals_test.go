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

func Test_GetAPIKeyPermissions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/msc/v1/organizations/org/credentials/v2/api-keys/kid" || r.Header.Get("Authorization") != "Bearer token" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"id":"kid","permissions":[{"organizationId":"org","capabilities":[{"resourceType":"DATA-PRODUCTS","action":"VIEW","filters":[]}]}]}`))
	}))
	defer server.Close()

	perms, err := GetAPIKeyPermissions(context.Background(), "token", server.URL, "org", "kid")
	if err != nil {
		t.Fatal(err)
	}
	if len(perms) != 1 || perms[0].Capabilities[0].ResourceType != "DATA-PRODUCTS" {
		t.Fatalf("unexpected permissions %+v", perms)
	}
	if _, err := GetAPIKeyPermissions(context.Background(), "token", server.URL, "org", "other"); err == nil {
		t.Fatal("expected an error for a missing key")
	}
}

func Test_Allows(t *testing.T) {
	perm := func(org string, caps ...Capability) []APIKeyPermission {
		return []APIKeyPermission{{OrganizationID: org, Capabilities: caps}}
	}
	all := func(resource string) []Capability {
		var caps []Capability
		for _, action := range []string{"LIST", "VIEW", "CREATE", "EDIT"} {
			caps = append(caps, Capability{ResourceType: resource, Action: action})
		}
		return caps
	}
	cases := []struct {
		name  string
		perms []APIKeyPermission
		want  bool
	}{
		{"admin key", perm("org", Capability{ResourceType: "*", Action: "*"}), true},
		{"every action on both", perm("org", append(all("SIGNALS"), all("DATA-PRODUCTS")...)...), true},
		{"any action on both", perm("org", Capability{ResourceType: "SIGNALS", Action: "*"}, Capability{ResourceType: "DATA-PRODUCTS", Action: "*"}), true},
		{"signals only", perm("org", Capability{ResourceType: "SIGNALS", Action: "*"}), false},
		{"edit without create", perm("org", Capability{ResourceType: "SIGNALS", Action: "EDIT"}, Capability{ResourceType: "DATA-PRODUCTS", Action: "*"}), false},
		{"read-only key", perm("org", Capability{ResourceType: "DATA-PRODUCTS", Action: "VIEW"}), false},
		{"another org", perm("other", Capability{ResourceType: "*", Action: "*"}), false},
		{"no org on permission", perm("", Capability{ResourceType: "*", Action: "*"}), true},
	}
	for _, c := range cases {
		if got := Allows(c.perms, "org", SignalsSetupCapabilities); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
