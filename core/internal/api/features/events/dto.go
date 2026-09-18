package events

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/navire-dev/navire/core/internal/api/bind"
	"github.com/navire-dev/navire/core/internal/validation"
)

type EventRequest struct {
	ID             string         `json:"id,omitempty"`
	TenantID       string         `json:"tenantID,omitempty" validate:"omitempty,max=80"`
	Priority       string         `json:"priority,omitempty" validate:"omitempty,oneof=low normal high critical"`
	IdempotencyKey string         `json:"idempotency_key,omitempty" validate:"omitempty,max=128"`
	Timestamp      time.Time      `json:"timestamp"`
	TemplateKey    string         `json:"key" validate:"required,max=50,navirekey"`
	Variant        string         `json:"variant,omitempty" validate:"omitempty,max=80,identifier"`
	Data           map[string]any `json:"data"`
}

func (r *EventRequest) validate(lim Limits) []bind.FieldError {
	issues := append(validation.Struct(r), validation.Data(r.Data, validation.DataLimits{
		MaxKeys:     lim.MaxDataKeys,
		MaxKeyLen:   lim.MaxKeyLen,
		MaxValueLen: lim.MaxStrValLen,
	})...)
	errs := make([]bind.FieldError, 0, len(issues))
	for _, issue := range issues {
		errs = append(errs, bind.FieldError{Field: issue.Field, Code: issue.Code, Msg: issue.Message})
	}
	return errs
}

func (r *EventRequest) EnsureIdempotencyKey() {
	if r.IdempotencyKey != "" {
		return
	}
	b, _ := json.Marshal(struct {
		TemplateKey string
		Variant     string
		Data        map[string]any
		Priority    string
	}{r.TemplateKey, r.Variant, r.Data, r.Priority})
	sum := sha256.Sum256(b)
	r.IdempotencyKey = hex.EncodeToString(sum[:])
}
