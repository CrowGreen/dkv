package replication

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/CrowGreen/dkv/internal/store"
)

type NodeState int

const (
	Follower NodeState = iota
	Candidate
	Leader
)

type LogEntry struct {
	Index     int64
	Term      int64
	Key       string
	Value     []byte
	Operation string // "PUT" or "DELETE"
}

type ReplicaNode struct {
	mu sync.Mutex

	id    string
	state NodeState
	term  int64

	store *store.Store
	log   []LogEntry

	commitIndex int64
	lastApplied int64

	peers       []string
	leaderID    string
	votedFor    string
	votes       int

	heartbeatCh chan struct{}
	stopCh      chan struct{}
}

func NewNode(id string, peers []string, s *store.Store) *ReplicaNode {
	return &ReplicaNode{
		id:          id,
		state:       Follower,
		store:       s,
		peers:       peers,
		heartbeatCh: make(chan struct{}, 1),
		stopCh:      make(chan struct{}),
	}
}

func (n *ReplicaNode) Start() {
	go n.runElectionTimer()
}

func (n *ReplicaNode) Stop() {
	close(n.stopCh)
}

func (n *ReplicaNode) IsLeader() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.state == Leader
}

func (n *ReplicaNode) GetState() (int64, NodeState) {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.term, n.state
}

// AppendEntry adds a new entry to the leader's log and replicates
func (n *ReplicaNode) AppendEntry(key string, value []byte, op string) (int64, error) {
	n.mu.Lock()
	if n.state != Leader {
		n.mu.Unlock()
		return 0, ErrNotLeader
	}

	entry := LogEntry{
		Index:     int64(len(n.log)) + 1,
		Term:      n.term,
		Key:       key,
		Value:     value,
		Operation: op,
	}
	n.log = append(n.log, entry)
	idx := entry.Index
	n.mu.Unlock()

	// apply locally
	n.applyEntry(entry)

	// replicate to peers async
	go n.replicateToPeers()

	return idx, nil
}

func (n *ReplicaNode) applyEntry(entry LogEntry) {
	switch entry.Operation {
	case "PUT":
		n.store.Put(entry.Key, entry.Value)
	case "DELETE":
		n.store.Delete(entry.Key)
	}
	n.mu.Lock()
	n.lastApplied = entry.Index
	n.mu.Unlock()
}

func (n *ReplicaNode) replicateToPeers() {
	// simplified: in production this would use gRPC AppendEntries RPCs
	n.mu.Lock()
	defer n.mu.Unlock()
	n.commitIndex = int64(len(n.log))
}

func (n *ReplicaNode) runElectionTimer() {
	timeout := 150*time.Millisecond + time.Duration(len(n.id)*50)*time.Millisecond

	for {
		select {
		case <-n.stopCh:
			return
		case <-n.heartbeatCh:
			// reset timer on heartbeat
			continue
		case <-time.After(timeout):
			n.startElection()
		}
	}
}

func (n *ReplicaNode) startElection() {
	n.mu.Lock()
	n.state = Candidate
	n.term++
	n.votedFor = n.id
	n.votes = 1
	currentTerm := n.term
	n.mu.Unlock()

	log.Printf("[%s] starting election for term %d", n.id, currentTerm)

	// simplified: auto-win if no peers
	n.mu.Lock()
	if len(n.peers) == 0 {
		n.state = Leader
		n.leaderID = n.id
		log.Printf("[%s] became leader (no peers) term=%d", n.id, n.term)
	}
	n.mu.Unlock()
}

func (n *ReplicaNode) HandleHeartbeat(leaderID string, term int64) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if term >= n.term {
		n.term = term
		n.state = Follower
		n.leaderID = leaderID
		n.votedFor = ""
	}

	select {
	case n.heartbeatCh <- struct{}{}:
	default:
	}
}

func (n *ReplicaNode) HandleVoteRequest(candidateID string, term int64) (int64, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if term < n.term {
		return n.term, false
	}

	if term > n.term {
		n.term = term
		n.state = Follower
		n.votedFor = ""
	}

	if n.votedFor == "" || n.votedFor == candidateID {
		n.votedFor = candidateID
		return n.term, true
	}

	return n.term, false
}

var ErrNotLeader = context.DeadlineExceeded // placeholder
// election timeout is too aggressive for 3-node cluster
