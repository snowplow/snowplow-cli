/**
 * Copyright (c) 2013-present Snowplow Analytics Ltd.
 * All rights reserved.
 * This software is made available by Snowplow Analytics, Ltd.,
 * under the terms of the Snowplow Limited Use License Agreement, Version 1.0
 * located at https://docs.snowplow.io/limited-use-license-1.0
 * BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
 * OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
 */

package console

import (
	"errors"
	"slices"
	"testing"
)

func Test_IsDeployed_Iglu(t *testing.T) {

	uri := "iglu:vendor/name/format/version"

	mock := &schemaDeployCheckProvider{
		[]string{uri},
		[]ListResponse{},
		func(h string) ([]Deployment, error) {
			return nil, nil
		},
	}

	found, _, _ := mock.IsDSDeployed(uri)

	if !found {
		t.Fatal("!found")
	}
}

func Test_IsDeployed_BuiltIn(t *testing.T) {

	uri := "iglu:com.snowplowanalytics.snowplow/page_view/jsonschema/1-0-0"

	mock := &schemaDeployCheckProvider{
		[]string{},
		[]ListResponse{},
		func(h string) ([]Deployment, error) {
			return nil, nil
		},
	}

	found, _, _ := mock.IsDSDeployed(uri)

	if !found {
		t.Fatal("!found")
	}
}

func Test_IsDeployed_ConsoleLucky(t *testing.T) {
	uri := "iglu:vendor/name/format/1-0-0"

	mock := &schemaDeployCheckProvider{
		[]string{},
		[]ListResponse{{
			Hash:        "hash",
			Vendor:      "vendor",
			Name:        "name",
			Format:      "format",
			Deployments: []Deployment{{"1-0-0", DEV, ""}},
		}},
		func(h string) ([]Deployment, error) {
			return nil, nil
		},
	}

	found, _, _ := mock.IsDSDeployed(uri)

	if !found {
		t.Fatal("!found")
	}
}

func Test_IsDeployed_ConsoleUnLucky(t *testing.T) {
	uri := "iglu:vendor/name/format/1-0-0"

	mock := &schemaDeployCheckProvider{
		[]string{},
		[]ListResponse{{
			Hash:        "hash",
			Vendor:      "vendor",
			Name:        "name",
			Format:      "format",
			Deployments: []Deployment{},
		}},
		func(h string) ([]Deployment, error) {
			return []Deployment{{"1-0-0", DEV, ""}}, nil
		},
	}

	found, _, _ := mock.IsDSDeployed(uri)

	if !found {
		t.Fatal("!found")
	}
}

func Test_IsDeployed_ConsoleAlternatives(t *testing.T) {
	uri := "iglu:vendor/name/format/1-0-10"

	mock := &schemaDeployCheckProvider{
		[]string{},
		[]ListResponse{{
			Hash:        "hash",
			Vendor:      "vendor",
			Name:        "name",
			Format:      "format",
			Deployments: []Deployment{},
		}},
		func(h string) ([]Deployment, error) {
			return []Deployment{{"2-0-0", DEV, ""}, {"1-0-0", DEV, ""}}, nil
		},
	}

	found, alts, _ := mock.IsDSDeployed(uri)

	if found {
		t.Fatal("!!found")
	}

	expected := []string{"2-0-0", "1-0-0"}

	if !slices.Equal(expected, alts) {
		t.Fatal("bad alternatives", expected, alts)
	}
}

func Test_IsDeployed_ConsoleAlternativesFail(t *testing.T) {
	uri := "iglu:vendor/name/format/1-0-10"

	mock := &schemaDeployCheckProvider{
		[]string{},
		[]ListResponse{{
			Hash:        "hash",
			Vendor:      "vendor",
			Name:        "name",
			Format:      "format",
			Deployments: []Deployment{},
		}},
		func(h string) ([]Deployment, error) {
			return nil, errors.New("fail")
		},
	}

	_, _, err := mock.IsDSDeployed(uri)

	if err == nil || err.Error() != "fail" {
		t.Fatal("should have errored")
	}
}
