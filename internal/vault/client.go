package vault

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
)

// Client is a wrapper around the Azure Key Vault secrets client.
type Client struct {
	client   *azsecrets.Client
	VaultURL string
}

// NewClient creates a new Key Vault client.
func NewClient(vaultURL string, cred azcore.TokenCredential) (*Client, error) {
	if vaultURL == "" {
		return nil, fmt.Errorf("vault URL is required")
	}

	// Create a new secrets client
	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Key Vault client: %w", err)
	}

	return &Client{
		client:   client,
		VaultURL: vaultURL,
	}, nil
}

// StoreSecret creates or updates a secret in the Key Vault.
func (c *Client) StoreSecret(ctx context.Context, secretName, value string) error {
	_, err := c.client.SetSecret(ctx, secretName, azsecrets.SetSecretParameters{Value: &value}, nil)
	if err != nil {
		return fmt.Errorf("failed to store secret '%s': %w", secretName, err)
	}
	return nil
}

// GetSecret retrieves a secret from the Key Vault.
func (c *Client) GetSecret(ctx context.Context, secretName string) (string, error) {
	resp, err := c.client.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get secret '%s': %w", secretName, err)
	}

	if resp.Value == nil {
		return "", fmt.Errorf("retrieved secret '%s' has a nil value", secretName)
	}

	return *resp.Value, nil
}

// DeleteSecret removes a secret from the Key Vault.
func (c *Client) DeleteSecret(ctx context.Context, secretName string) error {
	_, err := c.client.DeleteSecret(ctx, secretName, nil)
	if err != nil {
		return fmt.Errorf("failed to delete secret '%s': %w", secretName, err)
	}
	return nil
}

// ListSecrets retrieves all secret names from the Key Vault.
func (c *Client) ListSecrets(ctx context.Context) ([]string, error) {
	var secretNames []string

	pager := c.client.NewListSecretPropertiesPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}
		for _, secret := range page.Value {
			secretNames = append(secretNames, secret.ID.Name())
		}
	}

	return secretNames, nil
}

// SecretExists checks if a secret with the given name exists in the vault.
func (c *Client) SecretExists(ctx context.Context, secretName string) (bool, error) {
	_, err := c.client.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if ok := errors.As(err, &respErr); ok && respErr.StatusCode == 404 {
			return false, nil // Not found
		}
		return false, fmt.Errorf("failed to check for secret '%s': %w", secretName, err)
	}
	return true, nil
}

// As is a helper function to check for a specific error type in an error chain.
// This is no longer needed as we use errors.As directly.
