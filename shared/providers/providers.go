// Package providers contains the provider catalog shared by Core and DSPC.
// Provider implementations remain owned by DSPC; this package only defines
// their stable identifiers and activation policy.
package providers

import (
	"fmt"
	"strings"
)

type ID string

const (
	Gotify  ID = "gotify"
	Discord ID = "discord"
	Ntfy    ID = "ntfy"
	Slack   ID = "slack"
)

type Definition struct {
	ID     ID
	Active bool
}

var catalog = []Definition{
	{ID: Gotify, Active: true},
	{ID: Discord, Active: true},
	{ID: Ntfy, Active: true},
	{ID: Slack, Active: true},
}

func Lookup(slug string) (Definition, bool) {
	for _, definition := range catalog {
		if string(definition.ID) == slug {
			return definition, true
		}
	}
	return Definition{}, false
}

func Validate(slug string) error {
	if _, ok := Lookup(slug); !ok {
		return fmt.Errorf("invalid provider type %q (known types: %s)", slug, strings.Join(slugs(), ", "))
	}
	return nil
}

func IsActive(slug string) bool {
	definition, ok := Lookup(slug)
	return ok && definition.Active
}

// ActiveIDs returns the provider IDs enabled for the whole Navire installation.
// Core and DSPC use this shared list to build their local implementation registries.
func ActiveIDs() []ID {
	result := make([]ID, 0, len(catalog))
	for _, definition := range catalog {
		if definition.Active {
			result = append(result, definition.ID)
		}
	}
	return result
}

func ValidateActive(slug string) error {
	definition, ok := Lookup(slug)
	if !ok {
		return fmt.Errorf("invalid provider type %q (known types: %s)", slug, strings.Join(slugs(), ", "))
	}
	if !definition.Active {
		return fmt.Errorf("provider type %q is currently disabled", slug)
	}
	return nil
}

func slugs() []string {
	result := make([]string, 0, len(catalog))
	for _, definition := range catalog {
		result = append(result, string(definition.ID))
	}
	return result
}
