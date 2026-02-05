package reactor

import (
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
)

type MarkdownRenderer struct {
	renderer *glamour.TermRenderer
	cache    map[string]string
	width    int
	mu       sync.RWMutex
}

func NewMarkdownRenderer(width int) *MarkdownRenderer {
	r, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-4),
	)
	return &MarkdownRenderer{
		renderer: r,
		cache:    make(map[string]string),
		width:    width,
	}
}

func (m *MarkdownRenderer) Render(content string) string {
	m.mu.RLock()
	// Quick check if width changed (invalidate cache if so)
	if cached, ok := m.cache[content]; ok {
		m.mu.RUnlock()
		return cached
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check cache after lock
	if cached, ok := m.cache[content]; ok {
		return cached
	}

	rendered, err := m.renderer.Render(content)
	if err != nil {
		return content
	}

	m.cache[content] = rendered
	return rendered
}

func (m *MarkdownRenderer) SetWidth(width int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.width == width {
		return
	}
	m.width = width
	m.cache = make(map[string]string)
	m.renderer, _ = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-4),
	)
}
