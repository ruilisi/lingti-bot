package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

const (
	kimiDefaultBaseURL = "https://api.moonshot.cn/v1"
	kimiDefaultModel   = "moonshot-v1-8k"
)

// KimiProvider implements the Provider interface for Kimi (Moonshot AI)
type KimiProvider struct {
	client *openai.Client
	model  string
}

// KimiConfig holds Kimi provider configuration
type KimiConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// NewKimiProvider creates a new Kimi provider
func NewKimiProvider(cfg KimiConfig) (*KimiProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if cfg.Model == "" {
		cfg.Model = kimiDefaultModel
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = kimiDefaultBaseURL
	}

	config := openai.DefaultConfig(cfg.APIKey)
	config.BaseURL = baseURL

	return &KimiProvider{
		client: openai.NewClientWithConfig(config),
		model:  cfg.Model,
	}, nil
}

// Name returns the provider name
func (p *KimiProvider) Name() string {
	return "kimi"
}

// Chat sends messages and returns a response
func (p *KimiProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	// Convert messages to OpenAI format
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+1)

	// Add system message
	if req.SystemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		})
	}

	// Add conversation messages
	for _, msg := range req.Messages {
		messages = append(messages, p.toOpenAIMessage(msg))
	}

	// Convert tools to OpenAI format
	tools := make([]openai.Tool, 0, len(req.Tools))
	for _, tool := range req.Tools {
		var params map[string]any
		if err := json.Unmarshal(tool.InputSchema, &params); err != nil {
			params = map[string]any{"type": "object"}
		}
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
			},
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	// Build request
	chatReq := openai.ChatCompletionRequest{
		Model:     p.model,
		Messages:  messages,
		MaxTokens: maxTokens,
	}
	if len(tools) > 0 {
		chatReq.Tools = tools
	}

	// Call Kimi API
	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("kimi API error: %w", err)
	}

	return p.fromOpenAIResponse(resp), nil
}

// toOpenAIMessage converts a generic Message to OpenAI format
func (p *KimiProvider) toOpenAIMessage(msg Message) openai.ChatCompletionMessage {
	switch msg.Role {
	case "user":
		if msg.ToolResult != nil {
			content := msg.ToolResult.Content
			if content == "" {
				content = "(empty)"
			}
			return openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    content,
				ToolCallID: msg.ToolResult.ToolCallID,
			}
		}
		return openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: msg.Content,
		}

	case "assistant":
		m := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: msg.Content,
		}
		if len(msg.ToolCalls) > 0 {
			m.ToolCalls = make([]openai.ToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				m.ToolCalls[i] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Name,
						Arguments: string(tc.Input),
					},
				}
			}
		}
		return m

	case "tool":
		content := msg.Content
		if content == "" && msg.ToolResult != nil {
			content = msg.ToolResult.Content
		}
		if content == "" {
			content = "(empty)"
		}
		return openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    content,
			ToolCallID: msg.ToolResult.ToolCallID,
		}

	default:
		return openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: msg.Content,
		}
	}
}

// fromOpenAIResponse converts OpenAI response to generic format
func (p *KimiProvider) fromOpenAIResponse(resp openai.ChatCompletionResponse) ChatResponse {
	if len(resp.Choices) == 0 {
		return ChatResponse{}
	}

	choice := resp.Choices[0]
	var toolCalls []ToolCall

	for _, tc := range choice.Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: json.RawMessage(tc.Function.Arguments),
		})
	}

	finishReason := "stop"
	if choice.FinishReason == openai.FinishReasonToolCalls {
		finishReason = "tool_use"
	}

	return ChatResponse{
		Content:      choice.Message.Content,
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
	}
}
