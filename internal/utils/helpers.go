package utils

import (
	"fmt"
	"os"
	
	"github.com/fatih/color"
)

// PrintSuccess prints a success message.
func PrintSuccess(format string, a ...interface{}) {
	if os.Getenv("TESTING") == "1" {
		fmt.Fprintf(os.Stdout, format, a...)
	} else {
		color.Green(format, a...)
	}
}

// PrintError prints an error message and exits.
func PrintError(format string, a ...interface{}) {
	if os.Getenv("TESTING") == "1" {
		fmt.Fprintf(os.Stderr, format, a...)
	} else {
		color.Red(format, a...)
	}
}

// PrintInfo prints an informational message.
func PrintInfo(format string, a ...interface{}) {
	if os.Getenv("TESTING") == "1" {
		fmt.Fprintf(os.Stdout, format, a...)
	} else {
		color.Blue(format, a...)
	}
}

// PrintWarning prints a warning message.
func PrintWarning(format string, a ...interface{}) {
	if os.Getenv("TESTING") == "1" {
		fmt.Fprintf(os.Stdout, format, a...)
	} else {
		color.Yellow(format, a...)
	}
}
