package pgxmutex

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

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
func WithConn(conn conn) Option {
	return func(m *Mutex) error {
		m.conn = conn
		return nil
	}
}

// WithResourceID sets the lock ID for advisory locking.
func WithResourceID(id int64) Option {
	return func(m *Mutex) error {
		if id == 0 {
			return fmt.Errorf("resource ID must be provided")
		}
		m.so = getSingleton(id)
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
