package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/pltanton/lingti-bot/internal/tools"
)

const ServerName = "lingti-bot"

// ServerVersion is set via ldflags at build time
var ServerVersion = "1.2.4"

// NewServer creates a new MCP server with all tools registered
func NewServer() *server.MCPServer {
	s := server.NewMCPServer(ServerName, ServerVersion,
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithToolCapabilities(true),
	)

	registerFilesystemTools(s)
	registerShellTools(s)
	registerSystemTools(s)
	registerProcessTools(s)
	registerNetworkTools(s)
	registerCalendarTools(s)
	registerFileManagerTools(s)

	return s
}

func registerFilesystemTools(s *server.MCPServer) {
	// file_read
	s.AddTool(mcp.NewTool("file_read",
		mcp.WithDescription("Read the contents of a file"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file to read")),
	), tools.FileRead)

	// file_write
	s.AddTool(mcp.NewTool("file_write",
		mcp.WithDescription("Write content to a file"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file to write")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content to write to the file")),
	), tools.FileWrite)

	// file_list
	s.AddTool(mcp.NewTool("file_list",
		mcp.WithDescription("List contents of a directory"),
		mcp.WithString("path", mcp.Description("Path to the directory (default: current directory)")),
	), tools.FileList)

	// file_search
	s.AddTool(mcp.NewTool("file_search",
		mcp.WithDescription("Search for files matching a pattern"),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Glob pattern to match (e.g., *.go, *.txt)")),
		mcp.WithString("path", mcp.Description("Directory to search in (default: current directory)")),
	), tools.FileSearch)

	// file_info
	s.AddTool(mcp.NewTool("file_info",
		mcp.WithDescription("Get detailed information about a file"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file")),
	), tools.FileInfo)
}

func registerShellTools(s *server.MCPServer) {
	// shell_execute
	s.AddTool(mcp.NewTool("shell_execute",
		mcp.WithDescription("Execute a shell command"),
		mcp.WithString("command", mcp.Required(), mcp.Description("The command to execute")),
		mcp.WithNumber("timeout", mcp.Description("Timeout in seconds (default: 30)")),
		mcp.WithString("working_directory", mcp.Description("Working directory for the command")),
	), tools.ShellExecute)

	// shell_which
	s.AddTool(mcp.NewTool("shell_which",
		mcp.WithDescription("Find the path of an executable"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the executable to find")),
	), tools.ShellWhich)
}

func registerSystemTools(s *server.MCPServer) {
	// system_info
	s.AddTool(mcp.NewTool("system_info",
		mcp.WithDescription("Get system information (CPU, memory, OS)"),
	), tools.SystemInfo)

	// disk_usage
	s.AddTool(mcp.NewTool("disk_usage",
		mcp.WithDescription("Get disk usage information"),
		mcp.WithString("path", mcp.Description("Path to check (default: /)")),
	), tools.DiskUsage)

	// env_get
	s.AddTool(mcp.NewTool("env_get",
		mcp.WithDescription("Get an environment variable"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the environment variable")),
	), tools.EnvGet)

	// env_list
	s.AddTool(mcp.NewTool("env_list",
		mcp.WithDescription("List all environment variables"),
	), tools.EnvList)
}

func registerProcessTools(s *server.MCPServer) {
	// process_list
	s.AddTool(mcp.NewTool("process_list",
		mcp.WithDescription("List running processes"),
		mcp.WithString("filter", mcp.Description("Filter processes by name (optional)")),
	), tools.ProcessList)

	// process_info
	s.AddTool(mcp.NewTool("process_info",
		mcp.WithDescription("Get detailed information about a process"),
		mcp.WithNumber("pid", mcp.Required(), mcp.Description("Process ID")),
	), tools.ProcessInfo)

	// process_kill
	s.AddTool(mcp.NewTool("process_kill",
		mcp.WithDescription("Kill a process by PID"),
		mcp.WithNumber("pid", mcp.Required(), mcp.Description("Process ID to kill")),
	), tools.ProcessKill)
}

func registerNetworkTools(s *server.MCPServer) {
	// network_interfaces
	s.AddTool(mcp.NewTool("network_interfaces",
		mcp.WithDescription("List network interfaces"),
	), tools.NetworkInterfaces)

	// network_connections
	s.AddTool(mcp.NewTool("network_connections",
		mcp.WithDescription("List active network connections"),
		mcp.WithString("kind", mcp.Description("Connection type: tcp, udp, tcp4, tcp6, udp4, udp6, all (default: all)")),
	), tools.NetworkConnections)

	// network_ping
	s.AddTool(mcp.NewTool("network_ping",
		mcp.WithDescription("Ping a host (TCP connect test)"),
		mcp.WithString("host", mcp.Required(), mcp.Description("Host to ping")),
		mcp.WithString("port", mcp.Description("Port to connect to (default: 80)")),
		mcp.WithNumber("timeout", mcp.Description("Timeout in seconds (default: 5)")),
	), tools.NetworkPing)

	// network_dns_lookup
	s.AddTool(mcp.NewTool("network_dns_lookup",
		mcp.WithDescription("Perform DNS lookup for a hostname"),
		mcp.WithString("hostname", mcp.Required(), mcp.Description("Hostname to look up")),
	), tools.NetworkDNSLookup)
}

func registerCalendarTools(s *server.MCPServer) {
	// calendar_list_events
	s.AddTool(mcp.NewTool("calendar_list_events",
		mcp.WithDescription("List upcoming calendar events from macOS Calendar"),
		mcp.WithNumber("days", mcp.Description("Number of days to look ahead (default: 7)")),
	), tools.CalendarListEvents)

	// calendar_create_event
	s.AddTool(mcp.NewTool("calendar_create_event",
		mcp.WithDescription("Create a new event in macOS Calendar"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Event title")),
		mcp.WithString("start_time", mcp.Required(), mcp.Description("Start time (format: 2024-01-15 14:00)")),
		mcp.WithNumber("duration", mcp.Description("Duration in minutes (default: 60)")),
		mcp.WithString("calendar", mcp.Description("Calendar name (default: Calendar)")),
		mcp.WithString("location", mcp.Description("Event location")),
		mcp.WithString("notes", mcp.Description("Event notes")),
	), tools.CalendarCreateEvent)

	// calendar_list_calendars
	s.AddTool(mcp.NewTool("calendar_list_calendars",
		mcp.WithDescription("List available calendars"),
	), tools.CalendarListCalendars)

	// calendar_today
	s.AddTool(mcp.NewTool("calendar_today",
		mcp.WithDescription("Get today's agenda - all events scheduled for today"),
	), tools.CalendarToday)

	// calendar_search
	s.AddTool(mcp.NewTool("calendar_search",
		mcp.WithDescription("Search for events by keyword"),
		mcp.WithString("keyword", mcp.Required(), mcp.Description("Keyword to search for in event titles")),
		mcp.WithNumber("days", mcp.Description("Number of days to search ahead (default: 30)")),
	), tools.CalendarSearchEvents)

	// calendar_delete_event
	s.AddTool(mcp.NewTool("calendar_delete_event",
		mcp.WithDescription("Delete a calendar event by title"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Exact title of the event to delete")),
		mcp.WithString("calendar", mcp.Description("Calendar name to search in (optional)")),
		mcp.WithString("date", mcp.Description("Date of the event (format: 2024-01-15, optional)")),
	), tools.CalendarDeleteEvent)
}

func registerFileManagerTools(s *server.MCPServer) {
	// file_list_old
	s.AddTool(mcp.NewTool("file_list_old",
		mcp.WithDescription("List files that haven't been modified for a specified number of days"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Directory path to scan (e.g., ~/Desktop)")),
		mcp.WithNumber("days", mcp.Description("Minimum days since last modification (default: 30)")),
	), tools.FileListOld)

	// file_delete_old
	s.AddTool(mcp.NewTool("file_delete_old",
		mcp.WithDescription("Delete files that haven't been modified for a specified number of days"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Directory path to clean (e.g., ~/Desktop)")),
		mcp.WithNumber("days", mcp.Description("Minimum days since last modification (default: 30)")),
		mcp.WithBoolean("include_dirs", mcp.Description("Also delete old directories (default: false)")),
		mcp.WithBoolean("dry_run", mcp.Description("Only show what would be deleted without actually deleting (default: false)")),
	), tools.FileDeleteOld)

	// file_delete_list
	s.AddTool(mcp.NewTool("file_delete_list",
		mcp.WithDescription("Delete specific files by their paths"),
		mcp.WithArray("files", mcp.Required(), mcp.Description("Array of file paths to delete")),
	), tools.FileDeleteList)

	// file_trash
	s.AddTool(mcp.NewTool("file_trash",
		mcp.WithDescription("Move files to Trash instead of permanently deleting (macOS)"),
		mcp.WithArray("files", mcp.Required(), mcp.Description("Array of file paths to move to Trash")),
	), tools.FileMoveToTrash)
}
