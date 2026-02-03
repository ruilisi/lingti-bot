package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/pltanton/lingti-bot/internal/router"
)

// Agent processes messages using AI providers and tools
type Agent struct {
	provider Provider
}

// Config holds agent configuration
type Config struct {
	Provider string // "claude" or "deepseek" (default: "claude")
	APIKey   string
	BaseURL  string // Custom API base URL (optional)
	Model    string // Model name (optional, uses provider default)
}

// New creates a new Agent with the specified provider
func New(cfg Config) (*Agent, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	provider, err := createProvider(cfg)
	if err != nil {
		return nil, err
	}

	return &Agent{
		provider: provider,
	}, nil
}

// createProvider creates the appropriate AI provider based on config
func createProvider(cfg Config) (Provider, error) {
	switch strings.ToLower(cfg.Provider) {
	case "deepseek":
		return NewDeepSeekProvider(DeepSeekConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		})
	case "kimi", "moonshot":
		return NewKimiProvider(KimiConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		})
	case "claude", "anthropic", "":
		return NewClaudeProvider(ClaudeConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		})
	default:
		return nil, fmt.Errorf("unknown provider: %s (supported: claude, deepseek, kimi)", cfg.Provider)
	}
}

// handleBuiltinCommand handles special commands without calling AI
func (a *Agent) handleBuiltinCommand(msg router.Message) (router.Response, bool) {
	text := strings.TrimSpace(strings.ToLower(msg.Text))

	switch text {
	case "/whoami", "whoami", "我是谁", "我的id":
		return router.Response{
			Text: fmt.Sprintf("用户信息:\n- 用户ID: %s\n- 用户名: %s\n- 平台: %s\n- 频道ID: %s",
				msg.UserID, msg.Username, msg.Platform, msg.ChannelID),
		}, true
	case "/help", "help", "帮助":
		return router.Response{
			Text: "可用命令:\n- /whoami - 查看你的用户信息\n- /help - 显示帮助\n\n你也可以直接用自然语言和我对话！",
		}, true
	}

	return router.Response{}, false
}

// HandleMessage processes a message and returns a response
func (a *Agent) HandleMessage(ctx context.Context, msg router.Message) (router.Response, error) {
	log.Printf("[Agent] Processing message from %s: %s (provider: %s)", msg.Username, msg.Text, a.provider.Name())

	// Handle built-in commands
	if resp, handled := a.handleBuiltinCommand(msg); handled {
		return resp, nil
	}

	// Build the tools list
	tools := a.buildToolsList()

	// Create initial messages
	messages := []Message{
		{
			Role:    "user",
			Content: msg.Text,
		},
	}

	// System prompt
	systemPrompt := `You are a helpful AI assistant integrated with Slack. You have access to various tools to help users:

- File operations: list, read, write, search, delete old files
- Calendar: list events, create events, search events
- System info: CPU, memory, disk usage
- Shell commands: execute commands (be careful!)
- Network: interfaces, connections, ping, DNS lookup
- Process management: list, info, kill

When users ask you to do something, use the appropriate tools. Be concise in your responses.
Always confirm before performing destructive actions (delete, kill process, etc.).`

	// Call AI provider
	resp, err := a.provider.Chat(ctx, ChatRequest{
		Messages:     messages,
		SystemPrompt: systemPrompt,
		Tools:        tools,
		MaxTokens:    4096,
	})
	if err != nil {
		return router.Response{}, fmt.Errorf("AI error: %w", err)
	}

	// Handle tool use if needed
	for resp.FinishReason == "tool_use" {
		// Process tool calls
		toolResults := a.processToolCalls(ctx, resp.ToolCalls)

		// Add assistant response with tool calls
		messages = append(messages, Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Add tool results
		for _, result := range toolResults {
			messages = append(messages, Message{
				Role:       "user",
				ToolResult: &result,
			})
		}

		// Continue the conversation
		resp, err = a.provider.Chat(ctx, ChatRequest{
			Messages:     messages,
			SystemPrompt: systemPrompt,
			Tools:        tools,
			MaxTokens:    4096,
		})
		if err != nil {
			return router.Response{}, fmt.Errorf("AI error: %w", err)
		}
	}

	return router.Response{Text: resp.Content}, nil
}

// buildToolsList creates the tools list for the AI provider
func (a *Agent) buildToolsList() []Tool {
	return []Tool{
		// File operations
		{
			Name:        "file_read",
			Description: "Read the contents of a file",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"path": map[string]string{"type": "string", "description": "Path to the file"}},
				"required":   []string{"path"},
			}),
		},
		{
			Name:        "file_list",
			Description: "List contents of a directory",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"path": map[string]string{"type": "string", "description": "Directory path"}},
			}),
		},
		{
			Name:        "file_list_old",
			Description: "List files not modified for specified days",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]string{"type": "string", "description": "Directory path"},
					"days": map[string]string{"type": "number", "description": "Minimum days since modification"},
				},
				"required": []string{"path"},
			}),
		},
		{
			Name:        "file_trash",
			Description: "Move files to Trash",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"files": map[string]any{"type": "array", "items": map[string]string{"type": "string"}, "description": "File paths to trash"},
				},
				"required": []string{"files"},
			}),
		},
		// Calendar
		{
			Name:        "calendar_today",
			Description: "Get today's calendar events",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "calendar_list_events",
			Description: "List upcoming calendar events",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"days": map[string]string{"type": "number", "description": "Days ahead"}},
			}),
		},
		{
			Name:        "calendar_create_event",
			Description: "Create a calendar event",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":      map[string]string{"type": "string", "description": "Event title"},
					"start_time": map[string]string{"type": "string", "description": "Start time (YYYY-MM-DD HH:MM)"},
					"duration":   map[string]string{"type": "number", "description": "Duration in minutes"},
					"calendar":   map[string]string{"type": "string", "description": "Calendar name"},
				},
				"required": []string{"title", "start_time"},
			}),
		},
		// System
		{
			Name:        "system_info",
			Description: "Get system information (CPU, memory, OS)",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "shell_execute",
			Description: "Execute a shell command",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]string{"type": "string", "description": "Command to execute"},
					"timeout": map[string]string{"type": "number", "description": "Timeout in seconds"},
				},
				"required": []string{"command"},
			}),
		},
		{
			Name:        "process_list",
			Description: "List running processes",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"filter": map[string]string{"type": "string", "description": "Filter by name"}},
			}),
		},
	}
}

