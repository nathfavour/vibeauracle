package brain

import (
	"strings"

	"github.com/nathfavour/vibeauracle/auth"
	"github.com/nathfavour/vibeauracle/connect"
	vcontext "github.com/nathfavour/vibeauracle/context"
	"github.com/nathfavour/vibeauracle/copilot"
	"github.com/nathfavour/vibeauracle/internal/vibe"
	"github.com/nathfavour/vibeauracle/internal/watcher"
	"github.com/nathfavour/vibeauracle/model"
	"github.com/nathfavour/vibeauracle/prompt"
	"github.com/nathfavour/vibeauracle/sys"
	"github.com/nathfavour/vibeauracle/tooling"
	"github.com/nathfavour/vibeauracle/vault"
)

// Intent represents a user intent
type Intent = prompt.Intent

// CLICommand represents a CLI command
type CLICommand = vibe.CLICommand

// Request represents a user request or system trigger
type Request struct {
	ID       string
	Content  string
	Intent   Intent // Optional manual override
	Provider string // Optional provider override (e.g. github-models, ollama)
	Model    string // Optional model override
}

// Response represents the brain's output
type Response struct {
	Content   string
	Reasoning string
	Metadata  map[string]interface{}
	Error     error
}

func (r Response) GetContent() string {
	return r.Content
}

func (r Response) GetReasoning() string {
	return r.Reasoning
}

// Brain is the cognitive orchestrator
type Brain struct {
	model    *model.Model
	monitor  *sys.Monitor
	fs       sys.FS
	config   *sys.Config
	cm       *sys.ConfigManager
	auth     *auth.Handler
	vault    *vault.Vault
	memory   *vcontext.Memory
	prompts  *prompt.System
	tools    *tooling.Registry
	security *tooling.SecurityGuard
	sessions map[string]*tooling.Session
	activeSessionID string
	extMgr   *vibe.Manager
	connector *connect.Connector
	watcher   *watcher.Watcher

	// Copilot SDK integration
	copilotProvider  *copilot.Provider
	usingCopilotSDK  bool
	skillDirectories []SkillInfo

	// Loop Detection
	detector *LoopDetector

	// Callbacks
	OnStreamDelta       func(delta string)
	OnStreamDone        func(full string)
	OnUsage             func(usage model.Usage)
	OnFilesystemEvent   func(event watcher.Event)
}

// LoopDetector tracks agent actions to detect infinite loops
type LoopDetector struct {
	lastActions []string
	maxHistory  int
}

func NewLoopDetector(maxHistory int) *LoopDetector {
	return &LoopDetector{
		lastActions: make([]string, 0, maxHistory),
		maxHistory:  maxHistory,
	}
}

func (ld *LoopDetector) AddAction(action string) bool {
	// Normalize action string (trim whitespace, etc)
	action = strings.TrimSpace(action)

	// Check for repetition
	repeatCount := 0
	for _, a := range ld.lastActions {
		if a == action {
			repeatCount++
		}
	}

	// If we see the exact same response + tool result sequence 3 times, it's a loop
	if repeatCount >= 3 {
		return true
	}

	ld.lastActions = append(ld.lastActions, action)
	if len(ld.lastActions) > ld.maxHistory {
		ld.lastActions = ld.lastActions[1:]
	}
	return false
}

// ModelDiscovery represents a discovered model
type ModelDiscovery struct {
	Name     string
	Provider string
}
