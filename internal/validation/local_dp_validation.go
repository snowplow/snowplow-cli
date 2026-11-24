/*
Copyright (c) 2013-present Snowplow Analytics Ltd.
All rights reserved.
This software is made available by Snowplow Analytics, Ltd.,
under the terms of the Snowplow Limited Use License Agreement, Version 1.0
located at https://docs.snowplow.io/limited-use-license-1.0
BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
*/

package validation

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/snowplow/snowplow-cli/internal/console"
	"github.com/snowplow/snowplow-cli/internal/logging"
	"github.com/snowplow/snowplow-cli/internal/model"
)

type DPLookup struct {
	DataProducts map[string]model.DataProduct
	SourceApps   map[string]model.SourceApp

	Validations map[string]DPValidations
}

type DPValidations struct {
	Errors   []string
	Warnings []string
	Info     []string
	Debug    []string

	ErrorsWithPaths   map[string][]string
	WarningsWithPaths map[string][]string
}

func (v *DPValidations) concat(r DPValidations) {
	v.Errors = append(v.Errors, r.Errors...)
	v.Warnings = append(v.Warnings, r.Warnings...)
	v.Info = append(v.Info, r.Info...)
	v.Debug = append(v.Debug, r.Debug...)

	if v.ErrorsWithPaths == nil {
		v.ErrorsWithPaths = make(map[string][]string)
	}

	if v.WarningsWithPaths == nil {
		v.WarningsWithPaths = make(map[string][]string)
	}

	for k, rv := range r.ErrorsWithPaths {
		v.ErrorsWithPaths[k] = append(v.ErrorsWithPaths[k], rv...)
	}

	for k, rv := range r.WarningsWithPaths {
		v.WarningsWithPaths[k] = append(v.WarningsWithPaths[k], rv...)
	}
}

