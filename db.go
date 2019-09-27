package cluster

import (
	"errors"
	"sync/atomic"
	"unsafe"

	"github.com/nrwiersma/cluster/cluster/db"
)

// DBProvider represents an object that can provide a database.
type DBProvider interface {
	DB() *db.DB
}

// DB exposes the cluster database in a consistent way.
type DB struct {
	*db.DB

	prov DBProvider
	stop chan struct{}
}

// NewDB returns a database.
func NewDB(prov DBProvider) (*DB, error) {
	if prov == nil {
		return nil, errors.New("db: p cannot be nil")
	}

	d := &DB{
		prov: prov,
		stop: make(chan struct{}),
	}

	d.setupDB()

	return d, nil
}

func (d *DB) setupDB() {
	// Atomically swap the dbs. No locks, not mess, no fuss.
	dbPtr := (*unsafe.Pointer)(unsafe.Pointer(&d.DB))
	atomic.StorePointer(dbPtr, unsafe.Pointer(d.prov.DB()))

	go func() {
		abandon := d.DB.AbandonCh()
		select {
		case <-abandon:
			// Once the DB has been abandoned, get the latest db
			d.setupDB()

		case <-d.stop:
		}
	}()
}

// Close closes the database.
func (d *DB) Close() error {
	close(d.stop)
	return nil
}
