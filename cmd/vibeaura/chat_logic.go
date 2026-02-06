package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/nathfavour/vibeauracle/brain"
	"github.com/nathfavour/vibeauracle/sys"
)

func (m *model) loadDynamicCommands() {
	m.dynamicCommands = make(map[string]brain.CLICommand)
	for _, ext := range m.brain.Extensions() {
		if !ext.Enabled || ext.Manifest == nil {
			continue
		}
		for _, cmd := range ext.Manifest.CLICommands {
			slashName := "/" + cmd.Name
			m.dynamicCommands[slashName] = cmd
			// Add to auto-complete
			found := false
			for _, c := range allCommands {
				if c == slashName {
					found = true
					break
				}
			}
			if !found {
				allCommands = append(allCommands, slashName)
			}
		}
	}
}

func (m *model) loadTree(path string) {
	entries, _ := os.ReadDir(path)
	m.treeEntries = nil
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") || e.Name() == ".env" {
			m.treeEntries = append(m.treeEntries, e)
		}
	}
	m.isFileOpen = false
	m.updatePerusalContent()
}

func (m *model) openFile(path string) {
	content, err := os.ReadFile(path)
	if err == nil {
		m.isFileOpen = true
		m.currentPath = path
		m.editArea.SetValue(string(content))
		m.perusalVp.SetContent(string(content))
	}
}

func (m *model) updatePerusalContent() {
	if m.isFileOpen {
		return
	}
	var sb strings.Builder
	sb.WriteString(systemStyle.Render(" EXPLORER: "+m.currentPath) + "

")
	for i, entry := range m.treeEntries {
		cursor := "  "
		if i == m.treeCursor {
			cursor = "> "
		}
		icon := "📄 "
		if entry.IsDir() {
			icon = "📁 "
		}
		line := cursor + icon + entry.Name()
		if i == m.treeCursor {
			sb.WriteString(suggestionStyle.Render(line) + "
")
		} else {
			sb.WriteString(line + "
")
		}
	}
	m.perusalVp.SetContent(sb.String())
}

func (m *model) updateSuggestions(val string) {
	m.suggestions = nil
	m.suggestionIdx = 0
	m.triggerChar = ""
	m.isFilteringModels = false

	if val == "" {
		return
	}

	if strings.Contains(val, "/models /use") {
		m.isFilteringModels = true
		if len(m.allModelDiscoveries) == 0 {
			// Trigger discovery
			go func() {
				// We can't return Cmd here, so we'll just wait for the next Update cycle
			}()
		}

		// Everything after "/models /use " is the filter
		parts := strings.Split(val, "/models /use")
		filter := ""
		if len(parts) > 1 {
			filter = strings.TrimSpace(parts[1])
		}
		m.suggestionFilter = filter

		for _, d := range m.allModelDiscoveries {
			display := fmt.Sprintf("%s (%s)", shortenModelName(d.Name), d.Provider)
			if filter == "" || strings.Contains(strings.ToLower(display), strings.ToLower(filter)) {
				// We store the full identifier for applySuggestion, but display it nicely
				m.suggestions = append(m.suggestions, fmt.Sprintf("%s|%s", d.Provider, d.Name))
			}
		}
		return
	}

	// Handle trailing space for subcommand triggering
	if strings.HasSuffix(val, " ") {
		parts := strings.Fields(val)
		if len(parts) == 1 {
			if subs, ok := subCommands[parts[0]]; ok {
				m.suggestions = subs
				m.triggerChar = "" // Already has / in the subCommand string
				sort.Strings(m.suggestions)
				return
			}
		}
	}

	words := strings.Fields(val)
	if len(words) == 0 {
		if strings.HasSuffix(val, "/") {
			m.triggerChar = "/"
			m.suggestions = append([]string{}, allCommands...)
			sort.Strings(m.suggestions)
		} else if strings.HasSuffix(val, "#") {
			m.triggerChar = "#"
			m.suggestions = m.getFileSuggestions("")
		}
		return
	}

	lastWord := words[len(words)-1]

	// Check if we are typing a subcommand
	if len(words) > 1 {
		parentCmd := words[0]
		if subs, ok := subCommands[parentCmd]; ok {
			m.triggerChar = "" // Subcommands already have slashes
			for _, sub := range subs {
				if strings.HasPrefix(sub, lastWord) {
					m.suggestions = append(m.suggestions, sub)
				}
			}
			sort.Strings(m.suggestions)
			if len(m.suggestions) > 0 {
				return
			}
		}
	}

	if strings.HasPrefix(lastWord, "/") {
		m.triggerChar = "/"
		for _, cmd := range allCommands {
			if strings.HasPrefix(cmd, lastWord) {
				m.suggestions = append(m.suggestions, cmd)
			}
		}
		sort.Strings(m.suggestions)
	} else if strings.HasPrefix(lastWord, "#") {
		m.triggerChar = "#"
		m.suggestions = m.getFileSuggestions(lastWord[1:])
	}
}

func (m *model) getFileSuggestions(prefix string) []string {
	var suggestions []string
	root, _ := os.Getwd()

	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || len(suggestions) > 30 {
			return nil
		}

		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == "node_modules" || name == "vendor" || name == "bin" || name == "dist" {
				return filepath.SkipDir
			}
			if prefix != "" && !strings.HasPrefix(name, prefix) && !strings.HasPrefix(path, prefix) {
				return nil
			}
		}

		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}

		if prefix == "" || strings.HasPrefix(rel, prefix) || strings.HasPrefix(name, prefix) {
			suggestions = append(suggestions, rel)
		}

		return nil
	})

	sort.Strings(suggestions)
	return suggestions
}

