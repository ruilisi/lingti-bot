package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/pltanton/lingti-bot/internal/logger"
	"github.com/pltanton/lingti-bot/internal/router"
)

// Agent processes messages using AI providers and tools
type Agent struct {
	provider Provider
	memory   *ConversationMemory
	sessions *SessionStore
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
		memory:   NewMemory(50, 60*time.Minute), // Keep 50 messages, 60 min TTL
		sessions: NewSessionStore(),
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
	text := strings.TrimSpace(msg.Text)
	textLower := strings.ToLower(text)
	convKey := ConversationKey(msg.Platform, msg.ChannelID, msg.UserID)

	// Exact match commands
	switch textLower {
	case "/whoami", "whoami", "我是谁", "我的id":
		return router.Response{
			Text: fmt.Sprintf("用户信息:\n- 用户ID: %s\n- 用户名: %s\n- 平台: %s\n- 频道ID: %s",
				msg.UserID, msg.Username, msg.Platform, msg.ChannelID),
		}, true

	case "/help", "help", "帮助", "/commands":
		return router.Response{
			Text: `可用命令:

会话管理:
  /new, /reset    开始新对话，清除历史
  /status         查看当前会话状态

思考模式:
  /think off      关闭深度思考
  /think low      简单思考
  /think medium   中等思考（默认）
  /think high     深度思考

显示设置:
  /verbose on     显示详细执行过程
  /verbose off    隐藏执行过程

其他:
  /whoami         查看用户信息
  /model          查看当前模型
  /tools          列出可用工具
  /help           显示帮助

直接用自然语言和我对话即可！`,
		}, true

	case "/new", "/reset", "/clear", "新对话", "清除历史":
		a.memory.Clear(convKey)
		a.sessions.Clear(convKey)
		return router.Response{
			Text: "已开始新对话，历史记录和会话设置已重置。",
		}, true

	case "/status", "状态":
		history := a.memory.GetHistory(convKey)
		settings := a.sessions.Get(convKey)
		return router.Response{
			Text: fmt.Sprintf(`会话状态:
- 平台: %s
- 用户: %s
- 历史消息: %d 条
- 思考模式: %s
- 详细模式: %v
- AI 模型: %s`,
				msg.Platform, msg.Username, len(history),
				settings.ThinkingLevel, settings.Verbose, a.provider.Name()),
		}, true

	case "/model", "模型":
		return router.Response{
			Text: fmt.Sprintf("当前模型: %s", a.provider.Name()),
		}, true

	case "/tools", "工具", "工具列表":
		return router.Response{
			Text: `可用工具:

📁 文件操作:
  file_send, file_list, file_read, file_trash, file_list_old

📅 日历 (macOS):
  calendar_today, calendar_list_events, calendar_create_event
  calendar_search, calendar_delete

✅ 提醒事项 (macOS):
  reminders_list, reminders_add, reminders_complete, reminders_delete

📝 备忘录 (macOS):
  notes_list, notes_read, notes_create, notes_search

🌤 天气:
  weather_current, weather_forecast

🌐 网页:
  web_search, web_fetch, open_url

📋 剪贴板:
  clipboard_read, clipboard_write

🔔 通知:
  notification_send

📸 截图:
  screenshot

🎵 音乐 (macOS):
  music_play, music_pause, music_next, music_previous
  music_now_playing, music_volume, music_search

💻 系统:
  system_info, shell_execute, process_list`,
		}, true

	case "/verbose on", "详细模式开":
		a.sessions.SetVerbose(convKey, true)
		return router.Response{Text: "详细模式已开启"}, true

	case "/verbose off", "详细模式关":
		a.sessions.SetVerbose(convKey, false)
		return router.Response{Text: "详细模式已关闭"}, true

	case "/think off", "思考关":
		a.sessions.SetThinkingLevel(convKey, ThinkOff)
		return router.Response{Text: "思考模式已关闭"}, true

	case "/think low", "简单思考":
		a.sessions.SetThinkingLevel(convKey, ThinkLow)
		return router.Response{Text: "思考模式: 简单"}, true

	case "/think medium", "中等思考":
		a.sessions.SetThinkingLevel(convKey, ThinkMedium)
		return router.Response{Text: "思考模式: 中等"}, true

	case "/think high", "深度思考":
		a.sessions.SetThinkingLevel(convKey, ThinkHigh)
		return router.Response{Text: "思考模式: 深度"}, true
	}

	return router.Response{}, false
}

