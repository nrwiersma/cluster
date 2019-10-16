package rpc

import "time"

// Request represents an RPC request.
type Request interface {
	IsRead() bool
	AllowStaleRead() bool
}

// ReadRequest configures an RPC read request.
type ReadRequest struct {
	// If set will force a consistent read.
	ForceFresh bool

	// If set, wait until query exceeds given index. Must be provided
	// with MaxQueryTime.
	MinQueryIndex uint64

	// Provided with MinQueryIndex to wait for change.
	MaxQueryTime time.Duration
}

// IsRead determines if the request is a read request.
func (r ReadRequest) IsRead() bool {
	return true
}

// AllowStaleRead determines if reading is allowed on
// non-leader nodes.
func (r ReadRequest) AllowStaleRead() bool {
	return !r.ForceFresh
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

// ResponseMeta allows the response to contain metadata.
type ResponseMeta struct {
	// Is the index of the read.
	Index uint64
}
