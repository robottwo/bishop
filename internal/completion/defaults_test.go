package completion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robottwo/bishop/pkg/shellinput"
	"github.com/stretchr/testify/assert"
)

func TestDefaultCompleter_GetCompletions(t *testing.T) {
	completer := &DefaultCompleter{}

	tests := []struct {
		name      string
		command   string
		args      []string
		wantFound bool
		wantValue string // Check if at least one completion matches this
	}{
		{
			name:      "cd completion",
			command:   "cd",
			args:      []string{},
			wantFound: true,
			// We can't easily test result values as it depends on filesystem, but we expect found=true
		},
		{
			name:      "export completion",
			command:   "export",
			args:      []string{},
			wantFound: true,
			// Assumes PATH is in environment
			wantValue: "PATH",
		},
		{
			name:      "kill completion",
			command:   "kill",
			args:      []string{"-"},
			wantFound: true,
			wantValue: "-KILL",
		},
		{
			name:      "unknown command",
			command:   "unknown",
			args:      []string{},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := completer.GetCompletions(tt.command, tt.args, "", 0)
			assert.Equal(t, tt.wantFound, found)

			if tt.wantValue != "" {
				match := false
				for _, c := range got {
					if c.Value == tt.wantValue || (tt.command == "export" && c.Value == tt.wantValue) {
						match = true
						break
					}
				}
				assert.True(t, match, "Expected to find value %q in completions", tt.wantValue)
			}
		})
	}
}

func TestStaticCompleter(t *testing.T) {
	completer := NewStaticCompleter()

	tests := []struct {
		name      string
		command   string
		args      []string
		wantValue string
	}{
		{
			name:      "docker completion",
			command:   "docker",
			args:      []string{},
			wantValue: "run",
		},
		{
			name:      "docker completion filter",
			command:   "docker",
			args:      []string{"p"},
			wantValue: "ps",
		},
		{
			name:      "npm completion",
			command:   "npm",
			args:      []string{},
			wantValue: "install",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := completer.GetCompletions(tt.command, tt.args)

			match := false
			for _, c := range got {
				if c.Value == tt.wantValue {
					match = true
					break
				}
			}
			assert.True(t, match, "Expected to find value %q in completions", tt.wantValue)
		})
	}
}

func TestGitCompleter_Subcommands(t *testing.T) {
	completer := &GitCompleter{}

	// Test subcommands (empty args, line doesn't matter for subcommand completion)
	got := completer.GetCompletions([]string{}, "git ")

	expected := []string{"checkout", "commit", "add", "push", "pull", "status"}
	for _, exp := range expected {
		match := false
		for _, c := range got {
			if c.Value == exp {
				match = true
				break
			}
		}
		assert.True(t, match, "Expected to find git subcommand %q", exp)
	}
}

// Helper to avoid unused import error if we don't use the package explicitly
var _ = shellinput.CompletionCandidate{}

func TestParseSSHConfig(t *testing.T) {
	// Create a temp directory for test SSH config files
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		configFiles  map[string]string // filename -> content
		mainConfig   string            // which file is the main config
		wantHosts    []string
		notWantHosts []string
	}{
		{
			name: "basic host entries",
			configFiles: map[string]string{
				"config": `Host server1
	HostName example.com
	User admin

Host server2 server3
	HostName other.com
`,
			},
			mainConfig: "config",
			wantHosts:  []string{"server1", "server2", "server3"},
		},
		{
			name: "skip wildcards and negations",
			configFiles: map[string]string{
				"config": `Host *
	ServerAliveInterval 60

Host production
	HostName prod.example.com

Host !staging
	# This is a negation pattern

Host *.internal
	ProxyJump bastion
`,
			},
			mainConfig:   "config",
			wantHosts:    []string{"production"},
			notWantHosts: []string{"*", "!staging", "*.internal"},
		},
		{
			name: "case insensitive keywords",
			configFiles: map[string]string{
				"config": `HOST server1
host server2
Host server3
`,
			},
			mainConfig: "config",
			wantHosts:  []string{"server1", "server2", "server3"},
		},
		{
			name: "with Include directive",
			configFiles: map[string]string{
				"config": `Host mainserver
	HostName main.example.com

Include extra.conf
`,
				"extra.conf": `Host extrahost1
	HostName extra1.example.com

Host extrahost2
	HostName extra2.example.com
`,
			},
			mainConfig: "config",
			wantHosts:  []string{"mainserver", "extrahost1", "extrahost2"},
		},
		{
			name: "Include with glob pattern",
			configFiles: map[string]string{
				"config": `Include conf.d/*.conf
`,
				"conf.d/web.conf": `Host webserver
	HostName web.example.com
`,
				"conf.d/db.conf": `Host dbserver
	HostName db.example.com
`,
			},
			mainConfig: "config",
			wantHosts:  []string{"webserver", "dbserver"},
		},
		{
			name: "comments and empty lines",
			configFiles: map[string]string{
				"config": `# This is a comment
Host server1

   # Indented comment
Host server2

`,
			},
			mainConfig: "config",
			wantHosts:  []string{"server1", "server2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory structure
			testDir := t.TempDir()

			for filename, content := range tt.configFiles {
				fullPath := filepath.Join(testDir, filename)
				// Create parent directories if needed (e.g., conf.d/)
				if strings.Contains(filename, "/") {
					parentDir := filepath.Dir(fullPath)
					if err := os.MkdirAll(parentDir, 0755); err != nil {
						t.Fatalf("Failed to create directory %s: %v", parentDir, err)
					}
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", filename, err)
				}
			}

			hosts := make(map[string]bool)
			visited := make(map[string]bool)
			parseSSHConfig(filepath.Join(testDir, tt.mainConfig), testDir, hosts, visited)

			for _, want := range tt.wantHosts {
				assert.True(t, hosts[want], "Expected to find host %q", want)
			}
			for _, notWant := range tt.notWantHosts {
				assert.False(t, hosts[notWant], "Expected NOT to find host %q", notWant)
			}
		})
	}

	_ = tmpDir // silence unused warning
}

