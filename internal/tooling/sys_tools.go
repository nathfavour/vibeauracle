package tooling

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/nathfavour/vibeauracle/sys"
)

// Global Status Hook (injected by main)
var StatusReporter func(icon, step, msg string, extra ...string)

func ReportStatus(icon, step, msg string, extra ...string) {
	if StatusReporter != nil {
		StatusReporter(icon, step, msg, extra...)
	}
}
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
		Category:    CategoryFileSystem,
		Roles:       []AgentRole{RoleCoder, RoleEngineer},
		Complexity:  2,
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

func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	ReportStatus("📖", "exec", fmt.Sprintf("Reading file: %s", input.Path))

	content, err := t.fs.ReadFile(input.Path)
	if err != nil {
		ReportStatus("❌", "exec", fmt.Sprintf("Failed to read %s: %v", input.Path, err))
		return &ToolResult{Status: "error", Error: err}, err
	}

	ReportStatus("✅", "exec", fmt.Sprintf("Read %d bytes from %s", len(content), input.Path))
	return &ToolResult{
		Status:  "success",
		Content: string(content),
		Data:    map[string]interface{}{"size": len(content)},
	}, nil
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
		Description: "Create or overwrite a file with specific content.",
		Source:      "system",
		Category:    CategoryFileSystem,
		Roles:       []AgentRole{RoleCoder, RoleEngineer},
		Complexity:  5,
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

func (t *WriteFileTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	ReportStatus("💾", "exec", fmt.Sprintf("Writing to file: %s", input.Path))

	err := t.fs.WriteFile(input.Path, []byte(input.Content))
	if err != nil {
		ReportStatus("❌", "exec", fmt.Sprintf("Failed to write %s: %v", input.Path, err))
		return &ToolResult{Status: "error", Error: err}, err
	}

	ReportStatus("✅", "exec", fmt.Sprintf("Successfully wrote to %s", input.Path))
	return &ToolResult{
		Status:    "success",
		Content:   "File written successfully",
		Artifacts: []string{input.Path},
	}, nil
}

// ShellExecTool runs a shell command.
type ShellExecTool struct{}

func (t *ShellExecTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "sys_shell_exec",
		Description: "Execute a shell command.",
		Source:      "system",
		Category:    CategorySystem,
		Roles:       []AgentRole{RoleEngineer},
		Complexity:  8,
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

func (t *ShellExecTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	ReportStatus("🐚", "exec", fmt.Sprintf("Running: %s %v", input.Command, input.Args))

	cmd := exec.CommandContext(ctx, input.Command, input.Args...)
	output, err := cmd.CombinedOutput()
	status := "success"
	if err != nil {
		status = "error"
		ReportStatus("❌", "exec", fmt.Sprintf("Command failed: %v", err))
	} else {
		ReportStatus("✅", "exec", "Command completed successfully")
	}

	return &ToolResult{
		Status:  status,
		Content: string(output),
		Meta:    map[string]interface{}{"command": input.Command},
		Error:   err,
	}, nil // We return nil error here because the *execution* succeeded, even if the command failed, but we populate Error in struct
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
		Description: "Get a snapshot of current system resource usage.",
		Source:      "system",
		Category:    CategorySystem,
		Roles:       []AgentRole{RoleAll},
		Complexity:  1,
		Permissions: []Permission{PermRead},
		Parameters:  json.RawMessage(`{"type": "object"}`),
	}
}

func (t *SystemInfoTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	snap, err := t.monitor.GetSnapshot()
	if err != nil {
		return nil, err
	}
	return &ToolResult{
		Status:  "success",
		Content: fmt.Sprintf("CPU: %.1f%%, RAM: %.1f%%, CWD: %s", snap.CPUUsage, snap.MemoryUsage, snap.WorkingDir),
		Data:    snap,
	}, nil
}

// DeviceDeepDiveTool provides detailed hardware and environment information.
type DeviceDeepDiveTool struct{}

