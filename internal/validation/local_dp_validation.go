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
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/snowplow-product/snowplow-cli/internal/console"
	snplog "github.com/snowplow-product/snowplow-cli/internal/logging"
	"github.com/snowplow-product/snowplow-cli/internal/model"
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
}

func (v *DPValidations) concat(r DPValidations) {
	v.Errors = append(v.Errors, r.Errors...)
	v.Warnings = append(v.Warnings, r.Warnings...)
	v.Info = append(v.Info, r.Info...)
	v.Debug = append(v.Debug, r.Debug...)
}

func NewDPLookup(sdc console.SchemaDeployChecker, dp map[string]map[string]any) (*DPLookup, error) {

	probablyDps := map[string]model.DataProduct{}
	probablySap := map[string]model.SourceApp{}
	validation := map[string]DPValidations{}

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
					v.Errors = append(v.Errors, fmt.Sprintf("failed to decode data product %e", err))
				}
			case "source-application":
				var sa model.SourceApp
				if err := mapstructure.Decode(maybeDp, &sa); err == nil {
					probablySap[f] = sa
				} else {
					v.Errors = append(v.Errors, fmt.Sprintf("failed to decode source application %e", err))
				}
				v.concat(ValidateSAMinimum(sa))
				v.concat(ValidateSAAppIds(sa))
				v.concat(ValidateSAEntitiesCardinalities(sa))
				v.concat(ValidateSAEntitiesSources(sa))
				v.concat(ValidateSAEntitiesHaveNoRules(sa))
				v.concat(ValidateSAEntitiesSchemaDeployed(sdc, sa))
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
			for _, ref := range spec.SourceApplications {
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
								snplog.LogFatal(err)
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

func (lookup *DPLookup) SlogValidations(basePath string) error {
	for f, v := range lookup.Validations {
		rp, err := filepath.Rel(basePath, f)
		if err != nil {
			return err
		}
		for _, m := range v.Debug {
			slog.Debug("validating", "file", rp, "msg", m)
		}
		for _, m := range v.Info {
			slog.Info("validating", "file", rp, "msg", m)
		}
		for _, m := range v.Warnings {
			slog.Warn("validating", "file", rp, "msg", m)
		}
		for _, m := range v.Errors {
			slog.Error("validating", "file", rp, "msg", m)
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
		if len(v.Errors) > 0 {
			fmt.Printf("::error file=%s::%s\n", rp, strings.Join(v.Errors, "%0A"))
		}
	}

	return nil
}

func (lookup *DPLookup) ValidationErrorCount() int {
	count := 0
	for _, v := range lookup.Validations {
		count += len(v.Errors)
	}
	return count
}
