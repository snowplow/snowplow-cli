package cmd

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
)

func validateDs(validate *validator.Validate, ds DataStructure) error {

	type validation struct {
		field          string
		path           string
		validationType string
		value          any
		error          string
	}

	data, err := ds.parseData()
	if err != nil {
		return err
	}
	errs := validate.Struct(ds)
	errsData := validate.Struct(data)
	var allErrs []validation

	if errs != nil {
		for _, e := range errs.(validator.ValidationErrors) {
			path := fieldToPath(e.Namespace())
			res := validation{lowerFirstLetter(e.Field()), path, e.Tag(), e.Value(), e.Error()}
			allErrs = append(allErrs, res)
		}
	}

	if errsData != nil {
		for _, e := range errsData.(validator.ValidationErrors) {
			path := "dataStructure.data." + strings.TrimPrefix(fieldToPath(e.Namespace()), "dataStrucutreData.")
			res := validation{lowerFirstLetter(e.Field()), path, e.Tag(), e.Value(), e.Error()}
			allErrs = append(allErrs, res)
		}
	}
	var result []error

	if len(allErrs) > 0 {
		for _, err := range allErrs {
			switch err.validationType {
			case "required":
				result = append(result, fmt.Errorf("Required field %s is missing", err.path))
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

func ValidateLocalDs(dss map[string]DataStructure) []error {
	validate := validator.New(validator.WithRequiredStructEnabled())
	allErrors := []error{}
	for fileName, ds := range dss {
		errs := validateDs(validate, ds)
		if errs != nil {
			error := errors.Join(fmt.Errorf("Validation failed for %s", fileName), errs)
			allErrors = append(allErrors, error)
		}
	}
	return allErrors
}

func lowerFirstLetter(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size <= 1 {
		return s
	}
	lc := unicode.ToLower(r)
	if r == lc {
		return s
	}
	return string(lc) + s[size:]
}

func fieldToPath(s string) string {
	var parts []string

	for _, part := range strings.Split(s, ".") {
		parts = append(parts, lowerFirstLetter(part))
	}

	return strings.Join(parts, ".")
}
