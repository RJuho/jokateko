// Package store provides in-memory SQLite indexing and caching for Jokateko entities
// using pure Go modernc.org/sqlite and sqlc-generated type-safe queries.
package store

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var ddlSchema string

// DefaultMemoryDSN is the standard SQLite in-memory URI configured for high-performance in-memory operation:
// - cache=shared & mode=memory: shared in-memory database across connections
// - busy_timeout(10000): wait up to 10s for locks during concurrent access instead of immediately returning SQLITE_BUSY
// - journal_mode(MEMORY): stores rollback journal entirely in RAM for zero disk I/O while preserving atomic transaction rollback
// - synchronous(OFF): disables file-sync overhead for pure in-memory execution
const DefaultMemoryDSN = "file::memory:?cache=shared&mode=memory&_pragma=busy_timeout(10000)&_pragma=journal_mode(MEMORY)&_pragma=synchronous(OFF)"

// ErrNotFound indicates an entity was not found in the store.
var ErrNotFound = errors.New("entity not found")

// Store wraps an in-memory SQLite database connection and sqlc Queries
// with thread-safe concurrency controls and transaction helpers.
type Store struct {
	db      *sql.DB
	queries *Queries
	mu      sync.RWMutex
}

// Open initializes a new in-memory SQLite store, executes the embedded DDL schema,
// and returns a ready-to-use Store instance.
// If dsn is omitted or empty, DefaultMemoryDSN is used.
func Open(dsn ...string) (*Store, error) {
	connStr := DefaultMemoryDSN
	if len(dsn) > 0 && dsn[0] != "" {
		connStr = dsn[0]
	}

	db, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite in-memory database: %w", err)
	}

	// In-memory SQLite needs at least 1 idle connection to maintain its schema,
	// and serialized connection pooling avoids database locked errors.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping sqlite in-memory database: %w", err)
	}

	// Enable pragmas
	pragmas := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA temp_store = MEMORY;",
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to execute pragma %q: %w", pragma, err)
		}
	}

	// Execute embedded DDL schema
	if _, err := db.ExecContext(ctx, ddlSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to execute schema.sql: %w", err)
	}

	queries := New(db)

	return &Store{
		db:      db,
		queries: queries,
	}, nil
}

var memCounter atomic.Uint64

// OpenMemory is a convenience wrapper that opens an isolated, in-memory SQLite store.
func OpenMemory() (*Store, error) {
	id := memCounter.Add(1)
	dsn := fmt.Sprintf("file:mem_%d_%d?mode=memory&cache=shared&_pragma=busy_timeout(10000)&_pragma=journal_mode(MEMORY)&_pragma=synchronous(OFF)", id, time.Now().UnixNano())
	return Open(dsn)
}

// Queries returns the underlying sqlc Queries instance.
func (s *Store) Queries() *Queries {
	return s.queries
}

// DB returns the underlying *sql.DB instance.
func (s *Store) DB() *sql.DB {
	return s.db
}

// Close gracefully closes the SQLite database connection.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}

// WithTx executes a callback function inside an atomic SQLite transaction.
// If the callback returns an error, the transaction is rolled back.
// If the callback succeeds, the transaction is committed.
func (s *Store) WithTx(ctx context.Context, fn func(q *Queries) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	qtx := s.queries.WithTx(tx)
	if err := fn(qtx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// RLock acquires a read lock on the store.
func (s *Store) RLock() {
	s.mu.RLock()
}

// RUnlock releases a read lock on the store.
func (s *Store) RUnlock() {
	s.mu.RUnlock()
}

// Lock acquires a write lock on the store.
func (s *Store) Lock() {
	s.mu.Lock()
}

// Unlock releases a write lock on the store.
func (s *Store) Unlock() {
	s.mu.Unlock()
}
