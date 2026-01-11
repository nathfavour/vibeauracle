package tooling

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"

	"github.com/nathfavour/vibeauracle/sys"
)

// ReadFileTool reads the content of a file.
type ReadFileTool struct {
	fs sys.FS
}

func NewReadFileTool(f sys.FS) *ReadFileTool {
	return &ReadFileTool{fs: f}
}

func (t *ReadFileTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "sys_read_file",
		Description: "Read the content of a file from the filesystem.",
		Source:      "system",
		Permissions: []Permission{PermRead},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string", "description": "Absolute or relative path to the file"}
			},
			"required": ["path"]
		}`),
	}
}

func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (interface{}, error) {
	var input struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}
	content, err := t.fs.ReadFile(input.Path)
	if err != nil {
		return nil, err
	}
	return string(content), nil
}

// WriteFileTool creates or overwrites a file.
type WriteFileTool struct {
	fs sys.FS
}

func NewWriteFileTool(f sys.FS) *WriteFileTool {
	return &WriteFileTool{fs: f}
}

func (t *WriteFileTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "sys_write_file",
		Description: "Create or overwrite a file with specific content. Highly sensitive.",
		Source:      "system",
		Permissions: []Permission{PermWrite},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string", "description": "Path to the file to write"},
				"content": {"type": "string", "description": "Content to write to the file"}
			},
			"required": ["path", "content"]
		}`),
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, args json.RawMessage) (interface{}, error) {
	var input struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}
	err := t.fs.WriteFile(input.Path, []byte(input.Content))
	if err != nil {
		return nil, err
	}
	return "File written successfully", nil
}

// ShellExecTool runs a shell command.
type ShellExecTool struct{}

func (t *ShellExecTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "sys_shell_exec",
		Description: "Execute a shell command. Returns stdout and stderr. Use with extreme caution.",
		Source:      "system",
		Permissions: []Permission{PermExecute},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"command": {"type": "string", "description": "The command to execute"},
				"args": {"type": "array", "items": {"type": "string"}, "description": "Arguments for the command"}
			},
			"required": ["command"]
		}`),
	}
}

func (t *ShellExecTool) Execute(ctx context.Context, args json.RawMessage) (interface{}, error) {
	var input struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, input.Command, input.Args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("command failed: %w (output: %s)", err, string(output))
	}
	return string(output), nil
}

// SystemInfoTool provides a snapshot of system resources.
type SystemInfoTool struct {
	monitor *sys.Monitor
}

func NewSystemInfoTool(m *sys.Monitor) *SystemInfoTool {
	return &SystemInfoTool{monitor: m}
}

func (t *SystemInfoTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "sys_info",
		Description: "Get a snapshot of current system resource usage (CPU, Memory) and working directory.",
		Source:      "system",
		Permissions: []Permission{PermRead},
		Parameters:  json.RawMessage(`{"type": "object"}`),
	}
}

func (t *SystemInfoTool) Execute(ctx context.Context, args json.RawMessage) (interface{}, error) {
	return t.monitor.GetSnapshot()
}

// ListFilesTool lists files in a directory.
type ListFilesTool struct {
	fs sys.FS
}

func NewListFilesTool(f sys.FS) *ListFilesTool {
	return &ListFilesTool{fs: f}
}

func (t *ListFilesTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "sys_list_files",
		Description: "List files and directories in a given path.",
		Source:      "system",
		Permissions: []Permission{PermRead},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string", "description": "Path to list files from"}
			},
			"required": ["path"]
		}`),
	}
}

func (t *ListFilesTool) Execute(ctx context.Context, args json.RawMessage) (interface{}, error) {
	var input struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}
	return t.fs.ListFiles(input.Path)
}

// FetchURLTool fetches content from a URL.
type FetchURLTool struct{}

func (t *FetchURLTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "http_fetch",
		Description: "Fetch the content of a public URL (HTTP/HTTPS).",
		Source:      "system",
		Permissions: []Permission{PermNetwork},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {"type": "string", "description": "The URL to fetch"}
			},
			"required": ["url"]
		}`),
	}
}

func (t *FetchURLTool) Execute(ctx context.Context, args json.RawMessage) (interface{}, error) {
	var input struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", input.URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return string(body), nil
}
