// Package validation contains Core input contracts.
//
// Validation rejects data that does not belong to a contract. It deliberately
// does not rewrite user text, keys, configuration values, or secrets.
package validation

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-playground/validator/v10"
	sharedvalidation "github.com/navire-dev/navire/shared/validation"
)

type Issue struct {
	Field   string
	Code    string
	Message string
}

type DataLimits struct {
	MaxKeys     int
	MaxKeyLen   int
	MaxValueLen int
}

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

var structValidator = newStructValidator()

func newStructValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	mustRegisterValidation(v, "identifier", func(fl validator.FieldLevel) bool {
		value, ok := fl.Field().Interface().(string)
		return ok && identifierPattern.MatchString(value)
	})
	mustRegisterValidation(v, "navirekey", func(fl validator.FieldLevel) bool {
		value, ok := fl.Field().Interface().(string)
		return ok && sharedvalidation.IsKey(value)
	})
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			return field.Name
		}
		return name
	})
	return v
}

func mustRegisterValidation(v *validator.Validate, name string, fn validator.Func) {
	if err := v.RegisterValidation(name, fn); err != nil {
		panic(fmt.Errorf("register validation %q: %w", name, err))
	}
}

func Struct(value any) []Issue {
	err := structValidator.Struct(value)
	if err == nil {
		return nil
	}

	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return []Issue{{Code: "invalid", Message: "invalid request"}}
	}

	issues := make([]Issue, 0, len(validationErrors))
	for _, fieldErr := range validationErrors {
		issues = append(issues, Issue{
			Field:   fieldErr.Field(),
			Code:    fieldErr.Tag(),
			Message: validationMessage(fieldErr),
		})
	}
	return issues
}

func validationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "is required"
	case "max":
		return "is too long"
	case "identifier":
		return "must contain only letters, digits, '.', '_' or '-'"
	case "navirekey":
		return "must contain only lowercase letters, digits, '-' or '_' and be at most 50 characters"
	case "oneof":
		return "must be one of: " + strings.ReplaceAll(err.Param(), " ", ", ")
	default:
		return "is invalid"
	}
}

func Data(values map[string]any, limits DataLimits) []Issue {
	if len(values) == 0 {
		return nil
	}

	issues := make([]Issue, 0)
	if limits.MaxKeys > 0 && len(values) > limits.MaxKeys {
		issues = append(issues, Issue{
			Field:   "data",
			Code:    "too_many_keys",
			Message: "contains too many keys",
		})
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := values[key]
		field := dataField(key)
		if key == "" || (limits.MaxKeyLen > 0 && utf8.RuneCountInString(key) > limits.MaxKeyLen) || !identifierPattern.MatchString(key) {
			issues = append(issues, Issue{
				Field:   field,
				Code:    "invalid_key",
				Message: "must contain only letters, digits, '.', '_' or '-'",
			})
		}

		switch typed := value.(type) {
		case string:
			if !utf8.ValidString(typed) {
				issues = append(issues, Issue{Field: field, Code: "invalid_utf8", Message: "must be valid UTF-8"})
			}
			if limits.MaxValueLen > 0 && utf8.RuneCountInString(typed) > limits.MaxValueLen {
				issues = append(issues, Issue{Field: field, Code: "too_long", Message: "is too long"})
			}
		case float64, bool, nil:
		default:
			issues = append(issues, Issue{Field: field, Code: "unsupported_type", Message: "only string, number, bool, or null are allowed"})
		}
	}
	return issues
}

func dataField(key string) string {
	if identifierPattern.MatchString(key) {
		return "data." + key
	}
	return "data[" + strconv.Quote(key) + "]"
}
