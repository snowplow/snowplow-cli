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
	"errors"
	"fmt"
	"github.com/snowplow/snowplow-cli/internal/model"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

func validateDs(validate *validator.Validate, ds model.DataStructure) error {

	type validation struct {
		field          string
		path           string
		validationType string
		value          any
		error          string
	}

	data, err := ds.ParseData()
	if err != nil {
		return err
	}
	errs := validate.Struct(ds)
	errsData := validate.Struct(data)
	var allErrs []validation

	if errs != nil {
		for _, e := range errs.(validator.ValidationErrors) {
			path := "dataStructure." + strings.TrimPrefix(e.Namespace(), "DataStructure.")
			res := validation{e.Field(), path, e.Tag(), e.Value(), e.Error()}
			allErrs = append(allErrs, res)
		}
	}

	if errsData != nil {
		for _, e := range errsData.(validator.ValidationErrors) {
			path := "dataStructure.data." + strings.TrimPrefix(e.Namespace(), "DataStructureData.")
			res := validation{e.Field(), path, e.Tag(), e.Value(), e.Error()}
			allErrs = append(allErrs, res)
		}
	}
	var result []error

	if len(allErrs) > 0 {
		for _, err := range allErrs {
			switch err.validationType {
			case "required":
				result = append(result, fmt.Errorf("required field %s is missing", err.path))
			case "oneof":
				firstSentence := fmt.Sprintf("Invalid value %s at %s. Avaliable values are: ", err.value, err.path)
				var secondSentence string
				switch err.field {
				case "schemaType":
					secondSentence = "event, entity"
				case "apiVersion":
					secondSentence = "v1"
				case "resourceType":
					secondSentence = "data-structure"
				case "format":
					secondSentence = "jsonschema"
				}
				result = append(result, errors.New(firstSentence+secondSentence))
			default:
				result = append(result, errors.New(err.error))
			}
		}
		return errors.Join(result...)
	}

	return nil

}

func ValidateLocalDs(dss map[string]model.DataStructure) []error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		return field.Tag.Get("json")
	})
	allErrors := []error{}
	counts := make(map[string][]string)
	for fileName, ds := range dss {
		errs := validateDs(validate, ds)
		if errs != nil {
			error := errors.Join(fmt.Errorf("validation failed for %s", fileName), errs)
			allErrors = append(allErrors, error)
		}
		data, err := ds.ParseData()
		if err != nil {
			allErrors = append(allErrors, err)
		}
		key := fmt.Sprintf("%s/%s", data.Self.Vendor, data.Self.Name)
		counts[key] = append(counts[key], fileName)
	}
	for key, files := range counts {
		if len(files) > 1 {
			allErrors = append(allErrors, fmt.Errorf("the mapping between data structures and files should be unique. Files %s describe the same data structure %s", files, key))
		}
	}

	return allErrors
}
