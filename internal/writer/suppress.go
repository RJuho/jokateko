package writer

import (
	"crypto/sha256"
	"sync"
	"time"
)

type cacheEntry struct {
	hash      [32]byte
	isDeleted bool
	expiresAt time.Time
}

// SuppressionCache records recent self-originated file mutations to prevent
// circular watcher echo events when Jokateko writes files to disk.
type SuppressionCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

// NewSuppressionCache creates a thread-safe suppression cache with the specified TTL.
// If ttl <= 0, a default of 3 seconds is used.
func NewSuppressionCache(ttl time.Duration) *SuppressionCache {
	if ttl <= 0 {
		ttl = 3 * time.Second
	}
	return &SuppressionCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

// RecordWrite registers a recently written file path and its content hash.
func (c *SuppressionCache) RecordWrite(path string, content []byte) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cleanupExpiredLocked()

	hash := sha256.Sum256(content)
	c.entries[path] = cacheEntry{
		hash:      hash,
		isDeleted: false,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// RecordDelete registers a recently deleted file path.
func (c *SuppressionCache) RecordDelete(path string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cleanupExpiredLocked()

	c.entries[path] = cacheEntry{
		isDeleted: true,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// ShouldSuppressWrite checks if a write event on path matches the recorded hash and is still within TTL.
// If it matches, the entry is consumed and returns true.
func (c *SuppressionCache) ShouldSuppressWrite(path string, currentContent []byte) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[path]
	if !ok || entry.isDeleted || time.Now().After(entry.expiresAt) {
		return false
	}

	currentHash := sha256.Sum256(currentContent)
	if entry.hash == currentHash {
		delete(c.entries, path)
		return true
	}

	return false
}

// ShouldSuppressDelete checks if a delete event on path was self-originated and within TTL.
func (c *SuppressionCache) ShouldSuppressDelete(path string) bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[path]
	if !ok || !entry.isDeleted || time.Now().After(entry.expiresAt) {
		return false
	}

	delete(c.entries, path)
	return true
}

func (c *SuppressionCache) cleanupExpiredLocked() {
	now := time.Now()
	for p, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, p)
		}
	}
}