func (m *model) applySuggestion() (tea.Model, tea.Cmd) {
	if len(m.suggestions) == 0 {
		return m, nil
	}

	val := m.textarea.Value()
	suggestion := m.suggestions[m.suggestionIdx]

	// Handle model selection specialized format: provider|name
	if m.isFilteringModels && strings.Contains(suggestion, "|") {
		parts := strings.Split(suggestion, "|")
		provider := parts[0]
		modelName := parts[1]
		fullCmd := fmt.Sprintf("/models /use %s %s", provider, modelName)
		m.textarea.SetValue(fullCmd)
		m.textarea.SetCursor(len(m.textarea.Value()))
		m.suggestions = nil
		return m.handleSlashCommand(fullCmd)
	}

	// Determine if we are completing a subcommand or a top-level command
	words := strings.Fields(val)
	if len(words) == 0 {
		m.textarea.SetValue(suggestion)
	} else {
		// If the last word is what we're completing
		lastWord := words[len(words)-1]

		if strings.HasSuffix(val, " ") {
			// Context: User just typed a space, we are appending a new part
			m.textarea.SetValue(strings.TrimRight(val, " ") + " " + suggestion)
		} else if strings.HasPrefix(suggestion, lastWord) || (strings.HasPrefix(lastWord, "/") && strings.HasPrefix(suggestion, "/")) {
			// Context: User is partially typing the suggestion, replace the partial part
			words[len(words)-1] = suggestion
			m.textarea.SetValue(strings.Join(words, " "))
		} else {
			// Context: Unclear, safest to append with space
			m.textarea.SetValue(strings.TrimRight(val, " ") + " " + suggestion)
		}
	}

	m.textarea.SetCursor(len(m.textarea.Value()))
	m.suggestions = nil

	currentVal := strings.TrimSpace(m.textarea.Value())
	parts := strings.Fields(currentVal)

	// If we just completed a top-level command that has subcommands, add a space and show them
	if len(parts) == 1 {
		if _, ok := subCommands[parts[0]]; ok {
			m.textarea.SetValue(parts[0] + " ")
			m.textarea.SetCursor(len(m.textarea.Value()))
			m.updateSuggestions(m.textarea.Value())
			return m, nil
		}
	}

	// Auto-execute logic for no-arg commands/subcommands
	noArgSubs := map[string]map[string]bool{
		"/models":  {"/list": true},
		"/sys":     {"/stats": true, "/env": true, "/update": true, "/logs": true},
		"/mcp":     {"/list": true, "/logs": true},
		"/skill":   {"/list": true},
		"/agent":   {"/vibe": true, "/sdk": true},
		"/session": {"/list": true, "/clear": true},
	}

	if len(parts) == 1 {
		if _, hasSubs := subCommands[parts[0]]; !hasSubs {
			return m.handleSlashCommand(currentVal)
		}
	} else if len(parts) == 2 {
		if subs, ok := noArgSubs[parts[0]]; ok {
			if subs[parts[1]] {
				return m.handleSlashCommand(currentVal)
			}
		}
	}

	// Otherwise, add a trailing space for the next argument
	m.textarea.SetValue(currentVal + " ")
	m.textarea.SetCursor(len(m.textarea.Value()))
	return m, nil
}

