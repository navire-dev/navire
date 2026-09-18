package init

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/navire-dev/navire/navirectl/internal/config"
	"github.com/navire-dev/navire/navirectl/internal/runtime"
)

// New initializes a repository-local Navire configuration file.
func New(_ context.Context, rt *runtime.Context) error {
	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	reader, writer := promptIO(rt)
	prompts := bufio.NewReader(reader)
	configPath := filepath.Join(root, config.Filename)
	if err := confirmConfigOverwrite(prompts, writer, configPath); err != nil {
		return err
	}

	repositoryConfig, err := promptConfig(prompts, writer)
	if err != nil {
		return err
	}
	if err := repositoryConfig.Validate(); err != nil {
		return fmt.Errorf("validate repository configuration: %w", err)
	}

	if err := confirmCreation(prompts, writer, repositoryConfig); err != nil {
		return err
	}
	if err := prepareDirectories(prompts, writer, root, repositoryConfig); err != nil {
		return err
	}
	if err := writeConfig(root, configPath, repositoryConfig); err != nil {
		return err
	}

	rt.Logger.Success("initialized %s", config.Filename)
	return nil
}

func repositoryRoot() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve current directory: %w", err)
	}
	return root, nil
}

func confirmConfigOverwrite(reader *bufio.Reader, writer io.Writer, path string) error {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check %s: %w", config.Filename, err)
	}
	confirmed, err := promptConfirm(reader, writer, fmt.Sprintf("Overwrite %s?", config.Filename), false)
	if err != nil {
		return fmt.Errorf("confirm configuration overwrite: %w", err)
	}
	if !confirmed {
		return fmt.Errorf("%s already exists", config.Filename)
	}
	return nil
}

func promptConfig(reader *bufio.Reader, writer io.Writer) (config.Config, error) {
	templatesPath, err := promptPath(reader, writer, "templates", "templates")
	if err != nil {
		return config.Config{}, fmt.Errorf("read templates path: %w", err)
	}
	providersPath, err := promptPath(reader, writer, "providers", "providers")
	if err != nil {
		return config.Config{}, fmt.Errorf("read providers path: %w", err)
	}
	return config.Config{
		SchemaVersion: config.CurrentSchema,
		TemplatesPath: templatesPath,
		ProvidersPath: providersPath,
	}, nil
}

func confirmCreation(reader *bufio.Reader, writer io.Writer, cfg config.Config) error {
	if err := printSummary(writer, cfg); err != nil {
		return fmt.Errorf("write configuration summary: %w", err)
	}
	confirmed, err := promptConfirm(reader, writer, "Create repository configuration?", true)
	if err != nil {
		return fmt.Errorf("confirm repository configuration: %w", err)
	}
	if !confirmed {
		return errors.New("initialization canceled")
	}
	return nil
}

func prepareDirectories(reader *bufio.Reader, writer io.Writer, root string, cfg config.Config) error {
	if err := ensureDirectory(reader, writer, root, "templates", cfg.TemplatesPath); err != nil {
		return err
	}
	return ensureDirectory(reader, writer, root, "providers", cfg.ProvidersPath)
}

func writeConfig(root, path string, cfg config.Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", config.Filename, err)
	}
	data = append(data, '\n')
	if err := writeAtomically(path, data); err != nil {
		return fmt.Errorf("write %s: %w", config.Filename, err)
	}
	if _, err := config.Load(root); err != nil {
		return fmt.Errorf("validate written %s: %w", config.Filename, err)
	}
	return nil
}

func promptIO(rt *runtime.Context) (io.Reader, io.Writer) {
	reader := rt.Reader
	if reader == nil {
		reader = os.Stdin
	}
	writer := rt.Writer
	if writer == nil {
		writer = os.Stdout
	}
	return reader, writer
}

func printSummary(writer io.Writer, cfg config.Config) error {
	_, err := fmt.Fprintf(writer, "\nRepository configuration:\n  templates: %s/\n  providers: %s/\n\n", cfg.TemplatesPath, cfg.ProvidersPath)
	return err
}

func ensureDirectory(reader *bufio.Reader, writer io.Writer, root, label, relativePath string) error {
	path := filepath.Join(root, relativePath)
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s path %s is not a directory", label, relativePath)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check %s path: %w", label, err)
	}

	confirmed, err := promptConfirm(reader, writer, fmt.Sprintf("Create %s directory %s?", label, relativePath), true)
	if err != nil {
		return fmt.Errorf("confirm %s directory creation: %w", label, err)
	}
	if !confirmed {
		return fmt.Errorf("%s directory %s does not exist", label, relativePath)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create %s directory: %w", label, err)
	}
	return nil
}

func promptConfirm(reader *bufio.Reader, writer io.Writer, message string, defaultValue bool) (bool, error) {
	choice := "y/N"
	if defaultValue {
		choice = "Y/n"
	}
	if _, err := fmt.Fprintf(writer, "%s [%s]: ", message, choice); err != nil {
		return false, err
	}
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return defaultValue, nil
	}
	switch value {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("expected yes or no")
	}
}

func writeAtomically(path string, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(path), ".config-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()
	if err := temp.Chmod(0o644); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func promptPath(reader *bufio.Reader, writer io.Writer, label, defaultPath string) (string, error) {
	if _, err := fmt.Fprintf(writer, "%s path [%s]: ", label, defaultPath); err != nil {
		return "", err
	}
	value, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultPath, nil
	}
	if filepath.IsAbs(value) {
		return "", fmt.Errorf("%s path must be relative to the repository", label)
	}
	cleaned := filepath.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s path must stay inside the repository", label)
	}
	return cleaned, nil
}