// processToolCalls executes tool calls and returns results
func (a *Agent) processToolCalls(ctx context.Context, toolCalls []ToolCall) []ToolResult {
	results := make([]ToolResult, 0, len(toolCalls))

	for _, tc := range toolCalls {
		result := a.executeTool(ctx, tc.Name, tc.Input)
		results = append(results, ToolResult{
			ToolCallID: tc.ID,
			Content:    result,
			IsError:    false,
		})
	}

	return results
}

// executeTool runs a tool and returns the result
func (a *Agent) executeTool(ctx context.Context, name string, input json.RawMessage) string {
	log.Printf("[Agent] Executing tool: %s", name)

	// Parse input arguments
	var args map[string]any
	if err := json.Unmarshal(input, &args); err != nil {
		return fmt.Sprintf("Error parsing arguments: %v", err)
	}

	// Call tools directly
	return callToolDirect(ctx, name, args)
}

// callToolDirect calls a tool directly
func callToolDirect(ctx context.Context, name string, args map[string]any) string {
	switch name {
	case "system_info":
		return executeSystemInfo(ctx)
	case "calendar_today":
		return executeCalendarToday(ctx)
	case "calendar_list_events":
		days := 7
		if d, ok := args["days"].(float64); ok {
			days = int(d)
		}
		return executeCalendarListEvents(ctx, days)
	case "file_list":
		path := "."
		if p, ok := args["path"].(string); ok {
			path = p
		}
		return executeFileList(ctx, path)
	case "file_list_old":
		path := "."
		days := 30
		if p, ok := args["path"].(string); ok {
			path = p
		}
		if d, ok := args["days"].(float64); ok {
			days = int(d)
		}
		return executeFileListOld(ctx, path, days)
	case "shell_execute":
		cmd := ""
		if c, ok := args["command"].(string); ok {
			cmd = c
		}
		return executeShell(ctx, cmd)
	default:
		return fmt.Sprintf("Tool '%s' not implemented in direct mode", name)
	}
}

func jsonSchema(schema map[string]any) json.RawMessage {
	data, _ := json.Marshal(schema)
	return data
}