func (m *model) processRequest(content string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		req := brain.Request{
			ID:      uuid.NewString(),
			Content: content,
		}
		res, err := m.brain.Process(ctx, req)
		var resp brain.Response
		if err != nil {
			resp.Error = err
		} else {
			resp = res.(brain.Response)
		}
		return resp
	}
}

func (m *model) takeScreenshot() (tea.Model, tea.Cmd) {
	config := m.brain.GetConfig()
	dir := config.UI.ScreenshotDir

	if err := os.MkdirAll(dir, 0755); err != nil {
		m.messages = append(m.messages, errorStyle.Render(" Screenshot Error: ")+err.Error())
		return m, nil
	}

	timestamp := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("vibeaura_%s", timestamp)

	basePath := filepath.Join(dir, filename)
	svgPath := basePath + ".svg"
	pngPath := basePath + ".png"

	// Use current layout but ensure it's rendered for capture
	m.isCapturing = true
	rawView := m.View()
	m.isCapturing = false

	// Tier 2: Generate SVG but don't save yet if targeting PNG
	svgContent := convertAnsiToSVG(rawView)
	_ = os.WriteFile(svgPath, []byte(svgContent), 0644)

	// Tier 1: Try PNG
	err := convertToPNG(svgPath, pngPath)

	msg := systemStyle.Render(" SCREENSHOT CAPTURED ") + "
"

	if err == nil {
		// Highest Tier: PNG only
		_ = os.Remove(svgPath)
		msg += helpStyle.Render("🖼️ Saved PNG: " + pngPath)
	} else if svgContent != "" {
		// Middle Tier: SVG only
		msg += helpStyle.Render("📍 Saved SVG: " + svgPath)
		msg += "
" + errorStyle.Render(" PNG fail: ") + helpStyle.Render("install ffmpeg/rsvg")
	}

	m.messages = append(m.messages, msg)
	return m, m.asyncRender()
}

func (m *model) toggleRecording() (tea.Model, tea.Cmd) {
	if m.isRecording {
		m.isRecording = false
		msg := systemStyle.Render(" RECORDING STOPPED ") + "
" + helpStyle.Render("Processing frames in background...")
		m.messages = append(m.messages, msg)

		// Deep copy frames to avoid race conditions during background processing
		frames := make([]recordedFrame, len(m.recordedFrames))
		copy(frames, m.recordedFrames)
		m.recordedFrames = nil

		// Start encoding state
		m.isEncoding = true
		m.encodingCurrent = 0
		m.encodingTotal = len(frames)
		m.recordingErr = nil

		// Capture program and config for background use
		p := m.getProgram()
		outDir := m.brain.GetConfig().UI.ScreenshotDir

		go m.processRecording(m.recordingID, frames, p, outDir)
		return m, m.asyncRender()
	}

	// Dependency Check
	if err := checkRecordingDependencies(); err != nil {
		m.messages = append(m.messages, errorStyle.Render(" RECORDING UNAVAILABLE ")+"
"+helpStyle.Render(err.Error()))
		return m, m.asyncRender()
	}

	m.isRecording = true
	m.isDirty = true // Force capture of the first frame
	m.recordingID = uuid.New().String()
	m.recordedFrames = nil
	msg := systemStyle.Render(" RECORDING STARTED ") + "
" + helpStyle.Render("Capture interval: 41ms")
	m.messages = append(m.messages, msg)
	return m, tea.Batch(m.asyncRender(), recordTick())
}

func (m *model) getProgram() *tea.Program {
	return uiProgram
}

func getBestEncoder() string {
	if runtime.GOOS == "darwin" {
		return "h264_videotoolbox"
	}
	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		cmd := exec.Command("ffmpeg", "-hide_banner", "-encoders")
		if out, err := cmd.Output(); err == nil && strings.Contains(string(out), "h264_nvenc") {
			return "h264_nvenc"
		}
	}
	return "libx264"
}

