package pgxmutex

import (
	"context"
	"fmt"
	"time"
)

// Mutex is a distributed lock based on PostgreSQL advisory locks
type Mutex struct {
	conn conn
	ctx  context.Context
	so   *singleton
}

// NewMutex initializes a new Mutex with provided options.
func NewMutex(options ...Option) (*Mutex, error) {
	// Default configuration
	m := &Mutex{ctx: context.Background()}

	// Apply each option
	for _, opt := range options {
		if err := opt(m); err != nil {
			return nil, err
		}
	}

	// Check required fields
	if m.conn == nil {
		return nil, fmt.Errorf("database connection must be provided")
	}

	// Generate a lock ID if not provided
	if m.so == nil {
		m.so = getSingleton(time.Now().UnixNano())
	}

	return m, nil
}

// SyncMutex is a wrapper around Mutex that implements sync.Locker interface.
func (m *Mutex) SyncMutex() SyncMutex {
	return SyncMutex{m: m}
}

// GetResourceID returns the lock ID.
func (m *Mutex) GetResourceID() int64 {
	return m.so.id
}

// Lock tries to acquire the advisory lock, blocking until it's available.
func (m *Mutex) Lock() error {
	m.so.Lock()
	if _, err := m.conn.Exec(m.ctx, "SELECT pg_advisory_lock($1)", m.so.id); err != nil {
		m.so.Unlock()
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	return nil
}

// Unlock releases the advisory lock if it's currently held.
func (m *Mutex) Unlock() error {
	if _, err := m.conn.Exec(m.ctx, "SELECT pg_advisory_unlock($1)", m.so.id); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	m.so.Unlock()
	return nil
}

// TryLock attempts to acquire the advisory lock without blocking.
// Returns an error if unable to acquire the lock.
func (m *Mutex) TryLock() (bool, error) {
	if !m.so.TryLock() {
		return false, nil
	}

	var acquired bool
	if err := m.conn.QueryRow(m.ctx, "SELECT pg_try_advisory_lock($1)", m.so.id).Scan(&acquired); err != nil {
		m.so.Unlock()
		return false, fmt.Errorf("failed to attempt lock acquisition: %w", err)
	}

	if !acquired {
		m.so.Unlock()
	}

	return acquired, nil
}
