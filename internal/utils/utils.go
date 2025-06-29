package utils

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var debugMode bool

// IsDebug returns true if the ENVSYNC_DEBUG environment variable is set or debug mode is enabled.
func IsDebug() bool {
	return debugMode || os.Getenv("ENVSYNC_DEBUG") != ""
}

// SetDebugMode enables or disables debug mode programmatically.
func SetDebugMode(enabled bool) {
	debugMode = enabled
}

// PrintDebug prints a debug message if debugging is enabled.
func PrintDebug(format string, a ...interface{}) {
	if IsDebug() {
		if os.Getenv("TESTING") == "1" {
			fmt.Fprintf(os.Stderr, "DEBUG: "+format, a...)
		} else {
			color.Yellow("DEBUG: "+format, a...)
		}
	}
} 