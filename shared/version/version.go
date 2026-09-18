package version

import (
	"runtime"
	"time"
)

var (
	// These values are compile-time metadata and may be replaced with ldflags.
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
	GoVersion = runtime.Version()
)

// formatMap translates simple tokens into Go's reference time layout.
var formatMap = map[string]string{
	"YYYY": "2006",
	"MM":   "01",
	"DD":   "02",
	"hh":   "15",
	"mm":   "04",
	"ss":   "05",
	"TZ":   "MST",
}

// HumanBuildDate reformats an RFC3339 timestamp into a human-readable string.
func HumanBuildDate(rfc3339 string, loc *time.Location, format string) string {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return rfc3339
	}
	if loc == nil {
		loc = time.Local
	}

	layout := format
	for token, goLayout := range formatMap {
		layout = replaceAll(layout, token, goLayout)
	}

	return t.In(loc).Format(layout)
}

func replaceAll(s, old, replacement string) string {
	for {
		i := find(s, old)
		if i < 0 {
			break
		}
		s = s[:i] + replacement + s[i+len(old):]
	}
	return s
}

func find(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
