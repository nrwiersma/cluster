package cluster

import (
	"errors"
	"sync/atomic"
	"unsafe"

	"github.com/nrwiersma/cluster/cluster/state"
)

// StoreProvider represents an object that can provide a state store.
type StoreProvider interface {
	Store() *state.Store
}

// DB exposes the cluster database in a consistent way.
type DB struct {
	*state.Store

	prov StoreProvider
	stop chan struct{}
}

// NewDB returns a database.
func NewDB(prov StoreProvider) (*DB, error) {
	if prov == nil {
		return nil, errors.New("db: p cannot be nil")
	}

	d := &DB{
		prov: prov,
		stop: make(chan struct{}),
	}

	d.swapStore()
	go func() {
		for {
			abandon := d.Store.AbandonCh()
			select {
			case <-abandon:
				// Once the store has been abandoned, get the latest store
				d.swapStore()

			case <-d.stop:
				return
			}
		}
	}()

	return d, nil
}

// swapStore atomically swaps the state stores without locking.
func (d *DB) swapStore() {
	storePtr := (*unsafe.Pointer)(unsafe.Pointer(&d.Store))
	atomic.StorePointer(storePtr, unsafe.Pointer(d.prov.Store()))
}

// Close closes the database.
func (d *DB) Close() error {
	close(d.stop)
	return nil
}
