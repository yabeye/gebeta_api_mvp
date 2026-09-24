package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/yabeye/gebeta_api_mvp/common/apperrors"
)

// URLParamUUID extracts and parses a chi URL parameter as a UUID,
// returning apperrors.ErrValidation with a clear message if the
// parameter is missing or malformed. Use this for every route with an
// {id}-style path segment instead of parsing inline in the handler.
func URLParamUUID(r *http.Request, name string) (uuid.UUID, error) {
	raw := chi.URLParam(r, name)
	if raw == "" {
		return uuid.Nil, fmt.Errorf("%w: missing %q parameter", apperrors.ErrValidation, name)
	}

	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %q is not a valid id", apperrors.ErrValidation, name)
	}

	return id, nil
}

var validate = validator.New()

// DecodeAndValidate decodes JSON and validates the destination.
// Validation errors are returned using request JSON field names,
// without exposing Go struct/type names.
func DecodeAndValidate(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf(
			"%w: invalid request body: %s",
			apperrors.ErrValidation,
			formatJSONError(err),
		)
	}

	if err := validate.Struct(dst); err != nil {
		var validationErrors validator.ValidationErrors

		if errors.As(err, &validationErrors) {
			return fmt.Errorf(
				"%w: %s",
				apperrors.ErrValidation,
				formatValidationErrors(dst, validationErrors),
			)
		}

		return fmt.Errorf("%w: validation failed", apperrors.ErrValidation)
	}

	return nil
}

func formatValidationErrors(
	dst any,
	errs validator.ValidationErrors,
) string {
	var messages []string

	for _, err := range errs {
		field := jsonFieldName(dst, err.StructField())

		message := switchValidationMessage(
			field,
			err.Tag(),
			err.Param(),
		)

		messages = append(messages, message)
	}

	return strings.Join(messages, "; ")
}

func jsonFieldName(dst any, structField string) string {
	t := reflect.TypeOf(dst)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	field, ok := t.FieldByName(structField)
	if !ok {
		return structField
	}

	tag := field.Tag.Get("json")
	if tag == "" || tag == "-" {
		return structField
	}

	return strings.Split(tag, ",")[0]
}

func switchValidationMessage(
	field string,
	tag string,
	param string,
) string {
	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)

	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)

	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)

	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, param)

	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, param)

	case "oneof":
		return fmt.Sprintf("%s has an invalid value", field)

	default:
		return fmt.Sprintf("%s is invalid", field)
	}
}

func formatJSONError(err error) string {
	var syntaxErr *json.SyntaxError

	if errors.As(err, &syntaxErr) {
		return "malformed JSON"
	}

	var unknownField string
	if strings.Contains(err.Error(), "unknown field") {
		unknownField = strings.TrimPrefix(
			strings.TrimPrefix(err.Error(), `json: unknown field `),
			`"`,
		)

		unknownField = strings.TrimSuffix(unknownField, `"`)
		return fmt.Sprintf("unknown field: %s", unknownField)
	}

	return "malformed JSON"
}
