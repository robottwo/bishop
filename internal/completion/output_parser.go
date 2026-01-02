package completion

import (
	"encoding/json"
	"strings"

	"github.com/robottwo/bishop/pkg/shellinput"
)

type JsonCandidate struct {
	Value       string `json:"Value"`
	Display     string `json:"Display"`
	Description string `json:"Description"`
}

func ParseExternalCompletionOutput(output string) ([]shellinput.CompletionCandidate, error) {
	trimmedOutput := strings.TrimSpace(output)
	if trimmedOutput == "" {
		return []shellinput.CompletionCandidate{}, nil
	}

	// Try to parse as JSON first (Carapace style)
	if strings.HasPrefix(trimmedOutput, "[") {
		var candidates []shellinput.CompletionCandidate
		// Try parsing as simple list of strings
		var stringList []string
		if err := json.Unmarshal([]byte(trimmedOutput), &stringList); err == nil {
			for _, s := range stringList {
				candidates = append(candidates, shellinput.CompletionCandidate{Value: s})
			}
			return candidates, nil
		}

		// Try parsing as list of objects with Value/Display/Description
		var objList []JsonCandidate
		if err := json.Unmarshal([]byte(trimmedOutput), &objList); err == nil {
			for _, o := range objList {
				candidates = append(candidates, shellinput.CompletionCandidate{
					Value:       o.Value,
					Display:     o.Display,
					Description: o.Description,
				})
			}
			return candidates, nil
		}
	} else if strings.HasPrefix(trimmedOutput, "{") {
		// Try parsing as a single object with Value/Display/Description
		var obj JsonCandidate
		if err := json.Unmarshal([]byte(trimmedOutput), &obj); err == nil {
			return []shellinput.CompletionCandidate{{
				Value:       obj.Value,
				Display:     obj.Display,
				Description: obj.Description,
			}}, nil
		}
	}

	// Parse line-by-line (Bash/Zsh style)
	lines := strings.Split(trimmedOutput, "\n")
	completions := make([]shellinput.CompletionCandidate, 0, len(lines))
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}

		var candidate shellinput.CompletionCandidate

		// Try to parse as JSON (single object or array)
		if strings.HasPrefix(l, "{") {
			// Only try parsing as a single JSON object
			var obj JsonCandidate
			if err := json.Unmarshal([]byte(l), &obj); err == nil && obj.Value != "" {
				candidate.Value = obj.Value
				candidate.Display = obj.Display
				candidate.Description = obj.Description
				completions = append(completions, candidate)
				continue
			}
		} else if strings.HasPrefix(l, "[") {
			// Only try parsing as arrays
			// Try parsing as an array of JSON objects
			var objList []JsonCandidate
			if err := json.Unmarshal([]byte(l), &objList); err == nil && len(objList) > 0 {
				for _, o := range objList {
					completions = append(completions, shellinput.CompletionCandidate{
						Value:       o.Value,
						Display:     o.Display,
						Description: o.Description,
					})
				}
				continue
			}
			// Try parsing as a simple list of strings
			var stringList []string
			if err := json.Unmarshal([]byte(l), &stringList); err == nil && len(stringList) > 0 {
				for _, s := range stringList {
					completions = append(completions, shellinput.CompletionCandidate{Value: s})
				}
				continue
			}
		}

		// Check for tab delimiter (Value\tDescription)
		if strings.Contains(l, "\t") {
			parts := strings.SplitN(l, "\t", 2)
			candidate.Value = parts[0]
			if len(parts) > 1 {
				candidate.Description = parts[1]
			}
		} else if strings.Contains(l, ":") {
			// Check for colon delimiter (Value:Description) - Zsh style
			// Skip splitting if it looks like a URL, Windows path, or IPv6 literal.
			if looksLikeColonValue(l) {
				candidate.Value = l
			} else {
				parts := strings.SplitN(l, ":", 2)
				candidate.Value = parts[0]
				if len(parts) > 1 {
					candidate.Description = parts[1]
				}
			}
		} else {
			// Plain value
			candidate.Value = l
		}

		completions = append(completions, candidate)
	}

	return completions, nil
}

func looksLikeColonValue(value string) bool {
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "ssh://") {
		return true
	}

	if len(value) >= 3 && isAlpha(value[0]) && value[1] == ':' && (value[2] == '\\' || value[2] == '/') {
		return true
	}

	token := value
	if idx := strings.IndexAny(value, " \t"); idx != -1 {
		token = value[:idx]
	}

	return strings.Count(token, ":") > 1
}

func isAlpha(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
}
