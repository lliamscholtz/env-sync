package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewFileWatcher(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, ".env")
	
	// Create test file
	if err := os.WriteFile(testFile, []byte("TEST=value"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	tests := []struct {
		name        string
		enablePush  bool
		confirmPush bool
	}{
		{"pull only mode", false, false},
		{"auto-push mode", true, false},
		{"confirm-push mode", true, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			onChange := func() error { return nil }
			onPeriodic := func() error { return nil }
			
			watcher, err := NewFileWatcher(
				testFile,
				1*time.Second,
				100*time.Millisecond,
				onChange,
				onPeriodic,
				tt.enablePush,
				tt.confirmPush,
			)
			
			if err != nil {
				t.Fatalf("NewFileWatcher failed: %v", err)
			}
			
			if watcher.EnablePush != tt.enablePush {
				t.Errorf("Expected EnablePush=%v, got %v", tt.enablePush, watcher.EnablePush)
			}
			
			if watcher.ConfirmPush != tt.confirmPush {
				t.Errorf("Expected ConfirmPush=%v, got %v", tt.confirmPush, watcher.ConfirmPush)
			}
			
			if watcher.FilePath != testFile {
				t.Errorf("Expected FilePath=%s, got %s", testFile, watcher.FilePath)
			}
		})
	}
}

func TestFileWatcherConfiguration(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, ".env")
	
	// Create test file
	if err := os.WriteFile(testFile, []byte("TEST=value"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	onChange := func() error {
		return nil
	}
	
	onPeriodic := func() error {
		return nil
	}
	
	watcher, err := NewFileWatcher(
		testFile,
		100*time.Millisecond, // Very short interval for testing
		50*time.Millisecond,  // Short debounce for testing
		onChange,
		onPeriodic,
		true,  // Enable push
		false, // Don't confirm (auto-push for testing)
	)
	
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	
	// Test that configuration is set correctly
	if watcher.SyncInterval != 100*time.Millisecond {
		t.Errorf("Expected SyncInterval=100ms, got %v", watcher.SyncInterval)
	}
	
	if watcher.DebounceTime != 50*time.Millisecond {
		t.Errorf("Expected DebounceTime=50ms, got %v", watcher.DebounceTime)
	}
	
	if !watcher.EnablePush {
		t.Error("Expected EnablePush=true")
	}
	
	if watcher.ConfirmPush {
		t.Error("Expected ConfirmPush=false")
	}
}

func TestFileWatcherStartStop(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, ".env")
	
	// Create test file
	if err := os.WriteFile(testFile, []byte("TEST=value"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	onChange := func() error { return nil }
	onPeriodic := func() error { return nil }
	
	watcher, err := NewFileWatcher(
		testFile,
		1*time.Second,
		100*time.Millisecond,
		onChange,
		onPeriodic,
		false, // Disable push for this test
		false,
	)
	
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	
	// Test that watcher can start and stop without errors
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- watcher.Start(ctx)
	}()
	
	// Let it run for a bit
	time.Sleep(500 * time.Millisecond)
	
	// Stop the watcher
	cancel()
	
	// Wait for it to finish
	select {
	case err := <-done:
		if err != nil && err != context.Canceled {
			t.Errorf("Watcher Start failed: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("Watcher did not stop within timeout")
	}
}

func TestFileWatcherAtomicWrites(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, ".env")
	
	// Create test file
	if err := os.WriteFile(testFile, []byte("TEST=value"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	changeDetected := make(chan bool, 10) // Buffer for multiple changes
	onChange := func() error { 
		select {
		case changeDetected <- true:
		default:
		}
		return nil 
	}
	onPeriodic := func() error { return nil }
	
	watcher, err := NewFileWatcher(
		testFile,
		10*time.Second, // Long interval to avoid periodic calls during test
		50*time.Millisecond,  // Short debounce
		onChange,
		onPeriodic,
		true,  // Enable push
		false, // Don't confirm (auto-push for testing)
	)
	
	if err != nil {
		t.Fatalf("NewFileWatcher failed: %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	done := make(chan error, 1)
	go func() {
		done <- watcher.Start(ctx)
	}()
	
	// Wait a bit for watcher to start
	time.Sleep(100 * time.Millisecond)
	
	// Test multiple consecutive atomic writes
	for i := 1; i <= 3; i++ {
		// Simulate atomic write (create temp file, write, rename)
		tempFile := testFile + ".tmp"
		content := fmt.Sprintf("TEST=new_value_%d", i)
		if err := os.WriteFile(tempFile, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to create temp file %d: %v", i, err)
		}
		
		if err := os.Rename(tempFile, testFile); err != nil {
			t.Fatalf("Failed to rename temp file %d: %v", i, err)
		}
		
		// Wait for change detection
		select {
		case <-changeDetected:
			t.Logf("✅ Change %d detected successfully", i)
		case <-time.After(2 * time.Second):
			t.Errorf("❌ Atomic write change %d was not detected within timeout", i)
		}
		
		// Wait a bit between changes to avoid debouncing
		time.Sleep(100 * time.Millisecond)
	}
	
	cancel()
	
	// Wait for watcher to finish
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("Watcher did not stop within timeout")
	}
}