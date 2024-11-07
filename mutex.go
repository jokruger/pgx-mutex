package pgxmutex

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Conn interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...interface{}) pgx.Row
}

// Mutex is a distributed lock based on PostgreSQL advisory locks
type Mutex struct {
	conn Conn
	id   int64
	ctx  context.Context
	m    sync.Mutex
}

// Option is a functional option type for configuring Mutex.
type Option func(*Mutex) error

// WithConnStr creates new PGX connection from a connection string.
func WithConnStr(connStr string) Option {
	return func(m *Mutex) error {
		conn, err := pgx.Connect(context.Background(), connStr)
		if err != nil {
			return fmt.Errorf("failed to connect to database: %w", err)
		}
		m.conn = conn
		return nil
	}
}

// WithConn sets the custom DB connection.
func WithConn(conn Conn) Option {
	return func(m *Mutex) error {
		m.conn = conn
		return nil
	}
}

// WithResourceID sets the lock ID for advisory locking.
func WithResourceID(id int64) Option {
	return func(m *Mutex) error {
		m.id = id
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
	if m.conn == nil {
		return nil, fmt.Errorf("database connection must be provided")
	}

	// Generate a lock ID if not provided
	if m.id == 0 {
		m.id = time.Now().UnixNano()
	}

	return m, nil
}

// SyncMutex is a wrapper around Mutex that implements sync.Locker interface.
func (m *Mutex) SyncMutex() SyncMutex {
	return SyncMutex{m: m}
}

// Lock tries to acquire the advisory lock, blocking until it's available.
func (m *Mutex) Lock() error {
	m.m.Lock()
	defer m.m.Unlock()

	if _, err := m.conn.Exec(m.ctx, "SELECT pg_advisory_lock($1)", m.id); err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	return nil
}

// TryLock attempts to acquire the advisory lock without blocking.
// Returns an error if unable to acquire the lock.
func (m *Mutex) TryLock() (bool, error) {
	m.m.Lock()
	defer m.m.Unlock()

	var acquired bool
	if err := m.conn.QueryRow(m.ctx, "SELECT pg_try_advisory_lock($1)", m.id).Scan(&acquired); err != nil {
		return false, fmt.Errorf("failed to attempt lock acquisition: %w", err)
	}

	return acquired, nil
}

// Unlock releases the advisory lock if it's currently held.
func (m *Mutex) Unlock() error {
	m.m.Lock()
	defer m.m.Unlock()

	if _, err := m.conn.Exec(m.ctx, "SELECT pg_advisory_unlock($1)", m.id); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	return nil
}

// GetResourceID returns the lock ID.
func (m *Mutex) GetResourceID() int64 {
	return m.id
}
