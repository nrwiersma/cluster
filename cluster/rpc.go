package cluster

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	"github.com/nrwiersma/cluster/cluster/metadata"
	"github.com/nrwiersma/cluster/cluster/rpc"
)

// ErrRaftLayerClosed is returned when performing an action on a closed
// RaftLayer.
var ErrRaftLayerClosed = errors.New("RaftLayer closed")

func (a *Agent) setupRPC() (err error) {
	a.ln, err = net.ListenTCP("tcp", a.config.RPCAddr)
	if err != nil {
		return err
	}

	if a.config.RPCAdvertise == nil {
		a.config.RPCAdvertise = a.ln.Addr().(*net.TCPAddr)
	}

	a.raftLayer = NewRaftLayer(a.config.RPCAdvertise)
	return nil
}

func (a *Agent) listen(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if a.shutdown {
				return
			}

			a.log.Error("agent: error accepting connection", "error", err)
			continue
		}

		a.handleConn(conn)
	}
}

func (a *Agent) handleConn(conn net.Conn) {
	_ = a.raftLayer.HandOff(conn)
}

var ErrNoLeader = errors.New("agent: no leader")

func (a *Agent) forward(t rpc.MessageType, msg interface{}) (bool, interface{}, error) {
	select {
	case <-a.leaveCh:
		return true, nil, ErrNoLeader
	default:
	}

	// Check leadership
	isLeader, leader := a.getLeader()
	if isLeader {
		return false, nil, nil
	}
	if leader == nil {
		return true, nil, ErrNoLeader
	}

	//reply, err := a.connPool.Apply(leader.RPCAddr, t, msg)
	//if err != nil {
	//	// TODO: perhaps retry the request
	//	return true, nil, err
	//}
	//return true, reply, err
	return false, nil, nil
}

func (a *Agent) getLeader() (bool, *metadata.Agent) {
	// Check if we are the leader
	if a.IsLeader() {
		return true, nil
	}

	// Get the leader
	leader := a.raft.Leader()
	if leader == "" {
		return false, nil
	}

	return false, a.agentLookup.AgentByAddr(leader)
}

// RaftLayer allows a single listener to be used for
// both Raft and a custom RPC layer.
type RaftLayer struct {
	addr net.Addr

	connCh chan net.Conn

	closeOnce sync.Once
	closeCh   chan struct{}
}

// NewRaftLayer creates a Raft layer.
func NewRaftLayer(addr net.Addr) *RaftLayer {
	return &RaftLayer{
		addr:    addr,
		connCh:  make(chan net.Conn),
		closeCh: make(chan struct{}),
	}
}

func (l *RaftLayer) HandOff(conn net.Conn) error {
	select {
	case l.connCh <- conn:
		return nil
	case <-l.closeCh:
		return ErrRaftLayerClosed
	}
}

// Accept accepts a new connection.
func (l *RaftLayer) Accept() (net.Conn, error) {
	select {
	case conn := <-l.connCh:
		return conn, nil
	case <-l.closeCh:
		return nil, ErrRaftLayerClosed
	}
}

// Addr returns the address of the listener.
func (l *RaftLayer) Addr() net.Addr {
	return l.addr
}

// Dial creates a new Raft outgoing connection.
func (l *RaftLayer) Dial(address raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	d := &net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", string(address))
	if err != nil {
		return nil, err
	}

	// Write the Raft byte to set the mode
	// TODO: add the raft rpc byte
	//_, err = conn.Write([]byte{byte(pool.RPCRaft)})
	//if err != nil {
	//	conn.Close()
	//	return nil, err
	//}
	return conn, err
}

// Close closes the the Raft layer
func (l *RaftLayer) Close() error {
	l.closeOnce.Do(func() {
		close(l.closeCh)
	})
	return nil
}
