package writer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSuppressionCacheDefaultTTL(t *testing.T) {
	for _, ttl := range []time.Duration{0, -time.Second} {
		if c := NewSuppressionCache(ttl); c.ttl != 3*time.Second {
			t.Errorf("NewSuppressionCache(%v).ttl = %v, want 3s", ttl, c.ttl)
		}
	}
	if c := NewSuppressionCache(time.Minute); c.ttl != time.Minute {
		t.Errorf("ttl = %v, want 1m", c.ttl)
	}
}

// expire moves an entry's deadline into the past without sleeping.
func expire(c *SuppressionCache, path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := c.entries[path]
	e.expiresAt = time.Now().Add(-time.Second)
	c.entries[path] = e
}

func TestCleanupExpiredLocked(t *testing.T) {
	tests := []struct {
		name   string
		record func(c *SuppressionCache)
	}{
		{"on RecordWrite", func(c *SuppressionCache) { c.RecordWrite("/fresh", []byte("x")) }},
		{"on RecordDelete", func(c *SuppressionCache) { c.RecordDelete("/fresh") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewSuppressionCache(time.Minute)
			c.RecordWrite("/stale-write", []byte("a"))
			c.RecordDelete("/stale-delete")
			c.RecordWrite("/live", []byte("b"))
			expire(c, "/stale-write")
			expire(c, "/stale-delete")

			tt.record(c)

			c.mu.Lock()
			defer c.mu.Unlock()
			for _, p := range []string{"/stale-write", "/stale-delete"} {
				if _, ok := c.entries[p]; ok {
					t.Errorf("expired entry %q was not cleaned up", p)
				}
			}
			for _, p := range []string{"/live", "/fresh"} {
				if _, ok := c.entries[p]; !ok {
					t.Errorf("live entry %q was removed", p)
				}
			}
		})
	}
}

func TestShouldSuppressExpiredEntries(t *testing.T) {
	c := NewSuppressionCache(time.Minute)
	c.RecordWrite("/w", []byte("x"))
	c.RecordDelete("/d")
	expire(c, "/w")
	expire(c, "/d")

	if c.ShouldSuppressWrite("/w", []byte("x")) {
		t.Error("expired write must not be suppressed")
	}
	if c.ShouldSuppressDelete("/d") {
		t.Error("expired delete must not be suppressed")
	}
}

func TestSyncDirMissingDirectory(t *testing.T) {
	// Best-effort: an unopenable directory must not panic.
	syncDir(filepath.Join(t.TempDir(), "does-not-exist"))
	syncDir(t.TempDir())
}

func TestWriterNilCache(t *testing.T) {
	w := New(nil)
	if w.SuppressionCache() != nil {
		t.Error("expected nil suppression cache")
	}
	p := filepath.Join(t.TempDir(), "f.md")
	if err := w.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := w.RemoveFile(p); err != nil {
		t.Fatalf("RemoveFile: %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("file should be removed, stat err = %v", err)
	}
}
