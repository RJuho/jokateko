package writer_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/writer"
)

func TestAtomicWriter(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sub", "test.md")

	cache := writer.NewSuppressionCache(100 * time.Millisecond)
	w := writer.New(cache)

	content1 := []byte("# Hello World\n")
	if err := w.WriteFile(filePath, content1, 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify file was written
	readBytes, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(readBytes) != string(content1) {
		t.Errorf("expected %q, got %q", string(content1), string(readBytes))
	}

	// Verify suppression cache catches the write
	if !cache.ShouldSuppressWrite(filePath, content1) {
		t.Error("expected write to be suppressed immediately after write")
	}

	// Second check should return false (cache consumed)
	if cache.ShouldSuppressWrite(filePath, content1) {
		t.Error("expected second check to not be suppressed (already consumed)")
	}

	// Overwrite with new content
	content2 := []byte("# Updated Content\n")
	if err := w.WriteFile(filePath, content2, 0644); err != nil {
		t.Fatalf("second WriteFile failed: %v", err)
	}

	// Different content should not match old content
	if cache.ShouldSuppressWrite(filePath, content1) {
		t.Error("old content should not match new write in cache")
	}

	// Matching content suppresses
	if !cache.ShouldSuppressWrite(filePath, content2) {
		t.Error("expected content2 to be suppressed")
	}

	// Delete file
	if err := w.RemoveFile(filePath); err != nil {
		t.Fatalf("RemoveFile failed: %v", err)
	}

	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("expected file to be deleted, got err=%v", err)
	}

	if !cache.ShouldSuppressDelete(filePath) {
		t.Error("expected delete to be suppressed")
	}
}

func TestSuppressionCacheExpiration(t *testing.T) {
	cache := writer.NewSuppressionCache(20 * time.Millisecond)
	path := "/path/to/test.md"
	content := []byte("content")

	cache.RecordWrite(path, content)

	// Wait for TTL to expire
	time.Sleep(35 * time.Millisecond)

	if cache.ShouldSuppressWrite(path, content) {
		t.Error("expected expired entry to not be suppressed")
	}
}
