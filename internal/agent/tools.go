package agent

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/pltanton/lingti-bot/internal/tools"
)

// executeSystemInfo runs the system_info tool
func executeSystemInfo(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := tools.SystemInfo(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}

	return extractText(result)
}

// executeCalendarToday runs the calendar_today tool
func executeCalendarToday(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := tools.CalendarToday(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}

	return extractText(result)
}

// executeCalendarListEvents runs the calendar_list_events tool
func executeCalendarListEvents(ctx context.Context, days int) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"days": float64(days),
	}

	result, err := tools.CalendarListEvents(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}

	return extractText(result)
}

// executeFileList runs the file_list tool
func executeFileList(ctx context.Context, path string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"path": path,
	}

	result, err := tools.FileList(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}

	return extractText(result)
}

// executeFileListOld runs the file_list_old tool
func executeFileListOld(ctx context.Context, path string, days int) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"path": path,
		"days": float64(days),
	}

	result, err := tools.FileListOld(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}

	return extractText(result)
}

// executeShell runs the shell_execute tool
func executeShell(ctx context.Context, command string) string {
	// Safety check
	blocked := []string{"rm -rf /", "mkfs", "dd if="}
	cmdLower := strings.ToLower(command)
	for _, b := range blocked {
		if strings.Contains(cmdLower, b) {
			return "Command blocked for safety"
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	var result strings.Builder
	if stdout.Len() > 0 {
		result.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		result.WriteString("\nstderr: " + stderr.String())
	}
	if err != nil {
		result.WriteString("\nerror: " + err.Error())
	}

	return result.String()
}

// extractText extracts text content from MCP result
func extractText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}

	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			return textContent.Text
		}
	}

	return ""
}
