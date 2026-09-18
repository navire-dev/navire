package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	Filename         = ".config.json"
	CurrentSchema    = 1
	RuntimeConfigKey = "navirectl.config"
	RuntimeRootKey   = "navirectl.repository_root"
)

type Config struct {
	SchemaVersion int    `json:"schema_version"`
	TemplatesPath string `json:"templates_path"`
	ProvidersPath string `json:"providers_path"`
}

func Load(root string) (Config, error) {
	data, err := os.ReadFile(filepath.Join(root, Filename))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("navire repository is not initialized; run %q", "navirectl init")
		}
		return Config{}, fmt.Errorf("read %s: %w", Filename, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", Filename, err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate %s: %w", Filename, err)
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.SchemaVersion != CurrentSchema {
		return fmt.Errorf("unsupported schema version %d", c.SchemaVersion)
	}
	if err := validateRelativePath("templates", c.TemplatesPath); err != nil {
		return err
	}
	return validateRelativePath("providers", c.ProvidersPath)
}

func (c Config) TemplatesDir(root string) string {
	return filepath.Join(root, c.TemplatesPath)
}

func (c Config) ProvidersDir(root string) string {
	return filepath.Join(root, c.ProvidersPath)
}

func validateRelativePath(label, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s path is required", label)
	}
	if filepath.IsAbs(value) {
		return fmt.Errorf("%s path must be relative to the repository", label)
	}
	cleaned := filepath.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s path must stay inside the repository", label)
	}
	return nil
}