func NewDPLookup(cc console.CompatChecker, sdc console.SchemaDeployChecker, dp map[string]map[string]any, changedIdToFile map[string]string, validateAll bool, concurrency int) (*DPLookup, error) {
	result := &DPLookup{
		DataProducts: map[string]model.DataProduct{},
		SourceApps:   map[string]model.SourceApp{},
		Validations:  map[string]DPValidations{},
	}

	changedFiles := fileSet(changedIdToFile)

	for f, maybeDp := range dp {
		if apiV, ok := maybeDp["apiVersion"]; !ok || apiV != "v1" {
			continue
		}

		resourceType, ok := maybeDp["resourceType"]
		if !ok {
			result.Validations[f] = DPValidations{Errors: []string{"missing resourceType"}}
			continue
		}

		switch resourceType {
		case "data-product":
			result.Validations[f] = result.addDataProductAndValidateShapeAndCompat(cc, concurrency, f, maybeDp, validateAll || changedFiles[f])
		case "source-application":
			result.Validations[f] = result.addSourceAppAndValidate(sdc, f, maybeDp)
		default:
			result.Validations[f] = DPValidations{Debug: []string{fmt.Sprintf("ignoring, unknown resourceType: %s", resourceType)}}
		}
	}

	err := result.resolveRefs()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func fileSet(idToFile map[string]string) map[string]bool {
	files := make(map[string]bool)
	for _, file := range idToFile {
		files[file] = true
	}
	return files
}

func (lookup *DPLookup) SlogValidations(ctx context.Context, basePath string) error {
	logger := logging.LoggerFromContext(ctx)
	for f, v := range lookup.Validations {
		rp, err := filepath.Rel(basePath, f)
		if err != nil {
			return err
		}
		for _, m := range v.Debug {
			logger.Debug("validating", "file", rp, "msg", m)
		}
		for _, m := range v.Info {
			logger.Info("validating", "file", rp, "msg", m)
		}
		for _, m := range v.Warnings {
			logger.Warn("validating", "file", rp, "msg", m)
		}
		for k, se := range v.WarningsWithPaths {
			logger.Warn("validating", "file", rp, "path", k, "warnings", strings.Join(se, "\n")+"\n")
		}
		for _, m := range v.Errors {
			logger.Error("validating", "file", rp, "msg", m)
		}
		for k, se := range v.ErrorsWithPaths {
			logger.Error("validating", "file", rp, "path", k, "errors", strings.Join(se, "\n")+"\n")
		}
	}

	return nil
}

func (lookup *DPLookup) GhAnnotateValidations(basePath string) error {
	for f, v := range lookup.Validations {
		rp, err := filepath.Rel(basePath, f)
		if err != nil {
			return err
		}
		if len(v.Info) > 0 {
			fmt.Printf("::info file=%s::%s\n", rp, strings.Join(v.Info, "%0A"))
		}
		if len(v.Warnings) > 0 {
			fmt.Printf("::warn file=%s::%s\n", rp, strings.Join(v.Warnings, "%0A"))
		}
		for k, se := range v.WarningsWithPaths {
			fmt.Printf("::warn file=%s::%s%%0A%s\n", rp, k, strings.Join(se, "%0A"))
		}
		if len(v.Errors) > 0 {
			fmt.Printf("::error file=%s::%s\n", rp, strings.Join(v.Errors, "%0A"))
		}
		for k, se := range v.ErrorsWithPaths {
			fmt.Printf("::error file=%s::%s%%0A%s\n", rp, k, strings.Join(se, "%0A"))
		}
	}

	return nil
}

func (lookup *DPLookup) AddVersionMismatches(versionMismatches []model.EventSpecVersionMismatch, fixCommand string) {
	for _, m := range versionMismatches {
		path := fmt.Sprintf("/data/eventSpecifications/%d", m.Index)

		summary := fmt.Sprintf("%s: version %d does not match", m.Name, m.Declared)
		if !m.IsNew && len(m.Changed) > 0 && m.Expected > m.Declared {
			summary = fmt.Sprintf("%s: version %d needs a bump", m.Name, m.Declared)
		}
		messages := []string{summary}
		if m.IsNew {
			messages = append(messages, "new event specification")
		} else if len(m.Changed) > 0 {
			messages = append(messages, "changed: "+strings.Join(m.Changed, ", "))
		}
		if m.EntityCommentsRemoved {
			messages = append(messages, "entities: a comment set in Console will be removed")
		}
		messages = append(messages,
			fmt.Sprintf("expected version: %d", m.Expected),
			fmt.Sprintf("run `%s` to update it", fixCommand),
		)

		v := lookup.Validations[m.FileName]
		v.concat(DPValidations{ErrorsWithPaths: map[string][]string{path: messages}})
		lookup.Validations[m.FileName] = v
	}
}

func (lookup *DPLookup) ValidationErrorCount() int {
	count := 0
	for _, v := range lookup.Validations {
		count += len(v.Errors) + len(v.ErrorsWithPaths)
	}
	return count
}

func (lookup *DPLookup) addDataProductAndValidateShapeAndCompat(cc console.CompatChecker, concurrency int, f string, maybeDp map[string]any, checkCompat bool) DPValidations {
	dataProduct, v, shapeOk := lookup.addDataProductAndValidateShape(f, maybeDp)
	if !shapeOk || cc == nil {
		return v
	}
	if !checkCompat {
		v.Debug = append(v.Debug, fmt.Sprintf("skipping compatibility check for %s, since it was not changed, use --full to include it in validation", f))
		return v
	}
	v.concat(ValidateDPEventSpecCompat(cc, concurrency, dataProduct))
	return v
}

func (lookup *DPLookup) addDataProductAndValidateShape(f string, maybeDp map[string]any) (model.DataProduct, DPValidations, bool) {
	v := DPValidations{}

	var dp model.DataProduct
	if err := mapstructure.Decode(maybeDp, &dp); err == nil {
		lookup.DataProducts[f] = dp
	} else {
		v.Errors = append(v.Errors, fmt.Sprintf("failed to decode data product %s", err))
	}

	shapeValidations, shapeOk := ValidateDPShape(maybeDp)
	v.concat(shapeValidations)

	return dp, v, shapeOk
}

func (lookup *DPLookup) addSourceAppAndValidate(sdc console.SchemaDeployChecker, f string, maybeSa map[string]any) DPValidations {
	v := DPValidations{}

	var sa model.SourceApp
	if err := mapstructure.Decode(maybeSa, &sa); err == nil {
		lookup.SourceApps[f] = sa
	} else {
		v.Errors = append(v.Errors, fmt.Sprintf("failed to decode source application %s", err))
	}

	shapeValidations, shapeOk := ValidateSAShape(maybeSa)
	v.concat(shapeValidations)
	v.concat(ValidateSAEntitiesCardinalities(sa))
	if shapeOk && sdc != nil {
		v.concat(ValidateSAEntitiesSchemaDeployed(sdc, sa))
	}

	return v
}

func (lookup *DPLookup) allSourceAppsRelativeTo(filename string) ([]string, error) {
	relativeSas := []string{}
	for path := range lookup.SourceApps {
		relPath, err := filepath.Rel(filepath.Dir(filename), path)
		if err != nil {
			return nil, err
		}
		relativeSas = append(relativeSas, relPath)
	}

	return relativeSas, nil
}

func (lookup *DPLookup) resolveRefs() error {
	for dpFile, dp := range lookup.DataProducts {
		v, ok := lookup.Validations[dpFile]
		if !ok {
			v = DPValidations{}
		}

		resolvedSas, notFound, err := lookup.resolveSourceAppRefs(dpFile, dp.Data.SourceApplications)
		if err != nil {
			return err
		}
		v.Errors = append(v.Errors, notFound...)

		for _, spec := range dp.Data.EventSpecifications {
			notInParent, err := resolveExcludedSourceAppRefs(dpFile, spec, resolvedSas)
			if err != nil {
				return err
			}
			v.Errors = append(v.Errors, notInParent...)
		}

		lookup.Validations[dpFile] = v
	}

	return nil
}

func (lookup *DPLookup) resolveSourceAppRefs(dpFile string, refs []map[string]string) ([]string, []string, error) {
	resolvedSas := []string{}
	notFound := []string{}

	for _, ref := range refs {
		saRef, ok := ref["$ref"]
		if !ok {
			continue
		}

		absPath, err := filepath.Abs(filepath.Join(filepath.Dir(dpFile), saRef))
		if err != nil {
			return nil, nil, err
		}

		if _, ok := lookup.SourceApps[absPath]; ok {
			ref["$ref"] = absPath
			resolvedSas = append(resolvedSas, absPath)
			continue
		}

		available, err := lookup.allSourceAppsRelativeTo(dpFile)
		if err != nil {
			return nil, nil, err
		}
		notFound = append(notFound, fmt.Sprintf("source application $ref not found %s, available list %v", saRef, available))
	}

	return resolvedSas, notFound, nil
}

func resolveExcludedSourceAppRefs(dpFile string, spec model.EventSpec, resolvedSas []string) ([]string, error) {
	notInParent := []string{}

	for _, ref := range spec.ExcludedSourceApplications {
		saRef, ok := ref["$ref"]
		if !ok {
			continue
		}

		absPath, err := filepath.Abs(filepath.Join(filepath.Dir(dpFile), saRef))
		if err != nil {
			return nil, err
		}
		ref["$ref"] = absPath

		if slices.Contains(resolvedSas, absPath) {
			continue
		}

		relativeSas, err := pathsRelativeTo(dpFile, resolvedSas)
		if err != nil {
			logging.LogFatal(err)
		}

		notInParent = append(notInParent, fmt.Sprintf(
			"event spec source app not in parent data product list (event spec: %s, source app: %s), available list %v",
			spec.ResourceName,
			saRef,
			relativeSas,
		))
	}

	return notInParent, nil
}

func pathsRelativeTo(filename string, absPaths []string) ([]string, error) {
	relPaths := []string{}
	for _, absPath := range absPaths {
		relPath, err := filepath.Rel(filepath.Dir(filename), absPath)
		if err != nil {
			return nil, err
		}
		relPaths = append(relPaths, relPath)
	}

	return relPaths, nil
}
