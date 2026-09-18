package bind

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testRequest struct {
	Name string `json:"name"`
}

func testHandler() func(context.Context, testRequest) (any, error) {
	return func(_ context.Context, req testRequest) (any, error) {
		return map[string]string{"name": req.Name}, nil
	}
}

func TestJSONActionAcceptsContentTypeParameters(t *testing.T) {
	h := JSONAction(Options{
		RequireContentType:    "application/json",
		DisallowUnknownFields: true,
	}, nil, nil, testHandler())

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"navire"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestJSONActionRejectsSecondJSONValue(t *testing.T) {
	h := JSONAction(Options{RequireContentType: "application/json"}, nil, nil, testHandler())
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"one"}{"name":"two"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if !strings.Contains(rec.Body.String(), `"code":"invalid_json"`) {
		t.Fatalf("response = %s, want invalid_json", rec.Body.String())
	}
}

func TestJSONActionRejectsUnknownFields(t *testing.T) {
	h := JSONAction(Options{
		RequireContentType:    "application/json",
		DisallowUnknownFields: true,
	}, nil, nil, testHandler())
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"navire","extra":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestJSONActionDoesNotExposeHandlerError(t *testing.T) {
	h := JSONAction(Options{RequireContentType: "application/json"}, nil, nil,
		func(context.Context, testRequest) (any, error) {
			return nil, errTestInternal
		})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"navire"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if strings.Contains(rec.Body.String(), errTestInternal.Error()) {
		t.Fatalf("response leaked internal error: %s", rec.Body.String())
	}
}

func TestJSONActionWritesTypedPublicError(t *testing.T) {
	h := JSONAction(Options{RequireContentType: "application/json"}, nil, nil,
		func(context.Context, testRequest) (any, error) {
			return nil, publicTestError{}
		})
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"navire"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if !strings.Contains(rec.Body.String(), `"code":"template_not_found"`) || strings.Contains(rec.Body.String(), "database password") {
		t.Fatalf("response = %s, want public error without internal details", rec.Body.String())
	}
}

var errTestInternal = testError("database password leaked")

type testError string

func (e testError) Error() string { return string(e) }

type publicTestError struct{}

func (publicTestError) Error() string         { return "database password" }
func (publicTestError) HTTPStatus() int       { return http.StatusNotFound }
func (publicTestError) PublicCode() string    { return "template_not_found" }
func (publicTestError) PublicMessage() string { return "template not found" }
