package reactor

import (
	"sync"

	"github.com/charmbracelet/glamour"
)

type MarkdownRenderer struct {
	cache sync.Map
	width int
	mu    sync.Mutex // Protects width and pool recreation
	pool  *sync.Pool
}

func NewMarkdownRenderer(width int) *MarkdownRenderer {
	mr := &MarkdownRenderer{
		width: width,
	}
	mr.recreatePool(width)
	return mr
}

func (m *MarkdownRenderer) recreatePool(width int) {
	m.pool = &sync.Pool{
		New: func() interface{} {
			r, _ := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(width-4),
			)
			return r
		},
	}
}

func (m *MarkdownRenderer) Render(content string) string {
	// 1. Concurrent-safe cache check
	if cached, ok := m.cache.Load(content); ok {
		return cached.(string)
	}

	// 2. Get a renderer from the pool
	r := m.pool.Get().(*glamour.TermRenderer)
	defer m.pool.Put(r)

	rendered, err := r.Render(content)
	if err != nil {
		return content
	}

	m.cache.Store(content, rendered)
	return rendered
}

func (m *MarkdownRenderer) SetWidth(width int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.width == width {
		return
	}
	m.width = width
	// Invalidate cache and pool for new width
	m.cache = sync.Map{}
	m.recreatePool(width)
}

