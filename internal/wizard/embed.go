package wizard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "embed"
)

//go:embed bishrc.template
var bishrcTemplate []byte

// BishrcTemplate returns the default ~/.bishrc template content.
// Used by both the setup wizard and the config UI when creating a fresh .bishrc.
func BishrcTemplate() []byte {
	return bishrcTemplate
}

// EnsureBishrcConfigured ensures that ~/.bishrc exists and sources config_ui.
// For fresh installs, writes the full template. For existing files, appends the source line.
func EnsureBishrcConfigured() error {
	gshrcPath := filepath.Join(homeDir(), ".bishrc")

	content, err := os.ReadFile(gshrcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", gshrcPath, err)
	}

	if err == nil && strings.Contains(string(content), "config/bish/config_ui") {
		return nil
	}

	if os.IsNotExist(err) {
		if writeErr := os.WriteFile(gshrcPath, bishrcTemplate, 0644); writeErr != nil {
			return fmt.Errorf("failed to create %s: %w", gshrcPath, writeErr)
		}
		return nil
	}

	sourceSnippet := "\n# Source UI configuration\n[ -f ~/.config/bish/config_ui ] && source ~/.config/bish/config_ui\n"
	f, err := os.OpenFile(gshrcPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", gshrcPath, err)
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
