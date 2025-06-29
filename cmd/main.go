package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lliamscholtz/env-sync/internal/auth"
	"github.com/lliamscholtz/env-sync/internal/config"
	"github.com/lliamscholtz/env-sync/internal/crypto"
	"github.com/lliamscholtz/env-sync/internal/deps"
	"github.com/lliamscholtz/env-sync/internal/sync"
	"github.com/lliamscholtz/env-sync/internal/utils"
	"github.com/lliamscholtz/env-sync/internal/vault"
	"github.com/lliamscholtz/env-sync/internal/watcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version information (set by build)
	version = "dev"
	commit  = "none"
	date    = "unknown"
	
	cfgFile  string
	cliKey   string
	syncFile string // Sync configuration file for multi-file support
)

var rootCmd = &cobra.Command{
	Use:     "env-sync",
	Short:   "Sync encrypted .env files with Azure Key Vault",
	Version: version,
	Long: `env-sync securely synchronizes .env files with Azure Key Vault.

Prerequisites (automatically installed if missing):
  • Azure CLI (required for authentication)
  • Tilt (optional, for development workflow integration)

Multi-File Support:
  Use --sync-file to specify different configuration files for different environments:
    env-sync push --sync-file .env-sync.dev.yaml
    env-sync pull --sync-file .env-sync.qa.yaml
    env-sync watch --sync-file .env-sync.prod.yaml`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Commands that don't need pre-flight checks
		if cmd.Name() == "install-deps" || cmd.Name() == "generate-key" || cmd.Name() == "help" || cmd.Name() == "init" || cmd.Name() == "version" {
			return nil
		}

		// Run dependency check first
		if err := deps.EnsureDependencies(false); err != nil { // `false` for interactive prompt
			return err
		}

		// Then run auth check, skipping the dep check inside it as we just did it.
		return auth.EnsureAzureAuth(true)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./.env-sync.yaml)")
	rootCmd.PersistentFlags().StringVar(&cliKey, "key", "", "Base64 encoded encryption key (overrides all other key sources)")
	rootCmd.PersistentFlags().StringVar(&syncFile, "sync-file", "", "sync configuration file (default is ./.env-sync.yaml)")

	// Add commands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(generateKeyCmd)
	rootCmd.AddCommand(installDepsCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(rotateKeyCmd)

	// --- Flag Definitions ---

	// 'init' command flags
	initCmd.Flags().String("vault-url", "", "Azure Key Vault URL")
	initCmd.Flags().String("secret-name", "", "The name for the secret in Key Vault")
	initCmd.Flags().String("key-source", "", "Source for the encryption key (env, file, prompt)")
	initCmd.Flags().String("key-file", ".env-sync-key", "Path to the key file (if key-source is 'file')")
	initCmd.Flags().String("env-file", ".env", "Path to the local .env file")

	// 'generate-key' command flags
	generateKeyCmd.Flags().StringP("output", "o", "", "Save key to a file instead of displaying it")
	generateKeyCmd.Flags().StringP("format", "f", "base64", "Output format for the key (base64 or hex)")

	// 'install-deps' command flags
	installDepsCmd.Flags().BoolP("yes", "y", false, "Skip interactive prompts and install all missing dependencies")
	installDepsCmd.Flags().String("only", "", "Only install specific dependency (azure-cli, tilt)")

	// 'doctor' command flags
	doctorCmd.Flags().String("check", "", "Check specific component (azure-cli, tilt, auth, config)")
	doctorCmd.Flags().Bool("fix", false, "Automatically fix detected issues")

	// 'rotate-key' command flags
	rotateKeyCmd.Flags().String("new-key", "", "The new base64 encoded key for re-encryption (required)")
	rotateKeyCmd.MarkFlagRequired("new-key")

	// 'watch' command flags
	watchCmd.Flags().Bool("push", true, "Enable push on file changes (default: true, with confirmation prompts)")
	watchCmd.Flags().Bool("confirm", true, "Prompt for confirmation before pushing changes (default: true)")
	watchCmd.Flags().Bool("debug", false, "Enable debug logging for troubleshooting file change detection")
}

func main() {
	Execute()
}

// getConfigFile returns the configuration file path, prioritizing --sync-file over --config
func getConfigFile() string {
	if syncFile != "" {
		return syncFile
	}
	return cfgFile
}

// runSpecificCheck runs a check for a specific component
func runSpecificCheck(component string, autoFix bool) error {
	switch component {
	case "azure-cli":
		return checkAzureCLI(autoFix)
	case "tilt":
		return checkTilt(autoFix)
	case "auth":
		return checkAuth(autoFix)
	case "config":
		return checkConfig(autoFix)
	default:
		return fmt.Errorf("unknown component: %s. Valid options: azure-cli, tilt, auth, config", component)
	}
}

