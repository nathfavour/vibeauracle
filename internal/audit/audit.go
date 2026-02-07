package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type EventType string

const (
	EventInstall   EventType = "install"
	EventUninstall EventType = "uninstall"
	EventUpdate    EventType = "update"
	EventRollback  EventType = "rollback"
	EventSelfHeal  EventType = "self_heal"
	EventFailure   EventType = "failure"
)

type Entry struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      EventType              `json:"type"`
	Action    string                 `json:"action"`
	Version   string                 `json:"version"`
	Commit    string                 `json:"commit"`
	Status    string                 `json:"status"` // "success" or "error"
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

func Log(dataDir string, entry Entry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	auditDir := filepath.Join(dataDir, "audit")
	if err := os.MkdirAll(auditDir, 0755); err != nil {
		return fmt.Errorf("creating audit directory: %w", err)
	}

	logFile := filepath.Join(auditDir, "lifecycle.jsonl")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening audit log: %w", err)
	}
	defer f.Close()

	bytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling audit entry: %w", err)
	}

	if _, err := f.Write(append(bytes, '\n')); err != nil {
		return fmt.Errorf("writing audit entry: %w", err)
	}

	return nil
}

func LogSuccess(dataDir string, eventType EventType, action, version, commit, message string, metadata map[string]interface{}) error {
	return Log(dataDir, Entry{
		Type:     eventType,
		Action:   action,
		Version:  version,
		Commit:   commit,
		Status:   "success",
		Message:  message,
		Metadata: metadata,
	})
}

func LogFailure(dataDir string, eventType EventType, action, version, commit, message string, metadata map[string]interface{}) error {
	return Log(dataDir, Entry{
		Type:     eventType,
		Action:   action,
		Version:  version,
		Commit:   commit,
		Status:   "error",
		Message:  message,
		Metadata: metadata,
	})
}
