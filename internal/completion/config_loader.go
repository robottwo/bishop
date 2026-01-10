package completion

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigLoader handles loading completion data from embedded YAML files
type ConfigLoader struct {
	fs fs.FS
}

// NewConfigLoader creates a new ConfigLoader with the given filesystem
func NewConfigLoader(filesystem fs.FS) *ConfigLoader {
	return &ConfigLoader{
		fs: filesystem,
	}
}

// LoadAllCompletions loads all completion configurations from embedded YAML files
// Returns a map of command names to their completion entries
func (cl *ConfigLoader) LoadAllCompletions() (map[string][]UserCompletion, error) {
	completions := make(map[string][]UserCompletion)

	// Walk through all files in the embedded filesystem
	err := fs.WalkDir(cl.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-YAML files
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		// Read and parse the YAML file
		data, err := fs.ReadFile(cl.fs, path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		var config UserCompletionConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		// Merge commands from this file into the result map
		for command, entries := range config.Commands {
			completions[command] = entries
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load completions: %w", err)
	}

	return completions, nil
}

// LoadCompletionsFromFile loads completions from a single embedded YAML file
// This is useful for loading specific configuration files on demand
func (cl *ConfigLoader) LoadCompletionsFromFile(filename string) (map[string][]UserCompletion, error) {
	// Construct the path - assuming files are in data/ directory
	path := filepath.Join("data", filename)

	// Read the file
	data, err := fs.ReadFile(cl.fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	// Parse the YAML
	var config UserCompletionConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	return config.Commands, nil
}

// ListEmbeddedFiles returns a list of all YAML files in the embedded filesystem
// This can be useful for debugging or introspection
func (cl *ConfigLoader) ListEmbeddedFiles() ([]string, error) {
	var files []string

	err := fs.WalkDir(cl.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && (strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return files, nil
}