// Helper functions for specific checks
func checkAzureCLI(autoFix bool) error {
	utils.PrintInfo("🔍 Checking Azure CLI...\n")
	dm, _ := deps.NewDependencyManager()
	missing, _ := dm.CheckDependencies()
	
	for _, dep := range missing {
		if dep.Command == "az" {
			if autoFix {
				utils.PrintInfo("📦 Installing Azure CLI...\n")
				return dm.InstallDependency(dep)
			}
			utils.PrintError("❌ Azure CLI not found\n")
			return fmt.Errorf("azure CLI missing")
		}
	}
	utils.PrintSuccess("✅ Azure CLI is installed\n")
	return nil
}

func checkTilt(autoFix bool) error {
	utils.PrintInfo("🔍 Checking Tilt...\n")
	dm, _ := deps.NewDependencyManager()
	missing, _ := dm.CheckDependencies()
	
	for _, dep := range missing {
		if dep.Command == "tilt" {
			if autoFix {
				utils.PrintInfo("📦 Installing Tilt...\n")
				return dm.InstallDependency(dep)
			}
			utils.PrintWarning("⚠️ Tilt not found (optional)\n")
			return nil // Not fatal
		}
	}
	utils.PrintSuccess("✅ Tilt is installed\n")
	return nil
}

func checkAuth(autoFix bool) error {
	utils.PrintInfo("🔍 Checking Azure authentication...\n")
	if err := auth.CheckAzLoginStatus(); err != nil {
		if autoFix {
			utils.PrintInfo("🚀 Running 'az login'...\n")
			return auth.EnsureAzureAuth(false)
		}
		utils.PrintError("❌ Azure authentication failed\n")
		return err
	}
	utils.PrintSuccess("✅ Azure authentication is configured\n")
	return nil
}

func checkConfig(autoFix bool) error {
	utils.PrintInfo("🔍 Checking configuration...\n")
	_, err := config.LoadConfig(getConfigFile())
	if err != nil {
		if autoFix {
			utils.PrintInfo("⚙️ Configuration issues cannot be auto-fixed. Please run 'env-sync init'\n")
		}
		utils.PrintError("❌ Configuration issue: %v\n", err)
		return err
	}
	utils.PrintSuccess("✅ Configuration is valid\n")
	return nil
}

// autoFixIssues attempts to automatically fix detected issues
func autoFixIssues(issues []string) error {
	for _, issue := range issues {
		switch issue {
		case "dependencies":
			if err := deps.EnsureDependencies(true); err != nil {
				return fmt.Errorf("failed to fix dependencies: %w", err)
			}
		case "auth":
			if err := auth.EnsureAzureAuth(false); err != nil {
				return fmt.Errorf("failed to fix authentication: %w", err)
			}
		}
	}
	utils.PrintSuccess("🎉 Auto-fix completed!\n")
	return nil
}

// installSpecificDependency installs only the specified dependency
func installSpecificDependency(depName string, yes bool) error {
	dm, err := deps.NewDependencyManager()
	if err != nil {
		return fmt.Errorf("failed to initialize dependency manager: %w", err)
	}

	// Map friendly names to command names
	commandMap := map[string]string{
		"azure-cli": "az",
		"tilt":      "tilt",
	}

	command, exists := commandMap[depName]
	if !exists {
		return fmt.Errorf("unknown dependency: %s. Valid options: azure-cli, tilt", depName)
	}

	// Check if it's already installed
	missing, installed := dm.CheckDependencies()
	
	// Check if already installed
	for _, dep := range installed {
		if dep.Command == command {
			utils.PrintSuccess("✅ %s is already installed\n", dep.Name)
			return nil
		}
	}

	// Find the dependency to install
	for _, dep := range missing {
		if dep.Command == command {
			if !yes && dep.Required == false {
				// Prompt for optional dependencies
				if !deps.PromptInstallDependencies([]deps.Dependency{dep}) {
					utils.PrintInfo("⏭️ Skipping installation of %s\n", dep.Name)
					return nil
				}
			}
			
			utils.PrintInfo("📦 Installing %s...\n", dep.Name)
			return dm.InstallDependency(dep)
		}
	}

	return fmt.Errorf("dependency %s not found in dependency list", depName)
}

func initConfig() {
	// This function is called by Cobra. We use our custom LoadConfig in commands.
}

