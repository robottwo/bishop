package completion

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/robottwo/bishop/pkg/shellinput"
)

// DefaultCompleter handles built-in default completions for common commands
type DefaultCompleter struct{}

// GetCompletions tries to provide completions for the given command and context
func (d *DefaultCompleter) GetCompletions(command string, args []string, line string, pos int) ([]shellinput.CompletionCandidate, bool) {
	switch command {
	case "cd":
		return d.completeDirectories(args), true
	case "export", "unset":
		return d.completeEnvVars(args), true
	case "ssh", "scp", "sftp":
		return d.completeSSHHosts(args), true
	case "make":
		return d.completeMakeTargets(args), true
	case "kill":
		return d.completeKillSignals(args), true
	case "man", "help":
		// For now, just return nil to let it fall back or implementation TODO
		// Implementing full man page scanning is expensive for a default
		return nil, false
	}
	return nil, false
}

func (d *DefaultCompleter) completeDirectories(args []string) []shellinput.CompletionCandidate {
	// Use the last arg as prefix, or empty if none
	prefix := ""
	if len(args) > 0 {
		prefix = args[len(args)-1]
	}

	// Re-use getFileCompletions but filter for directories only
	// We need to access the current directory. For now, we assume current process CWD.
	// ideally this should come from context/runner.
	cwd, _ := os.Getwd()

	// We can use the existing getFileCompletions helper if we can filter its output
	// But getFileCompletions returns strings. We can parse them.
	// Or we implement a specific directory walker.
	// Let's reuse getFileCompletions for consistency and filter.
	candidates := getFileCompletions(prefix, cwd)

	var dirs []shellinput.CompletionCandidate
	for _, c := range candidates {
		// Check if it's a directory by looking at the Suffix field
		if c.Suffix == string(os.PathSeparator) {
			c.Description = "Directory"
			dirs = append(dirs, c)
		}
	}
	return dirs
}

func (d *DefaultCompleter) completeEnvVars(args []string) []shellinput.CompletionCandidate {
	prefix := ""
	if len(args) > 0 {
		prefix = args[len(args)-1]
	}

	var candidates []shellinput.CompletionCandidate
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		key := parts[0]
		if strings.HasPrefix(key, prefix) {
			candidates = append(candidates, shellinput.CompletionCandidate{
				Value:       key,
				Description: "Environment Variable",
			})
		}
	}
	return candidates
}

func (d *DefaultCompleter) completeSSHHosts(args []string) []shellinput.CompletionCandidate {
	prefix := ""
	if len(args) > 0 {
		prefix = args[len(args)-1]
	}

	hosts := make(map[string]bool)

	home, err := os.UserHomeDir()
	if err == nil {
		sshDir := filepath.Join(home, ".ssh")

		// Parse ~/.ssh/config (including Include directives)
		configPath := filepath.Join(sshDir, "config")
		visited := make(map[string]bool)
		parseSSHConfig(configPath, sshDir, hosts, visited)

		// Parse ~/.ssh/known_hosts
		knownHostsPath := filepath.Join(sshDir, "known_hosts")
		parseKnownHosts(knownHostsPath, hosts)
	}

	var candidates []shellinput.CompletionCandidate
	for host := range hosts {
		if strings.HasPrefix(host, prefix) {
			candidates = append(candidates, shellinput.CompletionCandidate{
				Value:       host,
				Description: "SSH Host",
			})
		}
	}
	return candidates
}

// parseSSHConfig parses an SSH config file, extracting Host entries and
// recursively processing Include directives. visited tracks parsed files
// to prevent infinite loops.
func parseSSHConfig(configPath, sshDir string, hosts map[string]bool, visited map[string]bool) {
	// Resolve to absolute path for deduplication
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return
	}
	if visited[absPath] {
		return
	}
	visited[absPath] = true

	file, err := os.Open(configPath)
	if err != nil {
		return
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Case-insensitive matching for SSH config keywords
		lineLower := strings.ToLower(line)

		if strings.HasPrefix(lineLower, "host ") {
			// Host can have multiple aliases: "Host foo bar"
			parts := strings.Fields(line)
			if len(parts) > 1 {
				for _, host := range parts[1:] {
					// Skip wildcards and negations
					if !strings.ContainsAny(host, "*?!") {
						hosts[host] = true
					}
				}
			}
		} else if strings.HasPrefix(lineLower, "include ") {
			// Include directive - can use glob patterns
			// Extract the pattern (everything after "Include ")
			pattern := strings.TrimSpace(line[8:])
			if pattern == "" {
				continue
			}

			// Expand ~ to home directory
			if strings.HasPrefix(pattern, "~") {
				home, err := os.UserHomeDir()
				if err == nil {
					pattern = filepath.Join(home, pattern[1:])
				}
			}

			// If pattern is relative, it's relative to ~/.ssh
			if !filepath.IsAbs(pattern) {
				pattern = filepath.Join(sshDir, pattern)
			}

			// Expand glob pattern
			matches, err := filepath.Glob(pattern)
			if err != nil {
				continue
			}

			for _, match := range matches {
				parseSSHConfig(match, sshDir, hosts, visited)
			}
		}
	}
}

