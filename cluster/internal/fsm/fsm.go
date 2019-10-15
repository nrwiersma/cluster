package fsm

import (
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/go-msgpack/codec"
	"github.com/hashicorp/raft"
	"github.com/nrwiersma/cluster/cluster/internal/rpc"
	"github.com/nrwiersma/cluster/cluster/state"
)

// msgpackHandle is a shared handle for encoding/decoding msgpack payloads
var msgpackHandle = &codec.MsgpackHandle{}

// handler is a function used to handle an FSM state request.
type handler func(buf []byte, index uint64) interface{}

// snapshoter is a function used to help snapshot the FSM state.
type snapshoter func(s *snapshot, sink raft.SnapshotSink, encoder *codec.Encoder) error

// restorer is a function used to load back a snapshot of the FSM state.
type restorer func(header *snapshotHeader, restore *state.Restore, decoder *codec.Decoder) error

// FSM is a finite state machine used by Raft to
// provide strong consistency.
type FSM struct {
	storeMu sync.Mutex
	store   *state.Store

	handlers    map[rpc.MessageType]handler
	snapshoters []snapshoter
	restorers   map[rpc.MessageType]restorer
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

	fsm.handlers = map[rpc.MessageType]handler{
		rpc.RegisterNodeRequestType:   fsm.handleRegisterNodeRequest,
		rpc.DeregisterNodeRequestType: fsm.handleDeregisterNodeRequest,
	}

	fsm.snapshoters = []snapshoter{
		snapshotNodes,
		snapshotIndexes, // It is pretty important that this goes last
	}

	fsm.restorers = map[rpc.MessageType]restorer{
		rpc.RegisterNodeRequestType: restoreNode,
		rpc.IndexRequestType:        restoreIndexes,
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
	return &snapshot{
		state:       f.store.Snapshot(),
		snapshoters: f.snapshoters,
	}, nil
}

// Restore restores the FSM to a previous state.
func (f *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	newStore, err := state.New()
	if err != nil {
		return err
	}

	restore := newStore.Restore()
	defer restore.Abort()

	dec := codec.NewDecoder(rc, msgpackHandle)

	var header snapshotHeader
	if err := dec.Decode(&header); err != nil {
		return err
	}

	msgType := make([]byte, 1)
	for {
		_, err := rc.Read(msgType)
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		msg := rpc.MessageType(msgType[0])
		if fn := f.restorers[msg]; fn != nil {
			if err := fn(&header, restore, dec); err != nil {
				return err
			}
			continue
		}

		return fmt.Errorf("unrecognized msg type %d", msg)
	}
	restore.Commit()

	f.storeMu.Lock()
	oldStore := f.store
	f.store = newStore
	f.storeMu.Unlock()

	oldStore.Abandon()
	return nil
}

func snapshotIndexes(s *snapshot, sink raft.SnapshotSink, enc *codec.Encoder) error {
	iter, err := s.state.Indexes()
	if err != nil {
		return err
	}

	for raw := iter.Next(); raw != nil; raw = iter.Next() {
		idx := raw.(*state.IndexEntry)

		_, _ = sink.Write([]byte{byte(rpc.IndexRequestType)})
		if err := enc.Encode(idx); err != nil {
			return err
		}
	}
	return nil
}

func restoreIndexes(header *snapshotHeader, restore *state.Restore, dec *codec.Decoder) error {
	var req state.IndexEntry
	if err := dec.Decode(&req); err != nil {
		return err
	}
	return restore.Index(&req)
}
