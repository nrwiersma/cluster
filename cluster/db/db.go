package db

import (
	"github.com/hashicorp/go-memdb"
)

// DB is a cluster state store.
type DB struct {
	schema *memdb.DBSchema
	db     *memdb.MemDB

	abandonCh chan struct{}
}

// New returns a cluster database.
func New() (*DB, error) {
	dbSchema := &memdb.DBSchema{
		Tables: make(map[string]*memdb.TableSchema),
	}

	db, err := memdb.NewMemDB(dbSchema)
	if err != nil {
		return nil, err
	}

	return &DB{
		db:        db,
		abandonCh: make(chan struct{}),
	}, nil
}

// Abandon is used to signal that the given state store has been abandoned.
// Calling this more than one time will panic.
func (d *DB) Abandon() {
	close(d.abandonCh)
}

// AbandonCh returns a channel you can wait on to know if the state store was
// abandoned.
func (d *DB) AbandonCh() <-chan struct{} {
	return d.abandonCh
}
