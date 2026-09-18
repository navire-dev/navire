package bind

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

type (
	// SanitizeFunc mutates the request in-place (closure can capture config)
	SanitizeFunc func(any)

	// ValidateFunc returns validation errors (closure can capture config)
	ValidateFunc func(any) []FieldError
)

type PublicError interface {
	error
	HTTPStatus() int
	PublicCode() string
	PublicMessage() string
}

type FieldError struct {
	Field string `json:"field"`
	Code  string `json:"code"` // "required", "invalid", "too_short", ...
	Msg   string `json:"message"`
}

type Options struct {
	MaxBytes              int64
	DisallowUnknownFields bool
	RequireContentType    string // e.g. "application/json"
	SuccessStatus         int
}

type errorBody struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

// JSONAction binds JSON → sanitize (optional) → validate (optional) → handler
func JSONAction[T any](
	opts Options,
	sanitize SanitizeFunc, // can be nil
	validate ValidateFunc, // can be nil
	handler func(context.Context, T) (any, error),
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Content-Type check
		if opts.RequireContentType != "" && !hasContentType(r.Header.Get("Content-Type"), opts.RequireContentType) {
			writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "content type must be application/json", nil)
			return
		}

		// 2. Max body size
		if opts.MaxBytes > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, opts.MaxBytes)
		}

		// 3. Parse JSON
		var t T
		dec := json.NewDecoder(r.Body)
		if opts.DisallowUnknownFields {
			dec.DisallowUnknownFields()
		}
		if err := dec.Decode(&t); err != nil {
			status := http.StatusBadRequest
			code := "invalid_json"
			message := "request body must contain valid JSON"
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				status = http.StatusRequestEntityTooLarge
				code = "request_too_large"
				message = "request body is too large"
			}
			writeError(w, status, code, message, nil)
			return
		}

		// A request must contain exactly one JSON value. Decode once more and
		// require EOF so trailing garbage and a second document are rejected.
		var extra json.RawMessage
		if err := dec.Decode(&extra); err != io.EOF {
			writeError(w, http.StatusBadRequest, "invalid_json", "request body must contain exactly one JSON value", nil)
			return
		}

		// 4. Sanitize (optional, in-place mutation)
		if sanitize != nil {
			sanitize(&t) // pass pointer for mutation
		}

		// 5. Validate (optional)
		if validate != nil {
			if errs := validate(&t); len(errs) > 0 {
				writeValidationError(w, errs)
				return
			}
		}

		// 6. Execute handler
		resp, err := handler(r.Context(), t)
		if err != nil {
			var publicErr PublicError
			if errors.As(err, &publicErr) {
				writeError(w, publicErr.HTTPStatus(), publicErr.PublicCode(), publicErr.PublicMessage(), nil)
				return
			}
			// Internal details belong in server logs, never in the API response.
			writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
			return
		}

		status := opts.SuccessStatus
		if status == 0 {
			status = http.StatusOK
		}
		writeJSON(w, status, resp)
	}
}

func hasContentType(value, expected string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	return err == nil && mediaType == expected
}

// writeJSON writes a JSON response with status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// writeValidationError writes a 422 with field errors.
func writeValidationError(w http.ResponseWriter, errs []FieldError) {
	writeError(w, http.StatusUnprocessableEntity, "validation_failed", "request validation failed", errs)
}

func writeError(w http.ResponseWriter, status int, code, message string, fields []FieldError) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: errorBody{
		Code:    code,
		Message: message,
		Fields:  fields,
	}})
}
