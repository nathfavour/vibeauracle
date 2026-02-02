package tooling

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// LSPManager manages the lifecycle of language servers.
type LSPManager struct {
	projectRoot string
	servers     map[string]*lspSession
	mu          sync.RWMutex
}

type lspSession struct {
	conn    jsonrpc2.Conn
	process *exec.Cmd
	cancel  context.CancelFunc
}

func NewLSPManager(projectRoot string) *LSPManager {
	if projectRoot == "" {
		projectRoot, _ = os.Getwd()
	}
	return &LSPManager{
		projectRoot: projectRoot,
		servers:     make(map[string]*lspSession),
	}
}

// GetSession returns or starts an LSP session for a specific language.
func (m *LSPManager) GetSession(ctx context.Context, language string) (*lspSession, error) {
	m.mu.RLock()
	if s, ok := m.servers[language]; ok {
		m.mu.RUnlock()
		return s, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check
	if s, ok := m.servers[language]; ok {
		return s, nil
	}

	s, err := m.startServer(ctx, language)
	if err != nil {
		return nil, err
	}

	m.servers[language] = s
	return s, nil
}

func (m *LSPManager) startServer(ctx context.Context, language string) (*lspSession, error) {
	var cmdName string
	var args []string

	switch language {
	case "go":
		cmdName = "gopls"
		args = []string{"serve"}
	case "python":
		cmdName = "pyright-langserver"
		args = []string{"--stdio"}
	case "typescript", "javascript":
		cmdName = "typescript-language-server"
		args = []string{"--stdio"}
	case "rust":
		cmdName = "rust-analyzer"
	default:
		return nil, fmt.Errorf("unsupported language for LSP: %s", language)
	}

	if _, err := exec.LookPath(cmdName); err != nil {
		return nil, fmt.Errorf("language server %s not found in PATH", cmdName)
	}

	cmdCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(cmdCtx, cmdName, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}

	stream := jsonrpc2.NewStream(struct {
		io.ReadCloser
		io.Writer
	}{stdout, stdin})
	
	// Use a no-op handler for now as we don't handle server-to-client notifications yet
	handler := func(ctx context.Context, reply jsonrpc2.Replier, req jsonrpc2.Request) error {
		return nil
	}
	conn := jsonrpc2.NewConn(stream)
	conn.Go(context.Background(), handler)

	session := &lspSession{
		conn:    conn,
		process: cmd,
		cancel:  cancel,
	}

	// Initialize the server
	params := &protocol.InitializeParams{
		RootURI: uri.File(m.projectRoot),
		Capabilities: protocol.ClientCapabilities{},
	}

	var result protocol.InitializeResult
	_, err = conn.Call(ctx, protocol.MethodInitialize, params, &result)
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("LSP initialize failed: %w", err)
	}

	err = conn.Notify(ctx, protocol.MethodInitialized, &protocol.InitializedParams{})
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("LSP initialized notification failed: %w", err)
	}

	return session, nil
}

func (s *lspSession) Close() {
	s.conn.Close()
	s.cancel()
	if s.process != nil {
		s.process.Process.Kill()
	}
}

func (m *LSPManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.servers {
		s.Close()
	}
	m.servers = make(map[string]*lspSession)
}

// MapExtensionToLanguage maps a file extension to an LSP language identifier.
func (m *LSPManager) MapExtensionToLanguage(ext string) string {
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".rs":
		return "rust"
	default:
		return ""
	}
}