func (t *DeviceDeepDiveTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "sys_device_deep_dive",
		Description: "Get detailed hardware, OS, and environment information beyond basic metrics.",
		Source:      "system",
		Category:    CategorySystem,
		Roles:       []AgentRole{RoleEngineer, RoleArchitect},
		Complexity:  4,
		Permissions: []Permission{PermRead, PermExecute},
		Parameters:  json.RawMessage(`{"type": "object"}`),
	}
}

func (t *DeviceDeepDiveTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	ReportStatus("🔍", "exec", "Performing device deep dive...")

	results := make(map[string]string)

	// OS info
	if out, err := exec.CommandContext(ctx, "uname", "-a").Output(); err == nil {
		results["os_kernel"] = strings.TrimSpace(string(out))
	}

	// CPU info (Linux specific, but common)
	if out, err := exec.CommandContext(ctx, "grep", "model name", "/proc/cpuinfo").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		if len(lines) > 0 {
			results["cpu_model"] = strings.TrimSpace(strings.Split(lines[0], ":")[1])
		}
	}

	// Memory info
	if out, err := exec.CommandContext(ctx, "free", "-h").Output(); err == nil {
		results["memory"] = strings.TrimSpace(string(out))
	}

	// Disk info
	if out, err := exec.CommandContext(ctx, "df", "-h", ".").Output(); err == nil {
		results["disk_usage"] = strings.TrimSpace(string(out))
	}

	// Environment variables (filtered for safety)
	if out, err := exec.CommandContext(ctx, "env").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		var filteredEnv []string
		for _, line := range lines {
			if strings.HasPrefix(line, "PATH=") || strings.HasPrefix(line, "SHELL=") || strings.HasPrefix(line, "EDITOR=") || strings.HasPrefix(line, "LANG=") {
				filteredEnv = append(filteredEnv, line)
			}
		}
		results["env_vars"] = strings.Join(filteredEnv, "\n")
	}

	var sb strings.Builder
	for k, v := range results {
		sb.WriteString(fmt.Sprintf("### %s\n%s\n\n", strings.ToUpper(k), v))
	}

	return &ToolResult{
		Status:  "success",
		Content: sb.String(),
		Data:    results,
	}, nil
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
		Category:    CategoryFileSystem,
		Roles:       []AgentRole{RoleCoder, RoleEngineer},
		Complexity:  2,
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

func (t *ListFilesTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	ReportStatus("📂", "exec", fmt.Sprintf("Listing files in: %s", input.Path))

	files, err := t.fs.ListFiles(input.Path)
	if err != nil {
		return &ToolResult{Status: "error", Error: err}, err
	}
	return &ToolResult{
		Status:  "success",
		Content: fmt.Sprintf("Found %d files", len(files)),
		Data:    files,
	}, nil
}

// FetchURLTool fetches content from a URL.
type FetchURLTool struct{}

func (t *FetchURLTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "http_fetch",
		Description: "Fetch the content of a public URL (HTTP/HTTPS).",
		Source:      "system",
		Category:    CategoryNetwork,
		Roles:       []AgentRole{RoleEngineer, RoleArchitect},
		Complexity:  4,
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

func (t *FetchURLTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	ReportStatus("🌐", "exec", fmt.Sprintf("Fetching URL: %s", input.URL))

	req, err := http.NewRequestWithContext(ctx, "GET", input.URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ReportStatus("❌", "exec", fmt.Sprintf("Request failed: %v", err))
		return &ToolResult{Status: "error", Error: err}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ToolResult{Status: "error", Error: err}, err
	}

	ReportStatus("✅", "exec", fmt.Sprintf("Fetched %d bytes", len(body)))

	return &ToolResult{
		Status:  "success",
		Content: string(body),
		Meta:    map[string]interface{}{"status_code": resp.StatusCode},
	}, nil
}
