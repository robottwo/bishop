package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robottwo/bishop/internal/environment"
	"github.com/robottwo/bishop/internal/wizard"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

// homeDir returns the user's home directory, using os.UserHomeDir() for portability
// across different platforms (including Windows where HOME is not typically set).
func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fall back to HOME env var if os.UserHomeDir() fails
		return os.Getenv("HOME")
	}
	return home
}

var (
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	// Full-screen box styles (matching ctrl-r history search)
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red for errors
	savedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // Green for success
)

// sessionConfigOverrides stores config values set via the UI that should override shell variables
// This prevents user's bash scripts from resetting values we just set
var sessionConfigOverrides = make(map[string]string)

// GetSessionOverride returns a session config override if one exists
func GetSessionOverride(key string) (string, bool) {
	val, ok := sessionConfigOverrides[key]
	return val, ok
}

type model struct {
	runner        *interp.Runner
	list          list.Model
	submenuList   list.Model
	selectionList list.Model
	state         state
	items         []settingItem
	textInput     textinput.Model
	activeSetting *settingItem
	activeSubmenu *menuItem
	quitting      bool
	width         int
	height        int
	errorMsg      string // Temporary error message to display
	savedMsg      string // Temporary saved confirmation message
}

type state int

const (
	stateList state = iota
	stateSubmenu
	stateEditing
	stateSelection
)

// menuItem represents a top-level menu entry (may have submenu)
type menuItem struct {
	title       string
	description string
	submenu     []settingItem // nil if this is a direct setting
	setting     *settingItem  // non-nil if this is a direct setting (no submenu)
}

func (m menuItem) Title() string       { return m.title }
func (m menuItem) Description() string { return m.description }
func (m menuItem) FilterValue() string { return m.title }

type settingItem struct {
	title       string
	description string
	envVar      string
	itemType    settingType
	options     []string // For list type
}

type settingType int

const (
	typeText settingType = iota
	typeList
	typeToggle
)

func (s settingItem) Title() string       { return s.title }
func (s settingItem) Description() string { return s.description }
func (s settingItem) FilterValue() string { return s.title }

// simpleItem implements list.Item
type simpleItem string

func (s simpleItem) Title() string       { return string(s) }
func (s simpleItem) Description() string { return "" }
func (s simpleItem) FilterValue() string { return string(s) }

