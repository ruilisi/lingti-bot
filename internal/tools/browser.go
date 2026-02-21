package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/proto"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/pltanton/lingti-bot/internal/browser"
	"github.com/pltanton/lingti-bot/internal/logger"
)

// BrowserStart launches a browser instance or connects to an existing Chrome.
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
	if c, ok := req.Params.Arguments["cdp_url"].(string); ok {
		opts.ConnectURL = c
	}

	b := browser.Instance()
	logger.Debug("[browser_start] headless=%v url=%q cdp_url=%q executable=%q", opts.Headless, opts.URL, opts.ConnectURL, opts.ExecutablePath)

	var startErr error
	if opts.ConnectURL == "" && opts.ExecutablePath == "" {
		// No explicit target: use EnsureRunning so we connect to the configured/running Chrome
		// (cdp_url from config, or 127.0.0.1:9222) instead of launching a fresh browser
		// that would have no login state.
		startErr = b.EnsureRunning()
	} else {
		startErr = b.Start(opts)
	}
	if startErr != nil {
		logger.Debug("[browser_start] failed: %v", startErr)
		return mcp.NewToolResultError(fmt.Sprintf("failed to start browser: %v", startErr)), nil
	}

	var msg string
	if opts.ConnectURL != "" {
		msg = fmt.Sprintf("Connected to existing Chrome at %s", opts.ConnectURL)
	} else {
		msg = "Connected to browser (auto-detected existing Chrome or started new)"
	}
	if opts.URL != "" {
		msg += fmt.Sprintf(", navigated to %s", opts.URL)
	}
	logger.Debug("[browser_start] %s", msg)
	return mcp.NewToolResultText(msg), nil
}

// BrowserStop closes the browser or disconnects from external Chrome.
func BrowserStop(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	b := browser.Instance()
	wasConnected := b.IsConnected()
	logger.Debug("[browser_stop] connected=%v", wasConnected)
	if err := b.Stop(); err != nil {
		logger.Debug("[browser_stop] failed: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to stop browser: %v", err)), nil
	}
	if wasConnected {
		logger.Debug("[browser_stop] disconnected (Chrome still running)")
		return mcp.NewToolResultText("Disconnected from browser (Chrome is still running)"), nil
	}
	logger.Debug("[browser_stop] stopped")
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

	logger.Debug("[browser_navigate] url=%q", url)
	b := browser.Instance()
	if err := b.EnsureRunning(); err != nil {
		logger.Debug("[browser_navigate] EnsureRunning failed: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to start browser: %v", err)), nil
	}

	page, err := b.NavigationPage()
	if err != nil {
		logger.Debug("[browser_navigate] NavigationPage failed: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get page: %v", err)), nil
	}

	logger.Debug("[browser_navigate] navigating...")
	if err := page.Navigate(url); err != nil {
		logger.Debug("[browser_navigate] Navigate failed: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to navigate: %v", err)), nil
	}

	logger.Debug("[browser_navigate] waiting for page load...")
	// WaitLoad may return "navigated or closed" on redirects — non-fatal.
	_ = page.WaitLoad()

	// Record this as the bot's current working page so snapshot/click/type
	// all operate on this tab rather than opening new ones.
	b.SetCurrentPage(page)

	info, err := page.Info()
	if err != nil {
		logger.Debug("[browser_navigate] done (no page info)")
		return mcp.NewToolResultText(fmt.Sprintf("Navigated to %s", url)), nil
	}
	logger.Debug("[browser_navigate] done: title=%q url=%q", info.Title, info.URL)
	return mcp.NewToolResultText(fmt.Sprintf("Navigated to %s (title: %s)", info.URL, info.Title)), nil
}

