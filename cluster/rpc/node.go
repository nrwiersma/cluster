package rpc

// NodesRequest is used to request a the nodes.
type NodesRequest struct {
	ReadRequest

	// Filter is a go-bexpr filter expression to filter the
	// nodes by before returning.
	Filter string
}

// NodesResponse are the nodes returned from
// a nodes request.
type NodesResponse struct {
	ResponseMeta

	// Nodes are the nodes found.
	Nodes []Node
}

// Node is a node.
type Node struct {
	ID      string
	Name    string
	Role    string
	Address string
	Health  string
	Meta    map[string]string
}