func TestParseKnownHosts(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantHosts    []string
		notWantHosts []string
	}{
		{
			name: "basic hostnames",
			content: `example.com ssh-rsa AAAAB3...
github.com ssh-ed25519 AAAAC3...
gitlab.com ssh-rsa AAAAB3...
`,
			wantHosts: []string{"example.com", "github.com", "gitlab.com"},
		},
		{
			name: "comma-separated hosts with IP",
			content: `server.example.com,192.168.1.100 ssh-rsa AAAAB3...
`,
			wantHosts:    []string{"server.example.com"},
			notWantHosts: []string{"192.168.1.100"},
		},
		{
			name: "skip hashed entries",
			content: `|1|abc123|def456 ssh-rsa AAAAB3...
plainhost.example.com ssh-rsa AAAAB3...
`,
			wantHosts:    []string{"plainhost.example.com"},
			notWantHosts: []string{"|1|abc123|def456"},
		},
		{
			name: "bracketed hosts with port",
			content: `[git.example.com]:2222 ssh-rsa AAAAB3...
[another.host]:22 ssh-ed25519 AAAAC3...
`,
			wantHosts: []string{"git.example.com", "another.host"},
		},
		{
			name: "skip IPv4 addresses",
			content: `192.168.1.1 ssh-rsa AAAAB3...
10.0.0.1 ssh-rsa AAAAB3...
host.example.com ssh-rsa AAAAB3...
`,
			wantHosts:    []string{"host.example.com"},
			notWantHosts: []string{"192.168.1.1", "10.0.0.1"},
		},
		{
			name: "skip IPv6 addresses",
			content: `::1 ssh-rsa AAAAB3...
2001:db8::1 ssh-rsa AAAAB3...
fe80::1 ssh-ed25519 AAAAC3...
host.example.com ssh-rsa AAAAB3...
`,
			wantHosts:    []string{"host.example.com"},
			notWantHosts: []string{"::1", "2001:db8::1", "fe80::1"},
		},
		{
			name: "comments and empty lines",
			content: `# This is a comment
example.com ssh-rsa AAAAB3...

# Another comment
server.local ssh-rsa AAAAB3...
`,
			wantHosts: []string{"example.com", "server.local"},
		},
		{
			name: "cert-authority marker",
			content: `@cert-authority *.example.com ssh-rsa AAAAB3...
regular.host ssh-rsa AAAAB3...
`,
			wantHosts:    []string{"regular.host"},
			notWantHosts: []string{"@cert-authority", "*.example.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file with test content
			tmpFile, err := os.CreateTemp("", "known_hosts_test")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatalf("Failed to write test content: %v", err)
			}
			tmpFile.Close()

			hosts := make(map[string]bool)
			parseKnownHosts(tmpFile.Name(), hosts)

			for _, want := range tt.wantHosts {
				assert.True(t, hosts[want], "Expected to find host %q", want)
			}
			for _, notWant := range tt.notWantHosts {
				assert.False(t, hosts[notWant], "Expected NOT to find host %q", notWant)
			}
		})
	}
}

func TestLooksLikeIPAddress(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		// IPv4
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"255.255.255.255", true},
		{"0.0.0.0", true},

		// IPv6
		{"::1", true},
		{"2001:db8::1", true},
		{"fe80::1", true},
		{"::ffff:192.168.1.1", true},

		// Not IP addresses
		{"example.com", false},
		{"localhost", false},
		{"server1", false},
		{"my-host.local", false},
		{"host123.example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := looksLikeIPAddress(tt.input)
			assert.Equal(t, tt.want, got, "looksLikeIPAddress(%q)", tt.input)
		})
	}
}
