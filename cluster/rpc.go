package cluster

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	rpc2 "github.com/nrwiersma/cluster/cluster/internal/rpc"
	"github.com/nrwiersma/cluster/cluster/metadata"
	"github.com/nrwiersma/cluster/cluster/rpc"
	"github.com/nrwiersma/cluster/cluster/server"
	"github.com/nrwiersma/cluster/cluster/state"
)

// ErrRaftLayerClosed is returned when performing an action on a closed
// RaftLayer.
var ErrRaftLayerClosed = errors.New("RaftLayer closed")

// ErrNoLeader is returned when no leader can be found.
var ErrNoLeader = errors.New("agent: no leader")

type agentStateDelegate struct {
	agent *Agent
}

func (d *agentStateDelegate) Store() *state.Store {
	return d.agent.fsm.Store()
}

func (d *agentStateDelegate) Apply(t rpc2.MessageType, msg interface{}) (interface{}, error) {
	return d.agent.raftApply(t, msg)
}

func (a *Agent) setupRPC() (err error) {
	a.srv = server.New(&agentStateDelegate{a})

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

func (a *Agent) forward(method string, req, resp interface{}) (bool, error) {
	select {
	case <-a.leaveCh:
		return true, ErrNoLeader
	default:
	}

	// Check leadership
	isLeader, leader := a.getLeader()
	if isLeader {
		return false, nil
	}
	if leader == nil {
		return true, ErrNoLeader
	}

	// If the request allows stale reads, lots it fall through to the current agent
	if rpcReq, ok := req.(rpc.Request); ok {
		if rpcReq.IsRead() && rpcReq.AllowStaleRead() {
			return false, nil
		}
	}

	//if err := a.connPool.RPC(leader.RPCAddr, method, req, resp); err != nil {
	//	// TODO: perhaps retry the request
	//	return true, err
	//}
	//return true, err
	return false, nil
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

// HandOff hands a connection off to raft.
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
