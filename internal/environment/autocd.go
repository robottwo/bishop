package environment

import (
	"strings"

	"mvdan.cc/sh/v3/interp"
)

// IsAutocdEnabled checks if BISH_AUTOCD is enabled
// Autocd allows users to change directories by typing just the path
func IsAutocdEnabled(runner *interp.Runner) bool {
	val := strings.ToLower(runner.Vars["BISH_AUTOCD"].String())
	return val == "1" || val == "true" || val == "yes" || val == "on"
}

// IsAutocdVerbose checks if BISH_AUTOCD_VERBOSE is enabled
// When enabled, prints "cd <path>" when autocd triggers
func IsAutocdVerbose(runner *interp.Runner) bool {
	val := strings.ToLower(runner.Vars["BISH_AUTOCD_VERBOSE"].String())
	return val == "1" || val == "true" || val == "yes" || val == "on"
}