func (m *model) processRecording(id string, frames []recordedFrame, p *tea.Program, outDir string) {
	if len(frames) == 0 {
		if p != nil {
			p.Send(recordingErrorMsg{Err: fmt.Errorf("no frames recorded")})
		}
		return
	}

	_ = os.MkdirAll(outDir, 0755)

	// 1. Parallelized rendering to memory with deduplication
	numFrames := len(frames)
	rgbDatas := make([][]byte, numFrames)
	var width, height int

	type renderResult struct {
		data []byte
		w, h int
		err  error
	}
	cache := make(map[string]*renderResult)
	var cacheMu sync.Mutex

	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU())
	var processedCount int32

	for i := range frames {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ansi := frames[idx].content
			cacheMu.Lock()
			res, ok := cache[ansi]
			cacheMu.Unlock()

			if !ok {
				data, w, h, err := renderAnsiToRGB(ansi)
				res = &renderResult{data, w, h, err}
				cacheMu.Lock()
				cache[ansi] = res
				cacheMu.Unlock()
			}

			if res.err == nil {
				rgbDatas[idx] = res.data
				cacheMu.Lock()
				if width == 0 {
					width = res.w
					height = res.h
				}
				cacheMu.Unlock()
			}

			newCount := atomic.AddInt32(&processedCount, 1)
			if p != nil && newCount%10 == 0 {
				p.Send(recordingProgressMsg{Current: int(newCount), Total: numFrames})
			}
		}(i)
	}
	wg.Wait()

	if p != nil {
		p.Send(recordingProgressMsg{Current: numFrames, Total: numFrames})
	}

	// 2. Assemble with FFmpeg using rawvideo pipe and GPU acceleration
	timestamp := time.Now().Format("2006-01-02_150405")
	finalPath := filepath.Join(outDir, fmt.Sprintf("vibeaura_rec_%s.mp4", timestamp))

	encoder := getBestEncoder()
	args := []string{
		"-y",
		"-framerate", "24",
		"-f", "rawvideo",
		"-pixel_format", "rgb24",
		"-video_size", fmt.Sprintf("%dx%d", width, height),
		"-i", "-",
		"-c:v", encoder,
	}

	if encoder == "libx264" {
		args = append(args, "-preset", "slower", "-crf", "17", "-tune", "stillimage")
	} else if encoder == "h264_nvenc" {
		args = append(args, "-preset", "slow", "-cq", "17", "-rc", "vbr")
	} else if encoder == "h264_videotoolbox" {
		args = append(args, "-realtime", "false", "-q:v", "90")
	}

	args = append(args,
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		"-vf", "scale='min(1920,iw)':-2:force_original_aspect_ratio=decrease:flags=lanczos,pad=ceil(iw/2)*2:ceil(ih/2)*2",
		finalPath,
	)

	cmd := exec.Command("ffmpeg", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		if p != nil {
			p.Send(recordingErrorMsg{Err: fmt.Errorf("ffmpeg pipe failed: %w", err)})
		}
		return
	}

	if err := cmd.Start(); err != nil {
		if p != nil {
			p.Send(recordingErrorMsg{Err: fmt.Errorf("ffmpeg start failed: %w", err)})
		}
		return
	}

	// Feed the pipe with frames
	for i, frame := range frames {
		data := rgbDatas[i]
		if data == nil {
			continue
		}
		for j := 0; j < frame.ticks; j++ {
			_, _ = stdin.Write(data)
		}
	}
	stdin.Close()
	_ = cmd.Wait()

	if p != nil {
		p.Send(recordingFinishedMsg{Path: finalPath})
	}
}

func (m *model) discoverModels() tea.Cmd {
	return func() tea.Msg {
		discoveries, err := m.brain.DiscoverModels(context.Background())
		if err != nil {
			return brain.Response{Error: err}
		}
		return discoveries
	}
}

func (m *model) pullOllamaModel(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.brain.PullModel(context.Background(), name)
		if err != nil {
			return brain.Response{Error: err}
		}
		return brain.Response{Content: "Successfully pulled " + name + ". You can now use it with /models /use ollama " + name}
	}
}

func (m *model) resumeIntervention(resumeFn func(string) (interface{}, error), choice string) tea.Cmd {
	return func() tea.Msg {
		result, err := resumeFn(choice)
		return interventionResultMsg{result: result, err: err}
	}
}

func (m *model) saveState() {
	state := chatState{
		Messages:      m.messages,
		Input:         m.textarea.Value(),
		PromptHistory: m.promptHistory,
		ShowSidebar:   m.showTree,
	}
	data, _ := json.Marshal(state)
	sessionID := m.brain.GetSessionID()
	_ = m.brain.StoreState(sessionID, data)
}
