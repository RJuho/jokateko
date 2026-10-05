package writer_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/writer"
)

func TestSuppressionCacheAccessor(t *testing.T) {
	cache := writer.NewSuppressionCache(time.Minute)
	if got := writer.New(cache).SuppressionCache(); got != cache {
		t.Errorf("SuppressionCache() = %p, want %p", got, cache)
	}
}

func TestNilSuppressionCacheIsSafe(t *testing.T) {
	var c *writer.SuppressionCache
	c.RecordWrite("/p", []byte("x"))
	c.RecordDelete("/p")
	if c.ShouldSuppressWrite("/p", []byte("x")) {
		t.Error("nil cache must not suppress writes")
	}
	if c.ShouldSuppressDelete("/p") {
		t.Error("nil cache must not suppress deletes")
	}
}

func TestSuppressionCacheKindMismatch(t *testing.T) {
	c := writer.NewSuppressionCache(time.Minute)

	c.RecordDelete("/deleted")
	if c.ShouldSuppressWrite("/deleted", nil) {
		t.Error("a recorded delete must not suppress a write event")
	}

	c.RecordWrite("/written", []byte("x"))
	if c.ShouldSuppressDelete("/written") {
		t.Error("a recorded write must not suppress a delete event")
	}
	if c.ShouldSuppressDelete("/unknown") {
		t.Error("unknown path must not be suppressed")
	}
	// The write entry is still intact after the mismatched delete check.
	if !c.ShouldSuppressWrite("/written", []byte("x")) {
		t.Error("write entry should still suppress the matching write")
	}
}

func TestWriteFileErrors(t *testing.T) {
	cache := writer.NewSuppressionCache(time.Minute)
	w := writer.New(cache)

	t.Run("parent path is a file", func(t *testing.T) {
		dir := t.TempDir()
		blocker := filepath.Join(dir, "blocker")
		if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		err := w.WriteFile(filepath.Join(blocker, "sub", "f.md"), []byte("y"), 0o644)
		if err == nil || !strings.Contains(err.Error(), "failed to create directory") {
			t.Errorf("expected mkdir error, got %v", err)
		}
	})

	t.Run("directory not writable", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("permission bits are not enforced for this user/platform")
		}
		dir := filepath.Join(t.TempDir(), "ro")
		if err := os.Mkdir(dir, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
		err := w.WriteFile(filepath.Join(dir, "f.md"), []byte("y"), 0o644)
		if err == nil || !strings.Contains(err.Error(), "failed to create temp file") {
			t.Errorf("expected create-temp error, got %v", err)
		}
	})

	t.Run("target is a non-empty directory", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "target")
		if err := os.MkdirAll(filepath.Join(target, "child"), 0o755); err != nil {
			t.Fatal(err)
		}
		err := w.WriteFile(target, []byte("y"), 0o644)
		if err == nil || !strings.Contains(err.Error(), "failed to atomically rename") {
			t.Errorf("expected rename error, got %v", err)
		}
		// The temp file must be cleaned up after a failed rename.
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".tmp") {
				t.Errorf("leftover temp file %q", e.Name())
			}
		}
	})
}

func TestWriteFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits not supported")
	}
	p := filepath.Join(t.TempDir(), "f.md")
	if err := writer.New(nil).WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestRemoveFile(t *testing.T) {
	cache := writer.NewSuppressionCache(time.Minute)
	w := writer.New(cache)

	t.Run("missing file is not an error", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "missing.md")
		if err := w.RemoveFile(p); err != nil {
			t.Errorf("RemoveFile(missing) = %v, want nil", err)
		}
		if !cache.ShouldSuppressDelete(p) {
			t.Error("delete should still be recorded for suppression")
		}
	})

	t.Run("non-empty directory fails", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "d")
		if err := os.MkdirAll(filepath.Join(dir, "child"), 0o755); err != nil {
			t.Fatal(err)
		}
		err := w.RemoveFile(dir)
		if err == nil || !strings.Contains(err.Error(), "failed to remove file") {
			t.Errorf("expected remove error, got %v", err)
		}
	})
}
