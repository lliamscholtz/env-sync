package utils

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintFunctions(t *testing.T) {
	// Set testing environment to use standard output
	oldTestingEnv := os.Getenv("TESTING")
	os.Setenv("TESTING", "1")
	defer os.Setenv("TESTING", oldTestingEnv)

	t.Run("PrintSuccess", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		PrintSuccess("Test success: %s", "message")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.Contains(t, output, "Test success: message")
	})

	t.Run("PrintInfo", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		PrintInfo("Test info: %s", "message")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.Contains(t, output, "Test info: message")
	})

	t.Run("PrintWarning", func(t *testing.T) {
		// Capture stdout (PrintWarning writes to stdout in testing mode)
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		PrintWarning("Test warning: %s", "message")

		w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.Contains(t, output, "Test warning: message")
	})

	t.Run("PrintError", func(t *testing.T) {
		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		PrintError("Test error: %s", "message")

		w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.Contains(t, output, "Test error: message")
	})
}

func TestPrintDebug(t *testing.T) {
	// Set testing environment
	oldTestingEnv := os.Getenv("TESTING")
	os.Setenv("TESTING", "1")
	defer os.Setenv("TESTING", oldTestingEnv)

	t.Run("debug enabled", func(t *testing.T) {
		// Enable debug
		oldDebugEnv := os.Getenv("ENVSYNC_DEBUG")
		os.Setenv("ENVSYNC_DEBUG", "1")
		defer os.Setenv("ENVSYNC_DEBUG", oldDebugEnv)

		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		PrintDebug("Debug message: %s", "test")

		w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		assert.Contains(t, output, "Debug message: test")
	})

	t.Run("debug disabled", func(t *testing.T) {
		// Disable debug
		oldDebugEnv := os.Getenv("ENVSYNC_DEBUG")
		os.Setenv("ENVSYNC_DEBUG", "")
		defer os.Setenv("ENVSYNC_DEBUG", oldDebugEnv)

		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		PrintDebug("Debug message: %s", "test")

		w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		// Should be empty when debug is disabled
		assert.Empty(t, strings.TrimSpace(output))
	})
} 