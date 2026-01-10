package completion

import (
	"io/fs"
	"strings"
	"testing"
)

// TestEmbedFS_NotEmpty verifies that the CompletionData embed.FS contains files
func TestEmbedFS_NotEmpty(t *testing.T) {
	entries, err := fs.ReadDir(CompletionData, "data")
	if err != nil {
		t.Fatalf("Failed to read embedded data directory: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("Expected embedded data directory to contain files, but it was empty")
	}

	// Count YAML files
	yamlCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml")) {
			yamlCount++
		}
	}

	if yamlCount == 0 {
		t.Fatal("Expected embedded data directory to contain YAML files, but found none")
	}

	t.Logf("Found %d YAML files in embedded data", yamlCount)
}

// TestEmbedFS_ExpectedFiles verifies that all expected YAML files are embedded
func TestEmbedFS_ExpectedFiles(t *testing.T) {
	expectedFiles := []string{
		"data/containers.yaml",
		"data/package_managers.yaml",
		"data/cloud.yaml",
		"data/databases.yaml",
		"data/developer_tools.yaml",
	}

	for _, expectedFile := range expectedFiles {
		_, err := fs.Stat(CompletionData, expectedFile)
		if err != nil {
			t.Errorf("Expected file %s not found in embedded filesystem: %v", expectedFile, err)
		}
	}
}