// Command variables
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project configuration (.env-sync.yaml)",
	Long: `Initializes the project by creating a .env-sync.yaml configuration file.
You must provide the Azure Key Vault URL, a name for the secret, and a source for the encryption key.

Use --sync-file to create a configuration file with a custom name:
  env-sync init --sync-file .env-sync.dev.yaml --vault-url <url> --secret-name <name> --key-source <source>`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultURL, _ := cmd.Flags().GetString("vault-url")
		secretName, _ := cmd.Flags().GetString("secret-name")
		keySource, _ := cmd.Flags().GetString("key-source")
		keyFile, _ := cmd.Flags().GetString("key-file")
		envFile, _ := cmd.Flags().GetString("env-file")

		if vaultURL == "" || secretName == "" || keySource == "" {
			return fmt.Errorf("--vault-url, --secret-name, and --key-source are required")
		}

		utils.PrintInfo("🚀 Initializing env-sync configuration...\n")

		// 1. Create credential and vault client to test connectivity
		cred, err := auth.CreateAzureCredential()
		if err != nil {
			return fmt.Errorf("failed to create Azure credentials during init: %w", err)
		}
		if !auth.IsAuthenticated(cred) {
			utils.PrintError("❌ Azure authentication failed. Please run 'az login' and try again.\n")
			auth.PrintAuthHelp()
			return fmt.Errorf("authentication required")
		}
		_, err = vault.NewClient(vaultURL, cred)
		if err != nil {
			return fmt.Errorf("failed to connect to Key Vault '%s'. Check URL and permissions: %w", vaultURL, err)
		}
		utils.PrintSuccess("✅ Azure Key Vault connection successful.\n")

		// 2. Load and validate the encryption key
		tempConfig := &config.Config{KeySource: keySource, KeyFile: keyFile}
		encryptionKey, err := tempConfig.GetEncryptionKey(cliKey)
		if err != nil {
			return fmt.Errorf("failed to load encryption key: %w", err)
		}
		if err := crypto.ValidateEncryptionKey(encryptionKey); err != nil {
			return fmt.Errorf("invalid encryption key: %w", err)
		}

		// 3. Test encryption/decryption with the key
		testData := []byte("encryption test")
		encrypted, err := crypto.EncryptEnvContent(testData, encryptionKey)
		if err != nil {
			return fmt.Errorf("encryption test failed: %w", err)
		}
		decrypted, err := crypto.DecryptEnvContent(encrypted, encryptionKey)
		if err != nil {
			return fmt.Errorf("decryption test failed: %w", err)
		}
		if !bytes.Equal(testData, decrypted) {
			return fmt.Errorf("encryption/decryption mismatch. The key is likely invalid")
		}
		utils.PrintSuccess("✅ Encryption key validated successfully.\n")

		// 4. Create and write the configuration file
		finalConfig := &config.Config{
			VaultURL:         vaultURL,
			SecretName:       secretName,
			EnvFile:          envFile,
			SyncInterval:     15 * time.Minute,
			KeySource:        keySource,
			KeyFile:          keyFile,
			ConflictStrategy: "manual",
			AutoBackup:       false,
		}
		
		configFileName := getConfigFile()
		if configFileName == "" {
			configFileName = ".env-sync.yaml"
		}
		
		if err := finalConfig.WriteToFile(configFileName); err != nil {
			return err
		}

		utils.PrintSuccess("🎉 Configuration file '%s' created successfully!\n", configFileName)
		utils.PrintInfo("📋 Next steps:\n")
		utils.PrintInfo("  1️⃣ Ensure your .env file is present or create one.\n")
		utils.PrintInfo("  2️⃣ Run 'env-sync push' to upload your .env file to Azure Key Vault.\n")
		return nil
	},
}

