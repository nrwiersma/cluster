package rpc

// NodesRequest is used to request a the nodes.
type NodesRequest struct {
	// Filter is a go-bexpr filter expression to filter the
	// nodes by before returning
	//Filter string

	ReadRequest
}

// NodesResponse are the nodes returned from
// a nodes request.
type NodesResponse struct {
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
