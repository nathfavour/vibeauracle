package reactor

import (
	"sync"

	"github.com/charmbracelet/glamour"
)

type MarkdownRenderer struct {
	// width -> content hash -> rendered string
	caches map[int]*sync.Map
	pools  map[int]*sync.Pool
	mu     sync.RWMutex
}

func NewMarkdownRenderer(width int) *MarkdownRenderer {
	mr := &MarkdownRenderer{
		caches: make(map[int]*sync.Map),
		pools:  make(map[int]*sync.Pool),
	}
	mr.getOrCreateResources(width)
	return mr
}

func (m *MarkdownRenderer) getOrCreateResources(width int) (*sync.Map, *sync.Pool) {
	m.mu.RLock()
	cache, okC := m.caches[width]
	pool, okP := m.pools[width]
	m.mu.RUnlock()

	if okC && okP {
		return cache, pool
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check
	if cache, ok := m.caches[width]; ok {
		return cache, m.pools[width]
	}

	newCache := &sync.Map{}
	newPool := &sync.Pool{
		New: func() interface{} {
			r, _ := glamour.NewTermRenderer(
				glamour.WithAutoStyle(),
				glamour.WithWordWrap(width-4),
			)
			return r
		},
	}
	m.caches[width] = newCache
	m.pools[width] = newPool
	return newCache, newPool
}

func (m *MarkdownRenderer) Render(content string, width int) string {
	cache, pool := m.getOrCreateResources(width)

	// 1. O(1) Cache hit
	if cached, ok := cache.Load(content); ok {
		return cached.(string)
	}

	// 2. Render only if missed
	r := pool.Get().(*glamour.TermRenderer)
	defer pool.Put(r)

	rendered, err := r.Render(content)
	if err != nil {
		return content
	}

	cache.Store(content, rendered)
	return rendered
}

func (m *MarkdownRenderer) SetWidth(width int) {
	m.getOrCreateResources(width)
}


