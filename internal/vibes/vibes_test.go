package vibes

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSplitFrontMatter(t *testing.T) {
	data := []byte(`---
name: test-vibe
version: 1.0.0
---
# Instructions
Do something.`)

	fm, body, err := splitFrontMatter(data)
	if err != nil {
		t.Fatalf("splitFrontMatter failed: %v", err)
	}

	if !bytes.Contains(fm, []byte("name: test-vibe")) {
		t.Errorf("frontMatter missing name: %s", string(fm))
	}
	if !bytes.Contains(body, []byte("# Instructions")) {
		t.Errorf("body missing instructions: %s", string(body))
	}
}

func TestParse(t *testing.T) {
	content := `---
name: my-vibe
version: 0.1.0
hooks:
  - on_startup
permissions:
  - config.read
---
The instructions go here.
`
	tmpFile := filepath.Join(t.TempDir(), "my.vibe.md")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	vibe, err := Parse(tmpFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if vibe.Spec.Name != "my-vibe" {
		t.Errorf("expected name my-vibe, got %s", vibe.Spec.Name)
	}
	if len(vibe.Spec.Hooks) != 1 || vibe.Spec.Hooks[0] != HookOnStartup {
		t.Errorf("unexpected hooks: %v", vibe.Spec.Hooks)
	}
	if !vibe.HasPermission(PermConfigRead) {
		t.Error("expected config.read permission")
	}
	if !strings.Contains(vibe.Instructions, "The instructions go here.") {
		t.Errorf("unexpected instructions: %s", vibe.Instructions)
	}
}

func TestRegistry(t *testing.T) {
	reg := NewRegistry()
	
	tmpDir := t.TempDir()
	content := `---
name: registry-vibe
version: 1.0.0
hooks:
  - on_command
---
Action!`
	if err := os.WriteFile(filepath.Join(tmpDir, "reg.vibe.md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	reg.AddDirectory(tmpDir)
	if err := reg.Scan(); err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	vibe, ok := reg.Get("registry-vibe")
	if !ok {
		t.Fatal("vibe not found in registry")
	}

	if vibe.Spec.Name != "registry-vibe" {
		t.Errorf("expected registry-vibe, got %s", vibe.Spec.Name)
	}

	vibesByHook := reg.ByHook(HookOnCommand)
	if len(vibesByHook) != 1 {
		t.Errorf("expected 1 vibe for hook, got %d", len(vibesByHook))
	}

	reg.Disable("registry-vibe")
	vibesByHook = reg.ByHook(HookOnCommand)
	if len(vibesByHook) != 0 {
		t.Error("expected 0 vibes for hook after disabling")
	}
}