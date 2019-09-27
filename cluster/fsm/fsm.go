package fsm

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/hashicorp/raft"
	"github.com/nrwiersma/cluster/cluster/db"
	"github.com/nrwiersma/cluster/cluster/rpc"
	"github.com/pkg/errors"
)

type handler func(buf []byte, index uint64) interface{}

// FSM is a finite state machine used by Raft to
// provide strong consistency.
type FSM struct {
	db *db.DB

	handlers map[rpc.MessageType]handler
}

// New returns an FSM for the given agent id.
func New() (*FSM, error) {
	db, err := db.New()
	if err != nil {
		return nil, err
	}

	fsm := &FSM{
		db: db,
	}

	fsm.handlers = map[rpc.MessageType]handler{
		rpc.RegisterNodeType:   fsm.handleRegisterNode,
		rpc.DeregisterNodeType: fsm.handleDeregisterNode,
	}

	return fsm, nil
}

// DB returns the current state database.
func (f *FSM) DB() *db.DB {
	return f.db
}

// Apply is invoked once a log has been committed.
func (f *FSM) Apply(l *raft.Log) interface{} {
	buf := l.Data
	msgType := rpc.MessageType(buf[0])

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

	fmt.Println("FSM SNAPSHOT")

	return &fsmSnapshot{b: rc}, nil
}

// Restore retores the FSM to a previous state.
func (f *FSM) Restore(rc io.ReadCloser) error {
	b, err := ioutil.ReadAll(rc)
	if err != nil {
		return errors.Wrap(err, "fsm: error reading snapshot")
	}

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