// HandleMessage processes a message and returns a response
func (a *Agent) HandleMessage(ctx context.Context, msg router.Message) (router.Response, error) {
	logger.Info("[Agent] Processing message from %s: %s (provider: %s)", msg.Username, msg.Text, a.provider.Name())

	// Handle built-in commands
	if resp, handled := a.handleBuiltinCommand(msg); handled {
		return resp, nil
	}

	// Generate conversation key
	convKey := ConversationKey(msg.Platform, msg.ChannelID, msg.UserID)

	// Build the tools list
	tools := a.buildToolsList()

	// Get conversation history
	history := a.memory.GetHistory(convKey)
	logger.Debug("[Agent] Conversation key: %s, history messages: %d", convKey, len(history))

	// Create messages with history
	messages := make([]Message, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, Message{
		Role:    "user",
		Content: msg.Text,
	})

	// Get system info for context
	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		homeDir = "~"
	}

	// Get session settings
	settings := a.sessions.Get(convKey)
	thinkingPrompt := ThinkingPrompt(settings.ThinkingLevel)

	// System prompt with actual paths
	systemPrompt := fmt.Sprintf(`You are 灵缇 (Lingti), a helpful AI assistant running on the user's computer.

## System Environment
- Operating System: %s
- Architecture: %s
- Home Directory: %s
- Desktop: %s/Desktop
- Documents: %s/Documents
- Downloads: %s/Downloads
- User: %s

## Available Tools

### File Operations
- file_send: Send/transfer a file to the user via messaging platform
- file_list: List directory contents (use ~/Desktop for desktop)
- file_read: Read file contents
- file_trash: Move files to trash (for delete operations)
- file_list_old: Find old files not modified for N days

### Calendar (macOS)
- calendar_today: Get today's events
- calendar_list_events: List upcoming events
- calendar_create_event: Create new event
- calendar_search: Search events
- calendar_delete: Delete event

### Reminders (macOS)
- reminders_list: List pending reminders
- reminders_add: Add new reminder
- reminders_complete: Mark as complete
- reminders_delete: Delete reminder

### Notes (macOS)
- notes_list: List notes
- notes_read: Read note content
- notes_create: Create new note
- notes_search: Search notes

### Weather
- weather_current: Current weather
- weather_forecast: Weather forecast

### Web
- web_search: Search the web (DuckDuckGo)
- web_fetch: Fetch URL content
- open_url: Open URL in browser

### Clipboard
- clipboard_read: Read clipboard
- clipboard_write: Write to clipboard

### System
- system_info: System information
- shell_execute: Execute shell command
- process_list: List processes
- notification_send: Send notification
- screenshot: Capture screen

### Music (macOS)
- music_play/pause/next/previous: Playback control
- music_now_playing: Current track info
- music_volume: Set volume
- music_search: Search and play

### Browser Automation (snapshot-then-act pattern)
- browser_start: Start new browser or connect to existing Chrome via cdp_url (e.g. "127.0.0.1:9222")
- browser_navigate: Navigate to a URL (auto-connects to Chrome on port 9222 if available, otherwise launches new)
- browser_snapshot: Capture accessibility tree with numbered refs
- browser_click: Click an element by ref number
- browser_type: Type text into element by ref number (optional submit with Enter)
- browser_press: Press keyboard key (Enter, Tab, Escape, etc.)
- browser_execute_js: Run JavaScript on the page (dismiss modals, extract data, etc.)
- browser_click_all: Click ALL elements matching a CSS selector with delay (batch like/follow)
- browser_screenshot: Take page screenshot
- browser_tabs: List all open tabs
- browser_tab_open: Open new tab
- browser_tab_close: Close a tab
- browser_status: Check browser state
- browser_stop: Close browser (or disconnect from external Chrome)

## Browser Automation Rules
You MUST follow the **snapshot-then-act** pattern for ALL browser interactions:
1. **Navigate** to the target website's homepage using browser_navigate
2. **Snapshot** the page using browser_snapshot to discover UI elements and their ref numbers
3. **Interact** with elements step by step using browser_click / browser_type / browser_press
4. **Re-snapshot** after any page change (click, navigation, form submit) to get updated refs

**CRITICAL: NEVER construct or guess URLs to skip UI interaction steps.**
- BAD: Directly navigating to https://www.xiaohongshu.com/search/关键词
- GOOD: Navigate to https://www.xiaohongshu.com → snapshot → find search box → type keyword → submit

Always simulate real user behavior: navigate to the base URL first, then use the page's UI elements (search boxes, buttons, menus) to accomplish the task step by step. Refs are invalidated after page changes — always re-snapshot.

**Handling modals/overlays:** If an element is blocked by a modal or overlay (error message mentions "element covered by"), use browser_execute_js to dismiss it. Example scripts:
- document.querySelector('.modal-overlay').remove()
- document.querySelector('.dialog-close-btn').click()
Then re-snapshot and continue.

## Important Rules
1. **ALWAYS use tools** - Never tell users to do things manually
2. **Be action-oriented** - Execute tasks, don't just describe them
3. **Use correct paths** - 桌面=~/Desktop, 下载=~/Downloads, 文档=~/Documents
4. **Full permission** - You have full permission to execute all tools
5. **Be concise** - Short, helpful responses
6. **NEVER claim success without tool execution** - If user asks to create/add/delete something, you MUST call the corresponding tool. Never say "已创建/已添加/已删除" unless you actually called the tool and it succeeded.
7. **Date format for calendar** - When creating calendar events, use YYYY-MM-DD HH:MM format. Convert relative dates (明天/下周一) to absolute dates based on today's date.

Current date: %s%s`, time.Now().Format("2006-01-02"), runtime.GOOS, runtime.GOARCH, homeDir, homeDir, homeDir, homeDir, msg.Username, thinkingPrompt)

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
	var pendingFiles []router.FileAttachment
	for resp.FinishReason == "tool_use" {
		// Process tool calls
		toolResults, files := a.processToolCalls(ctx, resp.ToolCalls)
		pendingFiles = append(pendingFiles, files...)

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

	// Save conversation to memory
	a.memory.AddExchange(convKey,
		Message{Role: "user", Content: msg.Text},
		Message{Role: "assistant", Content: resp.Content},
	)

	// Log response at verbose level
	logger.Verbose("[Agent] Response: %s", resp.Content)

	return router.Response{Text: resp.Content, Files: pendingFiles}, nil
}

