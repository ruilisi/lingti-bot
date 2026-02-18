package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// OpenAICompatProvider implements the Provider interface for any OpenAI-compatible API.
// This covers: MiniMax, Doubao, Zhipu/GLM, OpenAI/GPT, Gemini, Yi, StepFun, SiliconFlow, Grok, etc.
type OpenAICompatProvider struct {
	client       *openai.Client
	model        string
	providerName string
}

// OpenAICompatConfig holds configuration for an OpenAI-compatible provider
type OpenAICompatConfig struct {
	ProviderName string // Display name (e.g., "minimax", "openai")
	APIKey       string
	BaseURL      string
	Model        string
	DefaultURL   string // Default base URL if not specified
	DefaultModel string // Default model if not specified
}

// NewOpenAICompatProvider creates a new OpenAI-compatible provider
func NewOpenAICompatProvider(cfg OpenAICompatConfig) (*OpenAICompatProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if cfg.Model == "" {
		cfg.Model = cfg.DefaultModel
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = cfg.DefaultURL
	}

	config := openai.DefaultConfig(cfg.APIKey)
	config.BaseURL = baseURL

	return &OpenAICompatProvider{
		client:       openai.NewClientWithConfig(config),
		model:        cfg.Model,
		providerName: cfg.ProviderName,
	}, nil
}

// Name returns the provider name
func (p *OpenAICompatProvider) Name() string {
	return p.providerName
}

// Chat sends messages and returns a response
func (p *OpenAICompatProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+1)

	if req.SystemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		})
	}

	for _, msg := range req.Messages {
		messages = append(messages, p.toOpenAIMessage(msg))
	}

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

	chatReq := openai.ChatCompletionRequest{
		Model:     p.model,
		Messages:  messages,
		MaxTokens: maxTokens,
	}
	if len(tools) > 0 {
		chatReq.Tools = tools
	}

	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("%s API error: %w", p.providerName, err)
	}

	return p.fromOpenAIResponse(resp), nil
}

func (p *OpenAICompatProvider) toOpenAIMessage(msg Message) openai.ChatCompletionMessage {
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

func (p *OpenAICompatProvider) fromOpenAIResponse(resp openai.ChatCompletionResponse) ChatResponse {
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
