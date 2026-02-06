package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/pltanton/lingti-bot/internal/browser"

)

// BrowserStart launches a browser instance.
func BrowserStart(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	opts := browser.StartOptions{
		Headless: false,
	}

	if h, ok := req.Params.Arguments["headless"].(bool); ok {
		opts.Headless = h
	}
	if u, ok := req.Params.Arguments["url"].(string); ok {
		opts.URL = u
	}
	if p, ok := req.Params.Arguments["executable_path"].(string); ok {
		opts.ExecutablePath = p
	}

	b := browser.Instance()
	if err := b.Start(opts); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start browser: %v", err)), nil
	}

	mode := "headless"
	if !opts.Headless {
		mode = "headed"
	}
	msg := fmt.Sprintf("Browser started (%s)", mode)
	if opts.URL != "" {
		msg += fmt.Sprintf(", navigated to %s", opts.URL)
	}
	return mcp.NewToolResultText(msg), nil
}

// BrowserStop closes the browser.
func BrowserStop(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	b := browser.Instance()
	if err := b.Stop(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to stop browser: %v", err)), nil
	}
	return mcp.NewToolResultText("Browser stopped"), nil
}

// BrowserStatus returns the current browser state.
func BrowserStatus(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	info := browser.Instance().Status()
	data, _ := json.Marshal(info)
	return mcp.NewToolResultText(string(data)), nil
}

// BrowserNavigate navigates to a URL.
func BrowserNavigate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, ok := req.Params.Arguments["url"].(string)
	if !ok || url == "" {
		return mcp.NewToolResultError("url is required"), nil
	}

	b := browser.Instance()
	if err := b.EnsureRunning(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start browser: %v", err)), nil
	}

	page, err := b.ActivePage()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get page: %v", err)), nil
	}

	if err := page.Navigate(url); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to navigate: %v", err)), nil
	}

	if err := page.WaitLoad(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("page load error: %v", err)), nil
	}

	info, err := page.Info()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Navigated to %s", url)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Navigated to %s (title: %s)", info.URL, info.Title)), nil
}

// BrowserSnapshot captures the accessibility tree with numbered refs.
func BrowserSnapshot(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	b := browser.Instance()
	if err := b.EnsureRunning(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start browser: %v", err)), nil
	}

	page, err := b.ActivePage()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get page: %v", err)), nil
	}

	snapshot, refs, err := browser.Snapshot(page)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to capture snapshot: %v", err)), nil
	}

	b.SetRefs(refs)

	info, _ := page.Info()
	header := ""
	if info != nil {
		header = fmt.Sprintf("URL: %s\nTitle: %s\nRefs: %d\n\n", info.URL, info.Title, len(refs))
	}

	return mcp.NewToolResultText(header + snapshot), nil
}

// BrowserScreenshot captures a screenshot of the current page.
func BrowserScreenshot(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	b := browser.Instance()
	if err := b.EnsureRunning(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start browser: %v", err)), nil
	}

	page, err := b.ActivePage()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get page: %v", err)), nil
	}

	fullPage := false
	if fp, ok := req.Params.Arguments["full_page"].(bool); ok {
		fullPage = fp
	}

	var imgData []byte
	if fullPage {
		imgData, err = page.Screenshot(true, &proto.PageCaptureScreenshot{
			Format: proto.PageCaptureScreenshotFormatPng,
		})
	} else {
		imgData, err = page.Screenshot(false, &proto.PageCaptureScreenshot{
			Format: proto.PageCaptureScreenshotFormatPng,
		})
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to capture screenshot: %v", err)), nil
	}

	// Determine output path
	outputPath := ""
	if p, ok := req.Params.Arguments["path"].(string); ok && p != "" {
		outputPath = p
	} else {
		home, _ := os.UserHomeDir()
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		outputPath = filepath.Join(home, "Desktop", fmt.Sprintf("browser_screenshot_%s.png", timestamp))
	}

	if len(outputPath) > 0 && outputPath[0] == '~' {
		home, _ := os.UserHomeDir()
		outputPath = filepath.Join(home, outputPath[1:])
	}

	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid path: %v", err)), nil
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create directory: %v", err)), nil
	}

	if err := os.WriteFile(absPath, imgData, 0644); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to save screenshot: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Screenshot saved to: %s", absPath)), nil
}

// BrowserClick clicks an element by ref number.
func BrowserClick(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ref, ok := req.Params.Arguments["ref"].(float64)
	if !ok {
		return mcp.NewToolResultError("ref is required (number)"), nil
	}

	b := browser.Instance()
	page, err := b.ActivePage()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get page: %v", err)), nil
	}

	if err := browser.Click(page, b, int(ref)); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to click ref %d: %v", int(ref), err)), nil
	}

	entry, _ := b.GetRef(int(ref))
	return mcp.NewToolResultText(fmt.Sprintf("Clicked [%d] %s %q", int(ref), entry.Role, entry.Name)), nil
}

