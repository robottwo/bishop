package wizard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

func saveConfigToFile(config wizardConfig) error {
	configPath := filepath.Join(homeDir(), ".bish_config_ui")
	configDir := filepath.Dir(configPath)

	newEntries := make(map[string]string)

	if config.fastModel.provider != "" {
		newEntries["BISH_FAST_MODEL_PROVIDER"] = fmt.Sprintf("export BISH_FAST_MODEL_PROVIDER='%s'", config.fastModel.provider)
	}
	if config.fastModel.apiKey != "" {
		safeKey := strings.ReplaceAll(config.fastModel.apiKey, "'", "'\\''")
		newEntries["BISH_FAST_MODEL_API_KEY"] = fmt.Sprintf("export BISH_FAST_MODEL_API_KEY='%s'", safeKey)
	}
	if config.fastModel.baseURL != "" {
		newEntries["BISH_FAST_MODEL_BASE_URL"] = fmt.Sprintf("export BISH_FAST_MODEL_BASE_URL='%s'", config.fastModel.baseURL)
	}
	if config.fastModel.modelID != "" {
		newEntries["BISH_FAST_MODEL_ID"] = fmt.Sprintf("export BISH_FAST_MODEL_ID='%s'", config.fastModel.modelID)
	}

	if config.slowModel.provider != "" {
		newEntries["BISH_SLOW_MODEL_PROVIDER"] = fmt.Sprintf("export BISH_SLOW_MODEL_PROVIDER='%s'", config.slowModel.provider)
	}
	if config.slowModel.apiKey != "" {
		safeKey := strings.ReplaceAll(config.slowModel.apiKey, "'", "'\\''")
		newEntries["BISH_SLOW_MODEL_API_KEY"] = fmt.Sprintf("export BISH_SLOW_MODEL_API_KEY='%s'", safeKey)
	}
	if config.slowModel.baseURL != "" {
		newEntries["BISH_SLOW_MODEL_BASE_URL"] = fmt.Sprintf("export BISH_SLOW_MODEL_BASE_URL='%s'", config.slowModel.baseURL)
	}
	if config.slowModel.modelID != "" {
		newEntries["BISH_SLOW_MODEL_ID"] = fmt.Sprintf("export BISH_SLOW_MODEL_ID='%s'", config.slowModel.modelID)
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

	tmpFile, err := os.CreateTemp(configDir, ".bish_config_ui.*.tmp")
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

	if dir, err := os.Open(configDir); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}

	success = true

	gshrcPath := filepath.Join(homeDir(), ".bishrc")
	sourceSnippet := "\n# Source UI configuration\n[ -f ~/.bish_config_ui ] && source ~/.bish_config_ui\n"

	content, err := os.ReadFile(gshrcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", gshrcPath, err)
	}

	if err == nil && strings.Contains(string(content), ".bish_config_ui") {
		return nil
	}

	var f *os.File
	if os.IsNotExist(err) {
		f, err = os.Create(gshrcPath)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", gshrcPath, err)
		}
	} else {
		f, err = os.OpenFile(gshrcPath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", gshrcPath, err)
		}
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if _, err := f.WriteString(sourceSnippet); err != nil {
		return fmt.Errorf("failed to write to %s: %w", gshrcPath, err)
	}

	return nil
}
