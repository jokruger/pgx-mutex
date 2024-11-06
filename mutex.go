package pgxmutex

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Mutex is a distributed lock based on PostgreSQL advisory locks
type Mutex struct {
	pool   *pgxpool.Pool
	lockID int64
	ctx    context.Context
}

// Option is a functional option type for configuring Mutex.
type Option func(*Mutex) error

// WithConnStr creates new PGX connection from a connection string.
func WithConnStr(connStr string) Option {
	return func(m *Mutex) error {
		pool, err := pgxpool.New(context.Background(), connStr)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		m.pool = pool
		return nil
	}
}

// WithPool sets the PGX connection pool.
func WithPool(pool *pgxpool.Pool) Option {
	return func(m *Mutex) error {
		m.pool = pool
		return nil
	}
}

// WithLockID sets the lock ID for advisory locking.
func WithLockID(lockID int64) Option {
	return func(m *Mutex) error {
		m.lockID = lockID
		return nil
	}
}

// WithContext sets a custom context for the Mutex operations.
func WithContext(ctx context.Context) Option {
	return func(m *Mutex) error {
		m.ctx = ctx
		return nil
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
		if err := opt(m); err != nil {
			return nil, err
		}
	}

	// Check required fields
	if m.pool == nil {
		return nil, fmt.Errorf("database connection pool must be provided")
	}

	// Generate a lock ID if not provided
	if m.lockID == 0 {
		m.lockID = time.Now().UnixNano()
	}

	return m, nil
}

// SyncMutex is a wrapper around Mutex that implements sync.Locker interface.
func (m *Mutex) SyncMutex() SyncMutex {
	return SyncMutex{m: m}
}

// Lock tries to acquire the advisory lock, blocking until it's available.
func (m *Mutex) Lock() error {
	if err := m.pool.QueryRow(m.ctx, "SELECT pg_advisory_lock($1)", m.lockID).Scan(); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	return nil
}

// TryLock attempts to acquire the advisory lock without blocking.
// Returns an error if unable to acquire the lock.
func (m *Mutex) TryLock() (bool, error) {
	var acquired bool
	if err := m.pool.QueryRow(m.ctx, "SELECT pg_try_advisory_lock($1)", m.lockID).Scan(&acquired); err != nil {
		return false, fmt.Errorf("failed to attempt lock acquisition: %w", err)
	}
	return acquired, nil
}

// Unlock releases the advisory lock if it's currently held.
func (m *Mutex) Unlock() error {
	if err := m.pool.QueryRow(m.ctx, "SELECT pg_advisory_unlock($1)", m.lockID).Scan(); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}
