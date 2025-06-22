package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// execute is a helper function to run a command and capture its output.
func execute(args ...string) (string, error) {
	// Set testing environment variable
	oldTestingEnv := os.Getenv("TESTING")
	os.Setenv("TESTING", "1")
	defer os.Setenv("TESTING", oldTestingEnv)

	// Redirect stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	
	os.Stdout = wOut
	os.Stderr = wErr

	// Run the command
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()

	// Close writers
	wOut.Close()
	wErr.Close()
	
	// Restore stdout and stderr
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	
	// Read the output
	var stdoutBuf, stderrBuf bytes.Buffer
	io.Copy(&stdoutBuf, rOut)
	io.Copy(&stderrBuf, rErr)
	
	// Combine stdout and stderr
	combined := stdoutBuf.String() + stderrBuf.String()

	return combined, err
}

func TestGenerateKeyCommand(t *testing.T) {
	output, err := execute("generate-key")
	assert.NoError(t, err)
	assert.Contains(t, output, "🔑 Generated Encryption Key")
	assert.Contains(t, output, "📋 Team Distribution Instructions")
}

func TestDoctorCommand(t *testing.T) {
	// This test is expected to fail in CI where az-cli might not be logged in.
	// We are just checking that it runs and produces the expected sections.
	output, err := execute("doctor")

	// We expect an error because config and auth are likely not set up.
	assert.Error(t, err, "doctor command should fail in a clean test environment")

	assert.Contains(t, output, "--- Checking Dependencies ---")
	assert.Contains(t, output, "--- Checking Azure Authentication ---")
	assert.Contains(t, output, "--- Checking Configuration ---")
}

func TestInitCommandNoFlags(t *testing.T) {
	_, err := execute("init")
	assert.Error(t, err, "init should fail without required flags")
}

func TestHelpCommand(t *testing.T) {
	output, err := execute("help")
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(output, rootCmd.Long))
}

func TestVersionFlag(t *testing.T) {
	// Test the actual version flag now that it exists
	output, err := execute("--version")
	assert.NoError(t, err)
	assert.Contains(t, output, "env-sync")
}

func TestDoctorCommandWithFlags(t *testing.T) {
	t.Run("check specific component", func(t *testing.T) {
		// Test checking a specific component
		output, err := execute("doctor", "--check", "azure-cli")
		// This will likely error in test environment, but we can check the output format
		_ = err // Ignore error in test environment
		assert.Contains(t, output, "Checking Azure CLI")
	})

	t.Run("invalid component check", func(t *testing.T) {
		_, err := execute("doctor", "--check", "invalid-component")
		assert.Error(t, err)
	})

	t.Run("help shows examples", func(t *testing.T) {
		output, err := execute("doctor", "--help")
		assert.NoError(t, err)
		assert.Contains(t, output, "Examples:")
		assert.Contains(t, output, "--check azure-cli")
		assert.Contains(t, output, "--fix")
	})
}

func TestInstallDepsWithFlags(t *testing.T) {
	t.Run("install specific dependency", func(t *testing.T) {
		// Test the --only flag
		output, err := execute("install-deps", "--only", "azure-cli", "--yes")
		// In test environment this might fail, but we check the command parsing works
		// We expect either success or a specific error about the dependency
		if err != nil {
			// Should contain some indication it tried to work with azure-cli
			assert.True(t, strings.Contains(output, "azure-cli") || strings.Contains(output, "Azure CLI"))
		}
	})

	t.Run("invalid dependency name", func(t *testing.T) {
		_, err := execute("install-deps", "--only", "invalid-dep")
		assert.Error(t, err)
	})

	t.Run("help shows examples", func(t *testing.T) {
		output, err := execute("install-deps", "--help")
		assert.NoError(t, err)
		assert.Contains(t, output, "--only azure-cli")
		assert.Contains(t, output, "Examples:")
	})
}
