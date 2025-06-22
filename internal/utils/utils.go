package utils

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// IsDebug returns true if the ENVSYNC_DEBUG environment variable is set.
func IsDebug() bool {
	return os.Getenv("ENVSYNC_DEBUG") != ""
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