// buildToolsList creates the tools list for the AI provider
func (a *Agent) buildToolsList() []Tool {
	return []Tool{
		// === FILE OPERATIONS ===
		{
			Name:        "file_send",
			Description: "Send a file to the user via the messaging platform. Use this when the user asks you to send/transfer/share a file. Use ~ for home directory.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":       map[string]string{"type": "string", "description": "Path to the file (use ~ for home, e.g., ~/Desktop/report.pdf)"},
					"media_type": map[string]string{"type": "string", "description": "Media type: file, image, voice, or video (default: file)"},
				},
				"required": []string{"path"},
			}),
		},
		{
			Name:        "file_read",
			Description: "Read the contents of a file. Use ~ for home directory.",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"path": map[string]string{"type": "string", "description": "Path to the file (use ~ for home, e.g., ~/Desktop/file.txt)"}},
				"required":   []string{"path"},
			}),
		},
		{
			Name:        "file_list",
			Description: "List contents of a directory. Use ~/Desktop for desktop, ~/Downloads for downloads, etc.",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"path": map[string]string{"type": "string", "description": "Directory path (use ~ for home, e.g., ~/Desktop)"}},
			}),
		},
		{
			Name:        "file_list_old",
			Description: "List files not modified for specified days. Use ~/Desktop for desktop, etc.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]string{"type": "string", "description": "Directory path (use ~ for home, e.g., ~/Desktop)"},
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

		// === CALENDAR ===
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
				"properties": map[string]any{"days": map[string]string{"type": "number", "description": "Days ahead (default 7)"}},
			}),
		},
		{
			Name:        "calendar_create_event",
			Description: "Create a new calendar event",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":      map[string]string{"type": "string", "description": "Event title"},
					"start_time": map[string]string{"type": "string", "description": "Start time (YYYY-MM-DD HH:MM)"},
					"duration":   map[string]string{"type": "number", "description": "Duration in minutes (default 60)"},
					"calendar":   map[string]string{"type": "string", "description": "Calendar name (optional)"},
					"location":   map[string]string{"type": "string", "description": "Event location (optional)"},
					"notes":      map[string]string{"type": "string", "description": "Event notes (optional)"},
				},
				"required": []string{"title", "start_time"},
			}),
		},
		{
			Name:        "calendar_search",
			Description: "Search calendar events by keyword",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"keyword": map[string]string{"type": "string", "description": "Search keyword"},
					"days":    map[string]string{"type": "number", "description": "Days to search (default 30)"},
				},
				"required": []string{"keyword"},
			}),
		},
		{
			Name:        "calendar_delete",
			Description: "Delete a calendar event by title",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":    map[string]string{"type": "string", "description": "Event title to delete"},
					"calendar": map[string]string{"type": "string", "description": "Calendar name (optional)"},
					"date":     map[string]string{"type": "string", "description": "Date (YYYY-MM-DD) to narrow search (optional)"},
				},
				"required": []string{"title"},
			}),
		},

		// === REMINDERS ===
		{
			Name:        "reminders_list",
			Description: "List all pending reminders",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "reminders_add",
			Description: "Create a new reminder",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]string{"type": "string", "description": "Reminder title"},
					"list":  map[string]string{"type": "string", "description": "Reminder list name (default: Reminders)"},
					"due":   map[string]string{"type": "string", "description": "Due date (YYYY-MM-DD or YYYY-MM-DD HH:MM)"},
					"notes": map[string]string{"type": "string", "description": "Additional notes"},
				},
				"required": []string{"title"},
			}),
		},
		{
			Name:        "reminders_complete",
			Description: "Mark a reminder as complete",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"title": map[string]string{"type": "string", "description": "Reminder title"}},
				"required":   []string{"title"},
			}),
		},
		{
			Name:        "reminders_delete",
			Description: "Delete a reminder",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"title": map[string]string{"type": "string", "description": "Reminder title"}},
				"required":   []string{"title"},
			}),
		},

		// === NOTES ===
		{
			Name:        "notes_list",
			Description: "List notes in a folder",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"folder": map[string]string{"type": "string", "description": "Folder name (default: Notes)"},
					"limit":  map[string]string{"type": "number", "description": "Max notes to show (default 20)"},
				},
			}),
		},
		{
			Name:        "notes_read",
			Description: "Read a note's content",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"title": map[string]string{"type": "string", "description": "Note title"}},
				"required":   []string{"title"},
			}),
		},
		{
			Name:        "notes_create",
			Description: "Create a new note",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":  map[string]string{"type": "string", "description": "Note title"},
					"body":   map[string]string{"type": "string", "description": "Note content"},
					"folder": map[string]string{"type": "string", "description": "Folder name (default: Notes)"},
				},
				"required": []string{"title"},
			}),
		},
		{
			Name:        "notes_search",
			Description: "Search notes by keyword",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"keyword": map[string]string{"type": "string", "description": "Search keyword"}},
				"required":   []string{"keyword"},
			}),
		},

		// === WEATHER ===
		{
			Name:        "weather_current",
			Description: "Get current weather for a location",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"location": map[string]string{"type": "string", "description": "City name or location (e.g., 'London', 'Tokyo')"}},
			}),
		},
		{
			Name:        "weather_forecast",
			Description: "Get weather forecast for a location",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]string{"type": "string", "description": "City name or location"},
					"days":     map[string]string{"type": "number", "description": "Days to forecast (1-3)"},
				},
			}),
		},

		// === WEB ===
		{
			Name:        "web_search",
			Description: "Search the web using DuckDuckGo",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"query": map[string]string{"type": "string", "description": "Search query"}},
				"required":   []string{"query"},
			}),
		},
		{
			Name:        "web_fetch",
			Description: "Fetch content from a URL",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"url": map[string]string{"type": "string", "description": "URL to fetch"}},
				"required":   []string{"url"},
			}),
		},
		{
			Name:        "open_url",
			Description: "Open a URL in the default web browser",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"url": map[string]string{"type": "string", "description": "URL to open"}},
				"required":   []string{"url"},
			}),
		},

		// === CLIPBOARD ===
		{
			Name:        "clipboard_read",
			Description: "Read content from the clipboard",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "clipboard_write",
			Description: "Write content to the clipboard",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"content": map[string]string{"type": "string", "description": "Content to copy"}},
				"required":   []string{"content"},
			}),
		},

		// === NOTIFICATIONS ===
		{
			Name:        "notification_send",
			Description: "Send a system notification",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":    map[string]string{"type": "string", "description": "Notification title"},
					"message":  map[string]string{"type": "string", "description": "Notification message"},
					"subtitle": map[string]string{"type": "string", "description": "Subtitle (macOS only)"},
				},
				"required": []string{"title"},
			}),
		},

		// === SCREENSHOT ===
		{
			Name:        "screenshot",
			Description: "Capture a screenshot",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]string{"type": "string", "description": "Save path (default: Desktop)"},
					"type": map[string]string{"type": "string", "description": "Type: fullscreen, window, or selection"},
				},
			}),
		},

		// === MUSIC ===
		{
			Name:        "music_play",
			Description: "Start or resume music playback",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "music_pause",
			Description: "Pause music playback",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "music_next",
			Description: "Skip to the next track",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "music_previous",
			Description: "Go to the previous track",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "music_now_playing",
			Description: "Get currently playing track info",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "music_volume",
			Description: "Set music volume (0-100)",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"volume": map[string]string{"type": "number", "description": "Volume level 0-100"}},
				"required":   []string{"volume"},
			}),
		},
		{
			Name:        "music_search",
			Description: "Search and play music in Spotify",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"query": map[string]string{"type": "string", "description": "Search query (song, artist, album)"}},
				"required":   []string{"query"},
			}),
		},

		// === SYSTEM ===
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

		// === GIT & GITHUB ===
		{
			Name:        "git_status",
			Description: "Show git working tree status",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "git_log",
			Description: "Show recent git commits",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"limit": map[string]string{"type": "number", "description": "Number of commits (default 10)"}},
			}),
		},
		{
			Name:        "git_diff",
			Description: "Show git diff",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"staged": map[string]string{"type": "boolean", "description": "Show staged changes"},
					"file":   map[string]string{"type": "string", "description": "Specific file to diff"},
				},
			}),
		},
		{
			Name:        "git_branch",
			Description: "List git branches",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "github_pr_list",
			Description: "List GitHub pull requests (requires gh CLI)",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"state": map[string]string{"type": "string", "description": "Filter by state: open, closed, all"},
					"limit": map[string]string{"type": "number", "description": "Max results (default 10)"},
				},
			}),
		},
		{
			Name:        "github_pr_view",
			Description: "View a GitHub pull request",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"number": map[string]string{"type": "number", "description": "PR number"}},
				"required":   []string{"number"},
			}),
		},
		{
			Name:        "github_issue_list",
			Description: "List GitHub issues (requires gh CLI)",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"state": map[string]string{"type": "string", "description": "Filter by state: open, closed, all"},
					"limit": map[string]string{"type": "number", "description": "Max results (default 10)"},
				},
			}),
		},
		{
			Name:        "github_issue_view",
			Description: "View a GitHub issue",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"number": map[string]string{"type": "number", "description": "Issue number"}},
				"required":   []string{"number"},
			}),
		},
		{
			Name:        "github_issue_create",
			Description: "Create a GitHub issue",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":  map[string]string{"type": "string", "description": "Issue title"},
					"body":   map[string]string{"type": "string", "description": "Issue body"},
					"labels": map[string]string{"type": "string", "description": "Comma-separated labels"},
				},
				"required": []string{"title"},
			}),
		},
		{
			Name:        "github_repo_view",
			Description: "View current GitHub repository info",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},

		// === BROWSER AUTOMATION ===
		{
			Name:        "browser_start",
			Description: "Start a new browser or connect to an existing Chrome. Use cdp_url to attach to a Chrome launched with --remote-debugging-port (e.g. \"127.0.0.1:9222\"). Without cdp_url, launches a new Chrome instance.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cdp_url":  map[string]string{"type": "string", "description": "CDP address of existing Chrome (e.g. 127.0.0.1:9222). Chrome must be started with --remote-debugging-port flag."},
					"headless": map[string]string{"type": "boolean", "description": "Launch in headless mode (default: false, ignored when using cdp_url)"},
					"url":      map[string]string{"type": "string", "description": "Initial URL to navigate to"},
				},
			}),
		},
		{
			Name:        "browser_navigate",
			Description: "Navigate to a URL in the browser. Auto-starts browser if not running (connects to Chrome on port 9222 if available, otherwise launches new). Always navigate to the base site URL first, then use snapshot+click/type to interact with page elements step by step.",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"url": map[string]string{"type": "string", "description": "URL to navigate to"}},
				"required":   []string{"url"},
			}),
		},
		{
			Name:        "browser_snapshot",
			Description: "Capture the page accessibility tree with numbered refs. Use these ref numbers with browser_click/browser_type to interact with elements. MUST re-run after any page change.",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "browser_click",
			Description: "Click an element by its ref number from browser_snapshot",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"ref": map[string]string{"type": "number", "description": "Element ref number from browser_snapshot"}},
				"required":   []string{"ref"},
			}),
		},
		{
			Name:        "browser_type",
			Description: "Type text into an element by its ref number from browser_snapshot. Use submit=true to press Enter after typing.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ref":    map[string]string{"type": "number", "description": "Element ref number from browser_snapshot"},
					"text":   map[string]string{"type": "string", "description": "Text to type"},
					"submit": map[string]string{"type": "boolean", "description": "Press Enter after typing (default: false)"},
				},
				"required": []string{"ref", "text"},
			}),
		},
		{
			Name:        "browser_press",
			Description: "Press a keyboard key (Enter, Tab, Escape, Backspace, ArrowUp, ArrowDown, ArrowLeft, ArrowRight, Space, Delete, Home, End, PageUp, PageDown)",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"key": map[string]string{"type": "string", "description": "Key name to press"}},
				"required":   []string{"key"},
			}),
		},
		{
			Name:        "browser_execute_js",
			Description: "Execute JavaScript on the current page. Use to dismiss modals/overlays blocking interaction, extract data, or interact with elements not reachable via refs.",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"script": map[string]string{"type": "string", "description": "JavaScript code to execute in page context"}},
				"required":   []string{"script"},
			}),
		},
		{
			Name:        "browser_click_all",
			Description: "Click ALL elements matching a CSS selector (e.g. '.like-btn', '#follow'). Uses real mouse clicks with configurable delay between each. Useful for batch-liking, batch-following, etc.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"selector": map[string]string{"type": "string", "description": "CSS selector for elements to click (e.g. '.like-lottie')"},
					"delay_ms": map[string]string{"type": "number", "description": "Milliseconds to wait between clicks (default: 500)"},
				},
				"required": []string{"selector"},
			}),
		},
		{
			Name:        "browser_screenshot",
			Description: "Take a screenshot of the current page",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":      map[string]string{"type": "string", "description": "Output file path (default: ~/Desktop/browser_screenshot_<timestamp>.png)"},
					"full_page": map[string]string{"type": "boolean", "description": "Capture full scrollable page (default: false)"},
				},
			}),
		},
		{
			Name:        "browser_tabs",
			Description: "List all open browser tabs with their target IDs and URLs",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "browser_tab_open",
			Description: "Open a new browser tab",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"url": map[string]string{"type": "string", "description": "URL to open (default: about:blank)"}},
			}),
		},
		{
			Name:        "browser_tab_close",
			Description: "Close a browser tab by target ID, or close the active tab if no ID given",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"target_id": map[string]string{"type": "string", "description": "Target ID of the tab to close (from browser_tabs)"}},
			}),
		},
		{
			Name:        "browser_status",
			Description: "Check if the browser is running and get current state",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "browser_stop",
			Description: "Close the browser",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
	}
}

