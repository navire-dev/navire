package ingest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/navire-dev/navire/shared/logger"
)

const reloadDebounce = 300 * time.Millisecond

type reloadFunc func(context.Context) error

type debounceTimer struct {
	timer  *time.Timer
	active bool
}

func (d *debounceTimer) channel() <-chan time.Time {
	if !d.active {
		return nil
	}
	return d.timer.C
}

func (d *debounceTimer) reset() {
	if d.timer == nil {
		d.timer = time.NewTimer(reloadDebounce)
		d.active = true
		return
	}

	if !d.timer.Stop() {
		select {
		case <-d.timer.C:
		default:
		}
	}
	d.timer.Reset(reloadDebounce)
	d.active = true
}

func (d *debounceTimer) stop() {
	if d.timer != nil {
		d.timer.Stop()
	}
	d.active = false
}

// Watcher watches a targets directory and reconciles the complete directory.
type Watcher struct {
	targetsDir string
	reload     reloadFunc
	log        logger.Logger
}

func NewWatcher(targetsDir string, reload reloadFunc, log logger.Logger) *Watcher {
	if log == nil {
		log = logger.New("info", true)
	}

	return &Watcher{targetsDir: targetsDir, reload: reload, log: log}
}

// Start watches until ctx is cancelled. Bursts of filesystem events produce a
// single complete reconciliation, which either succeeds or leaves Redis intact.
func (w *Watcher) Start(ctx context.Context) error {
	watcher, err := w.open()
	if err != nil {
		return err
	}
	defer w.close(watcher)

	w.log.Info("watching targets directory for changes", logger.String("dir", w.targetsDir))
	return w.loop(ctx, watcher)
}

func (w *Watcher) open() (*fsnotify.Watcher, error) {
	if err := os.MkdirAll(w.targetsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create targets directory: %w", err)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create watcher: %w", err)
	}
	if err := watcher.Add(w.targetsDir); err != nil {
		_ = watcher.Close()
		return nil, fmt.Errorf("watch directory: %w", err)
	}
	return watcher, nil
}

func (w *Watcher) close(watcher *fsnotify.Watcher) {
	if err := watcher.Close(); err != nil {
		w.log.Error("failed to close watcher", logger.Error(err))
	}
}

func (w *Watcher) loop(ctx context.Context, watcher *fsnotify.Watcher) error {
	var debounce debounceTimer
	defer func() {
		debounce.stop()
	}()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			w.handleEvent(event, &debounce)

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			w.handleError(err)

		case <-debounce.channel():
			debounce.active = false
			w.reloadTargets(ctx)

		case <-ctx.Done():
			w.log.Info("watcher stopped")
			return ctx.Err()
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event, debounce *debounceTimer) {
	if !isReloadEvent(event) {
		return
	}

	w.log.Debug("targets change detected",
		logger.String("file", filepath.Base(event.Name)),
		logger.String("operation", event.Op.String()),
	)
	debounce.reset()
}

func (w *Watcher) handleError(err error) {
	w.log.Error("watcher error", logger.Error(err))
}

func (w *Watcher) reloadTargets(ctx context.Context) {
	if err := w.reload(ctx); err != nil {
		w.log.Error("targets reload rejected; keeping last valid configuration", logger.Error(err))
		return
	}
	w.log.Info("targets configuration reloaded")
}

func isReloadEvent(event fsnotify.Event) bool {
	ext := filepath.Ext(event.Name)
	if ext != ".yml" && ext != ".yaml" {
		return false
	}
	return event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename|fsnotify.Remove) != 0
}
