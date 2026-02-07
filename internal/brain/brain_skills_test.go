package brain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverSkills(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := filepath.Join(os.TempDir(), "vibeauracle-test-skills")
	err := os.MkdirAll(tmpDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to tmpDir to simulate project root
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Create .agent/skills structure
	skillPath := filepath.Join(tmpDir, ".agent", "skills")
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatalf("Failed to create skill dir: %v", err)
	}

	// Create a skill subdirectory
	mySkillPath := filepath.Join(skillPath, "my-skill")
	if err := os.MkdirAll(mySkillPath, 0755); err != nil {
		t.Fatalf("Failed to create my-skill dir: %v", err)
	}

	// Create SKILL.md
	if err := os.WriteFile(filepath.Join(mySkillPath, "SKILL.md"), []byte("# My Skill"), 0644); err != nil {
		t.Fatalf("Failed to write SKILL.md: %v", err)
	}

	// Create another nested .agent/skills in a sub-project
	subProjectPath := filepath.Join(tmpDir, "sub-project", ".agent", "skills")
	if err := os.MkdirAll(subProjectPath, 0755); err != nil {
		t.Fatalf("Failed to create sub-project skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subProjectPath, "test-skill", "SKILL.md"), []byte("# Test Skill"), 0644); err != nil {
		// Need to create subdirectory first
		os.MkdirAll(filepath.Join(subProjectPath, "test-skill"), 0755)
		os.WriteFile(filepath.Join(subProjectPath, "test-skill", "SKILL.md"), []byte("# Test Skill"), 0644)
	}

	b := &Brain{}
	skills := b.DiscoverSkills()

	if len(skills) != 2 {
		t.Errorf("Expected 2 skills, got %d", len(skills))
	}

	foundRoot := false
	foundSub := false
	for _, s := range skills {
		if s.Name == "my-skill" && strings.Contains(s.Path, ".agent/skills") {
			foundRoot = true
		}
		if s.Name == "test-skill" && strings.Contains(s.Path, "sub-project") {
			foundSub = true
		}
	}

	if !foundRoot {
		t.Error("my-skill not found")
	}
	if !foundSub {
		t.Error("test-skill not found")
	}
}
