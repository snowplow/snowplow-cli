/**
 * Copyright (c) 2013-present Snowplow Analytics Ltd.
 * All rights reserved.
 * This software is made available by Snowplow Analytics, Ltd.,
 * under the terms of the Snowplow Limited Use License Agreement, Version 1.0
 * located at https://docs.snowplow.io/limited-use-license-1.0
 * BY INSTALLING, DOWNLOADING, ACCESSING, USING OR DISTRIBUTING ANY PORTION
 * OF THE SOFTWARE, YOU AGREE TO THE TERMS OF SUCH LICENSE AGREEMENT.
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

	probablyDps := map[string]model.DataProduct{}
	probablySap := map[string]model.SourceApp{}
	validation := map[string]DPValidations{}

	changedFiles := make(map[string]bool)

	for _, file := range changedIdToFile {
		changedFiles[file] = true
	}

	for f, maybeDp := range dp {
		v := DPValidations{}

		if apiV, ok := maybeDp["apiVersion"]; !ok || apiV != "v1" {
			v.Errors = append(v.Errors, fmt.Sprintf("ignoring, unknown or missing apiVersion: %s", apiV))
			continue
		}

		if resourceType, ok := maybeDp["resourceType"]; ok {
			switch resourceType {
			case "data-product":
				var dp model.DataProduct
				if err := mapstructure.Decode(maybeDp, &dp); err == nil {
					probablyDps[f] = dp
				} else {
					v.Errors = append(v.Errors, fmt.Sprintf("failed to decode data product %s", err))
				}
				shapeValidations, ok := ValidateDPShape(maybeDp)
				v.concat(shapeValidations)
				changed := changedFiles[f]
				if ok {
					if validateAll {
						v.concat(ValidateDPEventSpecCompat(cc, concurrency, dp))
					} else {
						if changed {
							v.concat(ValidateDPEventSpecCompat(cc, concurrency, dp))
						} else {
							v.Debug = append(v.Debug, fmt.Sprintf("skipping compatibility check for %s, since it was not changed, use --full to include it in validation", f))
						}
					}
				}
			case "source-application":
				var sa model.SourceApp
				if err := mapstructure.Decode(maybeDp, &sa); err == nil {
					probablySap[f] = sa
				} else {
					v.Errors = append(v.Errors, fmt.Sprintf("failed to decode source application %s", err))
				}
				shapeValidations, ok := ValidateSAShape(maybeDp)
				v.concat(shapeValidations)
				v.concat(ValidateSAEntitiesCardinalities(sa))
				if ok {
					v.concat(ValidateSAEntitiesSchemaDeployed(sdc, sa))
				}
			default:
				v.Debug = append(v.Debug, fmt.Sprintf("ignoring, unknown resourceType: %s", resourceType))
			}
		} else {
			v.Errors = append(v.Errors, "missing resourceType")
		}

		validation[f] = v
	}

	result := &DPLookup{probablyDps, probablySap, validation}
	err := result.resolveRefs()
	if err != nil {
		return nil, err
	}

	return result, nil
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

		resolvedSas := []string{}

		v, ok := lookup.Validations[dpFile]
		if !ok {
			v = DPValidations{}
		}

		for _, ref := range dp.Data.SourceApplications {
			if saRef, ok := ref["$ref"]; ok {
				absPath, err := filepath.Abs(filepath.Join(filepath.Dir(dpFile), saRef))
				if err != nil {
					return err
				}

				if _, ok := lookup.SourceApps[absPath]; ok {
					ref["$ref"] = absPath
					resolvedSas = append(resolvedSas, absPath)
				} else {
					available, err := lookup.allSourceAppsRelativeTo(dpFile)
					if err != nil {
						return err
					}
					v.Errors = append(v.Errors, fmt.Sprintf("source application $ref not found %s, available list %v", saRef, available))
				}
			}
		}
		for _, spec := range dp.Data.EventSpecifications {
			for _, ref := range spec.ExcludedSourceApplications {
				if saRef, ok := ref["$ref"]; ok {
					absPath, err := filepath.Abs(filepath.Join(filepath.Dir(dpFile), saRef))
					if err != nil {
						return err
					}
					ref["$ref"] = absPath

					if !slices.Contains(resolvedSas, absPath) {

						relativeSas := []string{}
						for _, absSaPath := range resolvedSas {
							relPath, err := filepath.Rel(filepath.Dir(dpFile), absSaPath)
							if err != nil {
								logging.LogFatal(err)
							}
							relativeSas = append(relativeSas, relPath)
						}

						v.Errors = append(v.Errors, fmt.Sprintf(
							"event spec source app not in parent data product list (event spec: %s, source app: %s), available list %v",
							spec.ResourceName,
							saRef,
							relativeSas,
						))
					}
				}
			}
		}

		lookup.Validations[dpFile] = v
	}

	return nil
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

func (lookup *DPLookup) ValidationErrorCount() int {
	count := 0
	for _, v := range lookup.Validations {
		count += len(v.Errors) + len(v.ErrorsWithPaths)
	}
	return count
}
