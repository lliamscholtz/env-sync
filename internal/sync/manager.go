package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lliamscholtz/env-sync/internal/config"
	"github.com/lliamscholtz/env-sync/internal/crypto"
	"github.com/lliamscholtz/env-sync/internal/utils"
	"github.com/lliamscholtz/env-sync/internal/vault"
)

// SyncManager handles conflict-aware synchronization
type SyncManager struct {
	config      *config.Config
	vaultClient *vault.Client
	resolver    *ConflictResolver
	stateFile   string // Stores last known state for conflict detection
}

// SyncState tracks the last known state for conflict detection
type SyncState struct {
	LastSyncTime   time.Time `json:"last_sync_time"`
	LastKnownHash  string    `json:"last_known_hash"`
	LastSyncBy     string    `json:"last_sync_by"`
	ConflictCount  int       `json:"conflict_count"`
}

// NewSyncManager creates a new sync manager with conflict resolution
func NewSyncManager(cfg *config.Config, vaultClient *vault.Client, strategy ConflictStrategy, interactive bool) *SyncManager {
	stateFile := filepath.Join(filepath.Dir(cfg.EnvFile), ".env-sync-state.json")
	backupDir := filepath.Join(filepath.Dir(cfg.EnvFile), ".env-sync-backups")
	
	resolver := NewConflictResolver(strategy, backupDir, interactive)
	
	return &SyncManager{
		config:      cfg,
		vaultClient: vaultClient,
		resolver:    resolver,
		stateFile:   stateFile,
	}
}

// Push uploads local content with conflict detection
func (sm *SyncManager) Push(ctx context.Context, encryptionKey []byte) error {
	utils.PrintInfo("📤 Starting conflict-aware push...\n")
	
	// Read local file
	localContent, err := os.ReadFile(sm.config.EnvFile)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}
	
	// Get current remote content for conflict detection
	remoteEncrypted, err := sm.vaultClient.GetSecret(ctx, sm.config.SecretName)
	if err != nil {
		// If secret doesn't exist, this is the first push
		utils.PrintInfo("📝 First push - no conflict detection needed\n")
		return sm.performPush(ctx, string(localContent), encryptionKey)
	}
	
	// Decrypt remote content
	remoteContent, err := crypto.DecryptEnvContent(remoteEncrypted, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt remote content: %w", err)
	}
	
	// Load last known state
	state, err := sm.loadState()
	if err != nil {
		utils.PrintWarning("⚠️  Could not load sync state, assuming first sync: %v\n", err)
		state = &SyncState{}
	}
	
	// Detect conflicts
	conflict, err := sm.resolver.DetectConflict(string(localContent), string(remoteContent), state.LastKnownHash)
	if err != nil {
		return fmt.Errorf("failed to detect conflicts: %w", err)
	}
	
	var finalContent string
	if conflict != nil {
		utils.PrintWarning("⚠️  Conflict detected during push!\n")
		
		// Resolve the conflict
		resolvedContent, err := sm.resolver.ResolveConflict(ctx, conflict, sm.config.EnvFile)
		if err != nil {
			return fmt.Errorf("failed to resolve conflict: %w", err)
		}
		
		finalContent = resolvedContent
		state.ConflictCount++
		
		// Write resolved content back to local file
		if err := os.WriteFile(sm.config.EnvFile, []byte(finalContent), 0600); err != nil {
			return fmt.Errorf("failed to write resolved content to local file: %w", err)
		}
		
		utils.PrintSuccess("✅ Conflict resolved and local file updated\n")
	} else {
		finalContent = string(localContent)
		utils.PrintInfo("✅ No conflicts detected\n")
	}
	
	// Perform the push
	if err := sm.performPush(ctx, finalContent, encryptionKey); err != nil {
		return err
	}
	
	// Update state
	state.LastSyncTime = time.Now()
	state.LastKnownHash = calculateHash(finalContent)
	state.LastSyncBy = "push"
	
	if err := sm.saveState(state); err != nil {
		utils.PrintWarning("⚠️  Failed to save sync state: %v\n", err)
	}
	
	return nil
}

