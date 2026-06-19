package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	vipc "github.com/nathfavour/vibeauracle/pkg/ipc"
)

// DefaultEngine mutates codebases via vibeauracle UDS, falling back to CLI.
type DefaultEngine struct {
	Binary string
}

func NewDefaultEngine() *DefaultEngine {
	bin, err := exec.LookPath("vibeaura")
	if err != nil {
		bin = "vibeaura"
	}
	return &DefaultEngine{Binary: bin}
}

func (e *DefaultEngine) Mutate(ctx context.Context, req MutationRequest) (*MutationResult, error) {
	if req.Payload == "" {
		return nil, fmt.Errorf("payload is required")
	}

	if out, err := vipc.Query(req.Payload); err == nil {
		return &MutationResult{Success: true, ExitCode: 0, Output: strings.TrimSpace(out)}, nil
	}

	if req.WorkDir == "" {
		return nil, fmt.Errorf("workdir is required when vibeauracle UDS is unavailable")
	}

	workDir, err := filepath.Abs(req.WorkDir)
	if err != nil {
		return nil, err
	}

	args := []string{"direct", "-n", req.Payload}
	cmd := exec.CommandContext(ctx, e.Binary, args...)
	cmd.Dir = workDir
	cmd.Env = os.Environ()

	out, runErr := cmd.CombinedOutput()
	exitCode := 0
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			return nil, runErr
		}
	}

	return &MutationResult{
		Success:  exitCode == 0,
		ExitCode: exitCode,
		Output:   strings.TrimSpace(string(out)),
	}, nil
}
