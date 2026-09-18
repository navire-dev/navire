package catalog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	messageTemplate "github.com/navire-dev/navire/shared/template"
)

type Issue struct {
	File string
	Err  error
}

type Catalog struct {
	dir         string
	mu          sync.RWMutex
	definitions map[string]*messageTemplate.Definition
}

func NewCatalog(dir string) *Catalog {
	return &Catalog{dir: dir, definitions: make(map[string]*messageTemplate.Definition)}
}

func (c *Catalog) Reload() ([]Issue, error) {
	files, err := filesIn(c.dir)
	if err != nil {
		return nil, err
	}
	loaded := make(map[string]*messageTemplate.Definition, len(files))
	issues := make([]Issue, 0)
	for _, path := range files {
		definition, err := loadFile(path)
		if err != nil {
			issues = append(issues, Issue{File: filepath.Base(path), Err: err})
			continue
		}
		if _, exists := loaded[definition.Key]; exists {
			issues = append(issues, Issue{File: filepath.Base(path), Err: fmt.Errorf("duplicate template key %q", definition.Key)})
			continue
		}
		loaded[definition.Key] = definition
	}
	c.mu.Lock()
	c.definitions = loaded
	c.mu.Unlock()
	return issues, nil
}

func (c *Catalog) LoadByKey(key string) (*messageTemplate.Definition, error) {
	c.mu.RLock()
	definition, ok := c.definitions[key]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("template %q not found", key)
	}
	return definition, nil
}

func (c *Catalog) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.definitions)
}

func (c *Catalog) Watch(ctx context.Context, callback func([]Issue, error)) error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return fmt.Errorf("create templates directory: %w", err)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create template watcher: %w", err)
	}
	defer func() { _ = watcher.Close() }()
	if err := watchDirectories(watcher, c.dir); err != nil {
		return fmt.Errorf("watch templates directories: %w", err)
	}

	const debounce = 300 * time.Millisecond
	var timer *time.Timer
	var reload <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			stopTimer(timer)
			return nil
		case <-reload:
			reloadCatalog(c, callback)
			reload = nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			timer, reload = handleWatchEvent(watcher, event, callback, timer, reload, debounce)
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			callback(nil, fmt.Errorf("template watcher: %w", err))
		}
	}
}

func stopTimer(timer *time.Timer) {
	if timer != nil {
		timer.Stop()
	}
}

func reloadCatalog(c *Catalog, callback func([]Issue, error)) {
	issues, err := c.Reload()
	callback(issues, err)
}

func handleWatchEvent(
	watcher *fsnotify.Watcher,
	event fsnotify.Event,
	callback func([]Issue, error),
	timer *time.Timer,
	reload <-chan time.Time,
	debounce time.Duration,
) (*time.Timer, <-chan time.Time) {
	if event.Op&fsnotify.Create != 0 {
		watchCreatedDirectory(watcher, event.Name, callback)
	}
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) != 0 {
		timer, reload = scheduleReload(timer, debounce)
	}
	return timer, reload
}

func watchCreatedDirectory(watcher *fsnotify.Watcher, path string, callback func([]Issue, error)) {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return
	}
	if err := watchDirectories(watcher, path); err != nil {
		callback(nil, fmt.Errorf("watch templates directories: %w", err))
	}
}

func scheduleReload(timer *time.Timer, debounce time.Duration) (*time.Timer, <-chan time.Time) {
	stopTimer(timer)
	timer = time.NewTimer(debounce)
	return timer, timer.C
}

func watchDirectories(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if err := watcher.Add(path); err != nil {
			return err
		}
		return nil
	})
}

func LoadByKey(dir, key string) (*messageTemplate.Definition, error) {
	files, err := filesIn(dir)
	if err != nil {
		return nil, err
	}
	for _, path := range files {
		definition, err := loadFile(path)
		if err != nil {
			return nil, err
		}
		if definition.Key == key {
			return definition, nil
		}
	}
	return nil, fmt.Errorf("template %q not found", key)
}

func loadFile(path string) (*messageTemplate.Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read template %s: %w", filepath.Base(path), err)
	}
	definition, err := messageTemplate.Parse(data, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	if err := definition.Validate(); err != nil {
		return nil, fmt.Errorf("validate template %s: %w", filepath.Base(path), err)
	}
	return definition, nil
}

func filesIn(dir string) ([]string, error) {
	return messageTemplate.YAMLFiles(dir)
}
