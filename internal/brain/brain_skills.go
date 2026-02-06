package brain

import (
	"os"
	"path/filepath"
)

// SkillInfo represents a discovered agent skill.
type SkillInfo struct {
	Name string
	Path string
}

// DiscoverSkills finds all .agent/skills directories in the project tree.
// It returns a list of SkillInfo objects representing individual skills found.
func (b *Brain) DiscoverSkills() []SkillInfo {
	var skills []SkillInfo
	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	err = filepath.Walk(cwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip paths with errors
		}

		// Check if this is a directory named "skills" inside an ".agent" directory
		if info.IsDir() && info.Name() == "skills" {
			parent := filepath.Dir(path)
			if filepath.Base(parent) == ".agent" {
				// We found a .agent/skills directory
				// List subdirectories which are the individual skills
				entries, err := os.ReadDir(path)
				if err == nil {
					for _, entry := range entries {
						if entry.IsDir() {
							// Check if it has a SKILL.md
							skillMdPath := filepath.Join(path, entry.Name(), "SKILL.md")
							if _, err := os.Stat(skillMdPath); err == nil {
								skills = append(skills, SkillInfo{
									Name: entry.Name(),
									Path: filepath.Join(path, entry.Name()),
								})
							}
						}
					}
				}
				// We can skip walking deeper into this directory for speed
				return filepath.SkipDir
			}
		}

		// Skip hidden directories except .agent
		if info.IsDir() && info.Name() != ".agent" && len(info.Name()) > 1 && info.Name()[0] == '.' {
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return nil
	}

	return skills
}
