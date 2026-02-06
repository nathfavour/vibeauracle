package context

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func NewWindow(maxItems int) *Window {
	return &Window{
		Items:     make(map[string]*ContextItem),
		MaxLength: maxItems,
	}
}

// Add inserts or updates an item in the context window.
func (w *Window) Add(id, content, itemType string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if item, exists := w.Items[id]; exists {
		item.Frequency++
		item.LastUsed = time.Now()
		item.Content = content // Update content if it changed
		return
	}

	w.Items[id] = &ContextItem{
		ID:        id,
		Content:   content,
		Type:      itemType,
		Frequency: 1,
		LastUsed:  time.Now(),
	}

	w.prune()
}

// prune enforces the window size by removing ensuring least relevant items are dropped.
func (w *Window) prune() {
	if len(w.Items) <= w.MaxLength {
		return
	}

	type rankedItem struct {
		ID    string
		Score float64 // Higher is better
	}

	var ranked []rankedItem
	now := time.Now()

	for id, item := range w.Items {
		if item.Pinned {
			continue // Never prune pinned items
		}
		// Recency bias + Frequency weight
		hoursUnused := now.Sub(item.LastUsed).Hours()
		score := float64(item.Frequency) - (hoursUnused * 0.5)
		ranked = append(ranked, rankedItem{ID: id, Score: score})
	}

	// Sort by score ascending (lowest first)
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].Score < ranked[j].Score
	})

	// Remove items until we fit
	excess := len(w.Items) - w.MaxLength
	for i := 0; i < excess && i < len(ranked); i++ {
		delete(w.Items, ranked[i].ID)
	}
}

// GetContext returns the formatted context string, sorted by relevance.
func (w *Window) GetContext() string {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var activeItems []*ContextItem
	for _, item := range w.Items {
		activeItems = append(activeItems, item)
	}
	// Sort: Pinned first, then by recency/frequency
	sort.Slice(activeItems, func(i, j int) bool {
		if activeItems[i].Pinned != activeItems[j].Pinned {
			return activeItems[i].Pinned
		}
		return activeItems[i].LastUsed.After(activeItems[j].LastUsed)
	})

	var sb strings.Builder
	for _, item := range activeItems {
		sb.WriteString(fmt.Sprintf("[%s] (%s):
%s
---
", item.Type, item.ID, item.Content))
	}
	return sb.String()
}
