package deps

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// This test is limited because we can't easily mock exec.LookPath or the OS
// without a more complex setup. We can primarily test the logic based on the
// current OS. This test is more of a sanity check.
func TestGetPackageManager(t *testing.T) {
	// This will only run the test for the GOOS of the machine running the test.
	// To test all OSs, we would need to build and run the test on each.
	t.Logf("Running test on OS: %s", runtime.GOOS)

	pm, err := getPackageManager()

	switch runtime.GOOS {
	case "darwin":
		// On macOS, we expect brew or an error
		if err == nil {
			assert.Equal(t, "brew", pm, "Expected 'brew' on macOS if anything is found")
		} else {
			assert.Error(t, err, "Expected an error if no package manager is found")
		}
	case "linux":
		// On Linux, it could be apt, yum, or dnf
		if err == nil {
			assert.Contains(t, []string{"apt", "yum", "dnf"}, pm, "Expected a known Linux package manager")
		} else {
			assert.Error(t, err, "Expected an error if no package manager is found")
		}
	case "windows":
		// On Windows, it could be winget or choco
		if err == nil {
			assert.Contains(t, []string{"winget", "choco"}, pm, "Expected a known Windows package manager")
		} else {
			assert.Error(t, err, "Expected an error if no package manager is found")
		}
	default:
		t.Skipf("Skipping package manager test on unsupported OS: %s", runtime.GOOS)
	}
}
