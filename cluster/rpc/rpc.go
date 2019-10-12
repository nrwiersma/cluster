package rpc

// RPCRequest represents an RPC request.
type RPCRequest interface {
	IsRead() bool
	AllowStaleRead() bool
}

// ReadRequest configures an RPC read request.
type ReadRequest struct {
	AllowStale bool
}

// IsRead determines if the request is a read request.
func (r ReadRequest) IsRead() bool {
	return true
}

// AllowStaleRead determines if reading is allowed on
// non-leader nodes.
func (r ReadRequest) AllowStaleRead() bool {
	return r.AllowStale
}

// WriteRequest configures an RPC write request.
type WriteRequest struct{}

// IsRead determines if the request is a read request.
func (r WriteRequest) IsRead() bool {
	return false
}

// AllowStaleRead determines if reading is allowed on
// non-leader nodes.
func (r WriteRequest) AllowStaleRead() bool {
	return false
}
