package state

import (
	"fmt"

	"github.com/hashicorp/go-memdb"
)

// RaftIndex holds the raft index of a record.
type RaftIndex struct {
	Index uint64
}

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
			"index": indexTableSchema(),
			"nodes": nodesTableSchema(),
		},
	}

	db, err := memdb.NewMemDB(dbSchema)
	if err != nil {
		return nil, err
	}

	return &Store{
		schema:    dbSchema,
		db:        db,
		abandonCh: make(chan struct{}),
	}, nil
}

// Abandon is used to signal that the given state store has been abandoned.
// Calling this more than one time will panic.
func (s *Store) Abandon() {
	close(s.abandonCh)
}

// AbandonCh returns a channel you can wait on to know if the state store was
// abandoned.
func (s *Store) AbandonCh() <-chan struct{} {
	return s.abandonCh
}

// Snapshot is used to create a point-in-time snapshot of the entire db.
func (s *Store) Snapshot() *Snapshot {
	tx := s.db.Txn(false)

	var tables []string
	for table := range s.schema.Tables {
		tables = append(tables, table)
	}
	idx := maxIndex(tx, tables...)

	return &Snapshot{tx, idx}
}

// Restore is used to efficiently manage restoring a large amount of data into
// the state store. It works by doing all the restores inside of a single
// transaction.
func (s *Store) Restore() *Restore {
	return &Restore{s, s.db.Txn(true)}
}

// Snapshot is used to provide a point-in-time snapshot. It
// works by starting a read transaction against the whole state store.
type Snapshot struct {
	tx        *memdb.Txn
	lastIndex uint64
}

// LastIndex returns that last index that affects the snapshotted data.
func (s *Snapshot) LastIndex() uint64 {
	return s.lastIndex
}

// Close performs cleanup of a state snapshot.
func (s *Snapshot) Close() {
	s.tx.Abort()
}

// Restore is used to efficiently manage restoring a large amount of
// data to a state store.
type Restore struct {
	store *Store
	tx    *memdb.Txn
}

// Abort abandons the changes made by a restore. This or Commit should always be
// called.
func (r *Restore) Abort() {
	r.tx.Abort()
}

// Commit commits the changes made by a restore. This or Abort should always be
// called.
func (r *Restore) Commit() {
	r.tx.Commit()
}

// IndexEntry keeps a record of the last index per-table.
type IndexEntry struct {
	Table string
	Index uint64
}

// indexTableSchema returns a new table schema used for tracking various indexes for the Raft log.
func indexTableSchema() *memdb.TableSchema {
	return &memdb.TableSchema{
		Name: "index",
		Indexes: map[string]*memdb.IndexSchema{
			"id": {
				Name:         "id",
				AllowMissing: false,
				Unique:       true,
				Indexer: &memdb.StringFieldIndex{
					Field:     "Table",
					Lowercase: true,
				},
			},
		},
	}
}

func updateIndex(tx *memdb.Txn, tbl string, idx uint64) error {
	return tx.Insert("index", &IndexEntry{Table: tbl, Index: idx})
}

func maxIndex(tx *memdb.Txn, tables ...string) uint64 {
	var max uint64

	for _, table := range tables {
		ti, err := tx.First("index", "id", table)
		if err != nil {
			continue
		}

		if idx, ok := ti.(*IndexEntry); ok && idx.Index > max {
			max = idx.Index
		}
	}
	return max
}

// Indexes returns indexes for snapshotting.
func (s *Snapshot) Indexes() (memdb.ResultIterator, error) {
	iter, err := s.tx.Get("index", "id")
	if err != nil {
		return nil, err
	}
	return iter, nil
}

// Index is used to restore an index
func (r *Restore) Index(idx *IndexEntry) error {
	if err := r.tx.Insert("index", idx); err != nil {
		return fmt.Errorf("index insert failed: %v", err)
	}
	return nil
}