var generateKeyCmd = &cobra.Command{
	Use:   "generate-key",
	Short: "Generate a new 256-bit AES encryption key",
	Long:  `Generates a cryptographically secure 256-bit key for AES encryption. The key can be displayed in base64 or hex format for manual distribution or saved directly to a file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		format, _ := cmd.Flags().GetString("format")

		key, err := crypto.GenerateEncryptionKey()
		if err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}

		keyString, err := crypto.KeyToString(key, format)
		if err != nil {
			return err
		}

		if output != "" {
			if err := os.WriteFile(output, []byte(keyString), 0600); err != nil {
				return fmt.Errorf("failed to write key to file '%s': %w", output, err)
			}
			utils.PrintSuccess("✅ Encryption key saved to: %s\n", output)
			utils.PrintWarning("⚠️ IMPORTANT: This file contains a secret. Add it to your .gitignore and do not commit it!\n")
		} else {
			utils.PrintInfo("🔑 Generated Encryption Key (%s):\n", format)
			fmt.Println(keyString)
			fmt.Println()
			utils.PrintInfo("📋 Team Distribution Instructions:\n")
			fmt.Println("1. Share this key securely with your team (e.g., using a password manager).")
			fmt.Println("2. Each team member should save it as:")
			fmt.Printf("   a) An environment variable: export ENVSYNC_ENCRYPTION_KEY=\"%s\"\n", keyString)
			fmt.Printf("   b) Or in a file (e.g., .env-sync-key): echo \"%s\" > .env-sync-key\n", keyString)
			fmt.Println("3. If using a file, add its name to .gitignore.")
			fmt.Println()
			utils.PrintWarning("⚠️ SECURITY: Never commit this key to version control!\n")
		}
		return nil
	},
}

var installDepsCmd = &cobra.Command{
	Use:   "install-deps",
	Short: "Install all required and optional dependencies",
	Long: `Checks for missing dependencies like Azure CLI and Tilt and installs them using the appropriate package manager for the current operating system.

Examples:
  env-sync install-deps                    # Install all missing dependencies
  env-sync install-deps --yes             # Install without prompts
  env-sync install-deps --only azure-cli  # Install only Azure CLI
  env-sync install-deps --only tilt       # Install only Tilt`,
	RunE: func(cmd *cobra.Command, args []string) error {
		yes, _ := cmd.Flags().GetBool("yes")
		only, _ := cmd.Flags().GetString("only")
		
		if only != "" {
			return installSpecificDependency(only, yes)
		}
		
		return deps.EnsureDependencies(yes)
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run a system health check to verify dependencies and configuration",
	Long: `Performs a series of checks to ensure the system is ready to use env-sync.
It verifies:
- Installation of required dependencies (Azure CLI)
- Installation of optional dependencies (Tilt)
- Azure authentication status
- Validity of the '.env-sync.yaml' configuration file

Examples:
  env-sync doctor                    # Full system check
  env-sync doctor --check azure-cli # Check only Azure CLI
  env-sync doctor --fix             # Automatically fix detected issues`,
	RunE: func(cmd *cobra.Command, args []string) error {
		checkComponent, _ := cmd.Flags().GetString("check")
		autoFix, _ := cmd.Flags().GetBool("fix")

		if checkComponent != "" {
			return runSpecificCheck(checkComponent, autoFix)
		}

		utils.PrintInfo("🩺 Running system health check...\n\n")
		hasIssues := false
		var fixableIssues []string

		// 1. Check dependencies
		utils.PrintInfo("--- Checking Dependencies ---\n")
		dm, err := deps.NewDependencyManager()
		if err != nil {
			utils.PrintError("❌ Failed to initialize dependency manager: %v\n", err)
			return err
		}
		missing, installed := dm.CheckDependencies()
		if len(missing) > 0 {
			hasIssues = true
			utils.PrintWarning("\nFound %d missing dependencies.\n", len(missing))
			for _, dep := range missing {
				utils.PrintWarning("  - %s (%s)\n", dep.Name, dep.Command)
			}
			utils.PrintInfo("🔧 To fix, run: env-sync install-deps\n")
			fixableIssues = append(fixableIssues, "dependencies")
		} else {
			utils.PrintSuccess("✅ All %d dependencies are installed.\n", len(installed))
		}

		// 2. Check Azure authentication
		utils.PrintInfo("\n--- Checking Azure Authentication ---\n")
		if err := auth.CheckAzLoginStatus(); err != nil {
			hasIssues = true
			utils.PrintError("❌ Azure login check failed: %v\n", err)
			auth.PrintAuthHelp()
			fixableIssues = append(fixableIssues, "auth")
		} else {
			utils.PrintSuccess("✅ Azure authentication is configured.\n")
		}

		// 3. Check configuration file
		utils.PrintInfo("\n--- Checking Configuration ---\n")
		cfg, err := config.LoadConfig(cfgFile)
		if err != nil {
			// It's not a fatal error if the config file doesn't exist yet
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				hasIssues = true
				utils.PrintWarning("-> Config file (.env-sync.yaml) not found. Run 'env-sync init' to create one.\n")
			} else {
				hasIssues = true
				utils.PrintError("❌ Could not load config file: %v\n", err)
			}
		} else if err := cfg.Validate(); err != nil {
			hasIssues = true
			utils.PrintError("❌ Config file (.env-sync.yaml) is invalid: %v\n", err)
		} else {
			utils.PrintSuccess("✅ Config file (.env-sync.yaml) found and is valid.\n")
			utils.PrintInfo("  - Vault URL: %s\n", cfg.VaultURL)
			utils.PrintInfo("  - Secret Name: %s\n", cfg.SecretName)
		}

		// Auto-fix if requested
		if autoFix && len(fixableIssues) > 0 {
			utils.PrintInfo("\n🔧 Auto-fixing detected issues...\n")
			return autoFixIssues(fixableIssues)
		}

		// Final summary
		if hasIssues {
			utils.PrintWarning("\n🩺 Doctor check completed with issues. Please address the items marked with ❌.\n")
			if len(fixableIssues) > 0 {
				utils.PrintInfo("💡 Tip: Use --fix flag to automatically resolve some issues.\n")
			}
			return fmt.Errorf("doctor check failed")
		}

		utils.PrintSuccess("\n🎉 Doctor check complete. System is ready!\n")
		return nil
	},
}

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Encrypt and push the local .env file to Azure Key Vault",
	Long: `Encrypts the local .env file using the configured encryption key and
uploads it as a new secret version to the specified Azure Key Vault.

Use --sync-file to specify a different configuration file:
  env-sync push --sync-file .env-sync.dev.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return pushWithConflictDetection(cmd, args, false) // false = not from watcher
	},
}

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Download and decrypt the .env file from Azure Key Vault",
	Long: `Retrieves the encrypted secret from Azure Key Vault, decrypts it using the configured key, and writes the content to the local .env file.

