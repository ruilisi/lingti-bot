package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// CalendarListEvents lists calendar events (macOS)
func CalendarListEvents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	days := 7
	if d, ok := req.Params.Arguments["days"].(float64); ok && d > 0 {
		days = int(d)
	}

	script := fmt.Sprintf(`
		set output to ""
		set startDate to current date
		set endDate to startDate + (%d * days)

		tell application "Calendar"
			repeat with cal in calendars
				set calName to name of cal
				set evts to (every event of cal whose start date ≥ startDate and start date ≤ endDate)
				repeat with evt in evts
					set evtStart to start date of evt
					set evtTitle to summary of evt
					set output to output & calName & " | " & (evtStart as string) & " | " & evtTitle & linefeed
				end repeat
			end repeat
		end tell
		return output
	`, days)

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get events: %v", err)), nil
	}

	if len(output) == 0 {
		return mcp.NewToolResultText("No events found"), nil
	}

	return mcp.NewToolResultText(string(output)), nil
}

// CalendarCreateEvent creates a new calendar event (macOS)
func CalendarCreateEvent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, ok := req.Params.Arguments["title"].(string)
	if !ok {
		return mcp.NewToolResultError("title is required"), nil
	}

	startTime, ok := req.Params.Arguments["start_time"].(string)
	if !ok {
		return mcp.NewToolResultError("start_time is required (format: 2024-01-15 14:00)"), nil
	}

	duration := 60
	if d, ok := req.Params.Arguments["duration"].(float64); ok {
		duration = int(d)
	}

	calendar := "Calendar"
	if c, ok := req.Params.Arguments["calendar"].(string); ok && c != "" {
		calendar = c
	}

	location := ""
	if l, ok := req.Params.Arguments["location"].(string); ok {
		location = l
	}

	notes := ""
	if n, ok := req.Params.Arguments["notes"].(string); ok {
		notes = n
	}

	t, err := time.Parse("2006-01-02 15:04", startTime)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid start_time format, use YYYY-MM-DD HH:MM: %v", err)), nil
	}

	endTime := t.Add(time.Duration(duration) * time.Minute)

	// macOS requires weekday in the date string
	script := fmt.Sprintf(`
		tell application "Calendar"
			tell calendar "%s"
				make new event with properties {summary:"%s", start date:date "%s", end date:date "%s", location:"%s", description:"%s"}
			end tell
		end tell
		return "OK"
	`, escapeAppleScript(calendar), escapeAppleScript(title),
		t.Format("Monday, 2 January 2006 at 3:04:05 PM"),
		endTime.Format("Monday, 2 January 2006 at 3:04:05 PM"),
		escapeAppleScript(location), escapeAppleScript(notes))

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create event: %v - %s", err, output)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Created: %s on %s for %d minutes", title, startTime, duration)), nil
}

// CalendarListCalendars lists available calendars
func CalendarListCalendars(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	script := `
		tell application "Calendar"
			set output to ""
			repeat with cal in calendars
				set output to output & name of cal & linefeed
			end repeat
			return output
		end tell
	`

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to list calendars: %v", err)), nil
	}

	return mcp.NewToolResultText(string(output)), nil
}

// CalendarToday returns today's agenda
func CalendarToday(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	script := `
		set output to ""
		set todayStart to current date
		set time of todayStart to 0
		set todayEnd to todayStart + (1 * days)

		tell application "Calendar"
			repeat with cal in calendars
				set calName to name of cal
				try
					set evts to (every event of cal whose start date ≥ todayStart and start date < todayEnd)
					repeat with evt in evts
						set evtStart to start date of evt
						set evtEnd to end date of evt
						set evtTitle to summary of evt
						set startTime to time string of evtStart
						set endTime to time string of evtEnd
						set output to output & startTime & " - " & endTime & " | " & evtTitle & " [" & calName & "]" & linefeed
					end repeat
				end try
			end repeat
		end tell
		return output
	`

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get today's events: %v", err)), nil
	}

	if len(strings.TrimSpace(string(output))) == 0 {
		return mcp.NewToolResultText("No events scheduled for today"), nil
	}

	return mcp.NewToolResultText("Today's agenda:\n" + string(output)), nil
}

