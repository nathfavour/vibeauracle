package model

// Part represents a part of a message.
type Part struct {
	Text string `json:"text,omitempty"`
	// Add other fields like FunctionCall, FunctionResponse if needed
}

// Content represents a message in the conversation.
type Content struct {
	Role  string `json:"role"`
	Parts []Part `json:"parts"`
}

// GenerationConfig contains configuration for content generation.
type GenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	TopK            int     `json:"topK,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// VertexGenerateContentRequest is the inner request structure.
type VertexGenerateContentRequest struct {
	Contents         []Content         `json:"contents"`
	SystemInstruction *Content         `json:"systemInstruction,omitempty"`
	GenerationConfig  *GenerationConfig `json:"generationConfig,omitempty"`
}

// CAGenerateContentRequest is the outer request structure for Code Assist.
type CAGenerateContentRequest struct {
	Model        string                       `json:"model"`
	Project      string                       `json:"project,omitempty"`
	UserPromptID string                       `json:"user_prompt_id,omitempty"`
	Request      VertexGenerateContentRequest `json:"request"`
}

// Response structures

type UsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type Candidate struct {
	Content Content `json:"content"`
	FinishReason string `json:"finishReason"`
}

type VertexGenerateContentResponse struct {
	Candidates    []Candidate   `json:"candidates"`
	UsageMetadata UsageMetadata `json:"usageMetadata"`
}

type CAGenerateContentResponse struct {
	Response VertexGenerateContentResponse `json:"response"`
	TraceID  string                        `json:"traceId"`
}

// Quota structures

type BucketInfo struct {
	ModelID string `json:"modelId"`
}

type RetrieveUserQuotaResponse struct {
	Buckets []BucketInfo `json:"buckets"`
}
