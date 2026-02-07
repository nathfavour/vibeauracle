package context

import (
	"database/sql"
	"sync"
	"time"

	"github.com/nathfavour/vibeauracle/model"
	"github.com/philippgille/chromem-go"
)

// ContextItem represents a granular unit of information.
type ContextItem struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`      // "file", "user_prompt", "agent_reply", "system_state"
	Frequency int       `json:"frequency"` // How often this item is requested/referenced
	LastUsed  time.Time `json:"last_used"`
	Pinned    bool      `json:"pinned"` // Critical info that never leaves the window
}

// Window manages the rolling context of information.
type Window struct {
	Items     map[string]*ContextItem
	MaxLength int // Max tokens or items (simplified as item count for now)
	mu        sync.RWMutex
}

// Memory now wraps the Window system + DB persistence + Vector DB
type Memory struct {
	db       *sql.DB
	Window   *Window
	vdb      *chromem.DB
	embedder model.Provider
}
