package sync

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/lliamscholtz/env-sync/internal/utils"
)

// ConflictStrategy defines how to handle conflicts
type ConflictStrategy string

const (
	ConflictStrategyManual    ConflictStrategy = "manual"    // Prompt user to resolve
	ConflictStrategyLocal     ConflictStrategy = "local"     // Local changes win
	ConflictStrategyRemote    ConflictStrategy = "remote"    // Remote changes win
	ConflictStrategyMerge     ConflictStrategy = "merge"     // Attempt automatic merge
	ConflictStrategyBackup    ConflictStrategy = "backup"    // Create backup and merge
)

// ConflictInfo contains details about a detected conflict
type ConflictInfo struct {
	LocalHash     string
	RemoteHash    string
	ConflictTime  time.Time
	LocalChanges  map[string]string
	RemoteChanges map[string]string
	Conflicts     []string // Keys that have different values
}

// ConflictResolver handles environment file conflicts
type ConflictResolver struct {
	Strategy      ConflictStrategy
	BackupDir     string
	InteractiveMode bool
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(strategy ConflictStrategy, backupDir string, interactive bool) *ConflictResolver {
	return &ConflictResolver{
		Strategy:        strategy,
		BackupDir:       backupDir,
		InteractiveMode: interactive,
	}
}

// DetectConflict checks if local and remote content conflict
func (cr *ConflictResolver) DetectConflict(localContent, remoteContent, lastKnownHash string) (*ConflictInfo, error) {
	localHash := calculateHash(localContent)
	remoteHash := calculateHash(remoteContent)
	
	// No conflict if content is identical
	if localHash == remoteHash {
		return nil, nil
	}
	
	// No conflict if one side hasn't changed since last sync
	if localHash == lastKnownHash {
		// Only remote changed
		return nil, nil
	}
	if remoteHash == lastKnownHash {
		// Only local changed
		return nil, nil
	}
	
	// Both sides changed - we have a conflict!
	localEnv, err := parseEnvContent(localContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse local env: %w", err)
	}
	
	remoteEnv, err := parseEnvContent(remoteContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote env: %w", err)
	}
	
	conflicts := findConflictingKeys(localEnv, remoteEnv)
	
	return &ConflictInfo{
		LocalHash:     localHash,
		RemoteHash:    remoteHash,
		ConflictTime:  time.Now(),
		LocalChanges:  localEnv,
		RemoteChanges: remoteEnv,
		Conflicts:     conflicts,
	}, nil
}

// ResolveConflict resolves a detected conflict based on strategy
func (cr *ConflictResolver) ResolveConflict(ctx context.Context, conflict *ConflictInfo, localFile string) (string, error) {
	utils.PrintWarning("⚠️  Conflict detected! Both local and remote .env files have changes.\n")
	utils.PrintInfo("📊 Conflicting keys: %v\n", conflict.Conflicts)
	
	// Create backup regardless of strategy
	if err := cr.createBackup(localFile, conflict); err != nil {
		utils.PrintWarning("⚠️  Failed to create backup: %v\n", err)
	}
	
	switch cr.Strategy {
	case ConflictStrategyManual:
		return cr.resolveManually(conflict)
	case ConflictStrategyLocal:
		utils.PrintInfo("🏠 Using local changes (remote changes discarded)\n")
		return generateEnvContent(conflict.LocalChanges), nil
	case ConflictStrategyRemote:
		utils.PrintInfo("☁️  Using remote changes (local changes discarded)\n")
		return generateEnvContent(conflict.RemoteChanges), nil
	case ConflictStrategyMerge:
		return cr.resolveWithMerge(conflict)
	case ConflictStrategyBackup:
		return cr.resolveWithBackupMerge(conflict)
	default:
		return "", fmt.Errorf("unknown conflict strategy: %s", cr.Strategy)
	}
}

// resolveManually prompts user to resolve conflicts
func (cr *ConflictResolver) resolveManually(conflict *ConflictInfo) (string, error) {
	if !cr.InteractiveMode {
		return "", fmt.Errorf("conflict requires manual resolution but not in interactive mode")
	}
	
	merged := make(map[string]string)
	
	// Add non-conflicting keys from both sides
	for k, v := range conflict.LocalChanges {
		if !contains(conflict.Conflicts, k) {
			merged[k] = v
		}
	}
	for k, v := range conflict.RemoteChanges {
		if !contains(conflict.Conflicts, k) {
			merged[k] = v
		}
	}
	
	// Resolve each conflict manually
	for _, key := range conflict.Conflicts {
		localVal := conflict.LocalChanges[key]
		remoteVal := conflict.RemoteChanges[key]
		
		utils.PrintInfo("\n🔧 Conflict for key: %s\n", key)
		utils.PrintInfo("  Local:  %s\n", localVal)
		utils.PrintInfo("  Remote: %s\n", remoteVal)
		
		choice, err := cr.promptUserChoice(key)
		if err != nil {
			return "", err
		}
		
		switch choice {
		case "local":
			merged[key] = localVal
		case "remote":
			merged[key] = remoteVal
		case "edit":
			newVal, err := cr.promptUserEdit(key, localVal, remoteVal)
			if err != nil {
				return "", err
			}
			merged[key] = newVal
		}
	}
	
	return generateEnvContent(merged), nil
}

// resolveWithMerge attempts automatic merge with conflict markers
func (cr *ConflictResolver) resolveWithMerge(conflict *ConflictInfo) (string, error) {
	merged := make(map[string]string)
	
	// Add all non-conflicting keys
	for k, v := range conflict.LocalChanges {
		if !contains(conflict.Conflicts, k) {
			merged[k] = v
		}
	}
	for k, v := range conflict.RemoteChanges {
		if !contains(conflict.Conflicts, k) {
			merged[k] = v
		}
	}
	
	// For conflicts, create conflict markers
	for _, key := range conflict.Conflicts {
		localVal := conflict.LocalChanges[key]
		remoteVal := conflict.RemoteChanges[key]
		
		conflictMarker := fmt.Sprintf(`<<<<<<< LOCAL
%s
=======
%s
>>>>>>> REMOTE`, localVal, remoteVal)
		
		merged[key] = conflictMarker
	}
	
	utils.PrintWarning("⚠️  Automatic merge created conflict markers for manual resolution\n")
	utils.PrintInfo("📝 Please edit the .env file to resolve conflicts and run sync again\n")
	
	return generateEnvContent(merged), nil
}

// resolveWithBackupMerge creates backups and prefers local changes
func (cr *ConflictResolver) resolveWithBackupMerge(conflict *ConflictInfo) (string, error) {
	merged := make(map[string]string)
	
	// Start with remote changes as base
	for k, v := range conflict.RemoteChanges {
		merged[k] = v
	}
	
	// Overlay local changes (local wins for conflicts)
	for k, v := range conflict.LocalChanges {
		if contains(conflict.Conflicts, k) {
			utils.PrintInfo("🏠 Using local value for %s (remote value backed up)\n", k)
		}
		merged[k] = v
	}
	
	return generateEnvContent(merged), nil
}

// createBackup creates backup files for conflict resolution
func (cr *ConflictResolver) createBackup(localFile string, conflict *ConflictInfo) error {
	if cr.BackupDir == "" {
		cr.BackupDir = ".env-sync-backups"
	}
	
	// Create backup directory
	if err := os.MkdirAll(cr.BackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	
	timestamp := conflict.ConflictTime.Format("20060102-150405")
	
	// Backup local version
	localBackup := fmt.Sprintf("%s/local-%s.env", cr.BackupDir, timestamp)
	localContent := generateEnvContent(conflict.LocalChanges)
	if err := os.WriteFile(localBackup, []byte(localContent), 0600); err != nil {
		return fmt.Errorf("failed to create local backup: %w", err)
	}
	
	// Backup remote version
	remoteBackup := fmt.Sprintf("%s/remote-%s.env", cr.BackupDir, timestamp)
	remoteContent := generateEnvContent(conflict.RemoteChanges)
	if err := os.WriteFile(remoteBackup, []byte(remoteContent), 0600); err != nil {
		return fmt.Errorf("failed to create remote backup: %w", err)
	}
	
	utils.PrintInfo("💾 Backups created:\n")
	utils.PrintInfo("  Local:  %s\n", localBackup)
	utils.PrintInfo("  Remote: %s\n", remoteBackup)
	
	return nil
}

// promptUserChoice prompts user to choose resolution for a conflict
func (cr *ConflictResolver) promptUserChoice(key string) (string, error) {
	fmt.Printf("Choose resolution for %s:\n", key)
	fmt.Printf("  (l)ocal - use local value\n")
	fmt.Printf("  (r)emote - use remote value\n")
	fmt.Printf("  (e)dit - enter new value\n")
	fmt.Printf("Choice [l/r/e]: ")
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	
	input = strings.TrimSpace(strings.ToLower(input))
	switch input {
	case "l", "local":
		return "local", nil
	case "r", "remote":
		return "remote", nil
	case "e", "edit":
		return "edit", nil
	default:
		fmt.Printf("Invalid choice. Please enter l, r, or e.\n")
		return cr.promptUserChoice(key) // Retry
	}
}

// promptUserEdit prompts user to enter a new value
func (cr *ConflictResolver) promptUserEdit(key, localVal, remoteVal string) (string, error) {
	fmt.Printf("Enter new value for %s:\n", key)
	fmt.Printf("  Current local:  %s\n", localVal)
	fmt.Printf("  Current remote: %s\n", remoteVal)
	fmt.Printf("New value: ")
	
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(input), nil
}

// Helper functions

func calculateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

func parseEnvContent(content string) (map[string]string, error) {
	env := make(map[string]string)
	lines := strings.Split(content, "\n")
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Find the first = sign
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid env line %d: %s", i+1, line)
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove surrounding quotes if present
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || 
			(value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		
		env[key] = value
	}
	
	return env, nil
}

func generateEnvContent(env map[string]string) string {
	var lines []string
	
	// Sort keys for consistent output
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	for _, key := range keys {
		value := env[key]
		// Quote values that contain spaces or special characters
		if strings.ContainsAny(value, " \t\n\r") || strings.Contains(value, "=") {
			value = fmt.Sprintf(`"%s"`, value)
		}
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}
	
	return strings.Join(lines, "\n") + "\n"
}

func findConflictingKeys(local, remote map[string]string) []string {
	var conflicts []string
	
	for key, localVal := range local {
		if remoteVal, exists := remote[key]; exists && localVal != remoteVal {
			conflicts = append(conflicts, key)
		}
	}
	
	// Check for keys that exist in remote but not local
	for key := range remote {
		if _, exists := local[key]; !exists {
			// This is a new key in remote, not a conflict per se
			// But could be considered one depending on strategy
		}
	}
	
	sort.Strings(conflicts)
	return conflicts
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}