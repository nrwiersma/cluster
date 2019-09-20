package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-sockaddr"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
	"github.com/hashicorp/raft-boltdb"
	"github.com/hashicorp/serf/serf"
)

var (
	members  string
	serfPort int
)

func init() {
	flag.StringVar(&members, "members", "", "127.0.0.1:1111,127.0.0.1:2222")
	flag.IntVar(&serfPort, "serfPort", 0, "1111")
}

func main() {
	flag.Parse()

	var peers []string
	if members != "" {
		peers = strings.Split(members, ",")
	}

	ip, err := sockaddr.GetPrivateIP()
	if err != nil {
		log.Fatal(err)
	}

	serfEvents := make(chan serf.Event, 16)

	memberlistConfig := memberlist.DefaultLANConfig()
	memberlistConfig.BindAddr = ip
	memberlistConfig.BindPort = serfPort
	memberlistConfig.LogOutput = os.Stdout

	serfConfig := serf.DefaultConfig()
	serfConfig.NodeName = fmt.Sprintf("%s:%d", ip, serfPort)
	serfConfig.EventCh = serfEvents
	serfConfig.MemberlistConfig = memberlistConfig
	serfConfig.LogOutput = os.Stdout

	s, err := serf.Create(serfConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Join an existing cluster by specifying at least one known member.
	if len(peers) > 0 {
		_, err = s.Join(peers, false)
		if err != nil {
			log.Fatal(err)
		}
	}

	workDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	raftPort := serfPort + 1

	id := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s:%d", ip, raftPort))))

	dataDir := filepath.Join(workDir, "data", id)
	defer func() {
		os.RemoveAll(dataDir)
	}()

	fmt.Printf("Data Dir: %s", dataDir)
	err = os.RemoveAll(dataDir + "/raft/")
	if err != nil {
		log.Fatal(err)
	}

	if err = os.MkdirAll(dataDir, 0777); err != nil {
		log.Fatal(err)
	}

	raftDBPath := filepath.Join(dataDir, "raft.db")
	raftDB, err := raftboltdb.NewBoltStore(raftDBPath)
	if err != nil {
		log.Fatal(err)
	}

	snapshotStore, err := raft.NewFileSnapshotStore(dataDir, 1, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	raftAddr := ip + ":" + strconv.Itoa(raftPort)

	trans, err := raft.NewTCPTransport(raftAddr, nil, 3, 10*time.Second, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}

	c := raft.DefaultConfig()
	c.LogOutput = os.Stdout
	c.LocalID = raft.ServerID(raftAddr)
	c.SnapshotThreshold = 10
	c.TrailingLogs = 5

	r, err := raft.NewRaft(c, &fsm{}, raftDB, raftDB, snapshotStore, trans)
	if err != nil {
		log.Fatal(err)
	}

	bootstrapConfig := raft.Configuration{
		Servers: []raft.Server{
			{
				Suffrage: raft.Voter,
				ID:       raft.ServerID(raftAddr),
				Address:  raft.ServerAddress(raftAddr),
			},
		},
	}

	// Add known peers to bootstrap
	//for _, node := range peers {
	//	if node == raftAddr {
	//		continue
	//	}
	//
	//	bootstrapConfig.Servers = append(bootstrapConfig.Servers, raft.Server{
	//		Suffrage: raft.Voter,
	//		NodeID:       raft.ServerID(node),
	//		Address:  raft.ServerAddress(node),
	//	})
	//}

	f := r.BootstrapCluster(bootstrapConfig)
	if err := f.Error(); err != nil {
		log.Fatalf("error bootstrapping: %s", err)
	}

	ticker := time.NewTicker(3 * time.Second)
	dataTicker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			fmt.Printf("Showing peers known by %s:\n", raftAddr)
			future := r.VerifyLeader()
			if err = future.Error(); err != nil {
				fmt.Println("Node is a follower")
			} else {
				fmt.Println("Node is leader")
			}

			cfuture := r.GetConfiguration()

			if err = cfuture.Error(); err != nil {
				log.Fatalf("error getting config: %s", err)
			}

			configuration := cfuture.Configuration()
			for _, server := range configuration.Servers {
				fmt.Println(server.Address)
			}

		case <-dataTicker.C:
			if r.State() == raft.Leader {
				fmt.Println("Leader is writing")
				r.Apply([]byte("hello"), 3*time.Second)
			}

		case ev := <-serfEvents:
			fmt.Printf("SERF EVENT: %#v\n", ev)

			leader := r.VerifyLeader()

			if memberEvent, ok := ev.(serf.MemberEvent); ok {

				for _, member := range memberEvent.Members {
					changedPeer := member.Addr.String() + ":" + strconv.Itoa(int(member.Port+1))

					switch ev.EventType() {
					case serf.EventMemberJoin:
						if leader.Error() == nil {
							f := r.AddVoter(raft.ServerID(changedPeer), raft.ServerAddress(changedPeer), 0, 0)
							if f.Error() != nil {
								log.Fatalf("error adding voter: %s", f.Error())
							}

						}

					case serf.EventMemberLeave, serf.EventMemberFailed, serf.EventMemberReap:
						if leader.Error() == nil {
							f := r.RemoveServer(raft.ServerID(changedPeer), 0, 0)
							if f.Error() != nil {
								log.Fatalf("error removing server: %s", f.Error())
							}
						}
					}
				}
			}

		}
	}
}

type snapshot struct{}

func (s snapshot) Persist(sink raft.SnapshotSink) error {
	_, err := sink.Write([]byte("foobar"))
	return err
}

func (s snapshot) Release() {
	// Nothing to do
}

type fsm struct {
}

func (f *fsm) Apply(log *raft.Log) interface{} {
	fmt.Printf("FSM APPLY: %#v\n", log)
	return nil
}

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	fmt.Println("SNAPSHOT CALLED")
	return snapshot{}, nil
}

func (f *fsm) Restore(r io.ReadCloser) error {
	defer r.Close()

	fmt.Println("RESTORE CALLED")
	return nil
}
