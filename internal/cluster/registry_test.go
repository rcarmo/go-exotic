package cluster

import (
	"testing"
	"time"
)

func TestRegistryHeartbeatAndEvictStale(t *testing.T) {
	now := time.Date(2026, 5, 13, 22, 0, 0, 0, time.UTC)
	r := newRegistryWithClock(func() time.Time { return now })
	oldPeer, err := LocalPeer("old", "http://127.0.0.1:1", 1)
	if err != nil {
		t.Fatalf("LocalPeer old: %v", err)
	}
	freshPeer, err := LocalPeer("fresh", "http://127.0.0.1:2", 1)
	if err != nil {
		t.Fatalf("LocalPeer fresh: %v", err)
	}
	if err := r.Upsert(oldPeer); err != nil {
		t.Fatalf("Upsert old: %v", err)
	}
	now = now.Add(10 * time.Second)
	if err := r.Upsert(freshPeer); err != nil {
		t.Fatalf("Upsert fresh: %v", err)
	}
	if err := r.Heartbeat("fresh"); err != nil {
		t.Fatalf("Heartbeat: %v", err)
	}
	evicted := r.EvictStale(5 * time.Second)
	if len(evicted) != 1 || evicted[0].ID != "old" {
		t.Fatalf("evicted=%+v want old", evicted)
	}
	peers := r.Peers()
	if len(peers) != 1 || peers[0].ID != "fresh" {
		t.Fatalf("peers=%+v want fresh", peers)
	}
}

func TestRegistryValidation(t *testing.T) {
	if err := (*Registry)(nil).Upsert(Peer{}); err == nil {
		t.Fatal("nil Upsert accepted")
	}
	if err := (*Registry)(nil).Heartbeat("x"); err == nil {
		t.Fatal("nil Heartbeat accepted")
	}
	r := NewRegistry()
	if err := r.Upsert(Peer{}); err == nil {
		t.Fatal("Upsert accepted bad peer")
	}
	if err := r.Heartbeat(""); err == nil {
		t.Fatal("Heartbeat accepted empty id")
	}
	if err := r.Heartbeat("missing"); err == nil {
		t.Fatal("Heartbeat accepted missing peer")
	}
	if got := r.EvictStale(0); got != nil {
		t.Fatalf("EvictStale(0)=%v want nil", got)
	}
}

func TestRegistryPeersSortedAndCopied(t *testing.T) {
	r := NewRegistry()
	b, _ := LocalPeer("b", "http://127.0.0.1:2", 1)
	a, _ := LocalPeer("a", "http://127.0.0.1:1", 1)
	if err := r.Upsert(b); err != nil {
		t.Fatal(err)
	}
	if err := r.Upsert(a); err != nil {
		t.Fatal(err)
	}
	peers := r.Peers()
	if len(peers) != 2 || peers[0].ID != "a" || peers[1].ID != "b" {
		t.Fatalf("peers not sorted: %+v", peers)
	}
	peers[0].ID = "mutated"
	again := r.Peers()
	if again[0].ID != "a" {
		t.Fatalf("Peers aliases registry state: %+v", again)
	}
}
