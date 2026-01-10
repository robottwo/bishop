package completion

import (
	"testing"
	"testing/fstest"
)

func TestNewConfigLoader(t *testing.T) {
	// Create a test embedded filesystem
	testFS := fstest.MapFS{}

	loader := NewConfigLoader(testFS)
	if loader == nil {
		t.Fatal("NewConfigLoader returned nil")
	}
}

func TestConfigLoader_LoadAllCompletions(t *testing.T) {
	tests := []struct {
		name          string
		fs            fstest.MapFS
		expectedCmds  []string
		expectedError bool
		errorContains string
	}{
		{
			name: "load single YAML file with one command",
			fs: fstest.MapFS{
				"data/test.yaml": &fstest.MapFile{
					Data: []byte(`commands:
  docker:
    - value: run
      description: Run a container
    - value: build
      description: Build an image
`),
				},
			},
			expectedCmds:  []string{"docker"},
			expectedError: false,
		},
		{
			name: "load multiple YAML files",
			fs: fstest.MapFS{
				"data/containers.yaml": &fstest.MapFile{
					Data: []byte(`commands:
  docker:
    - value: run
      description: Run a container
  kubectl:
    - value: get
      description: Get resources
`),
				},
				"data/package_managers.yaml": &fstest.MapFile{
					Data: []byte(`commands:
  npm:
    - value: install
      description: Install packages
  yarn:
    - value: add
      description: Add package
`),
				},
			},
			expectedCmds:  []string{"docker", "kubectl", "npm", "yarn"},
			expectedError: false,
		},
		{
			name: "load YAML with missing descriptions",
			fs: fstest.MapFS{
				"data/test.yaml": &fstest.MapFile{
					Data: []byte(`commands:
  myapp:
    - value: start
    - value: stop
      description: Stop the app
`),
				},
			},
			expectedCmds:  []string{"myapp"},
			expectedError: false,
		},
		{
			name: "load .yml extension",
			fs: fstest.MapFS{
				"data/test.yml": &fstest.MapFile{
					Data: []byte(`commands:
  myapp:
    - value: run
`),
				},
			},
			expectedCmds:  []string{"myapp"},
			expectedError: false,
		},
		{
			name: "skip non-YAML files",
			fs: fstest.MapFS{
				"data/README.md": &fstest.MapFile{
					Data: []byte("This is a readme"),
				},
				"data/test.yaml": &fstest.MapFile{
					Data: []byte(`commands:
  docker:
    - value: run
`),
				},
				"data/config.txt": &fstest.MapFile{
					Data: []byte("Some text file"),
				},
			},
			expectedCmds:  []string{"docker"},
			expectedError: false,
		},
		{
			name: "empty filesystem",
			fs:   fstest.MapFS{},
			expectedCmds:  []string{},
			expectedError: false,
		},
		{
			name: "malformed YAML",
			fs: fstest.MapFS{
				"data/bad.yaml": &fstest.MapFile{
					Data: []byte(`commands:
  docker:
    - value: run
      description: [invalid yaml structure
`),
				},
			},
			expectedError: true,
			errorContains: "failed to parse",
		},
		{
			name: "YAML with no commands key",
			fs: fstest.MapFS{
				"data/empty.yaml": &fstest.MapFile{
					Data: []byte(`some_other_key: value`),
				},
			},
			expectedCmds:  []string{},
			expectedError: false,
		},
		{
			name: "empty YAML file",
			fs: fstest.MapFS{
				"data/empty.yaml": &fstest.MapFile{
					Data: []byte(``),
				},
			},
			expectedCmds:  []string{},
			expectedError: false,
		},
		{
			name: "YAML with empty commands",
			fs: fstest.MapFS{
				"data/empty_commands.yaml": &fstest.MapFile{
					Data: []byte(`commands: {}`),
				},
			},
			expectedCmds:  []string{},
			expectedError: false,
		},
		{
			name: "nested directories with YAML files",
			fs: fstest.MapFS{
				"data/containers/docker.yaml": &fstest.MapFile{
					Data: []byte(`commands:
  docker:
    - value: run
`),
				},
				"data/cloud/aws.yaml": &fstest.MapFile{
					Data: []byte(`commands:
  aws:
    - value: s3
`),
				},
			},
			expectedCmds:  []string{"docker", "aws"},
			expectedError: false,
		},
		{
			name: "multiple commands override each other (last wins)",
			fs: fstest.MapFS{
				"data/first.yaml": &fstest.MapFile{
					Data: []byte(`commands:
  docker:
    - value: run
      description: First definition
`),
				},
				"data/second.yaml": &fstest.MapFile{
					Data: []byte(`commands:
  docker:
    - value: build
      description: Second definition
`),
				},
			},
			expectedCmds:  []string{"docker"},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewConfigLoader(tt.fs)
			completions, err := loader.LoadAllCompletions()

			if tt.expectedError {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(completions) != len(tt.expectedCmds) {
				t.Errorf("Expected %d commands, got %d", len(tt.expectedCmds), len(completions))
			}

			for _, cmd := range tt.expectedCmds {
				if _, ok := completions[cmd]; !ok {
					t.Errorf("Expected command %q not found in completions", cmd)
				}
			}
		})
	}
}

