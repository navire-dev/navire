package ingest

import (
	"testing"

	"github.com/fsnotify/fsnotify"
)

func TestIsReloadEvent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		event fsnotify.Event
		want  bool
	}{
		{name: "yaml created", event: fsnotify.Event{Name: "gotify.yml", Op: fsnotify.Create}, want: true},
		{name: "yaml written", event: fsnotify.Event{Name: "gotify.yaml", Op: fsnotify.Write}, want: true},
		{name: "yaml renamed", event: fsnotify.Event{Name: "gotify.yml", Op: fsnotify.Rename}, want: true},
		{name: "yaml removed", event: fsnotify.Event{Name: "gotify.yml", Op: fsnotify.Remove}, want: true},
		{name: "chmod ignored", event: fsnotify.Event{Name: "gotify.yml", Op: fsnotify.Chmod}, want: false},
		{name: "non yaml ignored", event: fsnotify.Event{Name: "README.md", Op: fsnotify.Write}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isReloadEvent(tt.event); got != tt.want {
				t.Fatalf("isReloadEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}
