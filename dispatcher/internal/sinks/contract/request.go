package contract

import "net/http"

// PreparedRequest is the provider-specific HTTP representation of a
// notification. Providers build it; sender owns execution and retries.
type PreparedRequest struct {
	Provider string
	Method   string
	URL      string
	Headers  http.Header
	Body     []byte
}
