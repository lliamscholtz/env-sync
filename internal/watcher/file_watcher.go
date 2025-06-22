package watcher

import (
	"context"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/lliamscholtz/env-sync/internal/utils"
)

// FileWatcher monitors a file for changes and triggers a callback.
type FileWatcher struct {
	FilePath     string
	SyncInterval time.Duration
	DebounceTime time.Duration
	OnChangeFunc func() error
	watcher      *fsnotify.Watcher
	done         chan bool
}

// NewFileWatcher creates a new file watcher instance.
func NewFileWatcher(filePath string, syncInterval, debounceTime time.Duration, onChange func() error) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &FileWatcher{
		FilePath:     filePath,
		SyncInterval: syncInterval,
		DebounceTime: debounceTime,
		OnChangeFunc: onChange,
		watcher:      watcher,
		done:         make(chan bool),
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

	utils.PrintInfo("Watching for changes to %s...\n", w.FilePath)

	var lastChange time.Time
	ticker := time.NewTicker(w.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			utils.PrintInfo("Stopping watcher...\n")
			return nil
		case event, ok := <-w.watcher.Events:
			if !ok {
				return nil
			}
			// We only care about writes and creates
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				if time.Since(lastChange) > w.DebounceTime {
					utils.PrintInfo("Change detected in %s, syncing...\n", w.FilePath)
					if err := w.OnChangeFunc(); err != nil {
						utils.PrintError("Error during sync: %v\n", err)
					}
					lastChange = time.Now()
				}
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return nil
			}
			utils.PrintError("Watcher error: %v\n", err)
		case <-ticker.C:
			utils.PrintInfo("Periodic sync triggered...\n")
			if err := w.OnChangeFunc(); err != nil {
				utils.PrintError("Error during periodic sync: %v\n", err)
			}
		}
	}
}

// Stop gracefully shuts down the file watcher.
func (w *FileWatcher) Stop() {
	w.done <- true
}
