package db

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
	Address string
	Health  Health
	Meta    map[string]string
}

// EnsureNode upserts a node in the database.
func (db *DB) EnsureNode(idx uint64, node *Node) error {
	return nil
}

// DeleteNode deletes a node from the database with the given id.
func (db *DB) DeleteNode(idx uint64, id string) error {
	return nil
}
