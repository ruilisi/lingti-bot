package agent

import (
	"context"
	"encoding/json"
)

// Provider defines the interface for AI backends
type Provider interface {
	// Chat sends messages and returns a response, handling tool calls internally
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)

	// Name returns the provider name (e.g., "claude", "deepseek")
	Name() string
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Messages     []Message
	SystemPrompt string
	Tools        []Tool
	MaxTokens    int
	// ForceToolUse, when true, sets tool_choice="required" so the model must call a tool.
	// Use this during multi-step browser tasks to prevent premature stop responses.
	ForceToolUse bool
	// ThinkingBudget, when > 0, enables Claude extended thinking with the given token budget.
	// Only used by the Claude provider; other providers ignore this.
	ThinkingBudget int
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	Content          string
	ReasoningContent string // For thinking models (e.g. kimi-k2.5)
	ToolCalls        []ToolCall
	// FinishReason indicates why the model stopped: "stop", "tool_use", etc.
	FinishReason string
}

// Message represents a chat message
type Message struct {
	Role             string // "user", "assistant", "tool"
	Content          string
	ReasoningContent string           // For thinking models (e.g. kimi-k2.5)
	ToolCalls        []ToolCall       // For assistant messages with tool calls
	ToolResult *ToolResult      // For tool result messages
}

// ToolCall represents a tool invocation by the model
type ToolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolCallID string
	Content    string
	IsError    bool
}

// Tool defines a tool that can be used by the model
type Tool struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}
