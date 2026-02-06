package brain

import (
	"os"
	"path/filepath"
)

// DiscoverSkills finds all .agent/skills directories in the project tree.
// It searches starting from the current working directory.
func (b *Brain) DiscoverSkills() []string {
	var skillDirs []string
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
				// The SDK expects the parent directory that contains multiple skills,
				// OR it expects the path to a directory where each subdirectory is a skill.
				// Based on the instruction: "index all .agent/skills/ directories"
				// and "referencing the SKILL.md file within each subdirectory",
				// it seems .agent/skills/ is the container for multiple skills.
				
				// Add to list
				skillDirs = append(skillDirs, path)
				
				// We can skip walking deeper into this directory for speed,
				// but there might be nested .agent/skills in sub-projects.
				// For safety and thoroughness, we continue walking.
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

	return skillDirs
}
