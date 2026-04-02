/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2026 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package engine

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WatchEvent represents a file system change detected by the watcher.
type WatchEvent struct {
	Path      string
	Op        WatchOp
	Timestamp time.Time
}

// WatchOp describes the type of file system operation.
type WatchOp int

const (
	OpCreate WatchOp = iota
	OpWrite
	OpRemove
	OpRename
)

func (o WatchOp) String() string {
	switch o {
	case OpCreate:
		return "create"
	case OpWrite:
		return "write"
	case OpRemove:
		return "remove"
	case OpRename:
		return "rename"
	default:
		return "unknown"
	}
}

// WatchStats holds statistics about watcher activity.
type WatchStats struct {
	FilesWatched   int
	DirsWatched    int
	EventsReceived int
	LastEvent      time.Time
	Errors         int
}

// Watcher monitors file system changes and triggers re-indexing.
type Watcher struct {
	root       string
	ignoreList []string
	ignorer    *IgnoreMatcher
	debounceMs int
	fsWatcher  *fsnotify.Watcher

	mu      sync.Mutex
	stats   WatchStats
	pending map[string]WatchEvent // debounce buffer
	onEvent func(WatchEvent)      // callback for each debounced event
	onError func(error)           // callback for errors

	done chan struct{}
}

// WatcherOption configures a Watcher.
type WatcherOption func(*Watcher)

// WithDebounce sets the debounce interval in milliseconds.
func WithDebounce(ms int) WatcherOption {
	return func(w *Watcher) {
		if ms > 0 {
			w.debounceMs = ms
		}
	}
}

// WithOnEvent sets the callback for debounced file events.
func WithOnEvent(fn func(WatchEvent)) WatcherOption {
	return func(w *Watcher) { w.onEvent = fn }
}

// WithOnError sets the callback for watcher errors.
func WithOnError(fn func(error)) WatcherOption {
	return func(w *Watcher) { w.onError = fn }
}

// NewWatcher creates a new file system watcher.
func NewWatcher(root string, ignoreList []string, opts ...WatcherOption) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	w := &Watcher{
		root:       root,
		ignoreList: ignoreList,
		ignorer:    NewIgnoreMatcher(root, ignoreList),
		debounceMs: 500,
		fsWatcher:  fsw,
		pending:    make(map[string]WatchEvent),
		done:       make(chan struct{}),
	}

	for _, opt := range opts {
		opt(w)
	}

	return w, nil
}

// Start begins watching the project directory tree. It blocks until ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	if err := w.addDirectories(); err != nil {
		return err
	}

	go w.eventLoop(ctx)
	go w.debounceLoop(ctx)

	<-ctx.Done()
	close(w.done)
	return w.fsWatcher.Close()
}

// Stats returns a snapshot of watcher statistics.
func (w *Watcher) Stats() WatchStats {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.stats
}

func (w *Watcher) addDirectories() error {
	return filepath.Walk(w.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}

		name := info.Name()

		// Skip hidden directories
		if strings.HasPrefix(name, ".") && name != "." {
			return filepath.SkipDir
		}

		relPath, _ := filepath.Rel(w.root, path)
		if relPath != "." && w.ignorer.IsIgnored(relPath, true) {
			return filepath.SkipDir
		}

		if err := w.fsWatcher.Add(path); err != nil {
			log.Printf("watcher: failed to watch %s: %v", path, err)
			return nil
		}

		w.mu.Lock()
		w.stats.DirsWatched++
		w.mu.Unlock()

		return nil
	})
}

func (w *Watcher) eventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			w.handleFSEvent(event)

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			w.mu.Lock()
			w.stats.Errors++
			w.mu.Unlock()
			if w.onError != nil {
				w.onError(err)
			}
		}
	}
}

func (w *Watcher) handleFSEvent(event fsnotify.Event) {
	relPath, err := filepath.Rel(w.root, event.Name)
	if err != nil {
		return
	}

	// Skip hidden files
	for _, part := range strings.Split(relPath, string(filepath.Separator)) {
		if strings.HasPrefix(part, ".") {
			return
		}
	}

	// Skip ignored (checks all path components + gitignore/sarasignore patterns)
	info, statErr := os.Stat(event.Name)
	isDir := statErr == nil && info.IsDir()
	if w.ignorer.IsIgnored(relPath, isDir) {
		return
	}

	op := fsOpToWatchOp(event.Op)

	// If a new directory was created, start watching it (unless ignored)
	if event.Op.Has(fsnotify.Create) && isDir {
		w.fsWatcher.Add(event.Name)
		w.mu.Lock()
		w.stats.DirsWatched++
		w.mu.Unlock()
	}

	we := WatchEvent{
		Path:      relPath,
		Op:        op,
		Timestamp: time.Now(),
	}

	w.mu.Lock()
	w.stats.EventsReceived++
	w.stats.LastEvent = we.Timestamp
	w.pending[relPath] = we
	w.mu.Unlock()
}

func (w *Watcher) debounceLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(w.debounceMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.done:
			return
		case <-ticker.C:
			w.flushPending()
		}
	}
}

func (w *Watcher) flushPending() {
	w.mu.Lock()
	if len(w.pending) == 0 {
		w.mu.Unlock()
		return
	}

	events := make([]WatchEvent, 0, len(w.pending))
	for _, e := range w.pending {
		events = append(events, e)
	}
	w.pending = make(map[string]WatchEvent)
	w.mu.Unlock()

	if w.onEvent != nil {
		for _, e := range events {
			w.onEvent(e)
		}
	}
}

func fsOpToWatchOp(op fsnotify.Op) WatchOp {
	switch {
	case op.Has(fsnotify.Create):
		return OpCreate
	case op.Has(fsnotify.Write):
		return OpWrite
	case op.Has(fsnotify.Remove):
		return OpRemove
	case op.Has(fsnotify.Rename):
		return OpRename
	default:
		return OpWrite
	}
}