// BrowserType types text into an element by ref number.
func BrowserType(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ref, ok := req.Params.Arguments["ref"].(float64)
	if !ok {
		return mcp.NewToolResultError("ref is required (number)"), nil
	}
	text, ok := req.Params.Arguments["text"].(string)
	if !ok || text == "" {
		return mcp.NewToolResultError("text is required"), nil
	}

	submit := false
	if s, ok := req.Params.Arguments["submit"].(bool); ok {
		submit = s
	}

	b := browser.Instance()
	page, err := b.ActivePage()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get page: %v", err)), nil
	}

	if err := browser.Type(page, b, int(ref), text, submit); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to type into ref %d: %v", int(ref), err)), nil
	}

	msg := fmt.Sprintf("Typed %q into [%d]", text, int(ref))
	if submit {
		msg += " and pressed Enter"
	}
	return mcp.NewToolResultText(msg), nil
}

// BrowserPress presses a keyboard key.
func BrowserPress(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	key, ok := req.Params.Arguments["key"].(string)
	if !ok || key == "" {
		return mcp.NewToolResultError("key is required (e.g., Enter, Tab, Escape)"), nil
	}

	b := browser.Instance()
	page, err := b.ActivePage()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get page: %v", err)), nil
	}

	if err := browser.Press(page, key); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to press key: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Pressed %s", key)), nil
}

// BrowserExecuteJS runs JavaScript on the active page.
func BrowserExecuteJS(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	script, ok := req.Params.Arguments["script"].(string)
	if !ok || script == "" {
		return mcp.NewToolResultError("script is required"), nil
	}

	b := browser.Instance()
	if err := b.EnsureRunning(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start browser: %v", err)), nil
	}

	page, err := b.ActivePage()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get page: %v", err)), nil
	}

	result, err := browser.ExecuteJS(page, script)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("JS error: %v", err)), nil
	}

	return mcp.NewToolResultText(result), nil
}

// BrowserTabs lists all open tabs.
func BrowserTabs(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	b := browser.Instance()
	if !b.IsRunning() {
		return mcp.NewToolResultError("browser not running"), nil
	}

	pages, err := b.Rod().Pages()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list tabs: %v", err)), nil
	}

	type tabInfo struct {
		TargetID string `json:"target_id"`
		URL      string `json:"url"`
		Title    string `json:"title"`
	}

	var tabs []tabInfo
	for _, p := range pages {
		info, err := p.Info()
		if err != nil {
			continue
		}
		tabs = append(tabs, tabInfo{
			TargetID: string(info.TargetID),
			URL:      info.URL,
			Title:    info.Title,
		})
	}

	data, _ := json.MarshalIndent(tabs, "", "  ")
	return mcp.NewToolResultText(fmt.Sprintf("%d tabs:\n%s", len(tabs), string(data))), nil
}

// BrowserTabOpen opens a new tab.
func BrowserTabOpen(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	b := browser.Instance()
	if err := b.EnsureRunning(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start browser: %v", err)), nil
	}

	url := "about:blank"
	if u, ok := req.Params.Arguments["url"].(string); ok && u != "" {
		url = u
	}

	page, err := b.Rod().Page(proto.TargetCreateTarget{URL: url})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to open tab: %v", err)), nil
	}

	page.MustWaitLoad()

	info, err := page.Info()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Opened new tab: %s", url)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Opened new tab: %s (target_id: %s)", info.URL, info.TargetID)), nil
}

// BrowserTabClose closes a tab by target ID or the active tab.
func BrowserTabClose(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	b := browser.Instance()
	if !b.IsRunning() {
		return mcp.NewToolResultError("browser not running"), nil
	}

	targetID := ""
	if t, ok := req.Params.Arguments["target_id"].(string); ok {
		targetID = t
	}

	pages, err := b.Rod().Pages()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list tabs: %v", err)), nil
	}

	if targetID != "" {
		for _, p := range pages {
			info, err := p.Info()
			if err != nil {
				continue
			}
			if string(info.TargetID) == targetID {
				if err := p.Close(); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to close tab: %v", err)), nil
				}
				return mcp.NewToolResultText(fmt.Sprintf("Closed tab %s", targetID)), nil
			}
		}
		return mcp.NewToolResultError(fmt.Sprintf("tab %s not found", targetID)), nil
	}

	// Close the active (first) tab
	if len(pages) == 0 {
		return mcp.NewToolResultError("no tabs to close"), nil
	}
	if err := pages.First().Close(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to close tab: %v", err)), nil
	}
	return mcp.NewToolResultText("Closed active tab"), nil
}
