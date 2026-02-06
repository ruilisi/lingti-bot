package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	geminiDefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"
	geminiDefaultModel   = "gemini-3-flash-preview"
)

// GeminiProvider implements the Provider interface for Google Gemini
type GeminiProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// GeminiConfig holds Gemini provider configuration
type GeminiConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(cfg GeminiConfig) (*GeminiProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if cfg.Model == "" {
		cfg.Model = geminiDefaultModel
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = geminiDefaultBaseURL
	}

	return &GeminiProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		model:   cfg.Model,
		client:  &http.Client{},
	}, nil
}

// Name returns the provider name
func (p *GeminiProvider) Name() string {
	return "gemini"
}

// Gemini API request/response structures
type geminiRequest struct {
	Contents          []geminiContent   `json:"contents"`
	SystemInstruction *geminiContent    `json:"systemInstruction,omitempty"`
	Tools             []geminiTool      `json:"tools,omitempty"`
	GenerationConfig  *generationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
	ThoughtSignature string                  `json:"thoughtSignature,omitempty"`
}

type geminiFunctionCall struct {
	Name    string         `json:"name"`
	Args    map[string]any `json:"args"`
	ID      string         `json:"id,omitempty"`
	Thought bool           `json:"thought,omitempty"` // Keep for compatibility if boolean
}

type geminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations"`
}

type geminiFunctionDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type generationConfig struct {
	MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

// Chat sends messages and returns a response
func (p *GeminiProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	// Build Gemini request
	gemReq := geminiRequest{
		Contents: make([]geminiContent, 0),
		GenerationConfig: &generationConfig{
			MaxOutputTokens: req.MaxTokens,
		},
	}

	// Add system instruction
	if req.SystemPrompt != "" {
		gemReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.SystemPrompt}},
		}
	}

	// Convert messages to Gemini format
	for _, msg := range req.Messages {
		gemReq.Contents = append(gemReq.Contents, p.toGeminiContent(msg))
	}

	// Convert tools to Gemini format
	if len(req.Tools) > 0 {
		declarations := make([]geminiFunctionDeclaration, 0, len(req.Tools))
		for _, tool := range req.Tools {
			var params map[string]any
			if err := json.Unmarshal(tool.InputSchema, &params); err != nil {
				params = map[string]any{"type": "object"}
			}
			declarations = append(declarations, geminiFunctionDeclaration{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
			})
		}
		gemReq.Tools = []geminiTool{{FunctionDeclarations: declarations}}
	}

	// Build URL
	url := fmt.Sprintf("%s/models/%s:generateContent", p.baseURL, p.model)

	// Marshal request
	reqBody, err := json.Marshal(gemReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", p.apiKey)

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return ChatResponse{}, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var gemResp geminiResponse
	if err := json.Unmarshal(respBody, &gemResp); err != nil {
		return ChatResponse{}, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.fromGeminiResponse(gemResp), nil
}

// toGeminiContent converts a generic Message to Gemini format
func (p *GeminiProvider) toGeminiContent(msg Message) geminiContent {
	content := geminiContent{
		Parts: make([]geminiPart, 0),
	}

	switch msg.Role {
	case "user":
		content.Role = "user"
		if msg.ToolResult != nil {
			// Tool result message
			content.Parts = append(content.Parts, geminiPart{
				FunctionResponse: &geminiFunctionResponse{
					Name: msg.ToolResult.ToolCallID, // Use as function name
					Response: map[string]any{
						"result": msg.ToolResult.Content,
					},
				},
			})
		} else {
			content.Parts = append(content.Parts, geminiPart{Text: msg.Content})
		}

	case "assistant":
		content.Role = "model"
		if msg.Content != "" {
			content.Parts = append(content.Parts, geminiPart{Text: msg.Content})
		}
		// Add function calls
		for _, tc := range msg.ToolCalls {
			var args map[string]any
			json.Unmarshal(tc.Input, &args)

			fc := &geminiFunctionCall{
				Name: tc.Name,
				Args: args,
			}
			var thoughtSignature string
			if tc.Meta != nil {
				if id, ok := tc.Meta["id"].(string); ok {
					fc.ID = id
				}
				if thought, ok := tc.Meta["thought"].(bool); ok {
					fc.Thought = thought
				}
				if ts, ok := tc.Meta["thought_signature"].(string); ok {
					thoughtSignature = ts
				}
			}

			content.Parts = append(content.Parts, geminiPart{
				FunctionCall:     fc,
				ThoughtSignature: thoughtSignature,
			})
		}

	default:
		content.Role = "user"
		content.Parts = append(content.Parts, geminiPart{Text: msg.Content})
	}

	return content
}

// fromGeminiResponse converts Gemini response to generic format
func (p *GeminiProvider) fromGeminiResponse(resp geminiResponse) ChatResponse {
	if len(resp.Candidates) == 0 {
		return ChatResponse{}
	}

	candidate := resp.Candidates[0]
	var content string
	var toolCalls []ToolCall

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			content += part.Text
		}
		if part.FunctionCall != nil {
			// Convert function call to tool call
			argsJSON, _ := json.Marshal(part.FunctionCall.Args)

			// Capture duplicate metadata
			meta := make(map[string]any)
			if part.FunctionCall.ID != "" {
				meta["id"] = part.FunctionCall.ID
			}
			if part.FunctionCall.Thought {
				meta["thought"] = part.FunctionCall.Thought
			}
			if part.ThoughtSignature != "" {
				meta["thought_signature"] = part.ThoughtSignature
			}

			// Determine ID to use (Gemini requires name match for results, but might send ID now)
			id := part.FunctionCall.Name
			if part.FunctionCall.ID != "" {
				id = part.FunctionCall.ID
			}

			toolCalls = append(toolCalls, ToolCall{
				ID:    id,
				Name:  part.FunctionCall.Name,
				Input: argsJSON,
				Meta:  meta,
			})
		}
	}

	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_use"
	}

	return ChatResponse{
		Content:      content,
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
	}
}
