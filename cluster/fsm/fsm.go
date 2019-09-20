package fsm

import (
	"io"

	"github.com/hashicorp/raft"
)

type FSM struct {
}

func New(nodeID int32) *FSM {
	return &FSM{

	}
}

func (f *FSM) Apply(*raft.Log) interface{} {
	panic("implement me")
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	panic("implement me")
}

func (f *FSM) Restore(io.ReadCloser) error {
	panic("implement me")
}
