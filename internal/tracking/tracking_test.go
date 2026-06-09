/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package tracking

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	gt "github.com/snowplow/snowplow-golang-tracker/v3/tracker"
)

func TestGetSdJSON(t *testing.T) {
	_, err := getSdJSON("", "", "")
	if err == nil || err.Error() != "fatal: --sdjson or --schema URI plus a --json needs to be specified" {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = getSdJSON("", "iglu:com.acme/event/jsonschema/1-0-0", "")
	if err == nil || err.Error() != "fatal: --json needs to be specified" {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = getSdJSON("", "", "{\"e\":\"pv\"}")
	if err == nil || err.Error() != "fatal: --schema URI needs to be specified" {
		t.Fatalf("unexpected error: %v", err)
	}

	sdj, err := getSdJSON("", "iglu:com.acme/event/jsonschema/1-0-0", "{\"e\":\"pv\"}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := sdj.String(); got != "{\"data\":{\"e\":\"pv\"},\"schema\":\"iglu:com.acme/event/jsonschema/1-0-0\"}" {
		t.Fatalf("unexpected sdj: %s", got)
	}

	_, err = getSdJSON("", "iglu:com.acme/event/jsonschema/1-0-0", "{\"e\"}")
	if err == nil || err.Error() != "invalid character '}' after object key" {
		t.Fatalf("unexpected error: %v", err)
	}

	sdj, err = getSdJSON("{\"data\":{\"e\":\"pv\"},\"schema\":\"iglu:com.acme/event/jsonschema/1-0-0\"}", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := sdj.String(); got != "{\"data\":{\"e\":\"pv\"},\"schema\":\"iglu:com.acme/event/jsonschema/1-0-0\"}" {
		t.Fatalf("unexpected sdj: %s", got)
	}

	_, err = getSdJSON("{\"data\":{\"e\"},\"schema\":\"iglu:com.acme/event/jsonschema/1-0-0\"}", "", "")
	if err == nil || err.Error() != "invalid character '}' after object key" {
		t.Fatalf("unexpected error: %v", err)
	}

	sdj, err = getSdJSON("{\"data\":{\"timestamp\":1534429336},\"schema\":\"iglu:com.acme/event/jsonschema/1-0-0\"}", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := sdj.String(); got != "{\"data\":{\"timestamp\":1534429336},\"schema\":\"iglu:com.acme/event/jsonschema/1-0-0\"}" {
		t.Fatalf("unexpected sdj: %s", got)
	}
}

func TestGetEntities(t *testing.T) {
	if got, err := getEntities(""); err != nil || got != nil {
		t.Fatalf("expected empty string to yield no entities, got %v / %v", got, err)
	}

	entities, err := getEntities("[]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entities) != 0 {
		t.Fatalf("expected 0 entities, got %d", len(entities))
	}

	entities, err = getEntities("[{\"data\":{\"timestamp\":1534429336},\"schema\":\"iglu:com.acme/context_1/jsonschema/1-0-0\"},{\"data\":{\"timestamp\":1534429336},\"schema\":\"iglu:com.acme/context_1/jsonschema/1-0-0\"}]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(entities))
	}
	want := "{\"data\":{\"timestamp\":1534429336},\"schema\":\"iglu:com.acme/context_1/jsonschema/1-0-0\"}"
	if entities[0].String() != want || entities[1].String() != want {
		t.Fatalf("unexpected entities: %s / %s", entities[0].String(), entities[1].String())
	}
}

func TestInitTracker(t *testing.T) {
	trackerChan := make(chan int, 1)
	tracker := initTracker("com.acme", "myapp", "POST", "https", "", trackerChan, nil)
	if tracker == nil || tracker.Emitter == nil || tracker.Subject == nil {
		t.Fatal("tracker not fully initialised")
	}
	if tracker.AppId != "myapp" {
		t.Fatalf("expected app id myapp, got %s", tracker.AppId)
	}
}

func TestTrackSelfDescribingEventGood(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(200)
	}))
	defer server.Close()

	collector := strings.TrimPrefix(server.URL, "http://")
	httpClient := &http.Client{Timeout: time.Duration(5 * time.Second)}

	trackerChan := make(chan int, 1)
	tracker := initTracker(collector, "myapp", "GET", "http", "", trackerChan, httpClient)

	jsonDataMap, _ := stringToMap("{\"hello\":\"world\"}")
	sdj := gt.InitSelfDescribingJson("iglu:com.acme/event/jsonschema/1-0-0", jsonDataMap)

	statusCode := trackSelfDescribingEvent(tracker, trackerChan, sdj, nil)
	if statusCode != 200 {
		t.Fatalf("expected 200, got %d", statusCode)
	}
	if requests != 1 {
		t.Fatalf("expected 1 request, got %d", requests)
	}
}

func TestTrackSelfDescribingEventBad(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(404)
	}))
	defer server.Close()

	collector := strings.TrimPrefix(server.URL, "http://")
	httpClient := &http.Client{Timeout: time.Duration(5 * time.Second)}

	trackerChan := make(chan int, 1)
	tracker := initTracker(collector, "myapp", "POST", "http", "", trackerChan, httpClient)

	jsonDataMap, _ := stringToMap("{\"hello\":\"world\"}")
	sdj := gt.InitSelfDescribingJson("iglu:com.acme/event/jsonschema/1-0-0", jsonDataMap)

	statusCode := trackSelfDescribingEvent(tracker, trackerChan, sdj, nil)
	if statusCode != 404 {
		t.Fatalf("expected 404, got %d", statusCode)
	}
	if requests != 1 {
		t.Fatalf("expected 1 request, got %d", requests)
	}
}

