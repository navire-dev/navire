package apperrors

import "net/http"

type Error struct {
	Status  int
	Code    string
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Message
	}
	return e.Message + ": " + e.Cause.Error()
}

func (e *Error) Unwrap() error { return e.Cause }

func (e *Error) HTTPStatus() int       { return e.Status }
func (e *Error) PublicCode() string    { return e.Code }
func (e *Error) PublicMessage() string { return e.Message }

func NotFound(code, message string, cause error) error {
	return &Error{Status: http.StatusNotFound, Code: code, Message: message, Cause: cause}
}

func Unprocessable(code, message string, cause error) error {
	return &Error{Status: http.StatusUnprocessableEntity, Code: code, Message: message, Cause: cause}
}

func Unavailable(code, message string, cause error) error {
	return &Error{Status: http.StatusServiceUnavailable, Code: code, Message: message, Cause: cause}
}
