// Package validation contains validation helpers shared by Navire components.
package validation

import (
	"regexp"
	"unicode/utf8"
)

const MaxKeyLength = 50

var keyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func IsKey(value string) bool {
	return utf8.RuneCountInString(value) <= MaxKeyLength && keyPattern.MatchString(value)
}
