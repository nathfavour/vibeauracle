package vault

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVault_Fallback(t *testing.T) {
	tmpDir := t.TempDir()
	
	v, err := New("vibeauracle-test", tmpDir)
	if err != nil {
		t.Fatalf("failed to create vault: %v", err)
	}

	// Manually disable keyring for testing fallback specifically
	v.ring = nil

	err = v.Set("mykey", "mysecret")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := v.Get("mykey")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if val != "mysecret" {
		t.Errorf("expected mysecret, got %s", val)
	}

	// Verify file exists
	secretsPath := filepath.Join(tmpDir, "secrets.json")
	if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
		t.Error("fallback secrets file not created")
	}
}