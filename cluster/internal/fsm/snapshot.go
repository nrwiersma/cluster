package fsm

import (
	"github.com/hashicorp/go-msgpack/codec"
	"github.com/hashicorp/raft"
	"github.com/nrwiersma/cluster/cluster/state"
)

type snapshotHeader struct {
	LastIndex uint64
}

type snapshot struct {
	state *state.Snapshot

	snapshoters []snapshoter
}

// Persist saves the FSM snapshot out to the given sink.
func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	// Write the header
	header := snapshotHeader{
		LastIndex: s.state.LastIndex(),
	}
	encoder := codec.NewEncoder(sink, msgpackHandle)
	if err := encoder.Encode(&header); err != nil {
		sink.Cancel()
		return err
	}

	// Run all the snapshoters to write the FSM state.
	for _, fn := range s.snapshoters {
		if err := fn(s, sink, encoder); err != nil {
			sink.Cancel()
			return err
		}
	}
	return nil
}

// Release releases the snapshot.
func (s *snapshot) Release() {
	s.state.Close()
}
