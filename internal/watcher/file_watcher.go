package watcher

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lliamscholtz/env-sync/internal/utils"
)

// FileWatcher monitors a file for changes and triggers a callback.
type FileWatcher struct {
	FilePath        string
	SyncInterval    time.Duration
	DebounceTime    time.Duration
	OnChangeFunc    func() error // Called when file changes (push)
	OnPeriodicFunc  func() error // Called on periodic intervals (pull)
	EnablePush      bool         // Whether to push on file changes
	ConfirmPush     bool         // Whether to prompt user before push
	watcher         *fsnotify.Watcher
	done            chan bool
	lastPullTime    time.Time     // Timestamp of last pull operation
	lastWatchCheck  time.Time     // Timestamp of last watcher health check
}

// NewFileWatcher creates a new file watcher instance.
func NewFileWatcher(filePath string, syncInterval, debounceTime time.Duration, onChange func() error, onPeriodic func() error, enablePush, confirmPush bool) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &FileWatcher{
		FilePath:       filePath,
		SyncInterval:   syncInterval,
		DebounceTime:   debounceTime,
		OnChangeFunc:   onChange,
		OnPeriodicFunc: onPeriodic,
		EnablePush:     enablePush,
		ConfirmPush:    confirmPush,
		watcher:        watcher,
		done:           make(chan bool),
		lastWatchCheck: time.Now(),
	}, nil
}

// Start begins monitoring the file for changes.
func (w *FileWatcher) Start(ctx context.Context) error {
	defer w.watcher.Close()
	defer close(w.done)

	// Watch both the file and its parent directory
	// This handles atomic writes where editors create temp files and rename them
	err := w.watcher.Add(w.FilePath)
	if err != nil {
		return err
	}
	
	// Also watch the parent directory to catch atomic writes/renames
	parentDir := filepath.Dir(w.FilePath)
	err = w.watcher.Add(parentDir)
	if err != nil {
		utils.PrintDebug("⚠️ Could not watch parent directory %s: %v\n", parentDir, err)
		// Not fatal, continue with just file watching
	}

	if w.EnablePush {
		if w.ConfirmPush {
			utils.PrintInfo("🔍 Watching %s (file changes will prompt for push, periodic pulls enabled).\n", w.FilePath)
		} else {
			utils.PrintInfo("🔍 Watching %s (file changes will auto-push, periodic pulls enabled).\n", w.FilePath)
		}
	} else {
		utils.PrintInfo("🔍 Watching %s (file change push disabled, periodic pull only).\n", w.FilePath)
	}

	var lastChange time.Time
	ticker := time.NewTicker(w.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			utils.PrintInfo("🛑 Stopping watcher...\n")
			return nil
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			
			// Debug: Always log all events to help diagnose issues
			utils.PrintDebug("🔍 File event: %s -> %s (target: %s)\n", event.Name, event.Op.String(), w.FilePath)
			
			// Only process events related to our target file
			isTargetFile := event.Name == w.FilePath || filepath.Base(event.Name) == filepath.Base(w.FilePath)
			
			// Handle file removal/recreation (atomic writes often do this) - do this first, outside other conditions
			if event.Name == w.FilePath {
				if event.Op&fsnotify.Remove == fsnotify.Remove {
					utils.PrintDebug("📁 Target file removed, will re-watch when recreated\n")
				}
				
				if event.Op&fsnotify.Create == fsnotify.Create {
					utils.PrintDebug("📁 Target file recreated, ensuring it's being watched\n")
					// Remove and re-add to ensure clean watching state
					w.watcher.Remove(w.FilePath)
					if err := w.watcher.Add(w.FilePath); err != nil {
						utils.PrintError("❌ Could not re-watch file after recreation: %v\n", err)
					} else {
						utils.PrintDebug("✅ Successfully re-established watcher after file recreation\n")
					}
				}
			}
			
			// Only process file changes if push is enabled
			if w.EnablePush && isTargetFile {
				// We care about writes, creates, and also renames (common with editors)
				if event.Op&fsnotify.Write == fsnotify.Write || 
				   event.Op&fsnotify.Create == fsnotify.Create ||
				   event.Op&fsnotify.Rename == fsnotify.Rename {
					
					// Skip file changes that happen within 3 seconds of a pull operation
					// This prevents the pull from triggering a push
					if time.Since(w.lastPullTime) < 3*time.Second {
						utils.PrintDebug("⏳ Skipping event (within 3s of pull): %s\n", event.Op.String())
						continue
					}
					
					// Check debounce timing
					timeSinceLastChange := time.Since(lastChange)
					utils.PrintDebug("⏱️ Time since last change: %.2fs (debounce: %.2fs)\n", timeSinceLastChange.Seconds(), w.DebounceTime.Seconds())
					if timeSinceLastChange > w.DebounceTime {
						utils.PrintInfo("📝 Change detected in %s (event: %s)\n", w.FilePath, event.Op.String())
						
						// Check if we should confirm before pushing
						shouldPush := true
						if w.ConfirmPush {
							shouldPush = w.promptUserForPush()
						}
						
						if shouldPush {
							utils.PrintInfo("📤 Pushing changes to remote...\n")
							if err := w.OnChangeFunc(); err != nil {
								utils.PrintError("❌ Error during push: %v\n", err)
							} else {
								utils.PrintSuccess("✅ Successfully pushed encrypted .env file to Azure Key Vault.\n")
							}
						} else {
							utils.PrintInfo("⏭️  Skipping push (user declined)\n")
						}
						
						lastChange = time.Now()
					} else {
						utils.PrintDebug("⏳ Debouncing event (%.2fs since last): %s\n", timeSinceLastChange.Seconds(), event.Op.String())
					}
				} else {
					utils.PrintDebug("🚫 Ignoring event type: %s\n", event.Op.String())
				}
			} else if !isTargetFile {
				utils.PrintDebug("🔇 Ignoring non-target file event: %s\n", event.Name)
			} else {
				// File change detection disabled - only periodic pulls are active
				utils.PrintDebug("📋 Push disabled, ignoring event: %s\n", event.Op.String())
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			utils.PrintError("❌ Watcher error: %v\n", err)
		case <-ticker.C:
			// Record pull time before and after pull operation
			w.lastPullTime = time.Now()
			if err := w.OnPeriodicFunc(); err != nil {
				utils.PrintError("❌ Error during periodic pull: %v\n", err)
			}
			
			// Periodically check if the watcher is still active (every 5 minutes)
			if time.Since(w.lastWatchCheck) > 5*time.Minute {
				utils.PrintDebug("🔍 Performing watcher health check...\n")
				if err := w.ensureWatcherActive(); err != nil {
					utils.PrintError("❌ Failed to ensure watcher is active: %v\n", err)
				}
				w.lastWatchCheck = time.Now()
			}
		}
	}
}

