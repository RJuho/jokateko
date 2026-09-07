package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileOp represents the synthesized operation type for a debounced filesystem event.
type FileOp string

const (
	// OpWrite indicates a file was created or modified.
	OpWrite FileOp = "write"
	// OpDelete indicates a file was removed or renamed away.
	OpDelete FileOp = "delete"
)

// FileEvent represents a normalized, debounced filesystem event.
type FileEvent struct {
	Path string
	Op   FileOp
}

// Watcher monitors entity directories and configuration files, coalescing rapid events.
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	dirs      []string
	debounce  time.Duration
	eventsCh  chan FileEvent
	errorsCh  chan error
	closeOnce sync.Once
	closed    chan struct{}
}

// New creates a new filesystem watcher for the specified directories.
func New(dirs []string, debounce time.Duration) (*Watcher, error) {
	if debounce <= 0 {
		debounce = 50 * time.Millisecond
	}

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	for _, d := range dirs {
		if d == "" {
			continue
		}
		if err := os.MkdirAll(d, 0755); err != nil {
			_ = fsw.Close()
			return nil, fmt.Errorf("failed to create watch directory %q: %w", d, err)
		}
		if err := fsw.Add(d); err != nil {
			_ = fsw.Close()
			return nil, fmt.Errorf("failed to watch directory %q: %w", d, err)
		}
	}

	return &Watcher{
		fsWatcher: fsw,
		dirs:      dirs,
		debounce:  debounce,
		eventsCh:  make(chan FileEvent, 64),
		errorsCh:  make(chan error, 16),
		closed:    make(chan struct{}),
	}, nil
}

// Events returns the channel receiving debounced filesystem change events.
func (w *Watcher) Events() <-chan FileEvent {
	return w.eventsCh
}

// Errors returns the channel receiving filesystem watcher errors.
func (w *Watcher) Errors() <-chan error {
	return w.errorsCh
}

// isIgnored checks whether a filename represents an ephemeral editor or system file.
func isIgnored(path string) bool {
	base := filepath.Base(path)
	if strings.HasPrefix(base, ".") {
		return true
	}
	if strings.HasSuffix(base, "~") ||
		strings.HasSuffix(base, ".tmp") ||
		strings.HasSuffix(base, ".swp") ||
		strings.HasSuffix(base, ".bak") {
		return true
	}
	return false
}

// Start runs the debouncing event loop until ctx is canceled or the watcher is closed.
func (w *Watcher) Start(ctx context.Context) {
	go func() {
		defer close(w.eventsCh)
		defer close(w.errorsCh)

		pending := make(map[string]FileOp)
		var (
			timer   *time.Timer
			timerCh <-chan time.Time
		)

		resetTimer := func() {
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(w.debounce)
			timerCh = timer.C
		}

		flush := func() {
			for path, op := range pending {
				select {
				case w.eventsCh <- FileEvent{Path: path, Op: op}:
				case <-ctx.Done():
					return
				case <-w.closed:
					return
				}
			}
			pending = make(map[string]FileOp)
			timerCh = nil
		}

		for {
			select {
			case <-timerCh:
				flush()
			case err, ok := <-w.fsWatcher.Errors:
				if !ok {
					return
				}
				
				select {
				case <-w.closed:
					continue
				case <-ctx.Done():
					continue
				case w.errorsCh <- err:
				default:
				}
			case ev, ok := <-w.fsWatcher.Events:
				if !ok {
					return
				}

				select {
				case <-w.closed:
					continue
				case <-ctx.Done():
					continue
				default:
				}

				if isIgnored(ev.Name) {
					continue
				}

				// Only watch markdown or TOML configuration files
				ext := filepath.Ext(ev.Name)
				if ext != ".md" && ext != ".toml" {
					continue
				}

				cleanPath := filepath.Clean(ev.Name)
				if ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename) {
					pending[cleanPath] = OpDelete
					resetTimer()
				} else if ev.Has(fsnotify.Write) || ev.Has(fsnotify.Create) {
					pending[cleanPath] = OpWrite
					resetTimer()
				}
			}
		}
	}()
}

// Close terminates the underlying fsnotify watcher and stops event processing.
func (w *Watcher) Close() error {
	var err error
	w.closeOnce.Do(func() {
		close(w.closed)
		err = w.fsWatcher.Close()
	})
	return err
}
