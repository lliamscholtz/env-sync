package sync

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDetectConflict(t *testing.T) {
	resolver := NewConflictResolver(ConflictStrategyMerge, "", false)
	
	tests := []struct {
		name           string
		localContent   string
		remoteContent  string
		lastKnownHash  string
		expectConflict bool
		expectedKeys   []string
	}{
		{
			name:           "no conflict - identical content",
			localContent:   "KEY1=value1\nKEY2=value2\n",
			remoteContent:  "KEY1=value1\nKEY2=value2\n",
			lastKnownHash:  "",
			expectConflict: false,
		},
		{
			name:           "no conflict - only remote changed",
			localContent:   "KEY1=value1\nKEY2=value2\n",
			remoteContent:  "KEY1=value1\nKEY2=value2_updated\n",
			lastKnownHash:  calculateHash("KEY1=value1\nKEY2=value2\n"),
			expectConflict: false,
		},
		{
			name:           "no conflict - only local changed",
			localContent:   "KEY1=value1_updated\nKEY2=value2\n",
			remoteContent:  "KEY1=value1\nKEY2=value2\n",
			lastKnownHash:  calculateHash("KEY1=value1\nKEY2=value2\n"),
			expectConflict: false,
		},
		{
			name:           "conflict - both sides changed same key",
			localContent:   "KEY1=local_value\nKEY2=value2\n",
			remoteContent:  "KEY1=remote_value\nKEY2=value2\n",
			lastKnownHash:  calculateHash("KEY1=original_value\nKEY2=value2\n"),
			expectConflict: true,
			expectedKeys:   []string{"KEY1"},
		},
		{
			name:           "conflict - multiple keys changed",
			localContent:   "KEY1=local1\nKEY2=local2\nKEY3=same\n",
			remoteContent:  "KEY1=remote1\nKEY2=remote2\nKEY3=same\n",
			lastKnownHash:  calculateHash("KEY1=orig1\nKEY2=orig2\nKEY3=same\n"),
			expectConflict: true,
			expectedKeys:   []string{"KEY1", "KEY2"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflict, err := resolver.DetectConflict(tt.localContent, tt.remoteContent, tt.lastKnownHash)
			if err != nil {
				t.Fatalf("DetectConflict failed: %v", err)
			}
			
			if tt.expectConflict && conflict == nil {
				t.Error("Expected conflict but none detected")
				return
			}
			
			if !tt.expectConflict && conflict != nil {
				t.Error("Unexpected conflict detected")
				return
			}
			
			if conflict != nil {
				if len(conflict.Conflicts) != len(tt.expectedKeys) {
					t.Errorf("Expected %d conflicts, got %d", len(tt.expectedKeys), len(conflict.Conflicts))
				}
				
				for _, key := range tt.expectedKeys {
					if !contains(conflict.Conflicts, key) {
						t.Errorf("Expected conflict for key %s", key)
					}
				}
			}
		})
	}
}

func TestResolveConflictStrategies(t *testing.T) {
	tempDir := t.TempDir()
	localFile := filepath.Join(tempDir, ".env")
	
	localContent := "KEY1=local_value\nKEY2=shared\n"
	remoteContent := "KEY1=remote_value\nKEY2=shared\n"
	
	// Create local file
	if err := os.WriteFile(localFile, []byte(localContent), 0600); err != nil {
		t.Fatalf("Failed to create local file: %v", err)
	}
	
	conflict := &ConflictInfo{
		LocalHash:     calculateHash(localContent),
		RemoteHash:    calculateHash(remoteContent),
		ConflictTime:  time.Now(),
		LocalChanges:  map[string]string{"KEY1": "local_value", "KEY2": "shared"},
		RemoteChanges: map[string]string{"KEY1": "remote_value", "KEY2": "shared"},
		Conflicts:     []string{"KEY1"},
	}
	
	tests := []struct {
		name            string
		strategy        ConflictStrategy
		expectedContent string
		shouldContain   string
	}{
		{
			name:            "local strategy",
			strategy:        ConflictStrategyLocal,
			expectedContent: "KEY1=local_value\nKEY2=shared\n",
		},
		{
			name:            "remote strategy",
			strategy:        ConflictStrategyRemote,
			expectedContent: "KEY1=remote_value\nKEY2=shared\n",
		},
		{
			name:         "merge strategy",
			strategy:     ConflictStrategyMerge,
			shouldContain: "<<<<<<< LOCAL",
		},
		{
			name:            "backup strategy",
			strategy:        ConflictStrategyBackup,
			expectedContent: "KEY1=local_value\nKEY2=shared\n", // Local wins
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewConflictResolver(tt.strategy, tempDir, false)
			
			result, err := resolver.ResolveConflict(context.Background(), conflict, localFile)
			if err != nil {
				t.Fatalf("ResolveConflict failed: %v", err)
			}
			
			if tt.expectedContent != "" {
				// Normalize whitespace for comparison
				normalizeFunc := func(s string) string {
					return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
				}
				
				if normalizeFunc(result) != normalizeFunc(tt.expectedContent) {
					t.Errorf("Expected content %q, got %q", tt.expectedContent, result)
				}
			}
			
			if tt.shouldContain != "" && !strings.Contains(result, tt.shouldContain) {
				t.Errorf("Expected result to contain %q, got %q", tt.shouldContain, result)
			}
		})
	}
}