// processToolCalls executes tool calls and returns results plus any file attachments
func (a *Agent) processToolCalls(ctx context.Context, toolCalls []ToolCall) ([]ToolResult, []router.FileAttachment) {
	results := make([]ToolResult, 0, len(toolCalls))
	var files []router.FileAttachment

	for _, tc := range toolCalls {
		if tc.Name == "file_send" {
			content, file := executeFileSend(tc.Input)
			if file != nil {
				files = append(files, *file)
			}
			results = append(results, ToolResult{
				ToolCallID: tc.ID,
				Content:    content,
				IsError:    file == nil,
			})
			continue
		}

		result := a.executeTool(ctx, tc.Name, tc.Input)
		results = append(results, ToolResult{
			ToolCallID: tc.ID,
			Content:    result,
			IsError:    false,
		})
	}

	return results, files
}

// executeTool runs a tool and returns the result
func (a *Agent) executeTool(ctx context.Context, name string, input json.RawMessage) string {
	logger.Info("[Agent] Executing tool: %s", name)

	// Parse input arguments
	var args map[string]any
	if err := json.Unmarshal(input, &args); err != nil {
		return fmt.Sprintf("Error parsing arguments: %v", err)
	}

	// Call tools directly
	result := callToolDirect(ctx, name, args)

	// Log result at verbose level (truncate if too long)
	if len(result) > 500 {
		logger.Verbose("[Agent] Tool %s result: %s... (truncated)", name, result[:500])
	} else {
		logger.Verbose("[Agent] Tool %s result: %s", name, result)
	}

	return result
}

