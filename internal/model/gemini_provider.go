package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nathfavour/vibeauracle/auth"
)

const (
	CodeAssistEndpoint = "https://cloudcode-pa.googleapis.com/v1internal"
)

// GeminiProvider implements model.Provider for Gemini via spoofed CLI credentials.
type GeminiProvider struct {
	client    *http.Client
	modelName string
	project   string
	mu        sync.Mutex

	onDelta func(string)
	onDone  func(string)
	usageCB func(Usage)
}

// NewProvider creates a new Gemini provider using spoofed CLI credentials.
func NewProvider(modelName string) (*GeminiProvider, error) {
	if !auth.IsGeminiCLIInstalled() {
		return nil, fmt.Errorf("gemini-cli credentials not found. Please run 'gemini login' first")
	}

	// Try to auto-detect project
	project := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if project == "" {
		project = os.Getenv("GOOGLE_CLOUD_PROJECT_ID")
	}

	p := &GeminiProvider{
		modelName: modelName,
		project:   project,
	}

	// If no modelName provided, try to resolve from gemini-cli settings
	if p.modelName == "" {
		p.resolveDefaultModel()
	}

	return p, nil
}

func (p *GeminiProvider) resolveDefaultModel() {
	home, _ := os.UserHomeDir()
	settingsPath := filepath.Join(home, ".gemini", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		p.modelName = "" // Let ListModels decide if empty
		return
	}

	var settings struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(data, &settings); err != nil || settings.Model == "" {
		p.modelName = ""
		return
	}

	// Porting the resolution logic: if 'auto', let the dynamic discovery handle it
	if settings.Model == "auto" {
		p.modelName = ""
	} else {
		p.modelName = settings.Model
	}
}

func (p *GeminiProvider) Name() string {
	return "gemini-cli"
}

func (p *GeminiProvider) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.client != nil {
		return nil
	}

	client, err := auth.GetGeminiHTTPClient(ctx)
	if err != nil {
		return fmt.Errorf("initializing gemini auth client: %w", err)
	}

	p.client = client
	return nil
}

func (p *GeminiProvider) SetStreamCallbacks(onDelta func(string), onDone func(string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onDelta = onDelta
	p.onDone = onDone
}

func (p *GeminiProvider) SetUsageCallback(cb func(Usage)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.usageCB = cb
}

func (p *GeminiProvider) Generate(ctx context.Context, prompt string) (GeneratedResponse, error) {
	if err := p.Start(ctx); err != nil {
		return GeneratedResponse{}, err
	}

	if p.project == "" {
		p.loadProject(ctx)
	}

	modelToUse := p.modelName
	if modelToUse == "" {
		models, _ := p.ListModels(ctx)
		if len(models) > 0 {
			// Find first 'pro' or just the first one
			modelToUse = models[0]
			for _, m := range models {
				if strings.Contains(m, "pro") {
					modelToUse = m
					break
				}
			}
		} else {
			modelToUse = "gemini-2.5-pro" // Absolute last resort if quota fails
		}
	}

	reqBody := CAGenerateContentRequest{
		Model:   "models/" + modelToUse,
		Project: p.project,
		Request: VertexGenerateContentRequest{
			Contents: []Content{
				{
					Role: "user",
					Parts: []Part{
						{Text: prompt},
					},
				},
			},
			GenerationConfig: &GenerationConfig{
				Temperature: 0.7,
			},
		},
	}

	data, _ := json.Marshal(reqBody)
	url := fmt.Sprintf("%s:streamGenerateContent?alt=sse", CodeAssistEndpoint)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return GeneratedResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return GeneratedResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return GeneratedResponse{}, fmt.Errorf("api error (%d): %s", resp.StatusCode, string(body))
	}

	var fullContent strings.Builder
	var lastUsage Usage

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return GeneratedResponse{}, err
		}

		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		jsonData := line[6:]
		var chunk CAGenerateContentResponse
		if err := json.Unmarshal([]byte(jsonData), &chunk); err != nil {
			continue
		}

		if len(chunk.Response.Candidates) > 0 {
			for _, part := range chunk.Response.Candidates[0].Content.Parts {
				fullContent.WriteString(part.Text)
				if p.onDelta != nil {
					p.onDelta(part.Text)
				}
			}
		}

		if chunk.Response.UsageMetadata.TotalTokenCount > 0 {
			lastUsage = Usage{
				InputTokens:  chunk.Response.UsageMetadata.PromptTokenCount,
				OutputTokens: chunk.Response.UsageMetadata.CandidatesTokenCount,
				TotalTokens:  chunk.Response.UsageMetadata.TotalTokenCount,
			}
		}
	}

	res := GeneratedResponse{
		Content: fullContent.String(),
		Usage:   lastUsage,
	}

	if p.onDone != nil {
		p.onDone(res.Content)
	}
	if p.usageCB != nil {
		p.usageCB(res.Usage)
	}

	return res, nil
}

func (p *GeminiProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return nil, fmt.Errorf("embedding not implemented for gemini-cli yet")
}

func (p *GeminiProvider) loadProject(ctx context.Context) {
	url := fmt.Sprintf("%s:loadCodeAssist", CodeAssistEndpoint)
	reqBody := map[string]interface{}{
		"metadata": map[string]string{
			"ideType":    "GEMINI_CLI",
			"platform":   "LINUX_AMD64",
			"pluginType": "GEMINI",
		},
	}
	data, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var res struct {
		CloudaicompanionProject string `json:"cloudaicompanionProject"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err == nil {
		if res.CloudaicompanionProject != "" {
			p.project = res.CloudaicompanionProject
		}
	}
}

func (p *GeminiProvider) ListModels(ctx context.Context) ([]string, error) {
	if err := p.Start(ctx); err != nil {
		return nil, err
	}

	// First, try to load project ID if not set
	if p.project == "" {
		p.loadProject(ctx)
	}

	url := fmt.Sprintf("%s:retrieveUserQuota", CodeAssistEndpoint)
	reqBody := map[string]string{"project": p.project}
	data, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var quota RetrieveUserQuotaResponse
	if err := json.NewDecoder(resp.Body).Decode(&quota); err != nil {
		return nil, err
	}

	var models []string
	for _, b := range quota.Buckets {
		if b.ModelID != "" {
			models = append(models, strings.TrimPrefix(b.ModelID, "models/"))
		}
	}

	return models, nil
}
