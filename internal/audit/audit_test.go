package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAudit_Log(t *testing.T) {
	tmpDir := t.TempDir()

	err := LogSuccess(tmpDir, EventInstall, "install", "1.0.0", "abc", "Success!", nil)
	if err != nil {
		t.Fatalf("LogSuccess failed: %v", err)
	}

	err = LogFailure(tmpDir, EventUpdate, "update", "1.1.0", "def", "Failed!", map[string]interface{}{"code": 500})
	if err != nil {
		t.Fatalf("LogFailure failed: %v", err)
	}

	logFile := filepath.Join(tmpDir, "audit", "lifecycle.jsonl")
	f, err := os.Open(logFile)
	if err != nil {
		t.Fatalf("failed to open audit log: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		var entry Entry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Errorf("failed to unmarshal entry %d: %v", count, err)
		}
		if count == 0 {
			if entry.Type != EventInstall || entry.Status != "success" {
				t.Errorf("unexpected first entry: %+v", entry)
			}
		} else if count == 1 {
			if entry.Type != EventUpdate || entry.Status != "error" {
				t.Errorf("unexpected second entry: %+v", entry)
			}
			if entry.Metadata["code"].(float64) != 500 {
				t.Errorf("expected metadata code 500, got %v", entry.Metadata["code"])
			}
		}
		count++
	}

	if count != 2 {
		t.Errorf("expected 2 entries, got %d", count)
	}
}
