package cluster

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/serf/serf"
	"github.com/nrwiersma/cluster/pkg/log"
)

func (s *Server) setupSerf(config *serf.Config, ch chan serf.Event, path string) (*serf.Serf, error) {
	config.Init()
	config.NodeName = s.config.NodeName
	config.Tags["role"] = s.config.Role
	config.Tags["id"] = fmt.Sprintf("%d", s.config.NodeID)
	//if s.config.Bootstrap {
	//	config.Tags["bootstrap"] = "1"
	//}
	if s.config.NonVoter {
		config.Tags["non_voter"] = "1"
	}
	config.Tags["raft_addr"] = s.config.RPCAddr
	config.Tags["serf_addr"] = fmt.Sprintf("%s:%d", s.config.SerfConfig.MemberlistConfig.BindAddr, s.config.SerfConfig.MemberlistConfig.BindPort)
	config.Logger = log.NewBridge(s.config.Logger, log.Debug, fmt.Sprintf("serf/%d: ", s.config.NodeID))
	config.MemberlistConfig.Logger = log.NewBridge(s.config.Logger, log.Debug, fmt.Sprintf("memberlist/%d: ", s.config.NodeID))
	config.EventCh = ch
	config.EnableNameConflictResolution = false
	config.SnapshotPath = filepath.Join(s.config.DataDir, path)

	if err := ensurePath(config.SnapshotPath, false); err != nil {
		return nil, err
	}

	return serf.Create(config)
}

func (s *Server) eventHandler() {
	for {
		select {
		case e := <-s.eventCh:
			switch e.EventType() {
			case serf.EventMemberJoin:
				s.nodeJoin(e.(serf.MemberEvent))
				s.localMemberEvent(e.(serf.MemberEvent))
			case serf.EventMemberReap:
				s.localMemberEvent(e.(serf.MemberEvent))
			case serf.EventMemberLeave, serf.EventMemberFailed:
				s.nodeFailed(e.(serf.MemberEvent))
				s.localMemberEvent(e.(serf.MemberEvent))
			}

		case <-s.shutdownCh:
			return
		}
	}
}

func (s *Server) nodeJoin(e serf.MemberEvent) {
	panic("TODO")
}

func (s *Server) nodeFailed(e serf.MemberEvent) {
	panic("TODO")
}

func (s *Server) localMemberEvent(e serf.MemberEvent) {
	panic("TODO")
}

// TODO: Need to maybe bootstrap here, for Bootstrap expect