// callToolDirect calls a tool directly
func callToolDirect(ctx context.Context, name string, args map[string]any) string {
	switch name {
	// File operations
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
	case "file_trash":
		return executeFileTrash(ctx, args)
	case "file_read":
		path := ""
		if p, ok := args["path"].(string); ok {
			path = p
		}
		return executeFileRead(ctx, path)

	// Calendar
	case "calendar_today":
		return executeCalendarToday(ctx)
	case "calendar_list_events":
		days := 7
		if d, ok := args["days"].(float64); ok {
			days = int(d)
		}
		return executeCalendarListEvents(ctx, days)
	case "calendar_create_event":
		return executeCalendarCreate(ctx, args)
	case "calendar_search":
		return executeCalendarSearch(ctx, args)
	case "calendar_delete":
		return executeCalendarDelete(ctx, args)

	// Reminders
	case "reminders_list":
		return executeRemindersToday(ctx)
	case "reminders_add":
		return executeRemindersAdd(ctx, args)
	case "reminders_complete":
		title := ""
		if t, ok := args["title"].(string); ok {
			title = t
		}
		return executeRemindersComplete(ctx, title)
	case "reminders_delete":
		title := ""
		if t, ok := args["title"].(string); ok {
			title = t
		}
		return executeRemindersDelete(ctx, title)

	// Notes
	case "notes_list":
		return executeNotesList(ctx, args)
	case "notes_read":
		title := ""
		if t, ok := args["title"].(string); ok {
			title = t
		}
		return executeNotesRead(ctx, title)
	case "notes_create":
		return executeNotesCreate(ctx, args)
	case "notes_search":
		keyword := ""
		if k, ok := args["keyword"].(string); ok {
			keyword = k
		}
		return executeNotesSearch(ctx, keyword)

	// Weather
	case "weather_current":
		location := ""
		if l, ok := args["location"].(string); ok {
			location = l
		}
		return executeWeatherCurrent(ctx, location)
	case "weather_forecast":
		location := ""
		days := 3
		if l, ok := args["location"].(string); ok {
			location = l
		}
		if d, ok := args["days"].(float64); ok {
			days = int(d)
		}
		return executeWeatherForecast(ctx, location, days)

	// Web
	case "web_search":
		query := ""
		if q, ok := args["query"].(string); ok {
			query = q
		}
		return executeWebSearch(ctx, query)
	case "web_fetch":
		url := ""
		if u, ok := args["url"].(string); ok {
			url = u
		}
		return executeWebFetch(ctx, url)
	case "open_url":
		url := ""
		if u, ok := args["url"].(string); ok {
			url = u
		}
		return executeOpenURL(ctx, url)

	// Clipboard
	case "clipboard_read":
		return executeClipboardRead(ctx)
	case "clipboard_write":
		content := ""
		if c, ok := args["content"].(string); ok {
			content = c
		}
		return executeClipboardWrite(ctx, content)

	// Notification
	case "notification_send":
		return executeNotificationSend(ctx, args)

	// Screenshot
	case "screenshot":
		return executeScreenshot(ctx, args)

	// Music
	case "music_play":
		return executeMusicPlay(ctx)
	case "music_pause":
		return executeMusicPause(ctx)
	case "music_next":
		return executeMusicNext(ctx)
	case "music_previous":
		return executeMusicPrevious(ctx)
	case "music_now_playing":
		return executeMusicNowPlaying(ctx)
	case "music_volume":
		volume := 50.0
		if v, ok := args["volume"].(float64); ok {
			volume = v
		}
		return executeMusicVolume(ctx, volume)
	case "music_search":
		query := ""
		if q, ok := args["query"].(string); ok {
			query = q
		}
		return executeMusicSearch(ctx, query)

	// System
	case "system_info":
		return executeSystemInfo(ctx)
	case "shell_execute":
		cmd := ""
		if c, ok := args["command"].(string); ok {
			cmd = c
		}
		return executeShell(ctx, cmd)

	// Git & GitHub
	case "git_status":
		return executeGitStatus(ctx)
	case "git_log":
		return executeGitLog(ctx, args)
	case "git_diff":
		return executeGitDiff(ctx, args)
	case "git_branch":
		return executeGitBranch(ctx)
	case "github_pr_list":
		return executeGitHubPRList(ctx, args)
	case "github_pr_view":
		return executeGitHubPRView(ctx, args)
	case "github_issue_list":
		return executeGitHubIssueList(ctx, args)
	case "github_issue_view":
		return executeGitHubIssueView(ctx, args)
	case "github_issue_create":
		return executeGitHubIssueCreate(ctx, args)
	case "github_repo_view":
		return executeGitHubRepoView(ctx)

	// Browser automation
	case "browser_start":
		return executeBrowserStart(ctx, args)
	case "browser_navigate":
		url := ""
		if u, ok := args["url"].(string); ok {
			url = u
		}
		return executeBrowserNavigate(ctx, url)
	case "browser_snapshot":
		return executeBrowserSnapshot(ctx)
	case "browser_click":
		ref := 0
		if r, ok := args["ref"].(float64); ok {
			ref = int(r)
		}
		return executeBrowserClick(ctx, ref)
	case "browser_type":
		ref := 0
		text := ""
		submit := false
		if r, ok := args["ref"].(float64); ok {
			ref = int(r)
		}
		if t, ok := args["text"].(string); ok {
			text = t
		}
		if s, ok := args["submit"].(bool); ok {
			submit = s
		}
		return executeBrowserType(ctx, ref, text, submit)
	case "browser_press":
		key := ""
		if k, ok := args["key"].(string); ok {
			key = k
		}
		return executeBrowserPress(ctx, key)
	case "browser_execute_js":
		script := ""
		if s, ok := args["script"].(string); ok {
			script = s
		}
		return executeBrowserExecuteJS(ctx, script)
	case "browser_click_all":
		return executeBrowserClickAll(ctx, args)
	case "browser_screenshot":
		return executeBrowserScreenshot(ctx, args)
	case "browser_tabs":
		return executeBrowserTabs(ctx)
	case "browser_tab_open":
		return executeBrowserTabOpen(ctx, args)
	case "browser_tab_close":
		return executeBrowserTabClose(ctx, args)
	case "browser_status":
		return executeBrowserStatus(ctx)
	case "browser_stop":
		return executeBrowserStop(ctx)

	default:
		return fmt.Sprintf("Tool '%s' not implemented", name)
	}
}

func jsonSchema(schema map[string]any) json.RawMessage {
	data, _ := json.Marshal(schema)
	return data
}