func TestConfigLoader_LoadAllCompletions_VerifyContent(t *testing.T) {
	// Test that the actual content is correctly parsed
	fs := fstest.MapFS{
		"data/test.yaml": &fstest.MapFile{
			Data: []byte(`commands:
  docker:
    - value: run
      description: Run a container
    - value: build
      description: Build an image
    - value: ps
  kubectl:
    - value: get
      description: Get resources
`),
		},
	}

	loader := NewConfigLoader(fs)
	completions, err := loader.LoadAllCompletions()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify docker completions
	dockerCompletions, ok := completions["docker"]
	if !ok {
		t.Fatal("docker command not found")
	}
	if len(dockerCompletions) != 3 {
		t.Fatalf("Expected 3 docker completions, got %d", len(dockerCompletions))
	}

	// Check first completion
	if dockerCompletions[0].Value != "run" {
		t.Errorf("Expected first completion value to be 'run', got %q", dockerCompletions[0].Value)
	}
	if dockerCompletions[0].Description != "Run a container" {
		t.Errorf("Expected first completion description to be 'Run a container', got %q", dockerCompletions[0].Description)
	}

	// Check completion without description
	if dockerCompletions[2].Value != "ps" {
		t.Errorf("Expected third completion value to be 'ps', got %q", dockerCompletions[2].Value)
	}
	if dockerCompletions[2].Description != "" {
		t.Errorf("Expected third completion description to be empty, got %q", dockerCompletions[2].Description)
	}

	// Verify kubectl completions
	kubectlCompletions, ok := completions["kubectl"]
	if !ok {
		t.Fatal("kubectl command not found")
	}
	if len(kubectlCompletions) != 1 {
		t.Fatalf("Expected 1 kubectl completion, got %d", len(kubectlCompletions))
	}
}

func TestConfigLoader_LoadCompletionsFromFile(t *testing.T) {
	tests := []struct {
		name          string
		fs            fstest.MapFS
		filename      string
		expectedCmds  []string
		expectedError bool
		errorContains string
	}{
		{
			name: "load specific file successfully",
			fs: fstest.MapFS{
				"data/containers.yaml": &fstest.MapFile{
					Data: []byte(`commands:
  docker:
    - value: run
  kubectl:
    - value: get
`),
				},
			},
			filename:      "containers.yaml",
			expectedCmds:  []string{"docker", "kubectl"},
			expectedError: false,
		},
		{
			name: "load file with .yml extension",
			fs: fstest.MapFS{
				"data/test.yml": &fstest.MapFile{
					Data: []byte(`commands:
  myapp:
    - value: start
`),
				},
			},
			filename:      "test.yml",
			expectedCmds:  []string{"myapp"},
			expectedError: false,
		},
		{
			name: "file not found",
			fs: fstest.MapFS{
				"data/other.yaml": &fstest.MapFile{
					Data: []byte(`commands: {}`),
				},
			},
			filename:      "nonexistent.yaml",
			expectedError: true,
			errorContains: "failed to read",
		},
		{
			name: "malformed YAML in specific file",
			fs: fstest.MapFS{
				"data/bad.yaml": &fstest.MapFile{
					Data: []byte(`commands: [invalid yaml`),
				},
			},
			filename:      "bad.yaml",
			expectedError: true,
			errorContains: "failed to parse",
		},
		{
			name: "empty YAML file",
			fs: fstest.MapFS{
				"data/empty.yaml": &fstest.MapFile{
					Data: []byte(``),
				},
			},
			filename:      "empty.yaml",
			expectedCmds:  []string{},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewConfigLoader(tt.fs)
			completions, err := loader.LoadCompletionsFromFile(tt.filename)

			if tt.expectedError {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(completions) != len(tt.expectedCmds) {
				t.Errorf("Expected %d commands, got %d", len(tt.expectedCmds), len(completions))
			}

			for _, cmd := range tt.expectedCmds {
				if _, ok := completions[cmd]; !ok {
					t.Errorf("Expected command %q not found in completions", cmd)
				}
			}
		})
	}
}

