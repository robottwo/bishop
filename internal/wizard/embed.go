package wizard

import _ "embed"

//go:embed bishrc.template
var bishrcTemplate []byte

// BishrcTemplate returns the default ~/.bishrc template content.
// Used by both the setup wizard and the config UI when creating a fresh .bishrc.
func BishrcTemplate() []byte {
	return bishrcTemplate
}