// Stop gracefully shuts down the file watcher.
func (w *FileWatcher) Stop() {
	w.done <- true
}

// ensureWatcherActive checks if the watcher is still active and re-establishes it if needed
func (w *FileWatcher) ensureWatcherActive() error {
	// Check if file exists
	if _, err := os.Stat(w.FilePath); os.IsNotExist(err) {
		utils.PrintDebug("📁 Target file doesn't exist: %s\n", w.FilePath)
		return nil // File doesn't exist, nothing to watch
	}

	// Try to get the current watch list to see if our file is still being watched
	// fsnotify doesn't provide a direct way to check this, so we'll try to add it again
	// If it's already being watched, this will return an error we can ignore
	err := w.watcher.Add(w.FilePath)
	if err != nil {
		if strings.Contains(err.Error(), "already watching") || strings.Contains(err.Error(), "file already exists") {
			// File is already being watched, which is good
			utils.PrintDebug("✅ File watcher is still active for %s\n", w.FilePath)
			return nil
		} else {
			// Some other error occurred, try to remove and re-add
			utils.PrintDebug("⚠️ Re-establishing watcher for %s due to error: %v\n", w.FilePath, err)
			w.watcher.Remove(w.FilePath)
			err = w.watcher.Add(w.FilePath)
			if err != nil {
				return fmt.Errorf("failed to re-establish watcher: %w", err)
			}
			utils.PrintDebug("✅ Successfully re-established watcher for %s\n", w.FilePath)
		}
	} else {
		utils.PrintDebug("✅ Added %s to watcher (was not being watched)\n", w.FilePath)
	}

	return nil
}

// promptUserForPush prompts the user to confirm whether they want to push changes
func (w *FileWatcher) promptUserForPush() bool {
	fmt.Printf("\n🚀 Push changes to remote? [y/N]: ")
	
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		utils.PrintError("❌ Error reading input: %v\n", err)
		return false
	}
	
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
