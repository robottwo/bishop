package completion

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/robottwo/bishop/pkg/shellinput"
	"gopkg.in/yaml.v3"
)

// StaticCompleter handles static word lists for common commands
type StaticCompleter struct {
	completions map[string][]shellinput.CompletionCandidate
	mu          sync.RWMutex
}

// UserCompletionConfig represents user-defined completion configuration
type UserCompletionConfig struct {
	Commands map[string][]UserCompletion `yaml:"commands" json:"commands"`
}

// UserCompletion represents a single user-defined completion entry
type UserCompletion struct {
	Value       string `yaml:"value" json:"value"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

func NewStaticCompleter() *StaticCompleter {
	sc := &StaticCompleter{
		completions: make(map[string][]shellinput.CompletionCandidate),
	}
	sc.registerDefaults()
	sc.loadUserCompletions()
	return sc
}

func (s *StaticCompleter) registerDefaults() {
	// Load completions from embedded YAML configuration files
	loader := NewConfigLoader(CompletionData)
	completions, err := loader.LoadAllCompletions()
	if err != nil {
		// Log error but don't fail - allow the application to continue
		// Users can still add custom completions via config files
		return
	}

	// Register all loaded completions
	for command, userCompletions := range completions {
		s.RegisterUserCommand(command, userCompletions)
	}
}

// RegisterUserCommand allows users to register custom command completions at runtime
func (s *StaticCompleter) RegisterUserCommand(command string, subcommands []UserCompletion) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var candidates []shellinput.CompletionCandidate
	for _, sub := range subcommands {
		candidates = append(candidates, shellinput.CompletionCandidate{
			Value:       sub.Value,
			Description: sub.Description,
		})
	}
	s.completions[command] = candidates
}

// loadUserCompletions loads user-defined completions from config files
func (s *StaticCompleter) loadUserCompletions() {
	// Check for config in standard locations
	configPaths := getUserCompletionConfigPaths()

	for _, configPath := range configPaths {
		if _, err := os.Stat(configPath); err == nil {
			if err := s.loadCompletionsFromFile(configPath); err == nil {
				break // Successfully loaded from this path
			}
		}
	}
}

// getUserCompletionConfigPaths returns the paths to check for user completion config
func getUserCompletionConfigPaths() []string {
	var paths []string

	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		paths = append(paths, filepath.Join(xdgConfig, "bish", "completions.yaml"))
		paths = append(paths, filepath.Join(xdgConfig, "bish", "completions.json"))
	}

	// Then check home directory
	if home := os.Getenv("HOME"); home != "" {
		paths = append(paths, filepath.Join(home, ".config", "bish", "completions.yaml"))
		paths = append(paths, filepath.Join(home, ".config", "bish", "completions.json"))
		// Also check direct home directory location
		paths = append(paths, filepath.Join(home, ".bish_completions.yaml"))
		paths = append(paths, filepath.Join(home, ".bish_completions.json"))
	}

	return paths
}

// loadCompletionsFromFile loads completions from a YAML or JSON file
func (s *StaticCompleter) loadCompletionsFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var config UserCompletionConfig

	// Try YAML first (also handles JSON since YAML is a superset)
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return err
		}
	} else if strings.HasSuffix(path, ".json") {
		if err := json.Unmarshal(data, &config); err != nil {
			return err
		}
	} else {
		// Try YAML, then JSON
		if err := yaml.Unmarshal(data, &config); err != nil {
			if err := json.Unmarshal(data, &config); err != nil {
				return err
			}
		}
	}

	// Register user-defined completions
	for command, completions := range config.Commands {
		s.RegisterUserCommand(command, completions)
	}

	return nil
}

// ReloadUserCompletions reloads user-defined completions from config files
func (s *StaticCompleter) ReloadUserCompletions() {
	s.loadUserCompletions()
}

// GetCompletions returns completion suggestions for a command
func (s *StaticCompleter) GetCompletions(command string, args []string) []shellinput.CompletionCandidate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Only provide completion for the first argument (subcommand)
	if len(args) == 0 {
		if candidates, ok := s.completions[command]; ok {
			return candidates
		}
	}
	// Filter by prefix
	if len(args) == 1 {
		prefix := args[0]
		if candidates, ok := s.completions[command]; ok {
			var filtered []shellinput.CompletionCandidate
			for _, c := range candidates {
				if len(c.Value) >= len(prefix) && strings.HasPrefix(c.Value, prefix) {
					filtered = append(filtered, c)
				}
			}
			return filtered
		}
	}
	return nil
}

// GetRegisteredCommands returns a sorted list of all commands that have static completions
func (s *StaticCompleter) GetRegisteredCommands() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	commands := make([]string, 0, len(s.completions))
	for cmd := range s.completions {
		commands = append(commands, cmd)
	}
	sort.Strings(commands)
	return commands
}

// HasCommand returns true if the command has registered completions
func (s *StaticCompleter) HasCommand(command string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.completions[command]
	return ok
}
