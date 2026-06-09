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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	storagememory "github.com/snowplow/snowplow-golang-tracker/v3/pkg/storage/memory"
	gt "github.com/snowplow/snowplow-golang-tracker/v3/tracker"
)

type selfDescJSON struct {
	Schema string                 `json:"schema"`
	Data   map[string]interface{} `json:"data"`
}

func getSdJSON(sdjson string, schema string, jsonData string) (*gt.SelfDescribingJson, error) {
	if sdjson == "" && schema == "" && jsonData == "" {
		return nil, errors.New("fatal: --sdjson or --schema URI plus a --json needs to be specified")
	} else if sdjson != "" {
		res := selfDescJSON{}
		d := json.NewDecoder(strings.NewReader(sdjson))
		d.UseNumber()
		if err := d.Decode(&res); err != nil {
			return nil, err
		}
		return gt.InitSelfDescribingJson(res.Schema, res.Data), nil
	} else if schema != "" && jsonData == "" {
		return nil, errors.New("fatal: --json needs to be specified")
	} else if schema == "" && jsonData != "" {
		return nil, errors.New("fatal: --schema URI needs to be specified")
	} else {
		jsonDataMap, err := stringToMap(jsonData)
		if err != nil {
			return nil, err
		}
		return gt.InitSelfDescribingJson(schema, jsonDataMap), nil
	}
}

func getEntities(entities string) ([]gt.SelfDescribingJson, error) {
	if entities == "" {
		return nil, nil
	}
	res := []selfDescJSON{}
	d := json.NewDecoder(strings.NewReader(entities))
	d.UseNumber()
	if err := d.Decode(&res); err != nil {
		return nil, err
	}

	sdjArr := make([]gt.SelfDescribingJson, len(res))
	for i, entity := range res {
		sdj := gt.InitSelfDescribingJson(entity.Schema, entity.Data)
		sdjArr[i] = *sdj
	}
	return sdjArr, nil
}

func initTracker(collector string, appid string, method string, protocol string, ipAddress string, trackerChan chan int, httpClient *http.Client) *gt.Tracker {
	callback := func(s []gt.CallbackResult, f []gt.CallbackResult) {
		status := 0
		if len(s) == 1 {
			status = s[0].Status
		} else if len(f) == 1 {
			status = f[0].Status
		}
		trackerChan <- status
	}

	emitter := gt.InitEmitter(
		gt.RequireCollectorUri(collector),
		gt.RequireStorage(storagememory.Init()),
		gt.OptionCallback(callback),
		gt.OptionRequestType(method),
		gt.OptionProtocol(protocol),
		gt.OptionHttpClient(httpClient),
	)
	subject := gt.InitSubject()
	if ipAddress != "" {
		subject.SetIpAddress(ipAddress)
	}
	return gt.InitTracker(
		gt.RequireEmitter(emitter),
		gt.OptionSubject(subject),
		gt.OptionAppId(appid),
	)
}

func trackSelfDescribingEvent(tracker *gt.Tracker, trackerChan chan int, sdj *gt.SelfDescribingJson, entities []gt.SelfDescribingJson) int {
	tracker.TrackSelfDescribingEvent(gt.SelfDescribingEvent{
		Event:    sdj,
		Contexts: entities,
	})
	returnCode := <-trackerChan
	tracker.Emitter.Storage.DeleteAllEventRows()
	return returnCode
}

func parseStatusCode(statusCode int) int {
	switch statusCode / 100 {
	case 2, 3:
		return 0
	case 4:
		return 4
	case 5:
		return 5
	default:
		return 1
	}
}

func stringToMap(str string) (map[string]interface{}, error) {
	var jsonDataMap map[string]interface{}
	d := json.NewDecoder(strings.NewReader(str))
	d.UseNumber()
	if err := d.Decode(&jsonDataMap); err != nil {
		return nil, err
	}
	return jsonDataMap, nil
}

type TrackArgs struct {
	Collector string
	AppId     string
	Method    string
	Protocol  string
	Sdjson    string
	Schema    string
	Json      string
	IpAddress string
	Entities  string
}

func Track(args TrackArgs, httpClient *http.Client) (int, error) {
	if args.Collector == "" {
		return 1, errors.New("fatal: --collector needs to be specified")
	}

	method := strings.ToUpper(args.Method)
	if method != "GET" && method != "POST" {
		return 1, fmt.Errorf("fatal: --method must be GET or POST, got %q", args.Method)
	}
	protocol := strings.ToLower(args.Protocol)
	if protocol != "http" && protocol != "https" {
		return 1, fmt.Errorf("fatal: --protocol must be http or https, got %q", args.Protocol)
	}

	sdj, err := getSdJSON(args.Sdjson, args.Schema, args.Json)
	if err != nil {
		return 1, err
	}

	entities, err := getEntities(args.Entities)
	if err != nil {
		return 1, err
	}

	trackerChan := make(chan int, 1)
	tracker := initTracker(args.Collector, args.AppId, method, protocol, args.IpAddress, trackerChan, httpClient)
	statusCode := trackSelfDescribingEvent(tracker, trackerChan, sdj, entities)

	returnCode := parseStatusCode(statusCode)
	if returnCode != 0 {
		return returnCode, errors.New("error: event failed to send, check your collector endpoint and try again")
	}
	return 0, nil
}
