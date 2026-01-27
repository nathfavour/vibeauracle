package vibe

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/nathfavour/vibeauracle/internal/tooling"
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
		if err := json.Unmarshal(out, &v); err != nil {
			fmt.Printf("Warning: failed to parse manifest for %s: %v\n", name, err)
			continue
		}
		vibes = append(vibes, &v)
	}

	return vibes, nil
}

func RegisterInbuiltVibes(ctx context.Context, r *tooling.Registry) error {
	vibes, err := GetInbuiltVibes(ctx)
	if err != nil {
		return err
	}

	for _, v := range vibes {
		r.RegisterProvider(NewVibeProvider(v))
	}

	return r.Sync(ctx)
}
