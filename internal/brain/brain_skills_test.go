package brain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverSkills(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir, err := os.MkdirAll(filepath.Join(os.TempDir(), "vibeauracle-test-skills"), 0755)
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
		t.Errorf("Expected 2 skill directories, got %d", len(skills))
	}

	foundRoot := false
	foundSub := false
	for _, s := range skills {
		if filepath.Base(filepath.Dir(s)) == ".agent" && filepath.Base(s) == "skills" {
			if strings.Contains(s, "sub-project") {
				foundSub = true
			} else {
				foundRoot = true
			}
		}
	}

	if !foundRoot {
		t.Error("Root .agent/skills not found")
	}
	if !foundSub {
		t.Error("Sub-project .agent/skills not found")
	}
}
