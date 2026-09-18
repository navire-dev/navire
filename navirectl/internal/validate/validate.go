package validate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/navire-dev/navire/navirectl/internal/config"
	"github.com/navire-dev/navire/navirectl/internal/runtime"
	sharedtemplate "github.com/navire-dev/navire/shared/template"
	"gopkg.in/yaml.v3"
)

func Templates(_ context.Context, rt *runtime.Context) error {
	value, ok := rt.Value(config.RuntimeConfigKey)
	if !ok {
		return errors.New("repository configuration is not loaded")
	}
	cfg, ok := value.(config.Config)
	if !ok {
		return errors.New("repository configuration has an invalid type")
	}
	rootValue, ok := rt.Value(config.RuntimeRootKey)
	if !ok {
		return errors.New("repository root is not loaded")
	}
	root, ok := rootValue.(string)
	if !ok {
		return errors.New("repository root has an invalid type")
	}

	dir := cfg.TemplatesDir(root)
	files, err := sharedtemplate.YAMLFiles(dir)
	if err != nil {
		return fmt.Errorf("read templates directory: %w", err)
	}

	checked := 0
	invalid := 0
	for _, filename := range files {
		checked++
		displayName, err := filepath.Rel(dir, filename)
		if err != nil {
			displayName = filepath.Base(filename)
		}
		data, readErr := os.ReadFile(filename)
		if readErr != nil {
			rt.Logger.Error("✗ template %s is invalid: %s", displayName, humanizeError(readErr))
			invalid++
			continue
		}
		definition, parseErr := sharedtemplate.Parse(data, displayName)
		if parseErr == nil {
			parseErr = definition.Validate()
		}
		if parseErr != nil {
			rt.Logger.Error("✗ template %s is invalid: %s", displayName, humanizeError(parseErr))
			invalid++
			continue
		}
		rt.Logger.Success("✓ template %s is valid", definition.Key)
	}

	if checked == 0 {
		return fmt.Errorf("no YAML templates found in %s", dir)
	}
	if invalid > 0 {
		rt.Logger.Error("validation failed: %d of %d template(s) invalid", invalid, checked)
		return fmt.Errorf("%d of %d template(s) failed validation", invalid, checked)
	}
	rt.Logger.Success("validation complete: %d template(s) validated", checked)
	return nil
}

func humanizeError(err error) string {
	var typeErr *yaml.TypeError
	if errors.As(err, &typeErr) {
		messages := make([]string, 0, len(typeErr.Errors))
		for _, detail := range typeErr.Errors {
			const marker = ": field "
			if line, field, ok := strings.Cut(detail, marker); ok {
				field = strings.TrimSuffix(field, " not found in type template.Definition")
				messages = append(messages, fmt.Sprintf("unknown field %q (%s)", field, line))
				continue
			}
			messages = append(messages, detail)
		}
		return strings.Join(messages, "; ")
	}

	message := err.Error()
	if index := strings.Index(message, ": "); index >= 0 {
		message = message[index+2:]
	}
	return strings.TrimPrefix(message, "yaml: ")
}