func TestConfigLoader_LoadCompletionsFromFile_PathHandling(t *testing.T) {
	// Test that the data/ prefix is correctly added
	fs := fstest.MapFS{
		"data/test.yaml": &fstest.MapFile{
			Data: []byte(`commands:
  myapp:
    - value: run
`),
		},
	}

	loader := NewConfigLoader(fs)

	// Should work with just the filename
	completions, err := loader.LoadCompletionsFromFile("test.yaml")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(completions) != 1 {
		t.Errorf("Expected 1 command, got %d", len(completions))
	}

	if _, ok := completions["myapp"]; !ok {
		t.Error("Expected myapp command not found")
	}
}

func TestConfigLoader_ListEmbeddedFiles(t *testing.T) {
	tests := []struct {
		name          string
		fs            fstest.MapFS
		expectedFiles []string
		expectedError bool
	}{
		{
			name: "list YAML files",
			fs: fstest.MapFS{
				"data/containers.yaml": &fstest.MapFile{
					Data: []byte(`commands: {}`),
				},
				"data/databases.yaml": &fstest.MapFile{
					Data: []byte(`commands: {}`),
				},
			},
			expectedFiles: []string{"data/containers.yaml", "data/databases.yaml"},
			expectedError: false,
		},
		{
			name: "list both .yaml and .yml files",
			fs: fstest.MapFS{
				"data/test.yaml": &fstest.MapFile{
					Data: []byte(`commands: {}`),
				},
				"data/other.yml": &fstest.MapFile{
					Data: []byte(`commands: {}`),
				},
			},
			expectedFiles: []string{"data/test.yaml", "data/other.yml"},
			expectedError: false,
		},
		{
			name: "ignore non-YAML files",
			fs: fstest.MapFS{
				"data/README.md": &fstest.MapFile{
					Data: []byte("readme"),
				},
				"data/test.yaml": &fstest.MapFile{
					Data: []byte(`commands: {}`),
				},
				"data/config.json": &fstest.MapFile{
					Data: []byte("{}"),
				},
			},
			expectedFiles: []string{"data/test.yaml"},
			expectedError: false,
		},
		{
			name:          "empty filesystem",
			fs:            fstest.MapFS{},
			expectedFiles: []string{},
			expectedError: false,
		},
		{
			name: "nested directories",
			fs: fstest.MapFS{
				"data/level1/test.yaml": &fstest.MapFile{
					Data: []byte(`commands: {}`),
				},
				"data/level1/level2/deep.yaml": &fstest.MapFile{
					Data: []byte(`commands: {}`),
				},
			},
			expectedFiles: []string{"data/level1/test.yaml", "data/level1/level2/deep.yaml"},
			expectedError: false,
		},
		{
			name: "ignore directories",
			fs: fstest.MapFS{
				"data/test.yaml": &fstest.MapFile{
					Data: []byte(`commands: {}`),
				},
			},
			expectedFiles: []string{"data/test.yaml"},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewConfigLoader(tt.fs)
			files, err := loader.ListEmbeddedFiles()

			if tt.expectedError {
				if err == nil {
					t.Fatal("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(files) != len(tt.expectedFiles) {
				t.Errorf("Expected %d files, got %d. Files: %v", len(tt.expectedFiles), len(files), files)
			}

			// Check that all expected files are present
			for _, expectedFile := range tt.expectedFiles {
				found := false
				for _, file := range files {
					if file == expectedFile {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected file %q not found in result", expectedFile)
				}
			}
		})
	}
}

func TestConfigLoader_WithRealEmbeddedData(t *testing.T) {
	// Test with the actual embedded CompletionData
	loader := NewConfigLoader(CompletionData)

	// List embedded files
	files, err := loader.ListEmbeddedFiles()
	if err != nil {
		t.Fatalf("Failed to list embedded files: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("Expected embedded YAML files, got none")
	}

	// Verify expected files exist
	expectedFiles := []string{
		"data/containers.yaml",
		"data/package_managers.yaml",
		"data/cloud.yaml",
		"data/databases.yaml",
		"data/developer_tools.yaml",
	}

	for _, expectedFile := range expectedFiles {
		found := false
		for _, file := range files {
			if file == expectedFile {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected embedded file %q not found", expectedFile)
		}
	}

	// Load all completions
	completions, err := loader.LoadAllCompletions()
	if err != nil {
		t.Fatalf("Failed to load all completions: %v", err)
	}

	// Verify some expected commands exist
	expectedCommands := []string{
		"docker", "kubectl", "npm", "yarn", "go", "cargo",
		"aws", "gcloud", "az", "terraform",
		"redis-cli", "psql", "mysql", "mongosh",
		"gh", "code", "vim", "tmux", "curl",
	}

	for _, cmd := range expectedCommands {
		if _, ok := completions[cmd]; !ok {
			t.Errorf("Expected command %q not found in loaded completions", cmd)
		}
	}

	// Verify docker has expected subcommands
	dockerCompletions, ok := completions["docker"]
	if !ok {
		t.Fatal("docker command not found")
	}

	if len(dockerCompletions) == 0 {
		t.Fatal("Expected docker completions, got none")
	}

	// Check for some common docker subcommands
	expectedDockerCmds := []string{"run", "build", "ps", "images"}
	for _, expectedCmd := range expectedDockerCmds {
		found := false
		for _, completion := range dockerCompletions {
			if completion.Value == expectedCmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected docker subcommand %q not found", expectedCmd)
		}
	}
}

func TestConfigLoader_LoadCompletionsFromFile_RealData(t *testing.T) {
	loader := NewConfigLoader(CompletionData)

	// Test loading specific files
	files := []string{
		"containers.yaml",
		"package_managers.yaml",
		"cloud.yaml",
		"databases.yaml",
		"developer_tools.yaml",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			completions, err := loader.LoadCompletionsFromFile(file)
			if err != nil {
				t.Fatalf("Failed to load %s: %v", file, err)
			}

			if len(completions) == 0 {
				t.Errorf("Expected completions from %s, got none", file)
			}
		})
	}
}

func TestConfigLoader_ConcurrentAccess(t *testing.T) {
	// Test that ConfigLoader can be safely accessed concurrently
	loader := NewConfigLoader(CompletionData)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				// Concurrent LoadAllCompletions calls
				_, _ = loader.LoadAllCompletions()

				// Concurrent LoadCompletionsFromFile calls
				_, _ = loader.LoadCompletionsFromFile("containers.yaml")

				// Concurrent ListEmbeddedFiles calls
				_, _ = loader.ListEmbeddedFiles()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestConfigLoader_ErrorWrapping(t *testing.T) {
	// Test that errors are properly wrapped with context
	fs := fstest.MapFS{
		"data/bad.yaml": &fstest.MapFile{
			Data: []byte(`invalid: [yaml`),
		},
	}

	loader := NewConfigLoader(fs)

	// Test LoadAllCompletions error wrapping
	_, err := loader.LoadAllCompletions()
	if err == nil {
		t.Fatal("Expected error for malformed YAML")
	}
	if !containsString(err.Error(), "failed to parse") {
		t.Errorf("Expected error to contain 'failed to parse', got: %v", err)
	}
	if !containsString(err.Error(), "data/bad.yaml") {
		t.Errorf("Expected error to contain file path, got: %v", err)
	}

	// Test LoadCompletionsFromFile error wrapping
	_, err = loader.LoadCompletionsFromFile("bad.yaml")
	if err == nil {
		t.Fatal("Expected error for malformed YAML")
	}
	if !containsString(err.Error(), "failed to parse") {
		t.Errorf("Expected error to contain 'failed to parse', got: %v", err)
	}

	// Test file not found error
	_, err = loader.LoadCompletionsFromFile("nonexistent.yaml")
	if err == nil {
		t.Fatal("Expected error for non-existent file")
	}
	if !containsString(err.Error(), "failed to read") {
		t.Errorf("Expected error to contain 'failed to read', got: %v", err)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