// BrowserSnapshot captures the accessibility tree with numbered refs.
func BrowserSnapshot(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("[browser_snapshot] capturing accessibility tree...")
	b := browser.Instance()
	if err := b.EnsureRunning(); err != nil {
		logger.Debug("[browser_snapshot] EnsureRunning failed: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to start browser: %v", err)), nil
	}

	page, err := b.ActivePage()
	if err != nil {
		logger.Debug("[browser_snapshot] ActivePage failed: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get page: %v", err)), nil
	}

	snapshot, refs, err := browser.Snapshot(page)
	if err != nil {
		logger.Debug("[browser_snapshot] Snapshot failed: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to capture snapshot: %v", err)), nil
	}

	logger.Debug("[browser_snapshot] captured %d refs", len(refs))
	b.SetRefs(refs)

	info, _ := page.Info()
	header := ""
	if info != nil {
		header = fmt.Sprintf("URL: %s\nTitle: %s\nRefs: %d\n", info.URL, info.Title, len(refs))
	}

	// Save debug screenshot if debug mode is enabled
	if b.IsDebugMode() {
		timestamp := time.Now().Format("2006-01-02_15-04-05.000")
		filename := fmt.Sprintf("snapshot_%s.png", timestamp)
		screenshotPath := filepath.Join(b.DebugDir(), filename)

		screenshot, err := page.Screenshot(false, &proto.PageCaptureScreenshot{
			Format: proto.PageCaptureScreenshotFormatPng,
		})
		if err == nil {
			if err := os.WriteFile(screenshotPath, screenshot, 0644); err == nil {
				header += fmt.Sprintf("Screenshot: %s\n", screenshotPath)
			}
		}
	}

	header += "\n"
	return mcp.NewToolResultText(header + snapshot), nil
}

// BrowserScreenshot captures a screenshot of the current page.
func BrowserScreenshot(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	logger.Debug("[browser_screenshot] capturing...")
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
		logger.Debug("[browser_screenshot] failed to write: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to save screenshot: %v", err)), nil
	}

	logger.Debug("[browser_screenshot] saved to %s (%d bytes)", absPath, len(imgData))
	return mcp.NewToolResultText(fmt.Sprintf("Screenshot saved to: %s", absPath)), nil
}

// truncateSnapshot limits a snapshot string to maxLines lines to keep tool results concise.
// Full snapshots in every tool result bloat the context and cause DeepSeek to stop early.
func truncateSnapshot(snapshot string, maxLines int) string {
	lines := strings.SplitN(snapshot, "\n", maxLines+1)
	if len(lines) <= maxLines {
		return snapshot
	}
	return strings.Join(lines[:maxLines], "\n") + fmt.Sprintf("\n... (%d lines truncated — call browser_snapshot for full view)", len(strings.Split(snapshot, "\n"))-maxLines)
}

