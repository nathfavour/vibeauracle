package connect

import (
	"time"
)

// SharedSession represents a session that is being shared
type SharedSession struct {
	ID        string    `json:"id"`
	CWD       string    `json:"cwd"`
	StartTime time.Time `json:"start_time"`
	Type      string    `json:"type"` // "browser" or "tui"
	Status    string    `json:"status"` // "active", "closed"
}

// ShareManager handles the lifecycle of shared sessions
type ShareManager struct {
	activeSessions map[string]*SharedSession
}

func NewShareManager() *ShareManager {
	return &ShareManager{
		activeSessions: make(map[string]*SharedSession),
	}
}

func (sm *ShareManager) CreateSession(sessionType string) *SharedSession {
	id := "sess_" + time.Now().Format("20060102150405")
	session := &SharedSession{
		ID:        id,
		StartTime: time.Now(),
		Type:      sessionType,
		Status:    "active",
	}
	sm.activeSessions[id] = session
	return session
}
