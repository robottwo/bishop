package completion

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/robottwo/bishop/pkg/shellinput"
)

// DocumentationCompleter handles completions for documentation commands (man, info, help)
type DocumentationCompleter struct {
	manPages  map[string][]string // name -> list of sections
	infoPages []string            // list of info page names
	initOnce  sync.Once
	builtins  []string
}

// NewDocumentationCompleter creates a new DocumentationCompleter
func NewDocumentationCompleter() *DocumentationCompleter {
	return &DocumentationCompleter{
		builtins: []string{
			// Standard Shell Builtins
			"alias", "bg", "bind", "break", "builtin", "case", "cd", "command",
			"compgen", "complete", "continue", "declare", "dirs", "disown", "echo",
			"enable", "eval", "exec", "exit", "export", "fc", "fg", "getopts",
			"hash", "help", "history", "if", "jobs", "kill", "let", "local",
			"logout", "popd", "printf", "pushd", "pwd", "read", "readonly",
			"return", "set", "shift", "shopt", "source", "suspend", "test",
			"times", "trap", "type", "typeset", "ulimit", "umask", "unalias",
			"unset", "wait", "while",
			// Bishop specific
			"@!config", "@!new", "@!tokens", "@!subagents", "@!reload-subagents", "@!coach",
		},
	}
}

func (d *DocumentationCompleter) init() {
	d.initOnce.Do(func() {
		d.scanManPages()
		d.scanInfoPages()
	})
}

// GetCompletions returns completions for documentation commands
func (d *DocumentationCompleter) GetCompletions(command string, args []string, line string, pos int) ([]shellinput.CompletionCandidate, bool) {
	if command != "man" && command != "info" && command != "help" {
		return nil, false
	}

	// Initialize scanning lazily
	d.init()

	var candidates []shellinput.CompletionCandidate

	switch command {
	case "man":
		candidates = d.completeMan(args)
	case "info":
		candidates = d.completeInfo(args)
	case "help":
		candidates = d.completeHelp(args)
	}

	return candidates, true
}

func (d *DocumentationCompleter) completeMan(args []string) []shellinput.CompletionCandidate {
	prefix := ""
	if len(args) > 0 {
		prefix = args[len(args)-1]
	}

	// Check for section argument
	var sectionFilter string
	if len(args) > 1 {
		// If we have 2+ args, the first one might be a section
		possibleSection := args[0]
		if isSection(possibleSection) {
			sectionFilter = possibleSection
			// The prefix is the last argument
			prefix = args[len(args)-1]
		}
	}

	var candidates []shellinput.CompletionCandidate

	// Iterate over cached man pages
	for name, sections := range d.manPages {
		if strings.HasPrefix(name, prefix) {
			// If a section filter is active, check if this page has that section
			if sectionFilter != "" {
				hasSection := false
				for _, s := range sections {
					if strings.HasPrefix(s, sectionFilter) {
						hasSection = true
						break
					}
				}
				if !hasSection {
					continue
				}
			}

			candidates = append(candidates, shellinput.CompletionCandidate{
				Value:       name,
				Description: "Manual page (" + strings.Join(sections, ", ") + ")",
			})
		}
	}

	sortCandidates(candidates)
	return candidates
}

func (d *DocumentationCompleter) completeInfo(args []string) []shellinput.CompletionCandidate {
	prefix := ""
	if len(args) > 0 {
		prefix = args[len(args)-1]
	}

	var candidates []shellinput.CompletionCandidate
	for _, page := range d.infoPages {
		if strings.HasPrefix(page, prefix) {
			candidates = append(candidates, shellinput.CompletionCandidate{
				Value:       page,
				Description: "Info page",
			})
		}
	}

	sortCandidates(candidates)
	return candidates
}