func TestParseStatusCode(t *testing.T) {
	cases := map[int]int{200: 0, 300: 0, 404: 4, 501: 5, 600: 1}
	for status, want := range cases {
		if got := parseStatusCode(status); got != want {
			t.Fatalf("parseStatusCode(%d) = %d, want %d", status, got, want)
		}
	}
}

func TestStringToMap(t *testing.T) {
	m, err := stringToMap("{\"hello\":\"world\"}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["hello"] != "world" || len(m) != 1 {
		t.Fatalf("unexpected map: %v", m)
	}

	if _, err := stringToMap("{\"hello\"}"); err == nil {
		t.Fatal("expected error for malformed json")
	}

	m, err = stringToMap("{\"timestamp\":1534429336}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["timestamp"] != json.Number("1534429336") {
		t.Fatalf("expected json.Number, got %T %v", m["timestamp"], m["timestamp"])
	}
}

func TestTrackValidationError(t *testing.T) {
	// No sdjson/schema/json provided -> validation error, no network call.
	code, err := Track(TrackArgs{
		Collector: "com.acme",
		AppId:     "snowplowcli",
		Method:    "GET",
		Protocol:  "https",
		Entities:  "[]",
	}, nil)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestTrackSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	code, err := Track(TrackArgs{
		Collector: strings.TrimPrefix(server.URL, "http://"),
		AppId:     "snowplowcli",
		Method:    "GET",
		Protocol:  "http",
		Schema:    "iglu:com.acme/event/jsonschema/1-0-0",
		Json:      "{\"hello\":\"world\"}",
		Entities:  "[]",
	}, &http.Client{Timeout: time.Duration(5 * time.Second)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestTrackHttpError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	code, err := Track(TrackArgs{
		Collector: strings.TrimPrefix(server.URL, "http://"),
		AppId:     "snowplowcli",
		Method:    "GET",
		Protocol:  "http",
		Sdjson:    "{\"data\":{\"hello\":\"world\"},\"schema\":\"iglu:com.acme/event/jsonschema/1-0-0\"}",
		Entities:  "[]",
	}, &http.Client{Timeout: time.Duration(5 * time.Second)})
	if err == nil {
		t.Fatal("expected error for 5xx response")
	}
	if code != 5 {
		t.Fatalf("expected exit code 5, got %d", code)
	}
}

func TestTrackInvalidMethod(t *testing.T) {
	code, err := Track(TrackArgs{
		Collector: "com.acme",
		Method:    "PUT",
		Protocol:  "https",
		Sdjson:    "{\"data\":{},\"schema\":\"iglu:com.acme/event/jsonschema/1-0-0\"}",
		Entities:  "[]",
	}, nil)
	if err == nil {
		t.Fatal("expected validation error for method PUT")
	}
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestTrackInvalidProtocol(t *testing.T) {
	code, err := Track(TrackArgs{
		Collector: "com.acme",
		Method:    "GET",
		Protocol:  "ftp",
		Sdjson:    "{\"data\":{},\"schema\":\"iglu:com.acme/event/jsonschema/1-0-0\"}",
		Entities:  "[]",
	}, nil)
	if err == nil {
		t.Fatal("expected validation error for protocol ftp")
	}
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestTrackNormalizesMethodAndProtocolCase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer server.Close()

	code, err := Track(TrackArgs{
		Collector: strings.TrimPrefix(server.URL, "http://"),
		Method:    "get",
		Protocol:  "HTTP",
		Sdjson:    "{\"data\":{\"hello\":\"world\"},\"schema\":\"iglu:com.acme/event/jsonschema/1-0-0\"}",
		Entities:  "[]",
	}, &http.Client{Timeout: time.Duration(5 * time.Second)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestTrackRedirectSuccess(t *testing.T) {
	// A 3xx with no Location is not followed by the client; the tracker sees
	// the raw 3xx status, which parseStatusCode maps to exit 0.
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(301)
	}))
	defer server.Close()

	code, err := Track(TrackArgs{
		Collector: strings.TrimPrefix(server.URL, "http://"),
		Method:    "GET",
		Protocol:  "http",
		Sdjson:    "{\"data\":{\"hello\":\"world\"},\"schema\":\"iglu:com.acme/event/jsonschema/1-0-0\"}",
		Entities:  "[]",
	}, &http.Client{Timeout: time.Duration(5 * time.Second)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0 for 3xx, got %d", code)
	}
	if requests != 1 {
		t.Fatalf("expected 1 request, got %d", requests)
	}
}

func TestTrackSendsEntitiesAndIpAddress(t *testing.T) {
	// GET puts the payload in the query string: aid=app-id, ip=ip-address,
	// cx=base64(std)-encoded contexts. Capture and decode to confirm the
	// subject and entities actually reach the collector.
	var query url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		w.WriteHeader(200)
	}))
	defer server.Close()

	code, err := Track(TrackArgs{
		Collector: strings.TrimPrefix(server.URL, "http://"),
		AppId:     "myapp",
		Method:    "GET",
		Protocol:  "http",
		IpAddress: "1.2.3.4",
		Sdjson:    "{\"data\":{\"hello\":\"world\"},\"schema\":\"iglu:com.acme/event/jsonschema/1-0-0\"}",
		Entities:  "[{\"data\":{\"k\":\"v\"},\"schema\":\"iglu:com.acme/context_1/jsonschema/1-0-0\"}]",
	}, &http.Client{Timeout: time.Duration(5 * time.Second)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if got := query.Get("aid"); got != "myapp" {
		t.Fatalf("expected aid=myapp, got %q", got)
	}
	if got := query.Get("ip"); got != "1.2.3.4" {
		t.Fatalf("expected ip=1.2.3.4, got %q", got)
	}

	cx := query.Get("cx")
	if cx == "" {
		t.Fatal("expected encoded contexts (cx) param to be present")
	}
	decoded, err := base64.StdEncoding.DecodeString(cx)
	if err != nil {
		t.Fatalf("cx not valid base64: %v", err)
	}
	if !strings.Contains(string(decoded), "iglu:com.acme/context_1/jsonschema/1-0-0") {
		t.Fatalf("entity schema missing from contexts: %s", decoded)
	}
}

func TestTrackOmitsContextsWhenNoEntities(t *testing.T) {
	var query url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		w.WriteHeader(200)
	}))
	defer server.Close()

	code, err := Track(TrackArgs{
		Collector: strings.TrimPrefix(server.URL, "http://"),
		Method:    "GET",
		Protocol:  "http",
		Sdjson:    "{\"data\":{\"hello\":\"world\"},\"schema\":\"iglu:com.acme/event/jsonschema/1-0-0\"}",
		Entities:  "[]",
	}, &http.Client{Timeout: time.Duration(5 * time.Second)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if query.Get("co") != "" || query.Get("cx") != "" {
		t.Fatalf("expected no contexts params, got co=%q cx=%q", query.Get("co"), query.Get("cx"))
	}
}
