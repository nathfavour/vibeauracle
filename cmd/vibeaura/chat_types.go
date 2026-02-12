package main

import (
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathfavour/vibeauracle/brain"
	vmodel "github.com/nathfavour/vibeauracle/model"
	"github.com/nathfavour/vibeauracle/reactor"
)

type focus int

const (
	focusInput focus = iota // Input text area
	focusConvo              // Conversation viewport (scrollable)
	focusTree               // Tree/file pane (scrollable)
	focusEdit               // File editor (when editing a file)
)

type recordedFrame struct {
	content string
	ticks   int
}

type usageMsg vmodel.Usage

type model struct {
	viewport viewport.Model

	perusalVp viewport.Model

	messages []string

	textarea textarea.Model

	editArea textarea.Model

	err error

	brain *brain.Brain

	width int

	height int

	initialized bool

	showTree bool

	focus focus

	treeEntries []os.DirEntry

	treeCursor int

	currentPath string

	isFileOpen bool

	banner string

	suggestions []string

	suggestionIdx int

	triggerChar string // '/' or '#'

	isCapturing bool

	// Usage monitoring

	lastUsage vmodel.Usage

	// Recording state

	isRecording    bool
	recordingID    string
	recordedFrames []recordedFrame
	recordTicker   *time.Ticker
	isDirty        bool

	// Recording progress feedback
	isEncoding      bool
	encodingCurrent int
	encodingTotal   int
	recordingErr    error
	// Model selection & filtering
	allModelDiscoveries []brain.ModelDiscovery
	suggestionFilter    string
	isFilteringModels   bool

	// Thinking / Agentic Process State
	thinkingLog []StatusEvent
	isThinking  bool
	lastStatus  StatusEvent

	// Updater
	updater       *AsyncUpdateManager
	updateReady   bool
	updateVersion string

	// Action Confirmation / Intervention
	pendingIntervention *interventionState

	// Prompt History (arrow up/down to cycle)
	promptHistory []string
	historyIndex  int
	tempPrompt    string // Stores current input when browsing history

	// Streaming response (Copilot SDK)
	streamingContent strings.Builder
	isStreaming      bool
	wasStreaming     bool

	// Dynamic Commands from Extensions
	dynamicCommands map[string]brain.CLICommand

	// Anyisland Management
	isManaged bool

	// Auracle Mode
	isAuracleMode bool

	// Sidebar & Focus System
	sidebar *SidebarManager
	focusScores map[string]float64 // path -> score
	activeFiles []string           // ordered by score
	lastFocusUpdate time.Time

	// Non-blocking Engine
	reactor        *reactor.Reactor
	md             *reactor.MarkdownRenderer
	lastRenderTime time.Time

	// Memoization
	lastViewportWidth int
	lastMessageCount  int
	memoizedView      string
	lastStreamContent string

	// Buffering for O(1) updates
	historyRendered string // Fully rendered stable history
	activeBlock     string // Currently active thinking/streaming content
}

type layoutMsg struct {
	content     string
	wasAtBottom bool
	wasAtTop    bool
	prevOffset  int
}
type recordTickMsg time.Time

type recordingProgressMsg struct {
	Current int
	Total   int
}

type recordingFinishedMsg struct {
	Path string
}

type recordingErrorMsg struct {
	Err error
}

type checkUpdateTickMsg time.Time

// interventionState holds data for a pending user confirmation.
type interventionState struct {
	title     string
	choices   []string
	selected  int
	resume    func(choice string) (interface{}, error)
	requestID string // To track the original request
}

type StatusEvent struct {
	Icon    string
	Message string
	Step    string // "plan", "exec", "reflect"
	Type    string // "think", "decision", "action", "delegation", "modification"
	Extra   string // Optional detailed data (command, diff, etc)
}

// Global channel for streaming thinking steps
var StatusStream = make(chan StatusEvent, 100)

type statusMsg StatusEvent

// streamDeltaMsg represents a streaming chunk from the Copilot SDK
type streamDeltaMsg struct {
	Delta string
}

// streamDoneMsg signals streaming has completed
type streamDoneMsg struct {
	FullContent string
}

// interventionResultMsg is sent after the user makes a choice in an intervention.
type interventionResultMsg struct {
	result interface{}
	err    error
}

type chatState struct {
	Messages      []string `json:"messages"`
	Input         string   `json:"input"`
	PromptHistory []string `json:"prompt_history"`
	ShowSidebar   bool     `json:"show_sidebar"`
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	userStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EE6FF8")).
			Bold(true)

	aiStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04D9FF")).
		Bold(true)

	systemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Bold(true)

	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	highlight = lipgloss.Color("#7D56F4")

	tagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			Bold(true).
			Italic(true)

	suggestionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Background(lipgloss.Color("#222222"))

	selectedSuggestionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#7D56F4")).
				Bold(true)

	treeStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#444444")).
			PaddingLeft(2)

	activeBorder = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder(), true).
			BorderForeground(lipgloss.Color("#7D56F4"))

	inactiveBorder = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true).
			BorderForeground(lipgloss.Color("#444444"))

	// Intervention/Approval selector styles
	interventionBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FF8C00")).
				Padding(1, 2).
				MarginTop(1)

	interventionTitleStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF8C00")).
				Bold(true)

	interventionChoiceStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#AAAAAA")).
				PaddingLeft(2)

	interventionSelectedStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FAFAFA")).
					Background(lipgloss.Color("#FF8C00")).
					Bold(true).
					PaddingLeft(2)

	statusLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#7D56F4")).
				Padding(0, 1).
				Bold(true)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4")).
				Bold(true)

	auracleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#FF00D7")).
			Padding(0, 1).
			Bold(true)

	// Chat Bubble Styles
	userBubbleStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#EE6FF8")).
			Padding(0, 1).
			MarginLeft(2)

	aiBubbleStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#04D9FF")).
			Padding(0, 1).
			MarginRight(2)

	userLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EE6FF8")).
			Bold(true).
			MarginLeft(2)

	aiLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04D9FF")).
			Bold(true)

	envStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Italic(true)

	envValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	// Thinking / Agentic process styles
	thinkHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#5F5F5F")).
				Padding(0, 1).
				Bold(true)

	decisionHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#00AF00")).
				Padding(0, 1).
				Bold(true)

	delegationHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#00AFD7")).
				Padding(0, 1).
				Bold(true)

	modificationHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FAFAFA")).
				Background(lipgloss.Color("#D75F00")).
				Padding(0, 1).
				Bold(true)

	blockBodyStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("#444444")).
			PaddingLeft(2).
			MarginLeft(1)
)

var allCommands = []string{
	"/help", "/status", "/cwd", "/version", "/clear", "/exit", "/show-tree", "/sidebar", "/copy", "/shot", "/record", "/auth", "/mcp", "/sys", "/skill", "/models", "/agent", "/session", "/update", "/restart", "/heal", "/connect", "/share", "/auracle",
}

var subCommands = map[string][]string{
	"/auth":    {"/ollama", "/github-models", "/github-copilot", "/copilot-sdk", "/openai", "/anthropic"},
	"/mcp":     {"/list", "/add", "/logs", "/call"},
	"/sys":     {"/stats", "/env", "/update", "/logs"},
	"/skill":   {"/list", "/info", "/load", "/disable"},
	"/models":  {"/list", "/use", "/pull"},
	"/agent":   {"/vibe", "/sdk", "/custom"},
	"/session": {"/list", "/clear"},
	"/connect": {"/list", "/new", "/join", "/close", "/clients"},
	"/share":   {"/browser", "/tui", "/stop"},
}
