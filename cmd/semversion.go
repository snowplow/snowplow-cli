package cmd

import (
	"fmt"
	"strconv"
	"strings"
)

type semVersion struct {
	Major    uint64
	Revision uint64
	Addition uint64
}

func parseSemVer(v string) (*semVersion, error) {
	version := strings.Split(v, "-")

	var err error

	major, err := strconv.ParseUint(version[0], 10, 32)
	if err != nil {
		return nil, err
	}
	revision, err := strconv.ParseUint(version[1], 10, 32)
	if err != nil {
		return nil, err
	}
	addition, err := strconv.ParseUint(version[2], 10, 32)
	if err != nil {
		return nil, err
	}

	return &semVersion{major, revision, addition}, nil
}

func semNextVer(v semVersion, upgradeType string) semVersion {
	switch upgradeType {
	case "major":
		return semVersion{v.Major + 1, 0, 0}
	case "revision":
		return semVersion{v.Major, v.Revision + 1, 0}
	case "minor":
		return semVersion{v.Major, v.Revision, v.Addition + 1}
	}

	return semVersion{v.Major, v.Revision, v.Addition}
}

func semVerCmp(x semVersion, y semVersion) int {
	if x.Major > y.Major {
		return 1
	}
	if x.Major < y.Major {
		return -1
	}
	if x.Revision > y.Revision {
		return 1
	}
	if x.Revision < y.Revision {
		return -1
	}
	if x.Addition > y.Addition {
		return 1
	}
	if x.Addition < y.Addition {
		return -1
	}

	return 0
}

func (v *semVersion) String() string {
	return fmt.Sprintf("%d-%d-%d", v.Major, v.Revision, v.Addition)
}
