package state

import (
	"github.com/hashicorp/go-memdb"
)

// Store is a cluster state store.
type Store struct {
	schema *memdb.DBSchema
	db     *memdb.MemDB

	abandonCh chan struct{}
}

// New returns a cluster state store.
func New() (*Store, error) {
	dbSchema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			"nodes": nodesTableSchema(),
		},
	}

	db, err := memdb.NewMemDB(dbSchema)
	if err != nil {
		return nil, err
	}

	return &Store{
		db:        db,
		abandonCh: make(chan struct{}),
	}, nil
}

// Abandon is used to signal that the given state store has been abandoned.
// Calling this more than one time will panic.
func (d *Store) Abandon() {
	close(d.abandonCh)
}

// AbandonCh returns a channel you can wait on to know if the state store was
// abandoned.
func (d *Store) AbandonCh() <-chan struct{} {
	return d.abandonCh
}