// BrowserClick clicks an element by ref number.
func BrowserClick(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ref, ok := req.Params.Arguments["ref"].(float64)
	if !ok {
		return mcp.NewToolResultError("ref is required (number)"), nil
	}

	logger.Debug("[browser_click] ref=%d", int(ref))
	b := browser.Instance()
	page, err := b.ActivePage()
	if err != nil {
		logger.Debug("[browser_click] ActivePage failed: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get page: %v", err)), nil
	}

	// Record tab count before the click so we can detect if a new tab opens.
	tabsBefore := b.PageCount()

	// Try to click the element
	if err := browser.Click(page, b, int(ref)); err != nil {
		logger.Debug("[browser_click] Click failed: %v", err)
		// Check if this is a "ref not found" error - might need fresh snapshot
		errStr := err.Error()
		if containsString(errStr, "ref") && containsString(errStr, "not found") {
			logger.Debug("[browser_click] ref not found, auto-refreshing snapshot...")
			// Try automatic retry with fresh snapshot
			_, newRefs, snapErr := browser.Snapshot(page)
			if snapErr == nil {
				b.SetRefs(newRefs)
				logger.Debug("[browser_click] retrying click with %d new refs", len(newRefs))

				// Retry the click with updated refs
				if retryErr := browser.Click(page, b, int(ref)); retryErr == nil {
					entry, _ := b.GetRef(int(ref))
					logger.Debug("[browser_click] retry succeeded: [%d] %s %q", int(ref), entry.Role, entry.Name)
					return mcp.NewToolResultText(fmt.Sprintf("Clicked [%d] %s %q (after auto-refresh)", int(ref), entry.Role, entry.Name)), nil
				}
			}
		}

		// If retry failed or not applicable, capture fresh snapshot for AI to see current state
		logger.Debug("[browser_click] capturing snapshot for error context")
		snapshot, newRefs, snapErr := browser.Snapshot(page)
		if snapErr == nil {
			b.SetRefs(newRefs)
			return mcp.NewToolResultError(fmt.Sprintf(
				"Failed to click ref %d: %v\n\nCurrent page snapshot (refs may have changed):\n%s",
				int(ref), err, snapshot,
			)), nil
		}

		return mcp.NewToolResultError(fmt.Sprintf("failed to click ref %d: %v", int(ref), err)), nil
	}

	entry, _ := b.GetRef(int(ref))
	logger.Debug("[browser_click] clicked [%d] %s %q", int(ref), entry.Role, entry.Name)
	clickMsg := fmt.Sprintf("Clicked [%d] %s %q", int(ref), entry.Role, entry.Name)

	// Wait a moment for any new tab to open, then detect and switch to it.
	time.Sleep(500 * time.Millisecond)
	if switched := b.SwitchToNewestPage(tabsBefore); switched {
		logger.Debug("[browser_click] new tab detected, switched currentPage")
		clickMsg += "\n\n⚠ A new tab was opened by this click. Bot is now tracking the new tab."
	}

	// Get page URL/title from the (possibly new) active page.
	activePage, _ := b.ActivePage()
	if activePage != nil {
		info, _ := activePage.Info()
		if info != nil {
			clickMsg += fmt.Sprintf("\n\nNow on: %s\nTitle: %s", info.URL, info.Title)
		}
	}
	clickMsg += "\n\nCall browser_snapshot to see the current page elements and continue with the next action."

	logger.Debug("[browser_click] done, instructing model to snapshot")
	return mcp.NewToolResultText(clickMsg), nil
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

	logger.Debug("[browser_type] ref=%d text=%q submit=%v", int(ref), text, submit)
	b := browser.Instance()
	page, err := b.ActivePage()
	if err != nil {
		logger.Debug("[browser_type] ActivePage failed: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to get page: %v", err)), nil
	}

	// Try to type into the element
	if err := browser.Type(page, b, int(ref), text, submit); err != nil {
		logger.Debug("[browser_type] Type failed: %v", err)
		// Check if this is a "ref not found" error - might need fresh snapshot
		errStr := err.Error()
		if containsString(errStr, "ref") && containsString(errStr, "not found") {
			logger.Debug("[browser_type] ref not found, auto-refreshing snapshot...")
			// Try automatic retry with fresh snapshot
			_, newRefs, snapErr := browser.Snapshot(page)
			if snapErr == nil {
				b.SetRefs(newRefs)
				logger.Debug("[browser_type] retrying with %d new refs", len(newRefs))

				// Retry the type with updated refs
				if retryErr := browser.Type(page, b, int(ref), text, submit); retryErr == nil {
					msg := fmt.Sprintf("Typed %q into [%d] (after auto-refresh)", text, int(ref))
					if submit {
						msg += " and pressed Enter"
					}
					logger.Debug("[browser_type] retry succeeded")
					return mcp.NewToolResultText(msg), nil
				}
			}
		}

		// If retry failed or not applicable, capture fresh snapshot for AI to see current state
		logger.Debug("[browser_type] capturing snapshot for error context")
		snapshot, newRefs, snapErr := browser.Snapshot(page)
		if snapErr == nil {
			b.SetRefs(newRefs)
			return mcp.NewToolResultError(fmt.Sprintf(
				"Failed to type into ref %d: %v\n\nCurrent page snapshot (refs may have changed):\n%s",
				int(ref), err, snapshot,
			)), nil
		}

		return mcp.NewToolResultError(fmt.Sprintf("failed to type into ref %d: %v", int(ref), err)), nil
	}

	typeMsg := fmt.Sprintf("Typed %q into [%d]", text, int(ref))
	if submit {
		typeMsg += " and pressed Enter"
	}
	logger.Debug("[browser_type] %s", typeMsg)

	info, _ := page.Info()
	hint := typeMsg
	if !submit {
		hint += "\n\nText typed but NOT submitted — use browser_press key=\"Enter\" or click the submit button to trigger the action."
	} else {
		hint += "\n\nPage may have changed after submit. Call browser_snapshot to see current elements and continue."
	}
	// Include brief page info so the model knows where it is, without the full snapshot.
	if info != nil {
		hint += fmt.Sprintf("\nNow on: %s | Title: %s", info.URL, info.Title)
	}
	return mcp.NewToolResultText(hint), nil
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

	logger.Debug("[browser_press] key=%q", key)
	if err := browser.Press(page, key); err != nil {
		logger.Debug("[browser_press] failed: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to press key: %v", err)), nil
	}

	logger.Debug("[browser_press] pressed %s", key)
	return mcp.NewToolResultText(fmt.Sprintf("Pressed %s", key)), nil
}

