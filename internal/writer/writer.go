package writer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// Writer provides atomic filesystem mutation operations to guarantee data integrity
// on disk and coordinate with the watcher to prevent echo loops.
type Writer struct {
	suppressCache *SuppressionCache
}

// New creates an atomic Writer instance with the given suppression cache.
// If suppressCache is nil, writes proceed without suppression recording.
func New(suppressCache *SuppressionCache) *Writer {
	return &Writer{
		suppressCache: suppressCache,
	}
}

// WriteFile writes data to targetPath atomically:
// 1. Creates a temporary file in the same directory as targetPath
// 2. Writes the content to the temporary file
// 3. Executes fsync to ensure data is flushed to physical storage
// 4. Atomically renames the temporary file over targetPath
// 5. Registers the file in the suppression cache
func (w *Writer) WriteFile(targetPath string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", dir, err)
	}

	base := filepath.Base(targetPath)
	tempPattern := fmt.Sprintf(".%s.*.tmp", base)

	tmpFile, err := os.CreateTemp(dir, tempPattern)
	if err != nil {
		return fmt.Errorf("failed to create temp file in %q: %w", dir, err)
	}

	tmpPath := tmpFile.Name()
	var success bool
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write data to temp file %q: %w", tmpPath, err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file %q: %w", tmpPath, err)
	}

	if err := tmpFile.Chmod(perm); err != nil {
		return fmt.Errorf("failed to chmod temp file %q: %w", tmpPath, err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file %q: %w", tmpPath, err)
	}

	// Register in suppression cache right before atomic rename
	if w.suppressCache != nil {
		w.suppressCache.RecordWrite(targetPath, data)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		return fmt.Errorf("failed to atomically rename %q to %q: %w", tmpPath, targetPath, err)
	}

	success = true

	// Persist the rename itself; without this a crash can lose the new directory entry.
	syncDir(dir)
	return nil
}

// RemoveFile removes targetPath and registers the deletion in the suppression cache.
func (w *Writer) RemoveFile(targetPath string) error {
	if w.suppressCache != nil {
		w.suppressCache.RecordDelete(targetPath)
	}

	if err := os.Remove(targetPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove file %q: %w", targetPath, err)
	}

	return nil
}

// SuppressionCache returns the active suppression cache or nil.
func (w *Writer) SuppressionCache() *SuppressionCache {
	return w.suppressCache
}

// syncDir flushes directory metadata (such as a completed rename) to disk.
// It is best-effort: some platforms, notably Windows, cannot fsync directories.
func syncDir(dir string) {
	if runtime.GOOS == "windows" {
		return
	}
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	_ = d.Sync()
	_ = d.Close()
}