// CalendarSearchEvents searches for events by keyword
func CalendarSearchEvents(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keyword, ok := req.Params.Arguments["keyword"].(string)
	if !ok || keyword == "" {
		return mcp.NewToolResultError("keyword is required"), nil
	}

	days := 30
	if d, ok := req.Params.Arguments["days"].(float64); ok && d > 0 {
		days = int(d)
	}

	script := fmt.Sprintf(`
		set output to ""
		set searchTerm to "%s"
		set startDate to current date
		set endDate to startDate + (%d * days)

		tell application "Calendar"
			repeat with cal in calendars
				set calName to name of cal
				try
					set evts to (every event of cal whose start date ≥ startDate and start date ≤ endDate)
					repeat with evt in evts
						set evtTitle to summary of evt
						if evtTitle contains searchTerm then
							set evtStart to start date of evt
							set output to output & calName & " | " & (evtStart as string) & " | " & evtTitle & linefeed
						end if
					end repeat
				end try
			end repeat
		end tell
		return output
	`, escapeAppleScript(keyword), days)

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.Output()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to search events: %v", err)), nil
	}

	if len(strings.TrimSpace(string(output))) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No events found matching '%s'", keyword)), nil
	}

	return mcp.NewToolResultText(string(output)), nil
}

// CalendarDeleteEvent deletes an event by title (first match)
func CalendarDeleteEvent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, ok := req.Params.Arguments["title"].(string)
	if !ok || title == "" {
		return mcp.NewToolResultError("title is required"), nil
	}

	calendar := ""
	if c, ok := req.Params.Arguments["calendar"].(string); ok {
		calendar = c
	}

	date := ""
	if d, ok := req.Params.Arguments["date"].(string); ok {
		date = d
	}

	var script string
	if calendar != "" && date != "" {
		// Delete from specific calendar on specific date
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid date format, use YYYY-MM-DD: %v", err)), nil
		}
		script = fmt.Sprintf(`
			tell application "Calendar"
				tell calendar "%s"
					set targetDate to date "%s"
					set dayStart to targetDate
					set time of dayStart to 0
					set dayEnd to dayStart + (1 * days)
					set evts to (every event whose summary is "%s" and start date ≥ dayStart and start date < dayEnd)
					if (count of evts) > 0 then
						delete item 1 of evts
						return "Deleted"
					else
						return "NotFound"
					end if
				end tell
			end tell
		`, escapeAppleScript(calendar), t.Format("January 2, 2006"), escapeAppleScript(title))
	} else if calendar != "" {
		// Delete from specific calendar
		script = fmt.Sprintf(`
			tell application "Calendar"
				tell calendar "%s"
					set evts to (every event whose summary is "%s")
					if (count of evts) > 0 then
						delete item 1 of evts
						return "Deleted"
					else
						return "NotFound"
					end if
				end tell
			end tell
		`, escapeAppleScript(calendar), escapeAppleScript(title))
	} else {
		// Search all calendars
		script = fmt.Sprintf(`
			tell application "Calendar"
				repeat with cal in calendars
					set evts to (every event of cal whose summary is "%s")
					if (count of evts) > 0 then
						delete item 1 of evts
						return "Deleted"
					end if
				end repeat
				return "NotFound"
			end tell
		`, escapeAppleScript(title))
	}

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete event: %v - %s", err, output)), nil
	}

	result := strings.TrimSpace(string(output))
	if result == "NotFound" {
		return mcp.NewToolResultText(fmt.Sprintf("Event '%s' not found", title)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Deleted event: %s", title)), nil
}

func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
