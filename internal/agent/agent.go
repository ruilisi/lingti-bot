package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/liushuangls/go-anthropic/v2"
	"github.com/pltanton/lingti-bot/internal/router"
)

// Agent processes messages using Claude and MCP tools
type Agent struct {
	client *anthropic.Client
	model  string
}

// Config holds agent configuration
type Config struct {
	APIKey  string
	BaseURL string // Custom API base URL (optional)
	Model   string // Default: claude-sonnet-4-20250514
}

// New creates a new Agent
func New(cfg Config) (*Agent, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if cfg.Model == "" {
		cfg.Model = "claude-sonnet-4-20250514"
	}

	// Create client with optional custom base URL
	var client *anthropic.Client
	if cfg.BaseURL != "" {
		client = anthropic.NewClient(cfg.APIKey, anthropic.WithBaseURL(cfg.BaseURL))
	} else {
		client = anthropic.NewClient(cfg.APIKey)
	}

	return &Agent{
		client: client,
		model:  cfg.Model,
	}, nil
}

// HandleMessage processes a message and returns a response
func (a *Agent) HandleMessage(ctx context.Context, msg router.Message) (router.Response, error) {
	log.Printf("[Agent] Processing message from %s: %s", msg.Username, msg.Text)

	// Build the tools list from MCP server
	tools := a.buildToolsList()

	// Create the message request
	messages := []anthropic.Message{
		{
			Role: anthropic.RoleUser,
			Content: []anthropic.MessageContent{
				anthropic.NewTextMessageContent(msg.Text),
			},
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

	// Call Claude API with tools
	resp, err := a.client.CreateMessages(ctx, anthropic.MessagesRequest{
		Model:     anthropic.Model(a.model),
		MaxTokens: 4096,
		System:    systemPrompt,
		Messages:  messages,
		Tools:     tools,
	})

	if err != nil {
		return router.Response{}, fmt.Errorf("API error: %w", err)
	}

	// Handle tool use if needed
	for resp.StopReason == anthropic.MessagesStopReasonToolUse {
		// Process tool calls
		toolResults := a.processToolCalls(ctx, resp.Content)

		// Add assistant response and tool results to messages
		messages = append(messages, anthropic.Message{
			Role:    anthropic.RoleAssistant,
			Content: resp.Content,
		})
		messages = append(messages, anthropic.Message{
			Role:    anthropic.RoleUser,
			Content: toolResults,
		})

		// Continue the conversation
		resp, err = a.client.CreateMessages(ctx, anthropic.MessagesRequest{
			Model:     anthropic.Model(a.model),
			MaxTokens: 4096,
			System:    systemPrompt,
			Messages:  messages,
			Tools:     tools,
		})

		if err != nil {
			return router.Response{}, fmt.Errorf("API error: %w", err)
		}
	}

	// Extract text response
	var responseText string
	for _, content := range resp.Content {
		if content.Type == anthropic.MessagesContentTypeText && content.Text != nil {
			responseText += *content.Text
		}
	}

	return router.Response{Text: responseText}, nil
}

// buildToolsList creates the tools list for Claude
func (a *Agent) buildToolsList() []anthropic.ToolDefinition {
	return []anthropic.ToolDefinition{
		// File operations
		{
			Name:        "file_read",
			Description: "Read the contents of a file",
			InputSchema: jsonSchema(map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{"path": map[string]string{"type": "string", "description": "Path to the file"}},
				"required":   []string{"path"},
			}),
		},
		{
			Name:        "file_list",
			Description: "List contents of a directory",
			InputSchema: jsonSchema(map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{"path": map[string]string{"type": "string", "description": "Directory path"}},
			}),
		},
		{
			Name:        "file_list_old",
			Description: "List files not modified for specified days",
			InputSchema: jsonSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]string{"type": "string", "description": "Directory path"},
					"days": map[string]string{"type": "number", "description": "Minimum days since modification"},
				},
				"required": []string{"path"},
			}),
		},
		{
			Name:        "file_trash",
			Description: "Move files to Trash",
			InputSchema: jsonSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"files": map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}, "description": "File paths to trash"},
				},
				"required": []string{"files"},
			}),
		},
		// Calendar
		{
			Name:        "calendar_today",
			Description: "Get today's calendar events",
			InputSchema: jsonSchema(map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}),
		},
		{
			Name:        "calendar_list_events",
			Description: "List upcoming calendar events",
			InputSchema: jsonSchema(map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{"days": map[string]string{"type": "number", "description": "Days ahead"}},
			}),
		},
		{
			Name:        "calendar_create_event",
			Description: "Create a calendar event",
			InputSchema: jsonSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
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
			InputSchema: jsonSchema(map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}),
		},
		{
			Name:        "shell_execute",
			Description: "Execute a shell command",
			InputSchema: jsonSchema(map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]string{"type": "string", "description": "Command to execute"},
					"timeout": map[string]string{"type": "number", "description": "Timeout in seconds"},
				},
				"required": []string{"command"},
			}),
		},
		{
			Name:        "process_list",
			Description: "List running processes",
			InputSchema: jsonSchema(map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{"filter": map[string]string{"type": "string", "description": "Filter by name"}},
			}),
		},
	}
}

// processToolCalls executes tool calls and returns results
func (a *Agent) processToolCalls(ctx context.Context, content []anthropic.MessageContent) []anthropic.MessageContent {
	var results []anthropic.MessageContent

	for _, c := range content {
		if c.Type == anthropic.MessagesContentTypeToolUse {
			toolName := c.Name
			toolID := c.ID

			// Execute the tool via MCP
			result := a.executeTool(ctx, toolName, c.Input)

			results = append(results, anthropic.NewToolResultMessageContent(toolID, result, false))
		}
	}

	return results
}

// executeTool runs a tool and returns the result
func (a *Agent) executeTool(ctx context.Context, name string, input json.RawMessage) string {
	log.Printf("[Agent] Executing tool: %s", name)

	// Parse input arguments
	var args map[string]interface{}
	if err := json.Unmarshal(input, &args); err != nil {
		return fmt.Sprintf("Error parsing arguments: %v", err)
	}

	// Call tools directly
	result := callToolDirect(ctx, name, args)

	return result
}

// callToolDirect calls a tool directly (simplified implementation)
func callToolDirect(ctx context.Context, name string, args map[string]interface{}) string {
	// Import and call tools directly
	// This is a simplified approach - a full implementation would use MCP protocol

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

func jsonSchema(schema map[string]interface{}) json.RawMessage {
	data, _ := json.Marshal(schema)
	return data
}
