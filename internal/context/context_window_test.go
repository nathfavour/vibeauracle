package context

import (
	"strings"
	"testing"
	"time"
)

func TestWindow_Add(t *testing.T) {
	w := NewWindow(2)

	w.Add("id1", "content1", "file")
	if len(w.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(w.Items))
	}

	w.Add("id1", "content1-updated", "file")
	if len(w.Items) != 1 {
		t.Errorf("expected 1 item after update, got %d", len(w.Items))
	}
	if w.Items["id1"].Frequency != 2 {
		t.Errorf("expected frequency 2, got %d", w.Items["id1"].Frequency)
	}
	if w.Items["id1"].Content != "content1-updated" {
		t.Errorf("expected updated content, got %s", w.Items["id1"].Content)
	}

	w.Add("id2", "content2", "file")
	w.Add("id3", "content3", "file")

	if len(w.Items) != 2 {
		t.Errorf("expected 2 items after pruning, got %d", len(w.Items))
	}
}

func TestWindow_Pruning(t *testing.T) {
	w := NewWindow(2)

	// Add id1, then id2, then id3.
	// id1 and id2 are added. When id3 is added, one must be pruned.
	w.Add("id1", "c1", "file")
	w.Add("id2", "c2", "file")
	
	// Manually adjust LastUsed to ensure id1 is older and has lower score
	w.Items["id1"].LastUsed = time.Now().Add(-10 * time.Hour)
	w.Items["id2"].LastUsed = time.Now()

	w.Add("id3", "c3", "file")

	if _, exists := w.Items["id1"]; exists {
		t.Error("expected id1 to be pruned")
	}
	if _, exists := w.Items["id3"]; !exists {
		t.Error("expected id3 to exist")
	}
}

func TestWindow_Pinned(t *testing.T) {
	w := NewWindow(1)

	w.Add("pinned1", "important", "system")
	w.Items["pinned1"].Pinned = true
	w.Items["pinned1"].LastUsed = time.Now().Add(-100 * time.Hour)

	w.Add("new1", "less-important", "file")

	if _, exists := w.Items["pinned1"]; !exists {
		t.Error("pinned item was pruned")
	}
	if len(w.Items) > 1 {
		// If MaxLength is 1 and we have 1 pinned + 1 new, 
		// prune() should have removed "new1"
		if _, exists := w.Items["new1"]; exists {
			t.Error("expected new1 to be pruned to respect MaxLength even with pinned items")
		}
	}
}

func TestWindow_GetContext(t *testing.T) {
	w := NewWindow(5)
	w.Add("id1", "content1", "file")
	w.Add("id2", "content2", "user")
	
	w.Items["id2"].Pinned = true

	ctx := w.GetContext()
	if !strings.Contains(ctx, "[user] (id2)") {
		t.Errorf("context missing id2: %s", ctx)
	}
	if !strings.Contains(ctx, "[file] (id1)") {
		t.Errorf("context missing id1: %s", ctx)
	}

	// Verify order: pinned first
	id2Pos := strings.Index(ctx, "id2")
	id1Pos := strings.Index(ctx, "id1")
	if id2Pos > id1Pos {
		t.Error("expected pinned id2 to come before id1")
	}
}
