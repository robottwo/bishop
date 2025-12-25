//go:build !windows
// +build !windows

package bash

// reconstructWindowsPath attempts to reconstruct a Windows path that has lost its backslashes
// This is a stub implementation for non-Windows platforms that just returns the input unchanged
func reconstructWindowsPath(malformedPath string) string {
	return malformedPath
}
