package pgxmutex

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Mutex is a distributed lock based on PostgreSQL advisory locks
type Mutex struct {
	conn     *pgx.Conn
	lockID   int64
	lockHeld bool
	ctx      context.Context
}

// Option is a functional option type for configuring Mutex.
type Option func(*Mutex)

// WithConn sets the PGX connection.
func WithConn(conn *pgx.Conn) Option {
	return func(m *Mutex) {
		m.conn = conn
	}
}

// WithLockID sets the lock ID for advisory locking.
func WithLockID(lockID int64) Option {
	return func(m *Mutex) {
		m.lockID = lockID
	}
}

// WithContext sets a custom context for the Mutex operations.
func WithContext(ctx context.Context) Option {
	return func(m *Mutex) {
		m.ctx = ctx
	}
}

// NewMutex initializes a new Mutex with provided options.
func NewMutex(options ...Option) (*Mutex, error) {
	// Default configuration
	m := &Mutex{
		ctx: context.Background(),
	}

	// Apply each option
	for _, opt := range options {
		opt(m)
	}

	// Check required fields
	if m.conn == nil {
		return nil, fmt.Errorf("database connection must be provided")
	}

	// Generate a lock ID if not provided
	if m.lockID == 0 {
		m.lockID = time.Now().UnixNano()
	}

	return m, nil
}

// Lock tries to acquire the advisory lock, blocking until it's available.
func (m *Mutex) Lock() error {
	if err := m.conn.QueryRow(m.ctx, "SELECT pg_advisory_lock($1)", m.lockID).Scan(); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	m.lockHeld = true
	return nil
}

// TryLock attempts to acquire the advisory lock without blocking.
// Returns an error if unable to acquire the lock.
func (m *Mutex) TryLock() (bool, error) {
	var acquired bool
	if err := m.conn.QueryRow(m.ctx, "SELECT pg_try_advisory_lock($1)", m.lockID).Scan(&acquired); err != nil {
		return false, fmt.Errorf("failed to attempt lock acquisition: %w", err)
	}
	m.lockHeld = acquired
	return acquired, nil
}

// Unlock releases the advisory lock if it's currently held.
func (m *Mutex) Unlock() error {
	if !m.lockHeld {
		return fmt.Errorf("cannot unlock: lock not held by this instance")
	}

	if err := m.conn.QueryRow(m.ctx, "SELECT pg_advisory_unlock($1)", m.lockID).Scan(); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	m.lockHeld = false
	return nil
}
