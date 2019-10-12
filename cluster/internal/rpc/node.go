package rpc

import "github.com/nrwiersma/cluster/cluster/state"

// RegisterNodeRequest is used to register a node.
type RegisterNodeRequest struct {
	Node state.Node
}

// DeregisterNodeRequest is used to deregister a node.
type DeregisterNodeRequest struct {
	Node state.Node
}
