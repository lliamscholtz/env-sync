package auth

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/lliamscholtz/env-sync/internal/deps"
	"github.com/lliamscholtz/env-sync/internal/utils"
)

// CreateAzureCredential creates a new credential object for Azure authentication.
// It uses a chain of credential sources for flexibility.
func CreateAzureCredential() (azcore.TokenCredential, error) {
	// The new SDK versions require creating the actual credential type, not its options.
	cliCred, err := azidentity.NewAzureCLICredential(nil)
	if err != nil {
		utils.PrintWarning("⚠️ Could not create Azure CLI credential: %v\n", err)
	}

	managedIDCred, err := azidentity.NewManagedIdentityCredential(nil)
	if err != nil {
		utils.PrintWarning("⚠️ Could not create Managed Identity credential: %v\n", err)
	}

	envCred, err := azidentity.NewEnvironmentCredential(nil)
	if err != nil {
		// Suppress warning for missing environment variables as this is expected
		// when using Azure CLI authentication
		utils.PrintDebug("🔧 Could not create Environment credential: %v\n", err)
	}

	// Filter out any credentials that failed to initialize
	creds := []azcore.TokenCredential{}
	if cliCred != nil {
		creds = append(creds, cliCred)
	}
	if managedIDCred != nil {
		creds = append(creds, managedIDCred)
	}
	if envCred != nil {
		creds = append(creds, envCred)
	}

	if len(creds) == 0 {
		return nil, fmt.Errorf("all credential types failed to initialize")
	}

	cred, err := azidentity.NewChainedTokenCredential(creds, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential chain: %w", err)
	}
	return cred, nil
}

// IsAuthenticated checks if the provided credential can acquire a token.
func IsAuthenticated(cred azcore.TokenCredential) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use the correctly imported policy.TokenRequestOptions
	_, err := cred.GetToken(ctx, policy.TokenRequestOptions{Scopes: []string{"https://vault.azure.net/.default"}})
	return err == nil
}

// EnsureAzureAuth is a high-level function that checks for Azure CLI and authentication status.
func EnsureAzureAuth(skipDepCheck bool) error {
	utils.PrintInfo("🔐 Verifying Azure authentication...\n")
	if !skipDepCheck {
		// First, ensure Azure CLI dependency is met
		dm, err := deps.NewDependencyManager()
		if err != nil {
			return err
		}
		missing, _ := dm.CheckDependencies()
		azCliMissing := false
		for _, dep := range missing {
			if dep.Name == "Azure CLI" {
				azCliMissing = true
				break
			}
		}

		if azCliMissing {
			return fmt.Errorf("Azure CLI is not installed. Please run 'env-sync install-deps' to install it")
		}
	}

	// Then, check authentication status
	return checkAuthStatus()
}

// checkAuthStatus verifies if the user is logged into Azure.
func checkAuthStatus() error {
	cred, err := CreateAzureCredential()
	if err != nil {
		return err
	}

	if !IsAuthenticated(cred) {
		utils.PrintError("❌ Not authenticated with Azure.\n")
		PrintAuthHelp()
		return fmt.Errorf("authentication failed")
	}

	utils.PrintSuccess("✅ Azure authentication successful.\n")
	return nil
}

// CheckAzLoginStatus runs `az account show` to check login status.
func CheckAzLoginStatus() error {
	cmd := exec.Command("az", "account", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("not logged in to Azure CLI. Please run 'az login'.\nOutput: %s", string(output))
	}
	return nil
}

// PrintAuthHelp provides guidance on how to authenticate.
func PrintAuthHelp() {
	utils.PrintInfo("🔑 Please authenticate using one of the following methods:\n")
	utils.PrintInfo("  1️⃣ Run 'az login' to authenticate with the Azure CLI.\n")
	utils.PrintInfo("  2️⃣ Set environment variables (AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_TENANT_ID).\n")
	utils.PrintInfo("  3️⃣ If running in Azure, ensure Managed Identity is configured.\n")
}
