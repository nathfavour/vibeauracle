package tooling

import (
	"context"

	"github.com/nathfavour/vibeauracle/sys"
)

// SystemProvider provides core tools built into the system.
type SystemProvider struct {
	fs      sys.FS
	monitor *sys.Monitor
	guard   *SecurityGuard
}

func NewSystemProvider(f sys.FS, m *sys.Monitor, guard *SecurityGuard) *SystemProvider {
	return &SystemProvider{fs: f, monitor: m, guard: guard}
}

func (p *SystemProvider) Name() string { return "system" }

func (p *SystemProvider) Provide(ctx context.Context) ([]Tool, error) {
	tools := []Tool{
		NewReadFileTool(p.fs),
		NewWriteFileTool(p.fs),
		NewListFilesTool(p.fs),
		NewTraversalTool(p.fs),
		&ShellExecTool{},
		NewSystemInfoTool(p.monitor),
		&FetchURLTool{},
	}

	var secured []Tool
	for _, t := range tools {
		if p.guard != nil {
			secured = append(secured, WrapWithSecurity(t, p.guard))
		} else {
			secured = append(secured, t)
		}
	}

	return secured, nil
}

// Global Registry Setup
func Setup(f sys.FS, m *sys.Monitor, guard *SecurityGuard) *Registry {
	r := NewRegistry()
	r.RegisterProvider(NewSystemProvider(f, m, guard))
	// Future: r.RegisterProvider(NewVibeProvider())
	// Future: r.RegisterProvider(NewMCPProvider())

	_ = r.Sync(context.Background())
	return r
}
