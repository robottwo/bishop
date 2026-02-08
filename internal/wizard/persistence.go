package wizard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func saveConfigToFile(config wizardConfig) error {
	configPath := filepath.Join(homeDir(), ".bish_config_ui")
	configDir := filepath.Dir(configPath)

	var configEntries []string

	if config.fastModel.provider != "" {
		configEntries = append(configEntries, fmt.Sprintf("export BISH_FAST_MODEL_PROVIDER='%s'", config.fastModel.provider))
	}
	if config.fastModel.apiKey != "" {
		safeKey := strings.ReplaceAll(config.fastModel.apiKey, "'", "'\\''")
		configEntries = append(configEntries, fmt.Sprintf("export BISH_FAST_MODEL_API_KEY='%s'", safeKey))
	}
	if config.fastModel.baseURL != "" {
		configEntries = append(configEntries, fmt.Sprintf("export BISH_FAST_MODEL_BASE_URL='%s'", config.fastModel.baseURL))
	}
	if config.fastModel.modelID != "" {
		configEntries = append(configEntries, fmt.Sprintf("export BISH_FAST_MODEL_ID='%s'", config.fastModel.modelID))
	}

	if config.slowModel.provider != "" {
		configEntries = append(configEntries, fmt.Sprintf("export BISH_SLOW_MODEL_PROVIDER='%s'", config.slowModel.provider))
	}
	if config.slowModel.apiKey != "" {
		safeKey := strings.ReplaceAll(config.slowModel.apiKey, "'", "'\\''")
		configEntries = append(configEntries, fmt.Sprintf("export BISH_SLOW_MODEL_API_KEY='%s'", safeKey))
	}
	if config.slowModel.baseURL != "" {
		configEntries = append(configEntries, fmt.Sprintf("export BISH_SLOW_MODEL_BASE_URL='%s'", config.slowModel.baseURL))
	}
	if config.slowModel.modelID != "" {
		configEntries = append(configEntries, fmt.Sprintf("export BISH_SLOW_MODEL_ID='%s'", config.slowModel.modelID))
	}

	if len(configEntries) == 0 {
		return fmt.Errorf("no configuration to save")
	}

	var buf strings.Builder
	for _, entry := range configEntries {
		buf.WriteString(entry + "\n")
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
