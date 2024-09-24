package model

import (
	"fmt"
	"strconv"
	"strings"
)

type SemVersion struct {
	Major    uint64
	Revision uint64
	Addition uint64
}

func ParseSemVer(v string) (*SemVersion, error) {
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

	return &SemVersion{major, revision, addition}, nil
}

func SemNextVer(v SemVersion, upgradeType string) SemVersion {
	switch upgradeType {
	case "major":
		return SemVersion{v.Major + 1, 0, 0}
	case "revision":
		return SemVersion{v.Major, v.Revision + 1, 0}
	case "minor":
		return SemVersion{v.Major, v.Revision, v.Addition + 1}
	}

	return SemVersion{v.Major, v.Revision, v.Addition}
}

func SemVerCmp(x SemVersion, y SemVersion) int {
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

func (v *SemVersion) String() string {
	return fmt.Sprintf("%d-%d-%d", v.Major, v.Revision, v.Addition)
}
