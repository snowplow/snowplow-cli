/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package model

import "testing"

func TestParseSemVer(t *testing.T) {
	v, err := ParseSemVer("3-14-159")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *v != (SemVersion{3, 14, 159}) {
		t.Fatalf("unexpected version: %+v", *v)
	}
}

func TestParseSemVerInvalidPart(t *testing.T) {
	for _, in := range []string{"a-1-1", "1-b-1", "1-1-c"} {
		v, err := ParseSemVer(in)
		if err == nil {
			t.Errorf("%s: expected an error, got %+v", in, v)
		}
		if v != nil {
			t.Errorf("%s: expected no version, got %+v", in, v)
		}
	}
}

func TestSemNextVer(t *testing.T) {
	current := SemVersion{2, 3, 4}
	cases := map[string]SemVersion{
		"major":    {3, 0, 0},
		"revision": {2, 4, 0},
		"minor":    {2, 3, 5},
		"other":    {2, 3, 4},
	}
	for upgradeType, want := range cases {
		if got := SemNextVer(current, upgradeType); got != want {
			t.Errorf("%s: got %+v, want %+v", upgradeType, got, want)
		}
	}
}

func TestSemVerCmp(t *testing.T) {
	cases := []struct {
		x, y SemVersion
		want int
	}{
		{SemVersion{1, 2, 3}, SemVersion{1, 2, 3}, 0},
		{SemVersion{2, 0, 0}, SemVersion{1, 9, 9}, 1},
		{SemVersion{1, 9, 9}, SemVersion{2, 0, 0}, -1},
		{SemVersion{1, 2, 0}, SemVersion{1, 1, 9}, 1},
		{SemVersion{1, 1, 9}, SemVersion{1, 2, 0}, -1},
		{SemVersion{1, 1, 2}, SemVersion{1, 1, 1}, 1},
		{SemVersion{1, 1, 1}, SemVersion{1, 1, 2}, -1},
	}
	for _, c := range cases {
		if got := SemVerCmp(c.x, c.y); got != c.want {
			t.Errorf("SemVerCmp(%+v, %+v) = %d, want %d", c.x, c.y, got, c.want)
		}
	}
}
