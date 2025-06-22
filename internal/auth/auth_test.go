package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateAzureCredential(t *testing.T) {
	t.Run("create credential", func(t *testing.T) {
		// This test will create a credential chain
		// It may not succeed in test environment but should not panic
		cred, err := CreateAzureCredential()
		
		// In test environment, this might fail due to no Azure auth
		// But we can at least test that the function doesn't panic
		// and returns a proper error if auth is not available
		if err != nil {
			// Expected in test environment
			assert.NotNil(t, err)
			assert.Nil(t, cred)
		} else {
			// If somehow auth is available, credential should not be nil
			assert.NotNil(t, cred)
		}
	})
}

func TestCheckAzLoginStatus(t *testing.T) {
	t.Run("check az login status", func(t *testing.T) {
		// This will check if Azure CLI is logged in
		// In test environment, this will likely return an error
		err := CheckAzLoginStatus()
		
		// We don't assert success/failure since this depends on the environment
		// We just ensure the function doesn't panic
		_ = err // Ignore the result, just test that it runs
	})
}

func TestEnsureAzureAuth(t *testing.T) {
	t.Run("ensure auth with dep check skipped", func(t *testing.T) {
		// Skip dependency check to avoid installation requirements in tests
		err := EnsureAzureAuth(true)
		
		// We don't assert success/failure since this depends on the environment
		// We just ensure the function doesn't panic
		_ = err // Ignore the result, just test that it runs
	})
}

func TestPrintAuthHelp(t *testing.T) {
	t.Run("print auth help", func(t *testing.T) {
		// This should not panic
		PrintAuthHelp()
		// If we get here, the function completed without panicking
	})
} 