func initialModel(runner *interp.Runner) model {
	// Define submenu items for slow model (chat/agent)
	slowModelSettings := []settingItem{
		{
			title:       "Provider",
			description: "LLM provider to use",
			envVar:      "BISH_SLOW_MODEL_PROVIDER",
			itemType:    typeList,
			options:     []string{"ollama", "openai", "openrouter"},
		},
		{
			title:       "API Key",
			description: "API key for the provider",
			envVar:      "BISH_SLOW_MODEL_API_KEY",
			itemType:    typeText,
		},
		{
			title:       "Model ID",
			description: "Model identifier (e.g., qwen2.5:32b)",
			envVar:      "BISH_SLOW_MODEL_ID",
			itemType:    typeText,
		},
		{
			title:       "Base URL",
			description: "API endpoint URL (optional override)",
			envVar:      "BISH_SLOW_MODEL_BASE_URL",
			itemType:    typeText,
		},
	}

	// Define submenu items for fast model (completion/suggestions)
	fastModelSettings := []settingItem{
		{
			title:       "Provider",
			description: "LLM provider to use",
			envVar:      "BISH_FAST_MODEL_PROVIDER",
			itemType:    typeList,
			options:     []string{"ollama", "openai", "openrouter"},
		},
		{
			title:       "API Key",
			description: "API key for the provider",
			envVar:      "BISH_FAST_MODEL_API_KEY",
			itemType:    typeText,
		},
		{
			title:       "Model ID",
			description: "Model identifier (e.g., qwen2.5)",
			envVar:      "BISH_FAST_MODEL_ID",
			itemType:    typeText,
		},
		{
			title:       "Base URL",
			description: "API endpoint URL (optional override)",
			envVar:      "BISH_FAST_MODEL_BASE_URL",
			itemType:    typeText,
		},
	}

	// Direct settings (no submenu)
	assistantHeightSetting := settingItem{
		title:       "Assistant Height",
		description: "Height of the bottom assistant box",
		envVar:      "BISH_ASSISTANT_HEIGHT",
		itemType:    typeText,
	}
	safetyChecksSetting := settingItem{
		title:       "Safety Checks",
		description: "Enable/Disable approved command checks (session only)",
		envVar:      "BISH_AGENT_APPROVED_BASH_COMMAND_REGEX",
		itemType:    typeToggle,
	}
	defaultToYesSetting := settingItem{
		title:       "Default to Yes",
		description: "Prompts default to Yes when Enter is pressed",
		envVar:      "BISH_DEFAULT_TO_YES",
		itemType:    typeToggle,
	}

	// Top-level menu items
	items := []list.Item{
		menuItem{
			title:       "Configure Slow Model",
			description: "Chat and agent operations",
			submenu:     slowModelSettings,
		},
		menuItem{
			title:       "Configure Fast Model",
			description: "Auto-completion and suggestions",
			submenu:     fastModelSettings,
		},
		menuItem{
			title:       "Assistant Height",
			description: "Height of the bottom assistant box",
			setting:     &assistantHeightSetting,
		},
		menuItem{
			title:       "Safety Checks",
			description: "Enable/Disable approved command checks (session only)",
			setting:     &safetyChecksSetting,
		},
		menuItem{
			title:       "Default to Yes",
			description: "Prompts default to Yes when Enter is pressed",
			setting:     &defaultToYesSetting,
		},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedItemStyle
	delegate.Styles.SelectedDesc = selectedItemStyle.Foreground(lipgloss.Color("240"))

	l := list.New(items, delegate, 0, 0)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)
	l.SetShowHelp(false)

	subL := list.New([]list.Item{}, delegate, 0, 0)
	subL.SetShowStatusBar(false)
	subL.SetFilteringEnabled(false)
	subL.SetShowTitle(false)
	subL.SetShowHelp(false)

	selL := list.New([]list.Item{}, delegate, 0, 0)
	selL.SetShowStatusBar(false)
	selL.SetFilteringEnabled(false)
	selL.SetShowTitle(false)
	selL.SetShowHelp(false)

	ti := textinput.New()
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	ti.Focus()

	return model{
		runner:        runner,
		list:          l,
		submenuList:   subL,
		selectionList: selL,
		state:         stateList,
		items:         []settingItem{},
		textInput:     ti,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height)
		m.submenuList.SetWidth(msg.Width)
		m.submenuList.SetHeight(msg.Height)
		m.selectionList.SetWidth(msg.Width)
		m.selectionList.SetHeight(msg.Height)

	case tea.KeyMsg:
		// Clear any previous messages on new key press
		m.errorMsg = ""
		m.savedMsg = ""

		// Handle text editing state
		if m.state == stateEditing {
			// Check for quit keys first, before delegating to text input
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			}
			switch msg.Type {
			case tea.KeyEsc:
				if m.activeSubmenu != nil {
					m.state = stateSubmenu
				} else {
					m.state = stateList
				}
				return m, nil
			case tea.KeyEnter:
				newValue := m.textInput.Value()
				savedPath, err := saveConfig(m.activeSetting.envVar, newValue, m.runner)
				if err != nil {
					m.errorMsg = fmt.Sprintf("Failed to save %s: %v", m.activeSetting.envVar, err)
					return m, nil
				}
				m.savedMsg = fmt.Sprintf("Saved to %s", savedPath)
				if m.activeSubmenu != nil {
					m.state = stateSubmenu
				} else {
					m.state = stateList
				}
				return m, nil
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		// Handle selection list state
		if m.state == stateSelection {
			// Check for quit keys first, before delegating to selection list
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			}
			switch msg.Type {
			case tea.KeyEsc:
				if m.activeSubmenu != nil {
					m.state = stateSubmenu
				} else {
					m.state = stateList
				}
				return m, nil
			case tea.KeyEnter:
				if i, ok := m.selectionList.SelectedItem().(simpleItem); ok {
					newValue := string(i)
					savedPath, err := saveConfig(m.activeSetting.envVar, newValue, m.runner)
					if err != nil {
						m.errorMsg = fmt.Sprintf("Failed to save %s: %v", m.activeSetting.envVar, err)
						return m, nil
					}
					m.savedMsg = fmt.Sprintf("Saved to %s", savedPath)
					if m.activeSubmenu != nil {
						m.state = stateSubmenu
					} else {
						m.state = stateList
					}
					return m, nil
				}
			}
			m.selectionList, cmd = m.selectionList.Update(msg)
			return m, cmd
		}

		// Handle submenu state
		if m.state == stateSubmenu {
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
			case "esc":
				m.activeSubmenu = nil
				m.state = stateList
				return m, nil
			case "enter":
				if i, ok := m.submenuList.SelectedItem().(settingItem); ok {
					m.activeSetting = &i
					return m, m.handleSettingAction(&i)
				}
			}
			m.submenuList, cmd = m.submenuList.Update(msg)
			return m, cmd
		}

		// Handle main list state
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(menuItem); ok {
				// If this menu item has a submenu, navigate to it
				if item.submenu != nil {
					m.activeSubmenu = &item
					subItems := make([]list.Item, len(item.submenu))
					for idx, s := range item.submenu {
						subItems[idx] = s
					}
					m.submenuList.SetItems(subItems)
					m.submenuList.Title = item.title
					m.state = stateSubmenu
					return m, nil
				}
				// If this is a direct setting, handle it
				if item.setting != nil {
					m.activeSetting = item.setting
					return m, m.handleSettingAction(item.setting)
				}
			}
		}
	}

	if m.state == stateList {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleSettingAction processes the action for a setting item
func (m *model) handleSettingAction(s *settingItem) tea.Cmd {
	if s.itemType == typeToggle {
		curr := getEnv(m.runner, s.envVar)
		var newVal string
		if s.envVar == "BISH_AGENT_APPROVED_BASH_COMMAND_REGEX" {
			if strings.Contains(curr, `".*"`) || strings.Contains(curr, `".+"`) {
				newVal = "[]"
			} else {
				newVal = `[".*"]`
			}
		} else {
			// Handle both "true"/"false" and "1"/"0" formats
			if curr == "true" || curr == "1" {
				newVal = "false"
			} else {
				newVal = "true"
			}
		}
		savedPath, err := saveConfig(s.envVar, newVal, m.runner)
		if err != nil {
			m.errorMsg = fmt.Sprintf("Failed to save %s: %v", s.envVar, err)
		} else if savedPath == "" {
			// Session-only setting (like safety checks)
			m.savedMsg = "Saved (session only)"
		} else {
			m.savedMsg = fmt.Sprintf("Saved to %s", savedPath)
		}
		return nil
	}

	if s.itemType == typeList {
		items := make([]list.Item, len(s.options))
		for idx, opt := range s.options {
			items[idx] = simpleItem(opt)
		}
		m.selectionList.SetItems(items)
		m.selectionList.Title = "Select " + s.title
		m.state = stateSelection
		return nil
	}

	// typeText
	m.textInput.SetValue(getEnv(m.runner, s.envVar))
	m.state = stateEditing
	return nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	// Calculate available dimensions for content
	// Leave room for header (1), footer (1), and border (2)
	availableHeight := m.height - 4
	if availableHeight < 5 {
		availableHeight = 5
	}
	availableWidth := m.width - 4
	if availableWidth < 20 {
		availableWidth = 20
	}

	var content strings.Builder
	var title string
	var helpText string

	switch m.state {
	case stateEditing:
		title = fmt.Sprintf("Edit %s", m.activeSetting.title)
		helpText = "Enter: Save | Esc: Cancel | q: Quit"
		content.WriteString("\n" + m.textInput.View() + "\n")
	case stateSelection:
		title = "Select " + m.activeSetting.title
		helpText = "↑/↓: Navigate | Enter: Select | Esc: Back | q: Quit"
		content.WriteString(m.selectionList.View())
	case stateSubmenu:
		title = m.activeSubmenu.title
		helpText = "↑/↓: Navigate | Enter: Edit | Esc: Back | q: Quit"
		// Update submenu descriptions with current values
		items := m.submenuList.Items()
		for i, item := range items {
			if s, ok := item.(settingItem); ok {
				val := getEnv(m.runner, s.envVar)
				if val == "" {
					val = "(not set)"
				}
				s.description = fmt.Sprintf("Current: %s", val)
				items[i] = s
			}
		}
		m.submenuList.SetItems(items)
		content.WriteString(m.submenuList.View())
	default:
		title = "Config Menu"
		helpText = "↑/↓: Navigate | Enter: Select | q: Quit"
		// Update main menu descriptions with current values for direct settings
		items := m.list.Items()
		for i, item := range items {
			if mi, ok := item.(menuItem); ok {
				if mi.setting != nil {
					val := getEnv(m.runner, mi.setting.envVar)
					switch mi.setting.envVar {
					case "BISH_AGENT_APPROVED_BASH_COMMAND_REGEX":
						if strings.Contains(val, `".*"`) || strings.Contains(val, `".+"`) {
							val = "Disabled for this session"
						} else {
							val = "Enabled"
						}
					case "BISH_DEFAULT_TO_YES":
						if val == "1" || val == "true" {
							val = "Yes (prompts show [Y/n])"
						} else {
							val = "No (prompts show [y/N])"
						}
					}
					if val == "" {
						val = "(not set)"
					}
					mi.description = fmt.Sprintf("Current: %s", val)
					items[i] = mi
				}
			}
		}
		m.list.SetItems(items)
		content.WriteString(m.list.View())
	}

	// Build the full-screen box content
	var boxContent strings.Builder

	// Header with centered title
	titlePadding := (availableWidth - len(title)) / 2
	if titlePadding < 0 {
		titlePadding = 0
	}
	centeredTitle := strings.Repeat(" ", titlePadding) + title
	boxContent.WriteString(headerStyle.Render(centeredTitle) + "\n")

	// Content area - truncate to available height
	contentLines := strings.Split(content.String(), "\n")
	contentHeight := availableHeight - 2 // Leave room for header and footer
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
	}
	boxContent.WriteString(strings.Join(contentLines, "\n"))

	// Pad to fill available height
	currentLines := len(contentLines) + 1 // +1 for header
	for i := currentLines; i < availableHeight-1; i++ {
		boxContent.WriteString("\n")
	}

	// Footer with help text and status messages
	footerContent := helpStyle.Render(helpText)
	if m.errorMsg != "" {
		footerContent = errorStyle.Render(m.errorMsg) + "\n" + footerContent
	} else if m.savedMsg != "" {
		footerContent = savedStyle.Render(m.savedMsg) + "\n" + footerContent
	}
	boxContent.WriteString("\n" + footerContent)

	// Render in a box with rounded border (matching ctrl-r style)
	boxStyle := lipgloss.NewStyle().
		Width(availableWidth).
		Height(availableHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	return boxStyle.Render(boxContent.String())
}

// RunConfigUI launches the interactive configuration UI
func RunConfigUI(runner *interp.Runner) error {
	p := tea.NewProgram(initialModel(runner), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func getEnv(runner *interp.Runner, key string) string {
	// Safety Checks uses a session-only flag BISH_SAFETY_CHECKS_DISABLED
	if key == "BISH_AGENT_APPROVED_BASH_COMMAND_REGEX" {
		if runner.Vars["BISH_SAFETY_CHECKS_DISABLED"].String() == "true" {
			return `[".*"]` // Disabled for this session
		}
		return "[]" // Enabled (default)
	}

	// Check session overrides first (for settings modified via config UI)
	if val, ok := sessionConfigOverrides[key]; ok {
		return val
	}

	if v, ok := runner.Vars[key]; ok {
		return v.String()
	}
	return ""
}

func saveConfig(key, value string, runner *interp.Runner) (savedPath string, err error) {
	// Handle Safety Checks specially - only affects current session, not persisted
	// Uses BISH_SAFETY_CHECKS_DISABLED flag which is checked in GetApprovedBashCommandRegex
	if key == "BISH_AGENT_APPROVED_BASH_COMMAND_REGEX" {
		// value is either '[".*"]' (disabled) or '[]' (enabled)
		if strings.Contains(value, `".*"`) || strings.Contains(value, `".+"`) {
			// Disable safety checks for this session only
			runner.Vars["BISH_SAFETY_CHECKS_DISABLED"] = expand.Variable{
				Exported: true,
				Kind:     expand.String,
				Str:      "true",
			}
		} else {
			// Enable safety checks - remove the session flag
			delete(runner.Vars, "BISH_SAFETY_CHECKS_DISABLED")
		}
		// Don't persist this setting - it only affects the current session
		return "", nil
	}

	// Validate input before saving
	if err := environment.ValidateConfigValue(key, value); err != nil {
		return "", err
	}

	// For other settings, update current session
	runner.Vars[key] = expand.Variable{
		Exported: true,
		Kind:     expand.String,
		Str:      value,
	}

	// Store in session overrides to prevent bash scripts from resetting the value
	sessionConfigOverrides[key] = value

	// Sync to environment so changes take effect immediately
	environment.SyncVariableToEnv(runner, key)

	// Persist to file for future sessions (deduplicating entries)
	configPath := filepath.Join(homeDir(), ".config", "bish", "config_ui")
	configDir := filepath.Dir(configPath)

	// Acquire exclusive lock on a lock file to prevent concurrent writes
	lockPath := configPath + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to open lock file: %w", err)
	}
	defer func() {
		_ = flockUnlock(lockFile.Fd())
		_ = lockFile.Close()
	}()

	if err := flockExclusive(lockFile.Fd()); err != nil {
		return "", fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Read existing config entries while holding lock
	configEntries := make(map[string]string)
	var orderedKeys []string

	if content, err := os.ReadFile(configPath); err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Parse "export KEY='value'" or "export KEY=value"
			if strings.HasPrefix(line, "export ") {
				rest := strings.TrimPrefix(line, "export ")
				if idx := strings.Index(rest, "="); idx > 0 {
					k := rest[:idx]
					if _, exists := configEntries[k]; !exists {
						orderedKeys = append(orderedKeys, k)
					}
					// Extract value (handle quoted values)
					v := rest[idx+1:]
					v = strings.Trim(v, "'\"")
					configEntries[k] = v
				}
			}
		}
	}

	// Update or add the new key
	if _, exists := configEntries[key]; !exists {
		orderedKeys = append(orderedKeys, key)
	}
	configEntries[key] = value

	// Build new config content
	var buf strings.Builder
	for _, k := range orderedKeys {
		v := configEntries[k]
		safeValue := strings.ReplaceAll(v, "'", "'\\''")
		buf.WriteString(fmt.Sprintf("export %s='%s'\n", k, safeValue))
	}

	// Atomic write: write to temp file, fsync, rename over original
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(configDir, "config_ui.*.tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on any error
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.WriteString(buf.String()); err != nil {
		_ = tmpFile.Close()
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return "", fmt.Errorf("failed to fsync temp file: %w", err)
	}

	if err := tmpFile.Chmod(0600); err != nil {
		_ = tmpFile.Close()
		return "", fmt.Errorf("failed to set permissions on temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, configPath); err != nil {
		return "", fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Fsync the directory to ensure rename is persisted
	if dir, err := os.Open(configDir); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}

	success = true

	if err := wizard.EnsureBishrcConfigured(); err != nil {
		return "", fmt.Errorf("failed to ensure .bishrc configuration: %w", err)
	}

	return configPath, nil
}