// BrowserCommentZhihu posts a comment on a Zhihu answer using the verified JS recipe.
// It expands the first answer's comment section, types the comment, and submits.
func BrowserCommentZhihu(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	comment, ok := req.Params.Arguments["comment"].(string)
	if !ok || comment == "" {
		return mcp.NewToolResultError("comment is required"), nil
	}

	b := browser.Instance()
	page, err := b.ActivePage()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get page: %v", err)), nil
	}

	// Step 1a: if editor is already open, skip opening it.
	// Otherwise click "添加评论" directly if visible, else click "X条评论" to expand
	// the comment section first, then click "添加评论" inside it.
	r1, err := browser.ExecuteJS(page, `
		// Already open?
		if (document.querySelector('.public-DraftEditor-content')) { return 'editor already open'; }

		// Prefer clicking "添加评论" directly (zhuanlan pages show this right away)
		var addBtn = Array.from(document.querySelectorAll('button,span,a')).find(function(e) {
			return e.textContent.replace(/\u200b/g,'').trim() === '添加评论';
		});
		if (addBtn) { addBtn.click(); return 'clicked 添加评论'; }

		// On question pages "X条评论" toggles the list; "添加评论" appears inside.
		// Click the first answer's comment toggle to expand it.
		var toggleBtn = Array.from(document.querySelectorAll('button,span')).find(function(e) {
			var t = e.textContent.replace(/\u200b/g,'').trim();
			return /^[\d]+\s*条评论$/.test(t) || t === '评论';
		});
		if (toggleBtn) { toggleBtn.click(); return 'expanded: ' + toggleBtn.textContent.trim(); }

		return 'no comment button found';
	`)
	logger.Debug("[browser_comment_zhihu] step1a: %s", r1)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("step1a failed: %v", err)), nil
	}

	// Step 1b: if we expanded comments (not directly clicked 添加评论), now find and
	// click the "添加评论" input placeholder that appears inside the expanded section.
	// Poll up to 4s for it to appear.
	// IMPORTANT: textContent is recursive, so querying 'div' would match any ancestor
	// that contains "添加评论" in its subtree. We restrict to elements that either:
	//   (a) have no child elements (leaf nodes), or
	//   (b) are known interactive types (button, span, a)
	//   (c) are the DraftEditor placeholder div specifically
	if r1 != "editor already open" && r1 != "clicked 添加评论" {
		var addClicked bool
		for range 20 {
			time.Sleep(200 * time.Millisecond)
			res, _ := browser.ExecuteJS(page, `
				// Already open after expand?
				if (document.querySelector('.public-DraftEditor-content')) { return 'editor appeared'; }

				// Specific known selectors for the comment input area on Zhihu question pages
				var specific = document.querySelector(
					'.CommentInput, [class*="CommentInput"], ' +
					'.DraftEditor-root, [class*="comment-input"], ' +
					'[placeholder="添加评论"], [data-placeholder="添加评论"]'
				);
				if (specific) { specific.click(); return 'clicked specific'; }

				// button/span/a with exact text — safe (not recursive parent match)
				var btn = Array.from(document.querySelectorAll('button,span,a')).find(function(e) {
					return e.textContent.replace(/\u200b/g,'').trim() === '添加评论';
				});
				if (btn) { btn.click(); return 'clicked btn: ' + btn.tagName; }

				// Leaf div/label with exact text (no child elements that also contain the text)
				var leaf = Array.from(document.querySelectorAll('div,label')).find(function(e) {
					if (e.textContent.replace(/\u200b/g,'').trim() !== '添加评论') return false;
					// Make sure no child element also has this text (i.e. we are the real target)
					return !Array.from(e.children).some(function(c) {
						return c.textContent.replace(/\u200b/g,'').trim() === '添加评论';
					});
				});
				if (leaf) { leaf.click(); return 'clicked leaf: ' + leaf.className.slice(0,40); }

				return 'waiting';
			`)
			if res != "waiting" {
				addClicked = true
				break
			}
		}
		if !addClicked {
			return mcp.NewToolResultError(fmt.Sprintf("could not find 添加评论 after expanding comments (step1=%s)", r1)), nil
		}
	}

	// Step 1c: wait for the Draft.js editor to appear (poll up to 4s)
	var editorReady bool
	for range 20 {
		time.Sleep(200 * time.Millisecond)
		check, _ := browser.ExecuteJS(page, `document.querySelector('.public-DraftEditor-content') ? 'yes' : 'no'`)
		if check == "yes" {
			editorReady = true
			break
		}
	}
	if !editorReady {
		return mcp.NewToolResultError(fmt.Sprintf("editor did not appear after clicking 添加评论 (step1=%s)", r1)), nil
	}

	// Step 2: insert text into Draft.js editor via ClipboardEvent paste.
	// execCommand('insertText') inserts DOM text but does NOT update Draft.js's internal
	// EditorState, so the 发布 button stays disabled. Dispatching a paste ClipboardEvent
	// causes Draft.js to process the text through its own paste handler, updating state.
	jsonComment, _ := json.Marshal(comment)
	r2, err := browser.ExecuteJS(page, fmt.Sprintf(`
		var ed = document.querySelector('.public-DraftEditor-content');
		if (!ed) { return 'editor not found'; }
		ed.click();
		ed.focus();
		document.execCommand('selectAll', false);
		var dt = new DataTransfer();
		dt.setData('text/plain', %s);
		ed.dispatchEvent(new ClipboardEvent('paste', { clipboardData: dt, bubbles: true, cancelable: true }));
		return 'pasted';
	`, string(jsonComment)))
	logger.Debug("[browser_comment_zhihu] step2 paste: %s", r2)
	if err != nil || r2 == "editor not found" {
		return mcp.NewToolResultError(fmt.Sprintf("step2 paste failed: %v %s", err, r2)), nil
	}

	// Wait for Draft.js to process the paste event and re-render
	time.Sleep(600 * time.Millisecond)

	// Step 3: click the 发布 submit button.
	// IMPORTANT: document.querySelector('button.Button--primary') matches the search bar button first.
	// Must find specifically the button with text "发布".
	r3, err := browser.ExecuteJS(page, `
		var btn = Array.from(document.querySelectorAll('button.Button--primary')).find(function(b){ return b.textContent.trim() === '发布'; });
		if (btn && !btn.disabled) { btn.click(); return 'submitted'; }
		if (btn && btn.disabled) { return 'submit button is disabled (comment may be empty)'; }
		return 'submit button not found';
	`)
	logger.Debug("[browser_comment_zhihu] step3 submit: %s", r3)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("step3 submit failed: %v", err)), nil
	}

	if r3 == "submitted" {
		return mcp.NewToolResultText(fmt.Sprintf("Comment posted successfully: %q", comment)), nil
	}
	return mcp.NewToolResultError(fmt.Sprintf("comment may not have submitted: %s (expand=%s, type=%s)", r3, r1, r2)), nil
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

	logger.Debug("[browser_execute_js] script=%q", script)
	result, err := browser.ExecuteJS(page, script)
	if err != nil {
		logger.Debug("[browser_execute_js] failed: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("JS error: %v", err)), nil
	}

	logger.Debug("[browser_execute_js] done, result length=%d", len(result))
	return mcp.NewToolResultText(result), nil
}