// parseKnownHosts parses an SSH known_hosts file, extracting hostnames.
// Hashed hostnames (starting with |) are skipped as they cannot be reversed.
func parseKnownHosts(knownHostsPath string, hosts map[string]bool) {
	file, err := os.Open(knownHostsPath)
	if err != nil {
		return
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Format: hostname[,hostname2,...] keytype key [comment]
		// or: @marker hostname keytype key
		// Hashed: |1|base64salt|base64hash keytype key

		// Skip hashed entries (start with |)
		if strings.HasPrefix(line, "|") {
			continue
		}

		// Skip @cert-authority and @revoked markers
		if strings.HasPrefix(line, "@") {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}
			line = strings.Join(parts[1:], " ")
		}

		// First field contains hostname(s)
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}

		hostField := fields[0]

		// Handle bracketed hosts like [hostname]:port
		if strings.HasPrefix(hostField, "[") {
			// Extract hostname from [hostname]:port
			endBracket := strings.Index(hostField, "]")
			if endBracket > 1 {
				hostField = hostField[1:endBracket]
			}
		}

		// Hostnames can be comma-separated: host1,host2,192.168.1.1
		for _, h := range strings.Split(hostField, ",") {
			h = strings.TrimSpace(h)
			if h == "" {
				continue
			}

			// Handle bracketed form within comma-separated list
			if strings.HasPrefix(h, "[") {
				endBracket := strings.Index(h, "]")
				if endBracket > 1 {
					h = h[1:endBracket]
				}
			}

			// Skip IP addresses (simple heuristic: contains only digits and dots/colons)
			if looksLikeIPAddress(h) {
				continue
			}

			// Skip wildcards
			if strings.ContainsAny(h, "*?") {
				continue
			}

			hosts[h] = true
		}
	}
}

// looksLikeIPAddress returns true if the string looks like an IPv4 or IPv6 address.
func looksLikeIPAddress(s string) bool {
	// IPv4: all chars are digits or dots
	allIPv4 := true
	for _, c := range s {
		if !((c >= '0' && c <= '9') || c == '.') {
			allIPv4 = false
			break
		}
	}
	if allIPv4 && strings.Contains(s, ".") {
		return true
	}

	// IPv6: contains colons and only hex digits, colons, or dots (for mapped IPv4)
	if strings.Contains(s, ":") {
		allIPv6 := true
		for _, c := range s {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || c == ':' || c == '.') {
				allIPv6 = false
				break
			}
		}
		return allIPv6
	}

	return false
}

func (d *DefaultCompleter) completeMakeTargets(args []string) []shellinput.CompletionCandidate {
	prefix := ""
	if len(args) > 0 {
		prefix = args[len(args)-1]
	}

	// Look for Makefile in current directory
	cwd, _ := os.Getwd()
	makefiles := []string{"Makefile", "makefile", "GNUmakefile"}

	var candidates []shellinput.CompletionCandidate

	for _, mk := range makefiles {
		path := filepath.Join(cwd, mk)
		if file, err := os.Open(path); err == nil {
			defer func() {
				_ = file.Close()
			}()
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				// Simple regex for targets: starts with word characters, ends with colon
				// Exclude .PHONY etc.
				if strings.Contains(line, ":") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, ".") {
					parts := strings.SplitN(line, ":", 2)
					target := strings.TrimSpace(parts[0])
					// Handle multiple targets "clean install:"
					targets := strings.Fields(target)
					for _, t := range targets {
						if strings.HasPrefix(t, prefix) {
							candidates = append(candidates, shellinput.CompletionCandidate{
								Value:       t,
								Description: "Make target",
							})
						}
					}
				}
			}
			break // Only parse the first found makefile
		}
	}
	return candidates
}

func (d *DefaultCompleter) completeKillSignals(args []string) []shellinput.CompletionCandidate {
	prefix := ""
	if len(args) > 0 {
		prefix = args[len(args)-1]
	}

	// If prefix doesn't start with -, maybe they are typing a PID.
	// If it starts with -, they want a signal.
	if strings.HasPrefix(prefix, "-") {
		signals := []string{
			"-HUP", "-INT", "-QUIT", "-ILL", "-TRAP", "-ABRT", "-BUS", "-FPE",
			"-KILL", "-USR1", "-SEGV", "-USR2", "-PIPE", "-ALRM", "-TERM",
			"-STKFLT", "-CHLD", "-CONT", "-STOP", "-TSTP", "-TTIN", "-TTOU",
			"-URG", "-XCPU", "-XFSZ", "-VTALRM", "-PROF", "-WINCH", "-IO",
			"-PWR", "-SYS",
		}

		var candidates []shellinput.CompletionCandidate
		for _, sig := range signals {
			if strings.HasPrefix(sig, prefix) {
				candidates = append(candidates, shellinput.CompletionCandidate{
					Value:       sig,
					Description: "Signal",
				})
			}
		}
		return candidates
	}
	return nil
}
