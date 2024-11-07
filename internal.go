package pgxmutex

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type conn interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...interface{}) pgx.Row
}

type singleton struct {
	sync.Mutex
	id int64
}

var singletons = make(map[int64]*singleton)
var singletonsMutex sync.Mutex

func getSingleton(id int64) *singleton {
	singletonsMutex.Lock()
	defer singletonsMutex.Unlock()

	if s, ok := singletons[id]; ok {
		return s
	}

	s := &singleton{id: id}
	singletons[id] = s
	return s
}
