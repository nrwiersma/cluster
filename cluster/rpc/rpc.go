package rpc

import (
	"bytes"

	"github.com/hashicorp/go-msgpack/codec"
	"github.com/nrwiersma/cluster/cluster/db"
)

// MessageType is an RPC message type.
type MessageType int8

// RPC message types.
const (
	RegisterNodeType MessageType = iota
	DeregisterNodeType
)

// RegisterNode is used to register a node.
type RegisterNode struct {
	Node db.Node
}

// DeregisterNode is used to deregister a node.
type DeregisterNode struct {
	Node db.Node
}

// msgpackHandle is a shared handle for encoding/decoding of RPC objects.
var msgpackHandle = &codec.MsgpackHandle{}

// Decode decodes an RPC object without a type header.
func Decode(buf []byte, out interface{}) error {
	return codec.NewDecoder(bytes.NewReader(buf), msgpackHandle).Decode(out)
}

// Encode encodes an RPC object with a type header.
func Encode(t MessageType, msg interface{}) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte(uint8(t))
	err := codec.NewEncoder(&buf, msgpackHandle).Encode(msg)
	return buf.Bytes(), err
}