// TestEmbedFS_FilesReadable verifies that embedded files can be read
func TestEmbedFS_FilesReadable(t *testing.T) {
	entries, err := fs.ReadDir(CompletionData, "data")
	if err != nil {
		t.Fatalf("Failed to read embedded data directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		path := "data/" + entry.Name()
		data, err := fs.ReadFile(CompletionData, path)
		if err != nil {
			t.Errorf("Failed to read embedded file %s: %v", path, err)
			continue
		}

		if len(data) == 0 {
			t.Errorf("Embedded file %s is empty", path)
		}
	}
}

// TestEmbedFS_YAMLStructure verifies that embedded YAML files have valid structure
func TestEmbedFS_YAMLStructure(t *testing.T) {
	entries, err := fs.ReadDir(CompletionData, "data")
	if err != nil {
		t.Fatalf("Failed to read embedded data directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		path := "data/" + entry.Name()
		data, err := fs.ReadFile(CompletionData, path)
		if err != nil {
			t.Errorf("Failed to read embedded file %s: %v", path, err)
			continue
		}

		// Check that file starts with "commands:" which is the expected root key
		content := string(data)
		if !strings.Contains(content, "commands:") {
			t.Errorf("Embedded file %s does not contain 'commands:' key", path)
		}

		// Check that file contains "value:" and "description:" entries
		if !strings.Contains(content, "value:") {
			t.Errorf("Embedded file %s does not contain any 'value:' entries", path)
		}
		if !strings.Contains(content, "description:") {
			t.Errorf("Embedded file %s does not contain any 'description:' entries", path)
		}
	}
}

// TestEmbedFS_WithConfigLoader verifies that ConfigLoader can load from embedded FS
func TestEmbedFS_WithConfigLoader(t *testing.T) {
	loader := NewConfigLoader(CompletionData)
	if loader == nil {
		t.Fatal("NewConfigLoader returned nil")
	}

	completions, err := loader.LoadAllCompletions()
	if err != nil {
		t.Fatalf("Failed to load completions from embedded FS: %v", err)
	}

	if len(completions) == 0 {
		t.Fatal("Expected completions to be loaded from embedded FS, but got none")
	}

	t.Logf("Successfully loaded %d commands from embedded YAML files", len(completions))
}

// TestEmbedFS_ExpectedCommands verifies that expected commands are present in embedded data
func TestEmbedFS_ExpectedCommands(t *testing.T) {
	loader := NewConfigLoader(CompletionData)
	completions, err := loader.LoadAllCompletions()
	if err != nil {
		t.Fatalf("Failed to load completions: %v", err)
	}

	// Expected commands from each category
	expectedCommands := map[string][]string{
		"docker":         {"run", "build", "ps", "images", "pull", "push"},
		"kubectl":        {"get", "apply", "create", "delete"},
		"npm":            {"install", "run", "test", "build"},
		"yarn":           {"add", "install", "remove"},
		"go":             {"build", "run", "test", "mod"},
		"cargo":          {"build", "run", "test", "new"},
		"aws":            {"s3", "ec2", "lambda"},
		"terraform":      {"init", "plan", "apply", "destroy"},
		"psql":           {"\\c", "\\l", "\\dt"},
		"gh":             {"pr", "issue", "repo", "auth"},
		"systemctl":      {"start", "stop", "restart", "status"},
	}

	for command, expectedSubcmds := range expectedCommands {
		subcommands, exists := completions[command]
		if !exists {
			t.Errorf("Expected command %q not found in embedded completions", command)
			continue
		}

		if len(subcommands) == 0 {
			t.Errorf("Command %q has no subcommands", command)
			continue
		}

		// Check that at least some expected subcommands are present
		for _, expectedSubcmd := range expectedSubcmds {
			found := false
			for _, subcmd := range subcommands {
				if subcmd.Value == expectedSubcmd {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected subcommand %q not found in %q completions", expectedSubcmd, command)
			}
		}
	}
}

// TestEmbedFS_CommandDescriptions verifies that commands have descriptions
func TestEmbedFS_CommandDescriptions(t *testing.T) {
	loader := NewConfigLoader(CompletionData)
	completions, err := loader.LoadAllCompletions()
	if err != nil {
		t.Fatalf("Failed to load completions: %v", err)
	}

	commandsChecked := 0
	commandsWithoutDesc := 0

	for command, subcommands := range completions {
		for _, subcmd := range subcommands {
			commandsChecked++
			if subcmd.Description == "" {
				commandsWithoutDesc++
				t.Logf("Warning: %s %s has no description", command, subcmd.Value)
			}
		}
	}

	// Allow some commands without descriptions, but the majority should have them
	if commandsWithoutDesc > commandsChecked/10 {
		t.Errorf("Too many commands without descriptions: %d out of %d", commandsWithoutDesc, commandsChecked)
	}

	t.Logf("Checked %d commands, %d without descriptions", commandsChecked, commandsWithoutDesc)
}

// TestEmbedFS_IntegrationWithStaticCompleter verifies full integration with StaticCompleter
func TestEmbedFS_IntegrationWithStaticCompleter(t *testing.T) {
	// Create a new StaticCompleter which should load from embedded YAML files
	sc := NewStaticCompleter()
	if sc == nil {
		t.Fatal("NewStaticCompleter returned nil")
	}

	// Verify that commands are registered
	registeredCommands := sc.GetRegisteredCommands()
	if len(registeredCommands) == 0 {
		t.Fatal("Expected registered commands from embedded YAML, but got none")
	}

	t.Logf("StaticCompleter has %d registered commands from embedded YAML", len(registeredCommands))

	// Verify specific commands work
	testCases := []struct {
		command      string
		expectedCmds []string
	}{
		{
			command:      "docker",
			expectedCmds: []string{"run", "build", "ps"},
		},
		{
			command:      "kubectl",
			expectedCmds: []string{"get", "apply", "create"},
		},
		{
			command:      "npm",
			expectedCmds: []string{"install", "run", "test"},
		},
	}

	for _, tc := range testCases {
		completions := sc.GetCompletions(tc.command, nil)
		if len(completions) == 0 {
			t.Errorf("Expected completions for %q, got none", tc.command)
			continue
		}

		for _, expected := range tc.expectedCmds {
			found := false
			for _, c := range completions {
				if c.Value == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected %q to have subcommand %q from embedded YAML", tc.command, expected)
			}
		}
	}
}

// TestEmbedFS_LoadSpecificFile verifies loading a specific embedded file
func TestEmbedFS_LoadSpecificFile(t *testing.T) {
	loader := NewConfigLoader(CompletionData)

	testCases := []struct {
		filename        string
		expectedCommand string
	}{
		{
			filename:        "containers.yaml",
			expectedCommand: "docker",
		},
		{
			filename:        "package_managers.yaml",
			expectedCommand: "npm",
		},
		{
			filename:        "cloud.yaml",
			expectedCommand: "terraform",
		},
		{
			filename:        "databases.yaml",
			expectedCommand: "psql",
		},
		{
			filename:        "developer_tools.yaml",
			expectedCommand: "gh",
		},
	}

	for _, tc := range testCases {
		completions, err := loader.LoadCompletionsFromFile(tc.filename)
		if err != nil {
			t.Errorf("Failed to load %s: %v", tc.filename, err)
			continue
		}

		if len(completions) == 0 {
			t.Errorf("Expected completions from %s, got none", tc.filename)
			continue
		}

		if _, exists := completions[tc.expectedCommand]; !exists {
			t.Errorf("Expected command %q not found in %s", tc.expectedCommand, tc.filename)
		}
	}
}

// TestEmbedFS_ListEmbeddedFiles verifies ListEmbeddedFiles works with real embedded data
func TestEmbedFS_ListEmbeddedFiles(t *testing.T) {
	loader := NewConfigLoader(CompletionData)

	files, err := loader.ListEmbeddedFiles()
	if err != nil {
		t.Fatalf("Failed to list embedded files: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("Expected embedded files, got none")
	}

	// Verify that files have .yaml or .yml extension
	for _, file := range files {
		if !strings.HasSuffix(file, ".yaml") && !strings.HasSuffix(file, ".yml") {
			t.Errorf("Unexpected file extension: %s", file)
		}
	}

	t.Logf("Found %d embedded files: %v", len(files), files)
}

// TestEmbedFS_NoDataLoss verifies that the total number of completions is reasonable
func TestEmbedFS_NoDataLoss(t *testing.T) {
	loader := NewConfigLoader(CompletionData)
	completions, err := loader.LoadAllCompletions()
	if err != nil {
		t.Fatalf("Failed to load completions: %v", err)
	}

	// Count total completion entries
	totalEntries := 0
	for _, subcommands := range completions {
		totalEntries += len(subcommands)
	}

	// We know from the build progress notes that we should have:
	// containers.yaml: 199 entries
	// package_managers.yaml: 282 entries
	// cloud.yaml: 165 entries
	// databases.yaml: 166 entries
	// developer_tools.yaml: 318 entries
	// Total: ~1130 entries

	minExpectedEntries := 1000 // Allow some margin for changes
	if totalEntries < minExpectedEntries {
		t.Errorf("Expected at least %d completion entries, got %d - possible data loss", minExpectedEntries, totalEntries)
	}

	t.Logf("Successfully loaded %d total completion entries across %d commands", totalEntries, len(completions))
}
