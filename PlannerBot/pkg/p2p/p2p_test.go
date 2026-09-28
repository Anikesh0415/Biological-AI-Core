package p2p

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"planner_bot/pkg/store"
)

func createTestStore(t *testing.T, name string) *store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), name+".db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store %s: %v", name, err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestP2PSyncTwoNodes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	userCode := "PLAN-SYNC-TEST"

	// 1. Setup Node A (Laptop)
	storeA := createTestStore(t, "laptop")
	now := time.Now().UTC()
	_ = storeA.Save(&store.Reminder{
		ID:        "rem-1",
		Task:      "Finish homework",
		FireAt:    now.Add(2 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	})

	serverA, err := NewSyncServer("device-laptop", userCode, storeA)
	if err != nil {
		t.Fatalf("create serverA: %v", err)
	}
	if err := serverA.Start(); err != nil {
		t.Fatalf("start serverA: %v", err)
	}
	defer serverA.Close()

	// 2. Setup Node B (Phone)
	storeB := createTestStore(t, "phone")
	clientB := NewSyncClient("device-phone", userCode, storeB)

	// Peer representing Node A
	peerA := &Peer{
		ID:   "device-laptop",
		Name: "Laptop",
		IP:   "127.0.0.1",
		Port: serverA.Port(),
	}

	// 3. Node B syncs with Node A
	changes, err := clientB.SyncWithPeer(ctx, peerA, time.Time{})
	if err != nil {
		t.Fatalf("SyncWithPeer failed: %v", err)
	}
	if changes != 1 {
		t.Fatalf("expected 1 change synced to Node B, got %d", changes)
	}

	// Verify Reminder 1 now exists in Node B's store!
	rB, err := storeB.Get("rem-1")
	if err != nil {
		t.Fatalf("Reminder rem-1 should exist on Phone: %v", err)
	}
	if rB.Task != "Finish homework" {
		t.Errorf("expected Task='Finish homework', got %q", rB.Task)
	}

	// 4. Two-way concurrent mutation test:
	// Phone adds a new task: "Buy groceries"
	_ = storeB.Save(&store.Reminder{
		ID:        "rem-2",
		Task:      "Buy groceries",
		FireAt:    now.Add(4 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now.Add(1 * time.Minute),
		Version:   1,
	})

	// Laptop updates task 1: "Finish homework and math project"
	_ = storeA.Save(&store.Reminder{
		ID:        "rem-1",
		Task:      "Finish homework and math project",
		FireAt:    now.Add(2 * time.Hour),
		CreatedAt: now,
		UpdatedAt: now.Add(2 * time.Minute),
		Version:   2,
	})

	// Node B syncs again with Node A
	_, err = clientB.SyncWithPeer(ctx, peerA, time.Time{})
	if err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	// Node B should now have Laptop's newer version of rem-1
	updatedRB, _ := storeB.Get("rem-1")
	if updatedRB.Task != "Finish homework and math project" {
		t.Errorf("Node B should have updated rem-1, got %q", updatedRB.Task)
	}

	// Node A should now have Phone's rem-2
	rem2A, err := storeA.Get("rem-2")
	if err != nil {
		t.Fatalf("Node A should have received rem-2 from Phone: %v", err)
	}
	if rem2A.Task != "Buy groceries" {
		t.Errorf("Node A rem-2 task mismatch: %q", rem2A.Task)
	}
}

func TestP2PUnauthorizedNodeRejected(t *testing.T) {
	ctx := context.Background()

	validUserCode := "PLAN-VALID-CODE"
	evilUserCode := "PLAN-ATTACKER-CODE"

	storeA := createTestStore(t, "laptop")
	serverA, err := NewSyncServer("device-laptop", validUserCode, storeA)
	if err != nil {
		t.Fatalf("create serverA: %v", err)
	}
	if err := serverA.Start(); err != nil {
		t.Fatalf("start serverA: %v", err)
	}
	defer serverA.Close()

	// Attacker device (wrong pairing code) attempts sync
	storeAttacker := createTestStore(t, "attacker")
	clientAttacker := NewSyncClient("device-attacker", evilUserCode, storeAttacker)

	peerA := &Peer{
		ID:   "device-laptop",
		Name: "Laptop",
		IP:   "127.0.0.1",
		Port: serverA.Port(),
	}

	// Attacker sync attempt MUST fail during authentication handshake
	_, err = clientAttacker.SyncWithPeer(ctx, peerA, time.Time{})
	if err == nil {
		t.Fatal("expected sync with wrong pairing code to be rejected, but it succeeded!")
	}
	t.Logf("correctly rejected unauthorized peer: %v", err)
}

func TestP2PEngineLifecycle(t *testing.T) {
	storeEngine := createTestStore(t, "engine_test")

	engine, err := NewEngine(EngineConfig{
		DeviceID:     "test-engine-1",
		DeviceName:   "Test Laptop",
		UserCode:     "PLAN-ENGINE-TEST",
		Store:        storeEngine,
		SyncInterval: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := engine.Start(ctx); err != nil {
		t.Fatalf("engine.Start failed: %v", err)
	}

	if engine.Port() <= 0 {
		t.Fatalf("expected positive listening port, got %d", engine.Port())
	}
	if engine.PairingCode() != "PLAN-ENGINE-TEST" {
		t.Errorf("unexpected pairing code: %s", engine.PairingCode())
	}

	// Stop engine
	if err := engine.Close(); err != nil {
		t.Fatalf("engine.Close failed: %v", err)
	}
}
