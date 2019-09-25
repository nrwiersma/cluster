package fsm

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/hashicorp/raft"
	"github.com/pkg/errors"
)

type FSM struct{}

func New(nodeID int32) *FSM {
	return &FSM{}
}

func (f *FSM) Apply(l *raft.Log) interface{} {
	fmt.Printf("FSM APPLY: %#v\n", l)
	return nil
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	rc := ioutil.NopCloser(bytes.NewReader([]byte("snapshot")))

	fmt.Println("FSM SNAPSHOT")

	return &fsmSnapshot{b: rc}, nil
}

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
