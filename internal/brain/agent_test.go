package brain

import (
	"os"
	"testing"

	"github.com/nathfavour/vibeauracle/sys"
)

func setupTestBrain(t *testing.T) (*Brain, string) {
	tmpDir := t.TempDir()
	
	cm, err := sys.NewConfigManager()
	if err != nil {
		t.Fatal(err)
	}
	
	// Override data dir for testing
	cfg, _ := cm.Load()
	cfg.DataDir = tmpDir
	cm.Save(cfg)

	b := &Brain{
		cm:     cm,
		config: cfg,
	}
    
    // We need to set up a mock or temporary home for ConfigManager
    // but sys.NewConfigManager() uses UserHomeDir.
    // sys tests already handle this by setting HOME.
    
    return b, tmpDir
}

func TestSetAgentMode(t *testing.T) {
    // We need to be careful with ConfigManager because it reads from disk
    origHome := os.Getenv("HOME")
    tmpHome := t.TempDir()
    os.Setenv("HOME", tmpHome)
    defer os.Setenv("HOME", origHome)

    cm, _ := sys.NewConfigManager()
    cfg, _ := cm.Load()
    
    b := &Brain{
        cm: cm,
        config: cfg,
    }

    err := b.SetAgentMode("sdk")
    if err != nil {
        t.Fatalf("SetAgentMode failed: %v", err)
    }

    if b.config.Agent.Mode != "sdk" {
        t.Errorf("expected mode sdk, got %s", b.config.Agent.Mode)
    }

    err = b.SetAgentMode("invalid")
    if err == nil {
        t.Error("expected error for invalid mode")
    }
}

func TestCustomAgents(t *testing.T) {
    origHome := os.Getenv("HOME")
    tmpHome := t.TempDir()
    os.Setenv("HOME", tmpHome)
    defer os.Setenv("HOME", origHome)

    cm, _ := sys.NewConfigManager()
    cfg, _ := cm.Load()
    
    b := &Brain{
        cm: cm,
        config: cfg,
    }

    agent := sys.CustomAgent{
        Name: "test-agent",
        Prompt: "You are a test agent",
    }

    err := b.RegisterCustomAgent(agent)
    if err != nil {
        t.Fatalf("RegisterCustomAgent failed: %v", err)
    }

    agents := b.GetCustomAgents()
    if len(agents) != 1 || agents[0].Name != "test-agent" {
        t.Errorf("unexpected custom agents: %v", agents)
    }

    err = b.SetActiveCustomAgent("test-agent")
    if err != nil {
        t.Fatalf("SetActiveCustomAgent failed: %v", err)
    }

    if b.config.Agent.ActiveCustom != "test-agent" || b.config.Agent.Mode != "custom" {
        t.Errorf("active custom agent not set correctly: %v", b.config.Agent)
    }
}
