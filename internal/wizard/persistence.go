package wizard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// configUIPath returns the path to the UI-generated config file.
func configUIPath() string {
	return filepath.Join(homeDir(), ".config", "bish", "config_ui")
}

// wizardManagedKeys are the env var names the wizard writes.
var wizardManagedKeys = map[string]bool{
	"BISH_FAST_MODEL_PROVIDER": true,
	"BISH_FAST_MODEL_API_KEY":  true,
	"BISH_FAST_MODEL_BASE_URL": true,
	"BISH_FAST_MODEL_ID":       true,
	"BISH_SLOW_MODEL_PROVIDER": true,
	"BISH_SLOW_MODEL_API_KEY":  true,
	"BISH_SLOW_MODEL_BASE_URL": true,
	"BISH_SLOW_MODEL_ID":       true,
}

// extractExportKey returns the variable name from a line like
// "export FOO='bar'" or "", false if the line is not an export.
func extractExportKey(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "export ") {
		return "", false
	}
	rest := strings.TrimPrefix(trimmed, "export ")
	if idx := strings.IndexByte(rest, '='); idx > 0 {
		return rest[:idx], true
	}
	return "", false
}

// shellEscape escapes a value for safe inclusion inside single quotes.
// The standard POSIX technique: replace ' with '\”.
func shellEscape(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

func saveConfigToFile(config wizardConfig) error {
	configPath := configUIPath()
	configDir := filepath.Dir(configPath)

	newEntries := make(map[string]string)

	if config.fastModel.provider != "" {
		newEntries["BISH_FAST_MODEL_PROVIDER"] = fmt.Sprintf("export BISH_FAST_MODEL_PROVIDER='%s'", shellEscape(config.fastModel.provider))
	}
	if config.fastModel.apiKey != "" {
		newEntries["BISH_FAST_MODEL_API_KEY"] = fmt.Sprintf("export BISH_FAST_MODEL_API_KEY='%s'", shellEscape(config.fastModel.apiKey))
	}
	if config.fastModel.baseURL != "" {
		newEntries["BISH_FAST_MODEL_BASE_URL"] = fmt.Sprintf("export BISH_FAST_MODEL_BASE_URL='%s'", shellEscape(config.fastModel.baseURL))
	}
	if config.fastModel.modelID != "" {
		newEntries["BISH_FAST_MODEL_ID"] = fmt.Sprintf("export BISH_FAST_MODEL_ID='%s'", shellEscape(config.fastModel.modelID))
	}

	if config.slowModel.provider != "" {
		newEntries["BISH_SLOW_MODEL_PROVIDER"] = fmt.Sprintf("export BISH_SLOW_MODEL_PROVIDER='%s'", shellEscape(config.slowModel.provider))
	}
	if config.slowModel.apiKey != "" {
		newEntries["BISH_SLOW_MODEL_API_KEY"] = fmt.Sprintf("export BISH_SLOW_MODEL_API_KEY='%s'", shellEscape(config.slowModel.apiKey))
	}
	if config.slowModel.baseURL != "" {
		newEntries["BISH_SLOW_MODEL_BASE_URL"] = fmt.Sprintf("export BISH_SLOW_MODEL_BASE_URL='%s'", shellEscape(config.slowModel.baseURL))
	}
	if config.slowModel.modelID != "" {
		newEntries["BISH_SLOW_MODEL_ID"] = fmt.Sprintf("export BISH_SLOW_MODEL_ID='%s'", shellEscape(config.slowModel.modelID))
	}

	if len(newEntries) == 0 {
		return fmt.Errorf("no configuration to save")
	}

	// Read existing config and preserve non-wizard lines
	var preserved []string
	if existing, err := os.ReadFile(configPath); err == nil {
		for _, line := range strings.Split(string(existing), "\n") {
			if key, ok := extractExportKey(line); ok && wizardManagedKeys[key] {
				continue // drop old wizard entries; they'll be replaced
			}
			preserved = append(preserved, line)
		}
	}

	// Build output: preserved lines first, then new wizard entries
	var buf strings.Builder
	for _, line := range preserved {
		if strings.TrimSpace(line) == "" {
			continue
		}
		buf.WriteString(line + "\n")
	}
	for _, key := range []string{
		"BISH_FAST_MODEL_PROVIDER", "BISH_FAST_MODEL_API_KEY",
		"BISH_FAST_MODEL_BASE_URL", "BISH_FAST_MODEL_ID",
		"BISH_SLOW_MODEL_PROVIDER", "BISH_SLOW_MODEL_API_KEY",
		"BISH_SLOW_MODEL_BASE_URL", "BISH_SLOW_MODEL_ID",
	} {
		if entry, ok := newEntries[key]; ok {
			buf.WriteString(entry + "\n")
		}
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(configDir, "config_ui.*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.WriteString(buf.String()); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tmpFile.Chmod(0600); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, configPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	// Rename succeeded; temp path no longer exists, nothing to clean up.
	success = true

	// Sync the directory to ensure the rename is durable on disk.
	if dir, err := os.Open(configDir); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}

	return EnsureBishrcConfigured()
}
