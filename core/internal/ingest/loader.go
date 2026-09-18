package ingest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/navire-dev/navire/shared/providers"
	"gopkg.in/yaml.v3"
)

// LoadTargetsFile parses a single targets/*.yml file and resolves environment variables.
func LoadTargetsFile(path string) (*TargetsFile, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Resolve env vars (${VAR} → os.Getenv("VAR"))
	resolved := os.ExpandEnv(string(data))

	// Parse YAML
	var tf TargetsFile
	if err := yaml.Unmarshal([]byte(resolved), &tf); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	// Validate
	if err := tf.Validate(); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	return &tf, nil
}

// LoadAllTargets scans a directory and loads all targets/*.yml and *.yaml files.
// Returns a map[full_name]Endpoint where full_name = {type}_{purpose}.
// Example: "gotify_prod", "slack_cicd", "discord_warnings"
func LoadAllTargets(dir string) (map[string]Endpoint, error) {
	return loadAllTargets(dir, true)
}

func loadAllTargets(dir string, requireEndpoint bool) (map[string]Endpoint, error) {
	files, err := targetFiles(dir)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		if requireEndpoint {
			return nil, fmt.Errorf("no targets files found in %s", dir)
		}
		return map[string]Endpoint{}, nil
	}

	endpoints := make(map[string]Endpoint)

	for _, file := range files {
		tf, err := LoadTargetsFile(file)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", filepath.Base(file), err)
		}
		if !providers.IsActive(string(tf.Provider)) {
			continue
		}

		// Skip if the provider is disabled globally.
		if !tf.Enabled {
			continue // Skip all endpoints in this file
		}

		// Process each endpoint
		for _, ep := range tf.Endpoints {
			// Skip if endpoint disabled
			if !ep.Enabled {
				continue
			}

			// Auto-prefix: full_name = {provider}_{endpoint}
			// Example: gotify + prod → "gotify_prod"
			fullName := string(tf.Provider) + "_" + ep.Key

			// Check collision (duplicate full_name)
			if existing, exists := endpoints[fullName]; exists {
				return nil, fmt.Errorf(
					"duplicate endpoint '%s' found:\n"+
						"  - First:  provider=%s, key=%s\n"+
						"  - Second: provider=%s, key=%s, file=%s",
					fullName,
					existing.Provider, existing.Key,
					string(tf.Provider), ep.Key, filepath.Base(file))
			}

			// Set runtime fields
			ep.Provider = tf.Provider
			ep.FullName = fullName

			// Store endpoint
			endpoints[fullName] = ep
		}
	}

	if len(endpoints) == 0 {
		if requireEndpoint {
			return nil, fmt.Errorf("no enabled endpoints found in %s", dir)
		}
		return endpoints, nil
	}

	return endpoints, nil
}

func targetFiles(dir string) ([]string, error) {
	var files []string
	for _, extension := range []string{"*.yml", "*.yaml"} {
		matches, err := filepath.Glob(filepath.Join(dir, extension))
		if err != nil {
			return nil, fmt.Errorf("glob %s: %w", extension, err)
		}
		files = append(files, matches...)
	}
	return files, nil
}
