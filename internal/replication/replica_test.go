package replication

import (
	"testing"
	"time"

	"github.com/CrowGreen/dkv/internal/store"
)

func TestSingleNodeBecomesLeader(t *testing.T) {
	s := store.New()
	node := NewNode("node1", nil, s)
	node.Start()
	defer node.Stop()

	time.Sleep(300 * time.Millisecond)

	if !node.IsLeader() {
		t.Fatal("single node should become leader")
	}
}

func TestAppendEntry(t *testing.T) {
	s := store.New()
	node := NewNode("node1", nil, s)
	node.Start()
	defer node.Stop()

	time.Sleep(300 * time.Millisecond)

	idx, err := node.AppendEntry("foo", []byte("bar"), "PUT")
	if err != nil {
		t.Fatal(err)
	}
	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}

	entry, ok := s.Get("foo")
	if !ok || string(entry.Value) != "bar" {
		t.Fatal("entry not applied to store")
	}
}

func TestVoteRequest(t *testing.T) {
	s := store.New()
	node := NewNode("node1", []string{"node2"}, s)

	term, granted := node.HandleVoteRequest("node2", 1)
	if !granted {
		t.Fatal("should grant vote")
	}
	if term != 1 {
		t.Fatalf("expected term 1, got %d", term)
	}

	// second vote in same term should be denied for different candidate
	_, granted2 := node.HandleVoteRequest("node3", 1)
	if granted2 {
		t.Fatal("should not grant second vote in same term")
	}
}

func TestHeartbeatResetsElection(t *testing.T) {
	s := store.New()
	node := NewNode("node1", []string{"node2"}, s)
	node.Start()
	defer node.Stop()

	// send heartbeats to prevent election
	for i := 0; i < 5; i++ {
		node.HandleHeartbeat("node2", 1)
		time.Sleep(50 * time.Millisecond)
	}

	if node.IsLeader() {
		t.Fatal("node should stay follower with heartbeats")
	}
}
