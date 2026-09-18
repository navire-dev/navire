package models

import (
	"fmt"
	"strings"

	sharedproviders "github.com/navire-dev/navire/shared/providers"
	sharedvalidation "github.com/navire-dev/navire/shared/validation"
)

type AuthType string

const (
	AuthNone        AuthType = "none"
	AuthQuery       AuthType = "query"
	AuthHeader      AuthType = "header"
	AuthBearer      AuthType = "bearer"
	AuthPathSegment AuthType = "path_segment"
)

type AuthParam string

// Endpoint is the provider-neutral runtime representation of a notification
// destination. The Core may populate it from configuration; the Dispatcher
// consumes it when executing a delivery.
type Endpoint struct {
	Key         string             `yaml:"key"`
	Name        string             `yaml:"name,omitempty"`
	Description string             `yaml:"description,omitempty"`
	Enabled     bool               `yaml:"enabled"`
	URL         string             `yaml:"url"`
	Auth        Auth               `yaml:"auth"`
	Provider    sharedproviders.ID `yaml:"-" json:"provider"`
	FullName    string             `yaml:"-" json:"full_name"`
}

type Auth struct {
	Type  AuthType  `yaml:"type"`
	Param AuthParam `yaml:"param"`
	Value string    `yaml:"value"`
}

type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityNormal   Priority = "normal"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

type Rendered struct {
	Title    string
	Body     string
	Priority Priority
}

func (ep *Endpoint) Validate() error {
	if !sharedvalidation.IsKey(ep.Key) {
		return fmt.Errorf("key must contain 1 to %d lowercase letters, digits, '-' or '_' and start with a letter or digit", sharedvalidation.MaxKeyLength)
	}
	if ep.URL == "" {
		return fmt.Errorf("url is required")
	}
	if err := sharedvalidation.ValidateHTTPURL(ep.URL); err != nil {
		return err
	}
	return ep.Auth.Validate()
}

func (a *Auth) Validate() error {
	validTypes := []AuthType{AuthQuery, AuthHeader, AuthBearer, AuthPathSegment, AuthNone}
	valid := false
	for _, typ := range validTypes {
		if a.Type == typ {
			valid = true
			break
		}
	}
	if !valid {
		names := make([]string, len(validTypes))
		for i, typ := range validTypes {
			names[i] = string(typ)
		}
		return fmt.Errorf("invalid type %q (must be one of: %s)", a.Type, strings.Join(names, ", "))
	}
	if (a.Type == AuthQuery || a.Type == AuthHeader) && a.Param == "" {
		return fmt.Errorf("param is required for auth type %q", a.Type)
	}
	if a.Type != AuthNone && a.Value == "" {
		return fmt.Errorf("value is required for auth type %q", a.Type)
	}
	return nil
}
