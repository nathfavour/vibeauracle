package connect

import (
	"context"
	"testing"
)

func TestShareManager(t *testing.T) {
	sm := NewShareManager()
	sess := sm.CreateSession("tui", "rw", "user1", []string{"client1"})

	if sess.ID == "" {
		t.Error("expected session ID to be set")
	}
	if sess.Type != "tui" {
		t.Errorf("expected type tui, got %s", sess.Type)
	}
	if sess.Permissions != "rw" {
		t.Errorf("expected permissions rw, got %s", sess.Permissions)
	}
	if sess.TargetUserID != "user1" {
		t.Errorf("expected target user user1, got %s", sess.TargetUserID)
	}

	if _, ok := sm.activeSessions[sess.ID]; !ok {
		t.Error("session not found in activeSessions")
	}
}

func TestConnector(t *testing.T) {
	ctx := context.Background()
	c, err := NewConnector(ctx)
	if err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}
	defer c.Close()

	addr := c.GetAddress()
	if addr == "" {
		t.Error("expected non-empty address")
	}

	sess := c.CreateSharedSession("browser", "ro", "", nil)
	if sess.Type != "browser" {
		t.Errorf("expected browser session, got %s", sess.Type)
	}
}
