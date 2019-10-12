package fsm

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/hashicorp/raft"
	rpc2 "github.com/nrwiersma/cluster/cluster/internal/rpc"
	"github.com/nrwiersma/cluster/cluster/state"
	"github.com/pkg/errors"
)

type handler func(buf []byte, index uint64) interface{}

// FSM is a finite state machine used by Raft to
// provide strong consistency.
type FSM struct {
	store *state.Store

	handlers map[rpc2.MessageType]handler
}

// New returns an FSM for the given agent id.
func New() (*FSM, error) {
	store, err := state.New()
	if err != nil {
		return nil, err
	}

	fsm := &FSM{
		store: store,
	}

	fsm.handlers = map[rpc2.MessageType]handler{
		rpc2.RegisterNodeRequestType:   fsm.handleRegisterNodeRequest,
		rpc2.DeregisterNodeRequestType: fsm.handleDeregisterNodeRequest,
	}

	return fsm, nil
}

// Store returns the current state store.
func (f *FSM) Store() *state.Store {
	return f.store
}

// Apply is invoked once a log has been committed.
func (f *FSM) Apply(l *raft.Log) interface{} {
	buf := l.Data
	msgType := rpc2.MessageType(buf[0])

	if fn := f.handlers[msgType]; fn != nil {
		return fn(buf[1:], l.Index)
	}

	// We dont know how to handle this message type,
	// perhaps it is from a future version.
	return nil
}

// Snapshot creates a snapshot of the current state of the FSM.
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	rc := ioutil.NopCloser(bytes.NewReader([]byte("snapshot")))

	// TODO: Handle this
	fmt.Println("FSM SNAPSHOT")

	return &fsmSnapshot{b: rc}, nil
}

// Restore restores the FSM to a previous state.
func (f *FSM) Restore(rc io.ReadCloser) error {
	b, err := ioutil.ReadAll(rc)
	if err != nil {
		return errors.Wrap(err, "fsm: error reading snapshot")
	}

	// TODO: Handle this
	fmt.Printf("FSM RESTORE: %s\n", string(b))

	return nil
}

type fsmSnapshot struct {
	b io.ReadCloser
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	_, err := io.Copy(sink, s.b)
	return err
}

func (s *fsmSnapshot) Release() {
	_ = s.b.Close()
}
