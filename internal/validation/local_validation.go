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
	data, err := ds.ParseData()
	if err != nil {
		return err
	}
	errs := validate.Struct(ds)
	errsData := validate.Struct(data)

	dsErrs, err := dsFieldErrors(errs, "dataStructure.", "DataStructure.")
	if err != nil {
		return err
	}
	dataErrs, err := dsFieldErrors(errsData, "dataStructure.data.", "DataStructureData.")
	if err != nil {
		return err
	}

	return errors.Join(append(dsErrs, dataErrs...)...)
}

func dsFieldErrors(errs error, pathPrefix string, namespacePrefix string) ([]error, error) {
	if errs == nil {
		return nil, nil
	}
	var fieldErrs validator.ValidationErrors
	if !errors.As(errs, &fieldErrs) {
		return nil, errs
	}
	var result []error
	for _, e := range fieldErrs {
		path := pathPrefix + strings.TrimPrefix(e.Namespace(), namespacePrefix)
		result = append(result, dsFieldError(e, path))
	}
	return result, nil
}

var dsOneOfValues = map[string]string{
	"schemaType":   "event, entity",
	"apiVersion":   "v1",
	"resourceType": "data-structure",
	"format":       "jsonschema",
}

func dsFieldError(e validator.FieldError, path string) error {
	switch e.Tag() {
	case "required":
		return fmt.Errorf("required field %s is missing", path)
	case "oneof":
		firstSentence := fmt.Sprintf("Invalid value %s at %s. Available values are: ", e.Value(), path)
		return errors.New(firstSentence + dsOneOfValues[e.Field()])
	default:
		return errors.New(e.Error())
	}
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
			allErrors = append(allErrors, errors.Join(fmt.Errorf("validation failed for %s", fileName), errs))
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
