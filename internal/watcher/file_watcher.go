package watcher

import (
	"context"
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
	watcher         *fsnotify.Watcher
	done            chan bool
	lastPullTime    time.Time     // Timestamp of last pull operation
}

// NewFileWatcher creates a new file watcher instance.
func NewFileWatcher(filePath string, syncInterval, debounceTime time.Duration, onChange func() error, onPeriodic func() error) (*FileWatcher, error) {
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
		watcher:        watcher,
		done:           make(chan bool),
	}, nil
}

// Start begins monitoring the file for changes.
func (w *FileWatcher) Start(ctx context.Context) error {
	defer w.watcher.Close()
	defer close(w.done)

	err := w.watcher.Add(w.FilePath)
	if err != nil {
		return err
	}

	utils.PrintInfo("🔍 File changes to %s will be automatically pushed to Azure Key Vault.\n", w.FilePath)

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
			// We only care about writes and creates
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				// Skip file changes that happen within 10 seconds of a pull operation
				// This prevents the pull from triggering a push
				if time.Since(w.lastPullTime) < 10*time.Second {
					continue
				}
				if time.Since(lastChange) > w.DebounceTime {
					utils.PrintInfo("📝 Change detected in %s, pushing to remote...\n", w.FilePath)
					if err := w.OnChangeFunc(); err != nil {
						utils.PrintError("❌ Error during push: %v\n", err)
					}
					lastChange = time.Now()
				}
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
		}
	}
}

// Stop gracefully shuts down the file watcher.
func (w *FileWatcher) Stop() {
	w.done <- true
}
