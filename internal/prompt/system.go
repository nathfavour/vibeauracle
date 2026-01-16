package prompt

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nathfavour/vibeauracle/sys"
)

// System is the modular prompt engine: classify → layer instructions → build prompt → parse response.
type System struct {
	cfg         *sys.Config
	memory      Memory
	recommender Recommender

	// Budgeting to avoid unintended spend.
	recoUsed int
}

func New(cfg *sys.Config, memory Memory, recommender Recommender) *System {
	return &System{cfg: cfg, memory: memory, recommender: recommender}
}

// SetRecommender updates the active recommender.
func (s *System) SetRecommender(r Recommender) {
	s.recommender = r
}

// Build produces the prompt envelope for a user input.
func (s *System) Build(ctx context.Context, userText string, snapshot sys.Snapshot, toolDefs string) (Envelope, []Recommendation, error) {
	intent := ClassifyIntent(userText)
	if s.cfg != nil && s.cfg.Prompt.Mode != "" {
		// Config can force a mode. "auto" keeps classification.
		mode := strings.ToLower(strings.TrimSpace(s.cfg.Prompt.Mode))
		switch mode {
		case "auto":
			// keep
		case "ask":
			intent = IntentAsk
		case "plan":
			intent = IntentPlan
		case "crud":
			intent = IntentCRUD
		}
	}

	if !LooksLikePrompt(userText) {
		return Envelope{Intent: intent, Prompt: "", Instructions: nil, Metadata: map[string]any{"ignored": true}}, nil, nil
	}

	instructions := s.layers(intent, snapshot.WorkingDir)

	// Learning layer: cheap recall injection.
	var recall string
	if s.cfg != nil && s.cfg.Prompt.LearningEnabled && s.memory != nil {
		snips, _ := s.memory.Recall(userText)
		if len(snips) > 0 {
			recall = strings.Join(snips, "\n")
		}
	}

	prompt := s.compose(intent, instructions, recall, snapshot, toolDefs, userText)

	// Learning write-back: store a compact behavioral signal for future recall.
	if s.cfg != nil && s.cfg.Prompt.LearningEnabled && s.memory != nil {
		compact := userText
		if len(compact) > 160 {
			compact = compact[:160]
		}
		_ = s.memory.Store(fmt.Sprintf("prompt:%d", time.Now().UnixNano()), fmt.Sprintf("intent=%s text=%s", intent, compact))
	}

	recs, err := s.maybeRecommend(ctx, intent, userText, snapshot.WorkingDir)
	if err != nil {
		// Recommendations are best-effort and must never fail the main prompt.
		recs = nil
	}

	return Envelope{
		Intent:       intent,
		Prompt:       prompt,
		Instructions: instructions,
		Metadata: map[string]any{
			"working_dir": snapshot.WorkingDir,
			"cpu":         snapshot.CPUUsage,
			"mem":         snapshot.MemoryUsage,
		},
	}, recs, nil
}

func (s *System) layers(intent Intent, wd string) []string {
	layers := []string{}

	// Base system layer - ACTION FIRST (softer language for content filters)
	layers = append(layers, "You are vibe auracle, an AI coding assistant. You help users by executing tasks directly.")
	layers = append(layers, "Handle typos gracefully by interpreting the user's likely intent.")
	layers = append(layers, "Keep responses brief and focused on results.")

	// Project-Native Layer: Discover instructions and Repo identity
	if wd != "" {
		projectContext := s.discoverProjectInstructions(wd)
		repoMeta := s.getRepoMetadata()
		if projectContext != "" || repoMeta != "" {
			combined := ""
			if repoMeta != "" {
				combined += "REPOSITORY IDENTITY:\n" + repoMeta + "\n"
			}
			if projectContext != "" {
				combined += "PROJECT RULES:\n" + projectContext
			}
			layers = append(layers, combined)
		}
	}

	// Project layer (configurable)
	if s.cfg != nil {
		if strings.TrimSpace(s.cfg.Prompt.ProjectInstructions) != "" {
			layers = append(layers, "MANUAL INSTRUCTIONS:\n"+s.cfg.Prompt.ProjectInstructions)
		}
	}

	// Mode layer
	switch intent {
	case IntentAsk:
		layers = append(layers, "Mode: Answer questions clearly and concisely.")
	case IntentPlan:
		layers = append(layers, "Mode: Create a structured plan.")
	case IntentCRUD:
		layers = append(layers, "Mode: Execute file and code changes.")
	default:
		layers = append(layers, "Mode: Execute the requested task.")
	}

	return layers
}

