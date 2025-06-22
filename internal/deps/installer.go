package deps

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/lliamscholtz/env-sync/internal/utils"
)

// Dependency represents a command-line tool dependency.
type Dependency struct {
	Name        string
	Command     string
	Version     string
	InstallCmd  map[string]string
	DownloadURL map[string]string
	Required    bool
}

// DependencyManager handles checking and installing dependencies.
type DependencyManager struct {
	OS             string
	Arch           string
	PackageManager string
}

// NewDependencyManager creates a new dependency manager for the current OS.
func NewDependencyManager() (*DependencyManager, error) {
	pm, err := getPackageManager()
	if err != nil {
		utils.PrintWarning("Could not detect package manager: %v\n", err)
	}
	return &DependencyManager{
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		PackageManager: pm,
	}, nil
}

var dependencies = []Dependency{
	{
		Name:     "Azure CLI",
		Command:  "az",
		Version:  "2.50.0",
		Required: true,
		InstallCmd: map[string]string{
			"linux-apt":      "curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash",
			"linux-yum":      "sudo rpm --import https://packages.microsoft.com/keys/microsoft.asc && echo -e '[azure-cli]\nname=Azure CLI\nbaseurl=https://packages.microsoft.com/yumrepos/azure-cli\nenabled=1\ngpgcheck=1\ngpgkey=https://packages.microsoft.com/keys/microsoft.asc' | sudo tee /etc/yum.repos.d/azure-cli.repo && sudo yum install -y azure-cli",
			"darwin-brew":    "brew install azure-cli",
			"windows-winget": "winget install --id Microsoft.AzureCLI -e",
			"windows-choco":  "choco install azure-cli -y",
		},
		DownloadURL: map[string]string{
			"linux":   "https://docs.microsoft.com/en-us/cli/azure/install-azure-cli-linux",
			"darwin":  "https://docs.microsoft.com/en-us/cli/azure/install-azure-cli-macos",
			"windows": "https://docs.microsoft.com/en-us/cli/azure/install-azure-cli-windows",
		},
	},
	{
		Name:     "Tilt",
		Command:  "tilt",
		Version:  "0.30.0",
		Required: false,
		InstallCmd: map[string]string{
			"linux-curl":     "curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash",
			"darwin-brew":    "brew install tilt",
			"windows-choco":  "choco install tilt -y",
			"windows-winget": "winget install --id Tilt.Tilt -e",
		},
		DownloadURL: map[string]string{
			"linux":   "https://docs.tilt.dev/install.html#linux",
			"darwin":  "https://docs.tilt.dev/install.html#macos",
			"windows": "https://docs.tilt.dev/install.html#windows",
		},
	},
}

// CheckDependencies verifies if all dependencies are installed.
func (dm *DependencyManager) CheckDependencies() (missing []Dependency, installed []Dependency) {
	utils.PrintInfo("Checking dependencies...\n")
	for _, dep := range dependencies {
		_, err := exec.LookPath(dep.Command)
		if err != nil {
			utils.PrintWarning("-> Missing dependency: %s\n", dep.Name)
			missing = append(missing, dep)
		} else {
			utils.PrintSuccess("-> Found dependency: %s\n", dep.Name)
			installed = append(installed, dep)
		}
	}
	return
}

// InstallDependency attempts to install a single dependency.
func (dm *DependencyManager) InstallDependency(dep Dependency) error {
	key := dm.OS + "-" + dm.PackageManager
	if dep.Name == "Tilt" && (dm.OS == "linux" || (dm.OS == "darwin" && dm.PackageManager != "brew")) {
		key = dm.OS + "-curl" // Tilt has a universal curl installer for Linux/macOS
	}

	cmdStr, supported := dep.InstallCmd[key]
	if !supported {
		url, hasURL := dep.DownloadURL[dm.OS]
		if !hasURL {
			return fmt.Errorf("unsupported OS '%s' for installing %s", dm.OS, dep.Name)
		}
		return fmt.Errorf("package manager '%s' not supported for installing %s on %s. Please install manually from: %s", dm.PackageManager, dep.Name, dm.OS, url)
	}

	utils.PrintInfo("Attempting to install %s with command: %s\n", dep.Name, cmdStr)

	parts := strings.Fields(cmdStr)
	cmd := exec.Command(parts[0], parts[1:]...)
	if strings.Contains(cmdStr, "|") || strings.Contains(cmdStr, "&&") {
		// For complex shell commands, use a shell to execute
		shell := "/bin/sh"
		if dm.OS == "windows" {
			shell = "powershell"
			cmd = exec.Command(shell, "-Command", cmdStr)
		} else {
			cmd = exec.Command(shell, "-c", cmdStr)
		}
	} else {
		cmd = exec.Command(parts[0], parts[1:]...)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin // For prompts like sudo password

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	utils.PrintSuccess("%s installed successfully.\n", dep.Name)
	return nil
}

// PromptInstallDependencies asks the user if they want to install missing dependencies.
func PromptInstallDependencies(missing []Dependency) bool {
	if len(missing) == 0 {
		return true
	}

	fmt.Print("Some dependencies are missing. Would you like to install them now? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(input)) == "y"
}

// InstallAllMissing installs all dependencies that were found to be missing.
func (dm *DependencyManager) InstallAllMissing(missing []Dependency) error {
	if len(missing) == 0 {
		utils.PrintSuccess("All dependencies are already installed.\n")
		return nil
	}

	for _, dep := range missing {
		if err := dm.InstallDependency(dep); err != nil {
			utils.PrintError("Failed to install %s: %v\n", dep.Name, err)
			utils.PrintInfo("Please try installing it manually from: %s\n", dep.DownloadURL[dm.OS])
			if dep.Required {
				return fmt.Errorf("required dependency %s could not be installed", dep.Name)
			}
		}
	}
	return nil
}

// EnsureDependencies is a high-level function to run the full dependency check and installation flow.
func EnsureDependencies(yes bool) error {
	dm, err := NewDependencyManager()
	if err != nil {
		return fmt.Errorf("failed to initialize dependency manager: %w", err)
	}

	missing, _ := dm.CheckDependencies()
	if len(missing) == 0 {
		utils.PrintSuccess("All dependencies are in place.\n")
		return nil
	}

	install := yes
	if !install {
		install = PromptInstallDependencies(missing)
	}

	if install {
		return dm.InstallAllMissing(missing)
	}

	for _, dep := range missing {
		if dep.Required {
			return fmt.Errorf("missing required dependency: %s. Please install it to continue", dep.Name)
		}
	}

	utils.PrintWarning("Continuing without optional dependencies.\n")
	return nil
}

// getPackageManager determines the package manager available on the system.
func getPackageManager() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("brew"); err == nil {
			return "brew", nil
		}
	case "linux":
		if _, err := exec.LookPath("apt-get"); err == nil {
			return "apt", nil
		}
		if _, err := exec.LookPath("yum"); err == nil {
			return "yum", nil
		}
		if _, err := exec.LookPath("dnf"); err == nil {
			return "dnf", nil
		}
	case "windows":
		if _, err := exec.LookPath("winget"); err == nil {
			return "winget", nil
		}
		if _, err := exec.LookPath("choco"); err == nil {
			return "choco", nil
		}
	}
	return "", fmt.Errorf("no supported package manager found")
}