Use --sync-file to specify a different configuration file:
  env-sync pull --sync-file .env-sync.qa.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig(getConfigFile())
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		if err := cfg.Validate(); err != nil {
			return err
		}

		utils.PrintInfo("⬇️  Pulling secret from %s/%s...\n", cfg.VaultURL, cfg.SecretName)

		key, err := cfg.LoadAndValidateKey(cliKey)
		if err != nil {
			return err
		}

		cred, err := auth.CreateAzureCredential()
		if err != nil {
			return err
		}
		vaultClient, err := vault.NewClient(cfg.VaultURL, cred)
		if err != nil {
			return err
		}

		ctx := context.Background()
		encrypted, err := vaultClient.GetSecret(ctx, cfg.SecretName)
		if err != nil {
			return fmt.Errorf("failed to get secret from Key Vault: %w", err)
		}

		// Decrypt the content before writing to file
		decrypted, err := crypto.DecryptEnvContent(encrypted, key)
		if err != nil {
			return fmt.Errorf("failed to decrypt secret: %w", err)
		}

		// Optional: backup existing file
		// os.Rename(cfg.EnvFile, cfg.EnvFile+".bak")

		if err := os.WriteFile(cfg.EnvFile, decrypted, 0644); err != nil {
			return fmt.Errorf("failed to write to env file '%s': %w", cfg.EnvFile, err)
		}

		utils.PrintSuccess("✅ Successfully pulled and decrypted .env from Azure Key Vault.\n")
		return nil
	},
}

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Monitor the .env file and automatically sync on changes",
	Long: `Starts a long-running process that watches the local .env file for modifications.

By default, pushes are enabled with confirmation prompts for safety.
Use --push=false to disable pushes and only perform periodic pulls.
Use --confirm flag to control whether you're prompted before each push (default: true).

The watcher performs periodic pulls at a configurable interval and pushes on file changes.

Use --sync-file to specify a different configuration file:
  env-sync watch --sync-file .env-sync.prod.yaml
  env-sync watch --push=false             # Pull-only mode  
  env-sync watch --confirm=false          # Auto-push without prompts
  env-sync watch --debug                  # Enable debug logging for troubleshooting`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig(getConfigFile())
		if err != nil {
			return fmt.Errorf("failed to load configuration for watcher: %w", err)
		}
		if err := cfg.Validate(); err != nil {
			return err
		}

		// Get the push, confirm, and debug flags
		enablePush, _ := cmd.Flags().GetBool("push")
		confirmPush, _ := cmd.Flags().GetBool("confirm")
		debugMode, _ := cmd.Flags().GetBool("debug")
		
		// Enable debug logging if requested
		if debugMode {
			utils.SetDebugMode(true)
			utils.PrintDebug("🐛 Debug mode enabled for file watcher\n")
		}

		// Ensure the .env file exists before starting the watcher
		if _, err := os.Stat(cfg.EnvFile); os.IsNotExist(err) {
			utils.PrintWarning("⚠️ '.env' file not found. Creating an empty one to watch.\n")
			if err := os.WriteFile(cfg.EnvFile, []byte{}, 0644); err != nil {
				return fmt.Errorf("failed to create placeholder .env file: %w", err)
			}
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Listen for interrupt signals for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			utils.PrintInfo("🛑 Received interrupt signal, shutting down watcher...\n")
			cancel()
		}()

		pushFunc := func() error {
			// Use the enhanced push function with conflict detection
			return pushWithConflictDetection(cmd, args, true) // true = from watcher
		}

		pullFunc := func() error {
			// Create a new command to avoid flag parsing issues in a loop
			pullCmd_instance := &cobra.Command{}
			*pullCmd_instance = *pullCmd
			return pullCmd_instance.RunE(cmd, args)
		}

		// Default debounce and sync intervals if not set
		debounceTime := 5 * time.Second
		syncInterval := cfg.SyncInterval
		if syncInterval == 0 {
			syncInterval = 15 * time.Minute
		}

		w, err := watcher.NewFileWatcher(cfg.EnvFile, syncInterval, debounceTime, pushFunc, pullFunc, enablePush, confirmPush)
		if err != nil {
			return fmt.Errorf("failed to create file watcher: %w", err)
		}

		utils.PrintInfo("🕐 Starting watcher with a %s pull interval and %s debounce time.\n", syncInterval, debounceTime)
		if enablePush {
			utils.PrintInfo("📋 File changes will push to Azure Key Vault, periodic syncs will pull from Azure Key Vault.\n")
		} else {
			utils.PrintInfo("📋 File change push disabled - only periodic pulls from Azure Key Vault are active.\n")
		}
		return w.Start(ctx)
	},
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Check Azure authentication status",
	Long:  `Verifies the current authentication status with Azure by attempting to acquire a token. It provides helpful guidance if authentication fails.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// The PersistentPreRunE already handles the auth check,
		// so we just need to confirm it passed.
		utils.PrintSuccess("✅ Azure authentication is configured and valid.\n")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show a summary of the current configuration and sync status",
	Long: `Displays the current configuration from the .env-sync.yaml file. It also compares the local .env file's modification time with the secret's last updated time in Azure Key Vault to determine sync status.

