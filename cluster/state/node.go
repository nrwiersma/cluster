package state

import (
	"fmt"

	"github.com/hashicorp/go-memdb"
)

// Health is the health of a node.
type Health string

// Health constants.
const (
	HealthPassing  Health = "passing"
	HealthCritical        = "critical"
)

// Node is used to store info about a node.
type Node struct {
	ID      string
	Name    string
	Role    string
	Address string
	Health  Health
	Meta    map[string]string

	RaftIndex
}

// Same checks is the nodes look the same.
func (n *Node) Same(o *Node) bool {
	return n.ID == o.ID &&
		n.Name == o.Name &&
		n.Role == o.Role &&
		n.Address == o.Address &&
		n.Health == o.Health
}

func nodesTableSchema() *memdb.TableSchema {
	return &memdb.TableSchema{
		Name: "nodes",
		Indexes: map[string]*memdb.IndexSchema{
			"id": {
				Name:         "id",
				AllowMissing: false,
				Unique:       true,
				Indexer: &memdb.StringFieldIndex{
					Field: "ID",
				},
			},
			"name": {
				Name:         "name",
				AllowMissing: false,
				Unique:       true,
				Indexer: &memdb.StringFieldIndex{
					Field: "Name",
				},
			},
		},
	}
}

// Nodes returns the nodes for a given snapshot.
func (s *Snapshot) Nodes() (memdb.ResultIterator, error) {
	return s.tx.Get("nodes", "id")
}

// Node restores a node.
func (r *Restore) Node(idx uint64, node *Node) error {
	return ensureNodeTx(r.tx, idx, node)
}

// Node returns a node with the given id or nil.
func (s *Store) Node(id string) (uint64, *Node, error) {
	tx := s.db.Txn(false)
	defer tx.Abort()

	idx := maxIndex(tx, "nodes")
	node, err := tx.First("nodes", "id", id)
	if err != nil {
		return 0, nil, fmt.Errorf("db: node lookup failed: %s", err)
	}
	if node != nil {
		return idx, node.(*Node), nil
	}
	return idx, nil, nil
}

// Nodes returns all the nodes as well as a watch channel that will be closed
// when the nodes change.
func (s *Store) Nodes(ws memdb.WatchSet) (uint64, []*Node, error) {
	tx := s.db.Txn(false)
	defer tx.Abort()

	idx := maxIndex(tx, "nodes")
	iter, err := tx.Get("nodes", "id")
	if err != nil {
		return 0, nil, fmt.Errorf("node lookup failed: %s", err)
	}
	ws.Add(iter.WatchCh())

	var nodes []*Node
	for next := iter.Next(); next != nil; next = iter.Next() {
		nodes = append(nodes, next.(*Node))
	}
	return idx, nodes, nil
}

// EnsureNode inserts a node in the database. This is used by the FSM to
// add and update nodes.
func (s *Store) EnsureNode(idx uint64, node *Node) error {
	tx := s.db.Txn(true)
	defer tx.Abort()

	if err := ensureNodeTx(tx, idx, node); err != nil {
		return err
	}

	tx.Commit()
	return nil
}

func ensureNodeTx(tx *memdb.Txn, idx uint64, node *Node) error {
	node.Index = idx

	existing, err := tx.First("nodes", "id", node.ID)
	if err != nil {
		return err
	}

	// If the node already exists, check if an update is needed, otherwise leave it.
	// This will stop chatty updates about nodes from serf.
	if existing != nil && node.Same(existing.(*Node)) {
		return nil
	}

	if err := tx.Insert("nodes", node); err != nil {
		return fmt.Errorf("db: failed inserting node: %w", err)
	}
	if err := updateIndex(tx, "nodes", idx); err != nil {
		return fmt.Errorf("failed updating index: %w", err)
	}
	return nil
}

// DeleteNode deletes a node from the database with the given id. This is
// used by the FSM to delete nodes.
func (s *Store) DeleteNode(idx uint64, id string) error {
	tx := s.db.Txn(true)
	defer tx.Abort()

	node, err := tx.First("nodes", "id", id)
	if err != nil {
		return err
	}
	if node == nil {
		return nil
	}

	if err := tx.Delete("nodes", node); err != nil {
		return err
	}
	if err := updateIndex(tx, "nodes", idx); err != nil {
		return fmt.Errorf("failed updating index: %w", err)
	}

	tx.Commit()
	return nil
}
