package doctor

import (
	"os"
	"testing"
	"time"

	"github.com/nathfavour/vibeauracle/sys"
)

func TestDoctor_Cues(t *testing.T) {
	// Setup mock environment
	origHome := os.Getenv("HOME")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Start monitor (it runs in background)
	go monitor()
	
	Send("test", SignalInfo, "hello doc", nil)
    
    // We need to wait a bit for the monitor goroutine to pick it up
    time.Sleep(100 * time.Millisecond)

	logs := GetRecentLogs(10)
    found := false
    for _, l := range logs {
        if l.Source == "test" && l.Message == "hello doc" {
            found = true
            break
        }
    }
    
    if !found {
        t.Errorf("expected log not found in recent logs: %v", logs)
    }
}

// SignalInfo is not defined, I used SignalInit or something else?
// Looking at doctor.go: heartbeat, warning, error, panic, init, crash.
// I'll use SignalInit.

func TestDoctor_AnalyzeHealth(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cm, _ := sys.NewConfigManager()
	cfg, _ := cm.Load()

	// Initial health
	if h := AnalyzeHealth(); h != HealthGood {
		t.Errorf("expected Good health, got %v", h)
	}

	// Mock crash
	cfg.Health.CrashCount = 1
	cfg.Health.LastCrash = time.Now()
	cm.Save(cfg)

	if h := AnalyzeHealth(); h != HealthDegraded {
		t.Errorf("expected Degraded health, got %v", h)
	}

	// Mock catastrophic
	cfg.Health.CrashCount = 3
	cm.Save(cfg)
	if h := AnalyzeHealth(); h != HealthCatastrophic {
		t.Errorf("expected Catastrophic health, got %v", h)
	}
}

const SignalInfo SignalType = "init"
