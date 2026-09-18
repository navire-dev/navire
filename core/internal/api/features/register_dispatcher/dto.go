package register_dispatcher

import (
	"github.com/navire-dev/navire/core/internal/api/bind"
	"github.com/navire-dev/navire/core/internal/dispatchers/registration"
)

type (
	Request  = registration.Request
	Response = registration.Response
)

func Validate() bind.ValidateFunc {
	return func(value any) []bind.FieldError {
		request := value.(*Request)
		issues := make([]bind.FieldError, 0, 4)
		if err := registration.ValidateDispatcherID(request.DispatcherID); err != nil {
			issues = append(issues, bind.FieldError{Field: "dispatcher_id", Code: "invalid", Msg: err.Error()})
		}
		if request.Timestamp == 0 {
			issues = append(issues, bind.FieldError{Field: "timestamp", Code: "required", Msg: "timestamp is required"})
		}
		if request.Nonce == "" {
			issues = append(issues, bind.FieldError{Field: "nonce", Code: "required", Msg: "nonce is required"})
		}
		if request.Signature == "" {
			issues = append(issues, bind.FieldError{Field: "signature", Code: "required", Msg: "signature is required"})
		}
		return issues
	}
}