// Pull downloads remote content with conflict detection
func (sm *SyncManager) Pull(ctx context.Context, encryptionKey []byte) error {
	utils.PrintInfo("📥 Starting conflict-aware pull...\n")
	
	// Get remote content
	remoteEncrypted, err := sm.vaultClient.GetSecret(ctx, sm.config.SecretName)
	if err != nil {
		return fmt.Errorf("failed to get remote secret: %w", err)
	}
	
	// Decrypt remote content
	remoteContent, err := crypto.DecryptEnvContent(remoteEncrypted, encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt remote content: %w", err)
	}
	
	// Check if local file exists
	var localContent []byte
	if _, err := os.Stat(sm.config.EnvFile); err == nil {
		localContent, err = os.ReadFile(sm.config.EnvFile)
		if err != nil {
			return fmt.Errorf("failed to read local file: %w", err)
		}
	}
	
	// Load last known state
	state, err := sm.loadState()
	if err != nil {
		utils.PrintWarning("⚠️  Could not load sync state, assuming first sync: %v\n", err)
		state = &SyncState{}
	}
	
	var finalContent string
	if len(localContent) > 0 {
		// Detect conflicts
		conflict, err := sm.resolver.DetectConflict(string(localContent), string(remoteContent), state.LastKnownHash)
		if err != nil {
			return fmt.Errorf("failed to detect conflicts: %w", err)
		}
		
		if conflict != nil {
			utils.PrintWarning("⚠️  Conflict detected during pull!\n")
			
			// Resolve the conflict
			resolvedContent, err := sm.resolver.ResolveConflict(ctx, conflict, sm.config.EnvFile)
			if err != nil {
				return fmt.Errorf("failed to resolve conflict: %w", err)
			}
			
			finalContent = resolvedContent
			state.ConflictCount++
			
			utils.PrintSuccess("✅ Conflict resolved\n")
		} else {
			finalContent = string(remoteContent)
			utils.PrintInfo("✅ No conflicts detected\n")
		}
	} else {
		// No local file, just use remote content
		finalContent = string(remoteContent)
		utils.PrintInfo("📝 Creating local file from remote content\n")
	}
	
	// Write final content to local file
	if err := os.WriteFile(sm.config.EnvFile, []byte(finalContent), 0600); err != nil {
		return fmt.Errorf("failed to write to local file: %w", err)
	}
	
	// Update state
	state.LastSyncTime = time.Now()
	state.LastKnownHash = calculateHash(finalContent)
	state.LastSyncBy = "pull"
	
	if err := sm.saveState(state); err != nil {
		utils.PrintWarning("⚠️  Failed to save sync state: %v\n", err)
	}
	
	utils.PrintSuccess("✅ Pull completed successfully\n")
	return nil
}

// GetConflictStats returns conflict statistics
func (sm *SyncManager) GetConflictStats() (*SyncState, error) {
	return sm.loadState()
}

// performPush handles the actual push operation
func (sm *SyncManager) performPush(ctx context.Context, content string, encryptionKey []byte) error {
	// Encrypt content
	encryptedContent, err := crypto.EncryptEnvContent([]byte(content), encryptionKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt content: %w", err)
	}
	
	// Store in vault
	if err := sm.vaultClient.StoreSecret(ctx, sm.config.SecretName, encryptedContent); err != nil {
		return fmt.Errorf("failed to store secret: %w", err)
	}
	
	utils.PrintSuccess("✅ Content pushed successfully\n")
	return nil
}

// loadState loads the sync state from disk
func (sm *SyncManager) loadState() (*SyncState, error) {
	data, err := os.ReadFile(sm.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &SyncState{}, nil // Return empty state for first time
		}
		return nil, err
	}
	
	var state SyncState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	
	return &state, nil
}

// saveState saves the sync state to disk
func (sm *SyncManager) saveState(state *SyncState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	
	// Ensure directory exists
	dir := filepath.Dir(sm.stateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	return os.WriteFile(sm.stateFile, data, 0600)
}