// BrowserClickAll clicks all elements matching a CSS selector with delay.
func BrowserClickAll(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	selector, ok := req.Params.Arguments["selector"].(string)
	if !ok || selector == "" {
		return mcp.NewToolResultError("selector is required (CSS selector)"), nil
	}

	delay := 500 * time.Millisecond
	if d, ok := req.Params.Arguments["delay_ms"].(float64); ok && d > 0 {
		delay = time.Duration(d) * time.Millisecond
	}

	b := browser.Instance()
	page, err := b.ActivePage()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get page: %v", err)), nil
	}

	skipSelector := ""
	if s, ok := req.Params.Arguments["skip_selector"].(string); ok {
		skipSelector = s
	}

	logger.Debug("[browser_click_all] selector=%q skip=%q delay=%v", selector, skipSelector, delay)
	count, err := browser.ClickAll(page, selector, delay, skipSelector)
	if err != nil {
		logger.Debug("[browser_click_all] failed: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("failed to click elements: %v", err)), nil
	}

	logger.Debug("[browser_click_all] clicked %d elements", count)
	return mcp.NewToolResultText(fmt.Sprintf("Clicked %d elements matching %q", count, selector)), nil
}

// BrowserTabs lists all open tabs.
func BrowserTabs(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	b := browser.Instance()
	if err := b.EnsureRunning(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start browser: %v", err)), nil
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

	// WaitLoad may return "Inspected target navigated or closed" on redirects — treat as non-fatal.
	_ = page.WaitLoad()

	info, err := page.Info()
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Opened new tab: %s", url)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Opened new tab: %s (target_id: %s)", info.URL, info.TargetID)), nil
}

// BrowserTabClose closes a tab by target ID or the active tab.
func BrowserTabClose(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	b := browser.Instance()
	if err := b.EnsureRunning(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start browser: %v", err)), nil
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

// containsString is a helper to check if a string contains a substring (case-insensitive).
func containsString(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