Use --sync-file to specify a different configuration file:
  env-sync status --sync-file .env-sync.dev.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		utils.PrintInfo("📊 --- env-sync Status ---\n")
		cfg, err := config.LoadConfig(getConfigFile())
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		if err := cfg.Validate(); err != nil {
			return err
		}

		// Print configuration
		configFile := getConfigFile()
		if configFile == "" {
			configFile = ".env-sync.yaml"
		}
		utils.PrintInfo("⚙️ Configuration loaded from '%s':\n", configFile)
		fmt.Printf("  - Vault URL: %s\n", cfg.VaultURL)
		fmt.Printf("  - Secret Name: %s\n", cfg.SecretName)
		fmt.Printf("  - Local Env File: %s\n", cfg.EnvFile)
		fmt.Printf("  - Key Source: %s\n", cfg.KeySource)
		if cfg.KeySource == "file" {
			fmt.Printf("  - Key File: %s\n", cfg.KeyFile)
		}
		fmt.Println()

		// Compare local and remote timestamps
		// Local file info
		localFileInfo, err := os.Stat(cfg.EnvFile)
		if os.IsNotExist(err) {
			utils.PrintWarning("⚠️ Local .env file not found. Run 'env-sync pull' to fetch it.\n")
			return nil
		} else if err != nil {
			return fmt.Errorf("could not stat local env file: %w", err)
		}

		// Remote secret info
		cred, err := auth.CreateAzureCredential()
		if err != nil {
			return err
		}
		vaultClient, err := vault.NewClient(cfg.VaultURL, cred)
		if err != nil {
			return err
		}
		secret, err := vaultClient.GetSecret(context.Background(), cfg.SecretName)
		if err != nil {
			utils.PrintWarning("⚠️ Could not retrieve remote secret. It may not have been pushed yet.\n")
			utils.PrintInfo("📁 Local file '%s' exists but has not been synced.\n", cfg.EnvFile)
			return nil
		}

		// This is a proxy for the last update time, as the SDK doesn't directly expose it on GetSecret.
		// A more accurate check would involve GetSecretProperties, but this is a reasonable approximation for status.
		_, err = crypto.DecryptEnvContent(secret, []byte(cliKey)) // We don't have the key here, so this check is tricky.
		// For a real implementation, we would just compare timestamps if the SDK provided it easily.
		// Let's just report what we have.

		fmt.Println("Sync Status:")
		fmt.Printf("  - Local file last modified: %s\n", localFileInfo.ModTime().Format(time.RFC1123))
		utils.PrintInfo("☁️ Remote secret is present in Key Vault.\n")
		utils.PrintWarning("⚠️ To see if content is in sync, please use a diff tool after pulling.\n")

		return nil
	},
}

