package completion

import (
	"embed"
)

// CompletionData contains all embedded YAML completion configuration files.
// These files are embedded at compile time and provide default completions
// for common CLI tools.
//
//go:embed data/*.yaml
var CompletionData embed.FS