func (d *DocumentationCompleter) completeHelp(args []string) []shellinput.CompletionCandidate {
	prefix := ""
	if len(args) > 0 {
		prefix = args[len(args)-1]
	}

	var candidates []shellinput.CompletionCandidate
	seen := make(map[string]bool)

	// 1. Builtins
	for _, b := range d.builtins {
		if strings.HasPrefix(b, prefix) {
			candidates = append(candidates, shellinput.CompletionCandidate{
				Value:       b,
				Description: "Shell Builtin",
			})
			seen[b] = true
		}
	}

	// 2. Info Pages
	for _, page := range d.infoPages {
		if _, exists := seen[page]; exists {
			continue
		}
		if strings.HasPrefix(page, prefix) {
			candidates = append(candidates, shellinput.CompletionCandidate{
				Value:       page,
				Description: "Info page",
			})
			seen[page] = true
		}
	}

	// 3. Man Pages
	for name, sections := range d.manPages {
		if _, exists := seen[name]; exists {
			continue
		}
		if strings.HasPrefix(name, prefix) {
			candidates = append(candidates, shellinput.CompletionCandidate{
				Value:       name,
				Description: "Manual page (" + strings.Join(sections, ", ") + ")",
			})
		}
	}

	sortCandidates(candidates)
	return candidates
}

func (d *DocumentationCompleter) scanManPages() {
	d.manPages = make(map[string][]string)

	// Get MANPATH or default
	paths := getEnvPaths("MANPATH", []string{
		"/usr/share/man",
		"/usr/local/share/man",
		filepath.Join(os.Getenv("HOME"), ".local/share/man"),
	})

	// Regex to match man page files: name.section.gz or name.section
	// e.g. ls.1.gz -> name=ls, section=1
	re := regexp.MustCompile(`^(.+)\.([0-9][a-zA-Z]*)(\.gz)?$`)

	for _, dir := range paths {
		_ = filepath.WalkDir(dir, func(path string, dEntry os.DirEntry, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if dEntry.IsDir() {
				// We need to descend into subdirectories like man1, man2, etc.
				return nil
			}

			// Parse filename
			matches := re.FindStringSubmatch(dEntry.Name())
			if matches != nil {
				name := matches[1]
				section := matches[2]

				d.manPages[name] = append(d.manPages[name], section)
			}
			return nil
		})
	}

	// Deduplicate sections
	for name, sections := range d.manPages {
		d.manPages[name] = uniqueStrings(sections)
	}
}

func (d *DocumentationCompleter) scanInfoPages() {
	d.infoPages = []string{}

	// Get INFOPATH or default
	paths := getEnvPaths("INFOPATH", []string{
		"/usr/share/info",
	})

	// Regex for info files
	re := regexp.MustCompile(`^(.+)\.info(-[0-9]+)?(\.gz)?$`)

	seen := make(map[string]bool)

	for _, dir := range paths {
		_ = filepath.WalkDir(dir, func(path string, dEntry os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if dEntry.IsDir() {
				return nil
			}

			name := dEntry.Name()
			if strings.HasSuffix(name, ".png") {
				return nil
			}

			// Parse
			matches := re.FindStringSubmatch(name)
			if matches != nil {
				baseName := matches[1]
				if !seen[baseName] {
					d.infoPages = append(d.infoPages, baseName)
					seen[baseName] = true
				}
			}
			return nil
		})
	}
}

// Helpers

func getEnvPaths(envVar string, defaults []string) []string {
	val := os.Getenv(envVar)
	if val == "" {
		return defaults
	}
	// Handle empty parts in MANPATH (:: means system default).
	// For simplicity, we just split and append defaults if we see ::?
	// Standard behavior is complicated. Here we just return the split list.
	// If the user sets MANPATH, they usually want to override.
	return strings.Split(val, string(os.PathListSeparator))
}

func isSection(s string) bool {
	if s == "" {
		return false
	}
	// Check if it starts with a digit
	return unicode.IsDigit(rune(s[0]))
}

func uniqueStrings(input []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range input {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	sort.Strings(list)
	return list
}

func sortCandidates(candidates []shellinput.CompletionCandidate) {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Value < candidates[j].Value
	})
}