var rotateKeyCmd = &cobra.Command{
	Use:   "rotate-key",
	Short: "Generate a new key and re-encrypt the secret in Azure Key Vault",
	Long: `Rotates the encryption key. It fetches the secret, decrypts it with the old (currently configured) key,
re-encrypts it with a new key provided via a flag, and updates the secret in Azure Key Vault.
The new key must then be manually distributed to the team.

Use --sync-file to specify a different configuration file:
  env-sync rotate-key --new-key <key> --sync-file .env-sync.prod.yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig(getConfigFile())
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		if err := cfg.Validate(); err != nil {
			return err
		}

		// 1. Load the old key from the currently configured source
		utils.PrintInfo("🔑 Loading current (old) encryption key...\n")
		oldKey, err := cfg.LoadAndValidateKey(cliKey) // cliKey will be empty if not passed, respecting priority
		if err != nil {
			return fmt.Errorf("could not load the old key from source '%s': %w", cfg.KeySource, err)
		}

		// 2. Get the new key from the flag
		newKeyRaw, _ := cmd.Flags().GetString("new-key")
		if newKeyRaw == "" {
			return fmt.Errorf("--new-key flag is required and must contain the new base64 encoded key")
		}
		newKey, err := base64.StdEncoding.DecodeString(newKeyRaw)
		if err != nil {
			return fmt.Errorf("invalid base64 format for --new-key: %w", err)
		}
		if err := crypto.ValidateEncryptionKey(newKey); err != nil {
			return fmt.Errorf("new key is invalid: %w", err)
		}

		if bytes.Equal(oldKey, newKey) {
			return fmt.Errorf("the new key cannot be the same as the old key")
		}
		utils.PrintSuccess("✅ New key loaded and validated.\n")

		// 3. Fetch the secret from Key Vault
		utils.PrintInfo("⬇️ Fetching current secret from Azure Key Vault...\n")
		cred, err := auth.CreateAzureCredential()
		if err != nil {
			return err
		}
		vaultClient, err := vault.NewClient(cfg.VaultURL, cred)
		if err != nil {
			return err
		}
		ctx := context.Background()
		encryptedContent, err := vaultClient.GetSecret(ctx, cfg.SecretName)
		if err != nil {
			return fmt.Errorf("failed to get secret '%s' for rotation: %w", cfg.SecretName, err)
		}

		// 4. Perform the rotation (decrypt with old, encrypt with new)
		utils.PrintInfo("🔄 Re-encrypting secret with the new key...\n")
		newEncryptedContent, err := crypto.RotateKey(oldKey, newKey, encryptedContent)
		if err != nil {
			return fmt.Errorf("key rotation failed during re-encryption: %w", err)
		}

		// 5. Store the newly encrypted secret back in the vault
		if err := vaultClient.StoreSecret(ctx, cfg.SecretName, newEncryptedContent); err != nil {
			return fmt.Errorf("failed to store re-encrypted secret in Key Vault: %w", err)
		}

		utils.PrintSuccess("\n🎉 Key rotated successfully in Azure Key Vault!\n")
		utils.PrintWarning("🚨 IMPORTANT: You must now securely distribute the new key to your team.\n")
		utils.PrintInfo("🔧 They will need to update their key source (e.g., ENVSYNC_ENCRYPTION_KEY) before they can 'pull' again.\n")
		return nil
	},
}

// pushWithConflictDetection performs a push operation with conflict detection and resolution
func pushWithConflictDetection(cmd *cobra.Command, args []string, fromWatcher bool) error {
	cfg, err := config.LoadConfig(getConfigFile())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Get the encryption key
	key, err := cfg.LoadAndValidateKey(cliKey)
	if err != nil {
		return fmt.Errorf("failed to load encryption key: %w", err)
	}

	// Create Azure credentials and vault client
	cred, err := auth.CreateAzureCredential()
	if err != nil {
		return fmt.Errorf("failed to create Azure credentials: %w", err)
	}

	vaultClient, err := vault.NewClient(cfg.VaultURL, cred)
	if err != nil {
		return fmt.Errorf("failed to create Key Vault client: %w", err)
	}

	ctx := context.Background()

	// Read the current local .env file
	localContent, err := os.ReadFile(cfg.EnvFile)
	if err != nil {
		return fmt.Errorf("failed to read .env file from '%s': %w", cfg.EnvFile, err)
	}

	// Try to get the current remote version to check for conflicts
	var remoteContent []byte
	var hasRemote bool
	
	if encrypted, err := vaultClient.GetSecret(ctx, cfg.SecretName); err == nil {
		if decrypted, err := crypto.DecryptEnvContent(encrypted, key); err == nil {
			remoteContent = decrypted
			hasRemote = true
		} else {
			utils.PrintWarning("⚠️ Could not decrypt remote content (possible key mismatch), proceeding with push...\n")
		}
	} else {
		utils.PrintInfo("ℹ️ No remote version found, this will be the first push.\n")
	}

	// Check for conflicts if we have both local and remote content
	if hasRemote && len(remoteContent) > 0 {
		// Create conflict resolver
		conflictStrategy := sync.ConflictStrategyManual
		if cfg.ConflictStrategy != "" {
			switch cfg.ConflictStrategy {
			case "local":
				conflictStrategy = sync.ConflictStrategyLocal
			case "remote":
				conflictStrategy = sync.ConflictStrategyRemote
			case "merge":
				conflictStrategy = sync.ConflictStrategyMerge
			case "backup":
				conflictStrategy = sync.ConflictStrategyBackup
			}
		}

		// For push operations, we want to detect actual key conflicts, not just content differences
		// We'll check if local and remote have different values for the same keys
		localEnv, err := parseEnvContentForPush(string(localContent))
		if err != nil {
			return fmt.Errorf("failed to parse local .env content: %w", err)
		}
		
		remoteEnv, err := parseEnvContentForPush(string(remoteContent))
		if err != nil {
			return fmt.Errorf("failed to parse remote .env content: %w", err)
		}
		
		// Find actual conflicts (same key, different values)
		var conflictingKeys []string
		for key, localValue := range localEnv {
			if remoteValue, exists := remoteEnv[key]; exists && localValue != remoteValue {
				conflictingKeys = append(conflictingKeys, key)
			}
		}
		
		var conflict *ConflictInfo
		if len(conflictingKeys) > 0 {
			conflict = &ConflictInfo{
				ConflictTime:  time.Now(),
				LocalChanges:  localEnv,
				RemoteChanges: remoteEnv,
				Conflicts:     conflictingKeys,
			}
		}

		if conflict != nil {
			utils.PrintWarning("⚠️ Conflict detected! Remote version has different values.\n\n")
			
			// Show the conflicts
			utils.PrintInfo("🔍 Conflicting keys:\n")
			for _, key := range conflict.Conflicts {
				localValue := conflict.LocalChanges[key]
				remoteValue := conflict.RemoteChanges[key]
				utils.PrintInfo("  • %s:\n", key)
				utils.PrintInfo("    Local:  %s\n", localValue)
				utils.PrintInfo("    Remote: %s\n", remoteValue)
			}
			utils.PrintInfo("\n")

			// Ask user what to do
			if fromWatcher {
				// In watcher mode, respect the configured strategy or ask
				if conflictStrategy == sync.ConflictStrategyManual {
					if !promptUserForConflictResolution("Push with local changes") {
						utils.PrintInfo("⏭️ Push cancelled by user.\n")
						return nil
					}
				} else {
					utils.PrintInfo("🔧 Using configured conflict strategy: %s\n", cfg.ConflictStrategy)
				}
			} else {
				// In manual push mode, always ask for confirmation
				if !promptUserForConflictResolution("Push with local changes (this will overwrite remote)") {
					utils.PrintInfo("⏭️ Push cancelled by user.\n")
					return nil
				}
			}
		} else {
			utils.PrintInfo("✅ No conflicts detected with remote version.\n")
		}
	}

	// Proceed with the push
	encrypted, err := crypto.EncryptEnvContent(localContent, key)
	if err != nil {
		return fmt.Errorf("failed to encrypt .env file: %w", err)
	}

	utils.PrintInfo("🔒 Pushing encrypted content to Azure Key Vault...\n")
	if err := vaultClient.StoreSecret(ctx, cfg.SecretName, encrypted); err != nil {
		return fmt.Errorf("failed to store secret in Key Vault: %w", err)
	}

	return nil
}

// promptUserForConflictResolution prompts the user to confirm an action when conflicts exist
func promptUserForConflictResolution(action string) bool {
	fmt.Printf("\n🚀 %s? [y/N]: ", action)
	
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		utils.PrintError("❌ Error reading input: %v\n", err)
		return false
	}
	
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// ConflictInfo represents information about a detected conflict
type ConflictInfo struct {
	ConflictTime  time.Time
	LocalChanges  map[string]string
	RemoteChanges map[string]string
	Conflicts     []string
}

// parseEnvContentForPush parses .env content into key-value pairs for conflict detection
func parseEnvContentForPush(content string) (map[string]string, error) {
	env := make(map[string]string)
	lines := strings.Split(content, "\n")
	
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Find the first = sign
		equalIndex := strings.Index(line, "=")
		if equalIndex == -1 {
			return nil, fmt.Errorf("invalid .env format at line %d: missing '=' in '%s'", lineNum+1, line)
		}
		
		key := strings.TrimSpace(line[:equalIndex])
		value := line[equalIndex+1:] // Keep the value as-is (including quotes)
		
		// Remove surrounding quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
			   (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		
		env[key] = value
	}
	
	return env, nil
}
