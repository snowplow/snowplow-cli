/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package wizard

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const DefaultSkillsRef = "main"

const skillsPrefix = "plugins/snowplow/skills/"

// markerFile marks a skill folder as installed by snowplow-cli, so later runs
// update it but never touch a folder of the same name the user made.
const markerFile = ".snowplow-cli"

var skillsArchiveURL = "https://codeload.github.com/snowplow/skills/tar.gz/%s"

type Skill struct {
	Name  string
	Files map[string][]byte
}

func FetchSkills(ctx context.Context, ref string) ([]Skill, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(skillsArchiveURL, ref), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading snowplow/skills@%s: %s", ref, resp.Status)
	}
	return readSkills(io.LimitReader(resp.Body, 50<<20))
}

func readSkills(r io.Reader) ([]Skill, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)

	byName := map[string]*Skill{}
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		// Archive paths start with "<repo>-<ref>/".
		_, rel, ok := strings.Cut(h.Name, "/")
		if !ok || !strings.HasPrefix(rel, skillsPrefix) {
			continue
		}
		name, file, ok := strings.Cut(strings.TrimPrefix(rel, skillsPrefix), "/")
		if !ok || name == "" || file == "" || !safeRel(file) {
			continue
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		if byName[name] == nil {
			byName[name] = &Skill{Name: name, Files: map[string][]byte{}}
		}
		byName[name].Files[file] = b
	}

	var skills []Skill
	for _, s := range byName {
		if _, ok := s.Files["SKILL.md"]; ok {
			skills = append(skills, *s)
		}
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	if len(skills) == 0 {
		return nil, errors.New("no skills found in the archive")
	}
	return skills, nil
}

func safeRel(p string) bool {
	clean := path.Clean(p)
	return clean == p && !path.IsAbs(p) && clean != ".." && !strings.HasPrefix(clean, "../")
}

// InstallSkills writes skills into dir. It returns the names it skipped
// because a folder of that name exists that snowplow-cli didn't create.
func InstallSkills(dir string, skills []Skill) (skipped []string, err error) {
	for _, s := range skills {
		target := filepath.Join(dir, s.Name)
		if _, err := os.Stat(target); err == nil {
			if _, err := os.Stat(filepath.Join(target, markerFile)); err != nil {
				skipped = append(skipped, s.Name)
				continue
			}
			if err := os.RemoveAll(target); err != nil {
				return skipped, err
			}
		}
		for file, b := range s.Files {
			p := filepath.Join(target, filepath.FromSlash(file))
			if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
				return skipped, err
			}
			if err := os.WriteFile(p, b, 0644); err != nil {
				return skipped, err
			}
		}
		if err := os.WriteFile(filepath.Join(target, markerFile), []byte("Installed by snowplow-cli. Re-running setup replaces this folder.\n"), 0644); err != nil {
			return skipped, err
		}
	}
	return skipped, nil
}