func (s *System) compose(intent Intent, layers []string, recall string, snapshot sys.Snapshot, toolDefs string, userText string) string {
	b := strings.Builder{}
	b.WriteString("SYSTEM INSTRUCTIONS:\n")
	for _, l := range layers {
		b.WriteString("- ")
		b.WriteString(l)
		b.WriteString("\n")
	}

	if strings.TrimSpace(recall) != "" {
		b.WriteString("\nLEARNING/RECALL (local):\n")
		b.WriteString(recall)
		b.WriteString("\n")
	}

	b.WriteString("\nSYSTEM SNAPSHOT:\n")
	b.WriteString(fmt.Sprintf("CWD: %s\nCPU: %.2f%%\nMEM: %.2f%%\n", snapshot.WorkingDir, snapshot.CPUUsage, snapshot.MemoryUsage))

	if strings.TrimSpace(toolDefs) != "" {
		b.WriteString("\nAVAILABLE TOOLS:\n")
		b.WriteString(toolDefs)
		b.WriteString(`
TOOL USAGE:
You can use tools to complete tasks. To invoke a tool, output a JSON code block:

` + "```json" + `
{"tool": "TOOL_NAME", "parameters": {"param1": "value1"}}
` + "```" + `

Example - Create a file:
` + "```json" + `
{"tool": "sys_write_file", "parameters": {"path": "example.txt", "content": "Hello world"}}
` + "```" + `

Example - Read a file:
` + "```json" + `
{"tool": "sys_read_file", "parameters": {"path": "README.md"}}
` + "```" + `

Guidelines:
- Execute tool calls directly without asking for permission
- Handle typos by interpreting the user's intent
- Report results briefly after tool execution
- Current directory: ` + snapshot.WorkingDir + `

`)
	}

	b.WriteString("\nUSER PROMPT:\n")
	b.WriteString(userText)
	b.WriteString("\n")

	return b.String()
}

func (s *System) maybeRecommend(ctx context.Context, intent Intent, userText string, wd string) ([]Recommendation, error) {
	if s.cfg == nil || !s.cfg.Prompt.RecommendationsEnabled {
		return nil, nil
	}
	if s.recommender == nil {
		return nil, nil
	}
	if s.cfg.Prompt.RecommendationsMaxPerRun > 0 && s.recoUsed >= s.cfg.Prompt.RecommendationsMaxPerRun {
		return nil, nil
	}

	// Only recommend for codebase-relevant intents.
	if intent != IntentPlan && intent != IntentCRUD {
		return nil, nil
	}

	// Sampling: keep this extremely low by default.
	prob := s.cfg.Prompt.RecommendationsSampleRate
	if prob <= 0 {
		prob = 0.05
	}
	// Deterministic-ish sampling: hashless, time-bucket based.
	if (time.Now().UnixNano() % 1000) > int64(prob*1000) {
		return nil, nil
	}

	s.recoUsed++
	return s.recommender.Recommend(ctx, RecommendInput{Intent: intent, UserText: userText, WorkingDir: wd, Time: time.Now()})
}

// discoverProjectInstructions scans for project-specific instructions in standard locations.
func (s *System) discoverProjectInstructions(wd string) string {
	var sb strings.Builder
	paths := []string{
		filepath.Join(wd, ".github", "agents"),
		filepath.Join(wd, ".github", "vibeaura"),
	}

	for _, p := range paths {
		files, err := os.ReadDir(p)
		if err != nil {
			continue
		}

		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".md") {
				content, err := os.ReadFile(filepath.Join(p, f.Name()))
				if err == nil {
					sb.WriteString(fmt.Sprintf("\n--- Source: %s ---\n", f.Name()))
					sb.WriteString(string(content))
					sb.WriteString("\n")
				}
			}
		}
	}

	return sb.String()
}

// getRepoMetadata uses 'gh repo view' to get rich context about the current repository.
func (s *System) getRepoMetadata() string {
	cmd := exec.Command("gh", "repo", "view", "--json", "name,owner,description,stargazerCount,primaryLanguage,licenseInfo,url")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}