func TestParseEnvContent(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		expected    map[string]string
		expectError bool
	}{
		{
			name:    "simple key-value pairs",
			content: "KEY1=value1\nKEY2=value2\n",
			expected: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
		{
			name:    "quoted values",
			content: "KEY1=\"value with spaces\"\nKEY2='single quoted'\n",
			expected: map[string]string{
				"KEY1": "value with spaces",
				"KEY2": "single quoted",
			},
		},
		{
			name:    "comments and empty lines",
			content: "# This is a comment\nKEY1=value1\n\nKEY2=value2\n# Another comment\n",
			expected: map[string]string{
				"KEY1": "value1",
				"KEY2": "value2",
			},
		},
		{
			name:    "values with equals signs",
			content: "KEY1=value=with=equals\nKEY2=https://example.com?param=value\n",
			expected: map[string]string{
				"KEY1": "value=with=equals",
				"KEY2": "https://example.com?param=value",
			},
		},
		{
			name:        "invalid format",
			content:     "INVALID_LINE_WITHOUT_EQUALS\n",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseEnvContent(tt.content)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but none occurred")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d keys, got %d", len(tt.expected), len(result))
			}
			
			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("Missing key %s", key)
				} else if actualValue != expectedValue {
					t.Errorf("For key %s, expected %q, got %q", key, expectedValue, actualValue)
				}
			}
		})
	}
}

func TestGenerateEnvContent(t *testing.T) {
	env := map[string]string{
		"KEY1": "simple_value",
		"KEY2": "value with spaces",
		"KEY3": "value=with=equals",
	}
	
	result := generateEnvContent(env)
	
	// Check that all keys are present
	for key := range env {
		if !strings.Contains(result, key+"=") {
			t.Errorf("Generated content missing key %s", key)
		}
	}
	
	// Check that values with spaces are quoted
	if !strings.Contains(result, `"value with spaces"`) {
		t.Error("Values with spaces should be quoted")
	}
	
	// Parse the generated content to ensure it's valid
	parsed, err := parseEnvContent(result)
	if err != nil {
		t.Fatalf("Generated content is not valid: %v", err)
	}
	
	// Verify all values match
	for key, expectedValue := range env {
		if actualValue, exists := parsed[key]; !exists {
			t.Errorf("Parsed content missing key %s", key)
		} else if actualValue != expectedValue {
			t.Errorf("For key %s, expected %q, got %q", key, expectedValue, actualValue)
		}
	}
}

func TestCreateBackup(t *testing.T) {
	tempDir := t.TempDir()
	localFile := filepath.Join(tempDir, ".env")
	
	// Create local file
	localContent := "KEY1=local_value\n"
	if err := os.WriteFile(localFile, []byte(localContent), 0600); err != nil {
		t.Fatalf("Failed to create local file: %v", err)
	}
	
	backupDir := filepath.Join(tempDir, ".env-sync-backups")
	resolver := NewConflictResolver(ConflictStrategyBackup, backupDir, false)
	
	conflict := &ConflictInfo{
		ConflictTime:  time.Now(),
		LocalChanges:  map[string]string{"KEY1": "local_value"},
		RemoteChanges: map[string]string{"KEY1": "remote_value"},
	}
	
	err := resolver.createBackup(localFile, conflict)
	if err != nil {
		t.Fatalf("createBackup failed: %v", err)
	}
	
	// Check that backup files were created
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		t.Error("Backup directory was not created")
	}
	
	// Check for backup files (they should have timestamp in name)
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}
	
	var localBackupFound, remoteBackupFound bool
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "local-") && strings.HasSuffix(entry.Name(), ".env") {
			localBackupFound = true
		}
		if strings.HasPrefix(entry.Name(), "remote-") && strings.HasSuffix(entry.Name(), ".env") {
			remoteBackupFound = true
		}
	}
	
	if !localBackupFound {
		t.Error("Local backup file not found")
	}
	if !remoteBackupFound {
		t.Error("Remote backup file not found")
	}
}