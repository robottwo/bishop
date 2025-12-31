package completion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/robottwo/bishop/pkg/shellinput"
)

func TestStaticCompleter_NewStaticCompleter(t *testing.T) {
	sc := NewStaticCompleter()
	if sc == nil {
		t.Fatal("NewStaticCompleter returned nil")
	}
	if sc.completions == nil {
		t.Fatal("completions map is nil")
	}
}

func TestStaticCompleter_GetCompletions_Docker(t *testing.T) {
	sc := NewStaticCompleter()

	// Test getting all docker completions
	completions := sc.GetCompletions("docker", nil)
	if len(completions) == 0 {
		t.Fatal("Expected docker completions, got none")
	}

	// Verify some expected subcommands exist
	expectedCmds := []string{"run", "build", "ps", "images", "pull", "push"}
	for _, expected := range expectedCmds {
		found := false
		for _, c := range completions {
			if c.Value == expected {
				found = true
				// Verify description is present
				if c.Description == "" {
					t.Errorf("Expected description for docker %s, got empty", expected)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected to find docker subcommand %q", expected)
		}
	}
}

func TestStaticCompleter_GetCompletions_WithPrefix(t *testing.T) {
	sc := NewStaticCompleter()

	// Test filtering with prefix
	completions := sc.GetCompletions("docker", []string{"pu"})
	if len(completions) == 0 {
		t.Fatal("Expected docker completions for 'pu' prefix, got none")
	}

	// Should include 'pull' and 'push', not 'run'
	for _, c := range completions {
		if c.Value == "run" {
			t.Error("Expected 'run' to be filtered out when prefix is 'pu'")
		}
		if c.Value != "pull" && c.Value != "push" && c.Value != "pause" && c.Value != "plugin" {
			t.Errorf("Unexpected completion %q for prefix 'pu'", c.Value)
		}
	}
}

func TestStaticCompleter_GetCompletions_GithubCLI(t *testing.T) {
	sc := NewStaticCompleter()

	// Test gh CLI completions
	completions := sc.GetCompletions("gh", nil)
	if len(completions) == 0 {
		t.Fatal("Expected gh completions, got none")
	}

	// Verify some expected subcommands
	expectedCmds := []string{"pr", "issue", "repo", "auth", "workflow"}
	for _, expected := range expectedCmds {
		found := false
		for _, c := range completions {
			if c.Value == expected {
				found = true
				if c.Description == "" {
					t.Errorf("Expected description for gh %s, got empty", expected)
				}
				break
			}
		}
		if !found {
			t.Errorf("Expected to find gh subcommand %q", expected)
		}
	}
}

func TestStaticCompleter_GetCompletions_Go(t *testing.T) {
	sc := NewStaticCompleter()

	// Test go completions
	completions := sc.GetCompletions("go", nil)
	if len(completions) == 0 {
		t.Fatal("Expected go completions, got none")
	}

	// Verify some expected subcommands
	expectedCmds := []string{"build", "run", "test", "mod", "get"}
	for _, expected := range expectedCmds {
		found := false
		for _, c := range completions {
			if c.Value == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find go subcommand %q", expected)
		}
	}
}

func TestStaticCompleter_GetCompletions_Cargo(t *testing.T) {
	sc := NewStaticCompleter()

	// Test cargo completions
	completions := sc.GetCompletions("cargo", nil)
	if len(completions) == 0 {
		t.Fatal("Expected cargo completions, got none")
	}

	// Verify some expected subcommands
	expectedCmds := []string{"build", "run", "test", "new", "add", "clippy"}
	for _, expected := range expectedCmds {
		found := false
		for _, c := range completions {
			if c.Value == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find cargo subcommand %q", expected)
		}
	}
}

func TestStaticCompleter_GetCompletions_UnknownCommand(t *testing.T) {
	sc := NewStaticCompleter()

	// Test unknown command returns nil
	completions := sc.GetCompletions("unknowncommand12345", nil)
	if completions != nil {
		t.Errorf("Expected nil for unknown command, got %v", completions)
	}
}

func TestStaticCompleter_GetCompletions_MultipleArgs(t *testing.T) {
	sc := NewStaticCompleter()

	// Test that multiple args returns nil (we only complete first subcommand)
	completions := sc.GetCompletions("docker", []string{"run", "test"})
	if completions != nil {
		t.Errorf("Expected nil for multiple args, got %v", completions)
	}
}

func TestStaticCompleter_HasCommand(t *testing.T) {
	sc := NewStaticCompleter()

	// Test HasCommand for known commands
	knownCmds := []string{"docker", "kubectl", "npm", "yarn", "go", "cargo", "gh", "aws", "gcloud", "az"}
	for _, cmd := range knownCmds {
		if !sc.HasCommand(cmd) {
			t.Errorf("Expected HasCommand(%q) to return true", cmd)
		}
	}

	// Test HasCommand for unknown command
	if sc.HasCommand("unknowncommand12345") {
		t.Error("Expected HasCommand for unknown command to return false")
	}
}

func TestStaticCompleter_GetRegisteredCommands(t *testing.T) {
	sc := NewStaticCompleter()

	commands := sc.GetRegisteredCommands()
	if len(commands) == 0 {
		t.Fatal("Expected registered commands, got none")
	}

	// Verify commands are sorted
	for i := 1; i < len(commands); i++ {
		if commands[i-1] > commands[i] {
			t.Errorf("Commands not sorted: %s > %s", commands[i-1], commands[i])
		}
	}

	// Verify some expected commands are present
	expectedCmds := []string{"docker", "kubectl", "npm", "go", "cargo", "gh"}
	for _, expected := range expectedCmds {
		found := false
		for _, cmd := range commands {
			if cmd == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find command %q in registered commands", expected)
		}
	}
}

func TestStaticCompleter_RegisterUserCommand(t *testing.T) {
	sc := NewStaticCompleter()

	// Register a custom command
	userCompletions := []UserCompletion{
		{Value: "subcmd1", Description: "First subcommand"},
		{Value: "subcmd2", Description: "Second subcommand"},
		{Value: "subcmd3", Description: ""},
	}
	sc.RegisterUserCommand("mycustomcmd", userCompletions)

	// Verify the command is registered
	if !sc.HasCommand("mycustomcmd") {
		t.Fatal("Expected mycustomcmd to be registered")
	}

	// Verify completions
	completions := sc.GetCompletions("mycustomcmd", nil)
	if len(completions) != 3 {
		t.Fatalf("Expected 3 completions, got %d", len(completions))
	}

	// Verify values and descriptions
	for _, c := range completions {
		if c.Value == "subcmd1" && c.Description != "First subcommand" {
			t.Errorf("Wrong description for subcmd1: %q", c.Description)
		}
		if c.Value == "subcmd2" && c.Description != "Second subcommand" {
			t.Errorf("Wrong description for subcmd2: %q", c.Description)
		}
	}
}

func TestStaticCompleter_LoadUserCompletionsFromYAML(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a test YAML config
	yamlContent := `commands:
  myapp:
    - value: start
      description: Start the application
    - value: stop
      description: Stop the application
    - value: status
  otherapp:
    - value: deploy
      description: Deploy to production
    - value: rollback
`
	configPath := filepath.Join(tmpDir, "completions.yaml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create a new completer and manually load from file
	sc := &StaticCompleter{
		completions: make(map[string][]shellinput.CompletionCandidate),
	}
	sc.registerDefaults()

	// Load from test file
	if err := sc.loadCompletionsFromFile(configPath); err != nil {
		t.Fatalf("Failed to load completions from YAML: %v", err)
	}

	// Verify myapp completions
	if !sc.HasCommand("myapp") {
		t.Fatal("Expected myapp to be registered")
	}
	completions := sc.GetCompletions("myapp", nil)
	if len(completions) != 3 {
		t.Fatalf("Expected 3 completions for myapp, got %d", len(completions))
	}

	// Verify otherapp completions
	if !sc.HasCommand("otherapp") {
		t.Fatal("Expected otherapp to be registered")
	}
	completions = sc.GetCompletions("otherapp", nil)
	if len(completions) != 2 {
		t.Fatalf("Expected 2 completions for otherapp, got %d", len(completions))
	}
}

func TestStaticCompleter_LoadUserCompletionsFromJSON(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a test JSON config
	jsonContent := `{
  "commands": {
    "myapp": [
      {"value": "start", "description": "Start the application"},
      {"value": "stop", "description": "Stop the application"}
    ],
    "otherapp": [
      {"value": "deploy", "description": "Deploy to production"}
    ]
  }
}`
	configPath := filepath.Join(tmpDir, "completions.json")
	if err := os.WriteFile(configPath, []byte(jsonContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create a new completer and manually load from file
	sc := &StaticCompleter{
		completions: make(map[string][]shellinput.CompletionCandidate),
	}
	sc.registerDefaults()

	// Load from test file
	if err := sc.loadCompletionsFromFile(configPath); err != nil {
		t.Fatalf("Failed to load completions from JSON: %v", err)
	}

	// Verify myapp completions
	if !sc.HasCommand("myapp") {
		t.Fatal("Expected myapp to be registered")
	}
	completions := sc.GetCompletions("myapp", nil)
	if len(completions) != 2 {
		t.Fatalf("Expected 2 completions for myapp, got %d", len(completions))
	}
}

func TestStaticCompleter_CloudProviderCLIs(t *testing.T) {
	sc := NewStaticCompleter()

	// Test AWS CLI
	t.Run("aws", func(t *testing.T) {
		completions := sc.GetCompletions("aws", nil)
		if len(completions) == 0 {
			t.Fatal("Expected aws completions")
		}
		// Verify common AWS services
		expectedServices := []string{"s3", "ec2", "lambda", "iam", "ecs"}
		for _, svc := range expectedServices {
			found := false
			for _, c := range completions {
				if c.Value == svc {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected aws subcommand %q", svc)
			}
		}
	})

	// Test gcloud CLI
	t.Run("gcloud", func(t *testing.T) {
		completions := sc.GetCompletions("gcloud", nil)
		if len(completions) == 0 {
			t.Fatal("Expected gcloud completions")
		}
	})

	// Test Azure CLI
	t.Run("az", func(t *testing.T) {
		completions := sc.GetCompletions("az", nil)
		if len(completions) == 0 {
			t.Fatal("Expected az completions")
		}
	})
}

func TestStaticCompleter_DatabaseClients(t *testing.T) {
	sc := NewStaticCompleter()

	// Test psql
	t.Run("psql", func(t *testing.T) {
		completions := sc.GetCompletions("psql", nil)
		if len(completions) == 0 {
			t.Fatal("Expected psql completions")
		}
	})

	// Test mysql
	t.Run("mysql", func(t *testing.T) {
		completions := sc.GetCompletions("mysql", nil)
		if len(completions) == 0 {
			t.Fatal("Expected mysql completions")
		}
	})

	// Test redis-cli
	t.Run("redis-cli", func(t *testing.T) {
		completions := sc.GetCompletions("redis-cli", nil)
		if len(completions) == 0 {
			t.Fatal("Expected redis-cli completions")
		}
	})

	// Test mongosh
	t.Run("mongosh", func(t *testing.T) {
		completions := sc.GetCompletions("mongosh", nil)
		if len(completions) == 0 {
			t.Fatal("Expected mongosh completions")
		}
	})
}

func TestStaticCompleter_ContainerTools(t *testing.T) {
	sc := NewStaticCompleter()

	// Test docker-compose
	t.Run("docker-compose", func(t *testing.T) {
		completions := sc.GetCompletions("docker-compose", nil)
		if len(completions) == 0 {
			t.Fatal("Expected docker-compose completions")
		}
		expectedCmds := []string{"up", "down", "build", "logs", "ps"}
		for _, expected := range expectedCmds {
			found := false
			for _, c := range completions {
				if c.Value == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected docker-compose subcommand %q", expected)
			}
		}
	})

	// Test podman
	t.Run("podman", func(t *testing.T) {
		completions := sc.GetCompletions("podman", nil)
		if len(completions) == 0 {
			t.Fatal("Expected podman completions")
		}
	})

	// Test helm
	t.Run("helm", func(t *testing.T) {
		completions := sc.GetCompletions("helm", nil)
		if len(completions) == 0 {
			t.Fatal("Expected helm completions")
		}
		expectedCmds := []string{"install", "upgrade", "uninstall", "list", "repo"}
		for _, expected := range expectedCmds {
			found := false
			for _, c := range completions {
				if c.Value == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected helm subcommand %q", expected)
			}
		}
	})
}

func TestStaticCompleter_Editors(t *testing.T) {
	sc := NewStaticCompleter()

	// Test vim
	t.Run("vim", func(t *testing.T) {
		completions := sc.GetCompletions("vim", nil)
		if len(completions) == 0 {
			t.Fatal("Expected vim completions")
		}
	})

	// Test nvim
	t.Run("nvim", func(t *testing.T) {
		completions := sc.GetCompletions("nvim", nil)
		if len(completions) == 0 {
			t.Fatal("Expected nvim completions")
		}
	})

	// Test code (VS Code)
	t.Run("code", func(t *testing.T) {
		completions := sc.GetCompletions("code", nil)
		if len(completions) == 0 {
			t.Fatal("Expected code completions")
		}
	})
}

func TestStaticCompleter_SystemTools(t *testing.T) {
	sc := NewStaticCompleter()

	// Test systemctl
	t.Run("systemctl", func(t *testing.T) {
		completions := sc.GetCompletions("systemctl", nil)
		if len(completions) == 0 {
			t.Fatal("Expected systemctl completions")
		}
		expectedCmds := []string{"start", "stop", "restart", "status", "enable", "disable"}
		for _, expected := range expectedCmds {
			found := false
			for _, c := range completions {
				if c.Value == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected systemctl subcommand %q", expected)
			}
		}
	})

	// Test tmux
	t.Run("tmux", func(t *testing.T) {
		completions := sc.GetCompletions("tmux", nil)
		if len(completions) == 0 {
			t.Fatal("Expected tmux completions")
		}
	})

	// Test curl
	t.Run("curl", func(t *testing.T) {
		completions := sc.GetCompletions("curl", nil)
		if len(completions) == 0 {
			t.Fatal("Expected curl completions")
		}
	})

	// Test jq
	t.Run("jq", func(t *testing.T) {
		completions := sc.GetCompletions("jq", nil)
		if len(completions) == 0 {
			t.Fatal("Expected jq completions")
		}
	})
}

func TestStaticCompleter_PackageManagers(t *testing.T) {
	sc := NewStaticCompleter()

	// Test apt
	t.Run("apt", func(t *testing.T) {
		completions := sc.GetCompletions("apt", nil)
		if len(completions) == 0 {
			t.Fatal("Expected apt completions")
		}
		expectedCmds := []string{"install", "remove", "update", "upgrade", "search"}
		for _, expected := range expectedCmds {
			found := false
			for _, c := range completions {
				if c.Value == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected apt subcommand %q", expected)
			}
		}
	})

	// Test brew
	t.Run("brew", func(t *testing.T) {
		completions := sc.GetCompletions("brew", nil)
		if len(completions) == 0 {
			t.Fatal("Expected brew completions")
		}
	})

	// Test pip
	t.Run("pip", func(t *testing.T) {
		completions := sc.GetCompletions("pip", nil)
		if len(completions) == 0 {
			t.Fatal("Expected pip completions")
		}
		expectedCmds := []string{"install", "uninstall", "list", "freeze", "show"}
		for _, expected := range expectedCmds {
			found := false
			for _, c := range completions {
				if c.Value == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected pip subcommand %q", expected)
			}
		}
	})
}

func TestStaticCompleter_IaC(t *testing.T) {
	sc := NewStaticCompleter()

	// Test terraform
	t.Run("terraform", func(t *testing.T) {
		completions := sc.GetCompletions("terraform", nil)
		if len(completions) == 0 {
			t.Fatal("Expected terraform completions")
		}
		expectedCmds := []string{"init", "plan", "apply", "destroy", "validate"}
		for _, expected := range expectedCmds {
			found := false
			for _, c := range completions {
				if c.Value == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected terraform subcommand %q", expected)
			}
		}
	})
}

func TestStaticCompleter_ConcurrentAccess(t *testing.T) {
	sc := NewStaticCompleter()

	// Run concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = sc.GetCompletions("docker", nil)
				_ = sc.GetCompletions("kubectl", []string{"get"})
				_ = sc.HasCommand("npm")
				_ = sc.GetRegisteredCommands()
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
