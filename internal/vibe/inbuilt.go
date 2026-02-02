package vibe

import (
	"context"
	"bytes"
        "encoding/json"
	"fmt"
	"os/exec"

	"github.com/nathfavour/vibeauracle/tooling"
)

func GetInbuiltVibes(ctx context.Context) ([]*Vibe, error) {
	inbuilts := []string{"auracrab", "autocommiter"}
	var vibes []*Vibe

	for _, name := range inbuilts {
		// Check if the tool is installed
		if _, err := exec.LookPath(name); err != nil {
			continue
		}

		// Fetch manifest
		cmd := exec.CommandContext(ctx, name, "vibe-manifest")
		out, err := cmd.Output()
		if err != nil {
			fmt.Printf("Warning: failed to fetch manifest for %s: %v\n", name, err)
			continue
		}

		var v Vibe
		out = bytes.TrimSpace(out); if idx := bytes.Index(out, []byte("{")); idx != -1 { out = out[idx:] }; if err := json.Unmarshal(out, &v); err != nil {
			fmt.Printf("Warning: failed to parse manifest for %s: %v\n", name, err)
			continue
		}
		vibes = append(vibes, &v)
	}

	return vibes, nil
}

func RegisterExtensions(ctx context.Context, m *Manager, r *tooling.Registry) error {
	extensions := m.List()

	for _, ext := range extensions {
		if !ext.Enabled {
			continue
		}

		// Check if the tool is installed by name
		if _, err := exec.LookPath(ext.Name); err != nil {
			continue
		}

		// Fetch manifest if not already present
		if ext.Manifest == nil {
			// Try to get it from the binary
			cmd := exec.CommandContext(ctx, ext.Name, "vibe-manifest")
			out, err := cmd.Output()
			if err == nil {
				var v Vibe
				out = bytes.TrimSpace(out)
				if idx := bytes.Index(out, []byte("{")); idx != -1 {
					out = out[idx:]
				}
				if err := json.Unmarshal(out, &v); err == nil {
					ext.Manifest = &v
				}
			}

			// Fallback to hardcoded professional defaults if binary fails
			if ext.Manifest == nil {
				ext.Manifest = m.getDefaultManifest(ext.Name)
			}
			
			if ext.Manifest != nil {
				_ = m.Save(ext)
			}
		}

		if ext.Manifest != nil {
			r.RegisterProvider(NewVibeProvider(ext.Manifest))
		}
	}

	return r.Sync(ctx)
}
