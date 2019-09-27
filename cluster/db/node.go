package db

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
	ID     string
	Health Health
	Meta   map[string]string
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
		},
	}
}

// Node returns a node with the given id or nil.
func (d *DB) Node(id string) (*Node, error) {
	tx := d.db.Txn(false)
	defer tx.Abort()

	node, err := tx.First("nodes", "id", id)
	if err != nil {
		return nil, fmt.Errorf("db: node lookup failed: %s", err)
	}
	if node != nil {
		return node.(*Node), nil
	}
	return nil, nil
}

// Nodes returns all the nodes.
func (d *DB) Nodes() ([]*Node, error) {
	tx := d.db.Txn(false)
	defer tx.Abort()

	it, err := tx.Get("nodes", "id")
	if err != nil {
		return nil, fmt.Errorf("node lookup failed: %s", err)
	}
	var nodes []*Node
	for next := it.Next(); next != nil; next = it.Next() {
		nodes = append(nodes, next.(*Node))
	}
	return nodes, nil
}

// EnsureNode upserts a node in the database.
func (d *DB) EnsureNode(idx uint64, node *Node) error {
	tx := d.db.Txn(true)
	defer tx.Abort()

	if err := tx.Insert("nodes", node); err != nil {
		return fmt.Errorf("db: failed inserting node: %w", err)
	}

	tx.Commit()
	return nil
}

// DeleteNode deletes a node from the database with the given id.
func (d *DB) DeleteNode(idx uint64, id string) error {
	tx := d.db.Txn(true)
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
	return nil
}
