package context

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"github.com/nathfavour/vibeauracle/model"
	"github.com/nathfavour/vibeauracle/sys"
	"github.com/philippgille/chromem-go"
)

func NewMemory(embedder model.Provider) *Memory {
	home, _ := os.UserHomeDir()
	dbDir := filepath.Join(home, ".vibeauracle")
	os.MkdirAll(dbDir, 0755)

	vdb := chromem.NewDB()

	dbPath := filepath.Join(dbDir, "vibe.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return &Memory{Window: NewWindow(50), vdb: vdb, embedder: embedder}
	}

	// Initialize tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS memory (
			key TEXT PRIMARY KEY,
			value TEXT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS app_state (
			id TEXT PRIMARY KEY,
			data TEXT,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS project_knowledge (
			root_path TEXT PRIMARY KEY,
			git_sha TEXT,
			logical_map TEXT,
			last_indexed TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		fmt.Printf("Error initializing database tables: %v\n", err)
	}

	return &Memory{
		db:       db,
		Window:   NewWindow(50),
		vdb:      vdb,
		embedder: embedder,
	}
}

// SetEmbedder updates the underlying provider used for embeddings
func (m *Memory) SetEmbedder(p model.Provider) {
	m.embedder = p
}

// AddToWindow pushes content into the short-term rolling context.
func (m *Memory) AddToWindow(id, content, itemType string) {
	if m.Window != nil {
		m.Window.Add(id, content, itemType)
	}
}

// Store adds a fact or snippet to the long-term db memory.
func (m *Memory) Store(key string, value string) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}
	_, err := m.db.Exec("INSERT OR REPLACE INTO memory (key, value) VALUES (?, ?)", key, value)
	return err
}

// SyncProject indexes the codebase semantically.
func (m *Memory) SyncProject(ctx context.Context, rootPath string) error {
	if m.vdb == nil || m.embedder == nil {
		return fmt.Errorf("vector db or embedder not initialized")
	}

	colName := "project_" + filepath.Base(rootPath)

	embeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
		res, err := m.embedder.Embed(ctx, []string{text})
		if err != nil {
			return nil, err
		}
		return res[0], nil
	}

	col, err := m.vdb.GetOrCreateCollection(colName, nil, embeddingFunc)
	if err != nil {
		return err
	}

	return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if strings.Contains(path, "/.git/") || strings.Contains(path, "/node_modules/") || strings.Contains(path, "/vendor/") {
			return nil
		}

		ext := filepath.Ext(path)
		isCode := false
		for _, e := range []string{`.go`, `.py`, `.ts`, `.js`, `.rs`, `.md`, `.txt`} {
			if ext == e {
				isCode = true
				break
			}
		}

		if !isCode {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(rootPath, path)
		doc := chromem.Document{
			ID:      relPath,
			Content: string(content),
			Metadata: map[string]string{
				"path": relPath,
				"ext":  ext,
			},
		}

		return col.AddDocuments(ctx, []chromem.Document{doc}, 1)
	})
}

// DiscoverProjectRules scans for instruction files like .cursorrules.
func (m *Memory) DiscoverProjectRules(rootPath string) string {
	var rules []string
	candidates := []string{
		".cursorrules",
		".github/copilot-instructions.md",
		"VIBE.md",
		"CONTRIBUTING.md",
	}

	for _, c := range candidates {
		p := filepath.Join(rootPath, c)
		if data, err := os.ReadFile(p); err == nil {
			rules = append(rules, fmt.Sprintf("---\nRules from %s ---\n%s", c, string(data)))
		}
	}

	return strings.Join(rules, "\n\n")
}

// Recall retrieves relevant snippets from window, DB, and Vector DB.
func (m *Memory) Recall(ctx context.Context, query string, rootPath string) ([]string, error) {
	var results []string

	if m.Window != nil {
		results = append(results, "--- Current Context Window ---")
		results = append(results, m.Window.GetContext())
	}

	if m.vdb != nil && m.embedder != nil {
		colName := "project_" + filepath.Base(rootPath)

		embeddingFunc := func(ctx context.Context, text string) ([]float32, error) {
			res, err := m.embedder.Embed(ctx, []string{text})
			if err != nil {
				return nil, err
			}
			return res[0], nil
		}

		col, err := m.vdb.GetOrCreateCollection(colName, nil, embeddingFunc)
		if err == nil {
			queryRes, err := col.Query(ctx, query, 3, nil, nil)
			if err == nil && len(queryRes) > 0 {
				results = append(results, "--- Semantic Project Knowledge ---")
				for _, res := range queryRes {
					results = append(results, fmt.Sprintf("[File: %s]\n%s", res.Metadata["path"], res.Content))
				}
			}
		}
	}

	if m.db != nil {
		rows, err := m.db.Query("SELECT value FROM memory WHERE value LIKE ? LIMIT 3", "%"+query+"%")
		if err == nil {
			defer rows.Close()
			results = append(results, "--- Historical Snippets ---")
			for rows.Next() {
				var s string
				if err := rows.Scan(&s); err == nil {
					results = append(results, s)
				}
			}
		}
	}

	return results, nil
}

// SaveState persists arbitrary application state (JSON)
func (m *Memory) SaveState(id string, state interface{}) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	_, err = m.db.Exec("INSERT OR REPLACE INTO app_state (id, data) VALUES (?, ?)", id, string(data))
	return err
}

// LoadState retrieves persisted application state
func (m *Memory) LoadState(id string, target interface{}) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}
	var data string
	err := m.db.QueryRow("SELECT data FROM app_state WHERE id = ?", id).Scan(&data)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), target)
}

// ClearState removes a specific state entry
func (m *Memory) ClearState(id string) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}
	_, err := m.db.Exec("DELETE FROM app_state WHERE id = ?", id)
	return err
}

// ListStates returns all stored state IDs matching a prefix
func (m *Memory) ListStates(prefix string) ([]string, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	rows, err := m.db.Query("SELECT id FROM app_state WHERE id LIKE ?", prefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// SaveProjectKnowledge stores logical info about a project
func (m *Memory) SaveProjectKnowledge(ctx sys.ProjectContext) error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}
	data, err := json.Marshal(ctx.LogicalMap)
	if err != nil {
		return err
	}
	_, err = m.db.Exec(`
		INSERT OR REPLACE INTO project_knowledge (root_path, git_sha, logical_map, last_indexed)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
		ctx.RootPath, ctx.GitSHA, string(data))
	return err
}

// GetProjectKnowledge retrieves logical info for a project
func (m *Memory) GetProjectKnowledge(rootPath string) (*sys.ProjectContext, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var gitSHA, logicalMapStr string
	var lastIndexed time.Time
	err := m.db.QueryRow(`
		SELECT git_sha, logical_map, last_indexed 
		FROM project_knowledge WHERE root_path = ?`,
		rootPath).Scan(&gitSHA, &logicalMapStr, &lastIndexed)
	if err != nil {
		return nil, err
	}

	var logicalMap map[string]string
	if err := json.Unmarshal([]byte(logicalMapStr), &logicalMap); err != nil {
		return nil, err
	}

	return &sys.ProjectContext{
		RootPath:    rootPath,
		GitSHA:      gitSHA,
		LogicalMap:  logicalMap,
		LastIndexed: lastIndexed,
	}, nil
}
