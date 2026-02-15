package cmd

import (
	"fmt"
	"os"

	"github.com/pltanton/lingti-bot/internal/config"
	"github.com/pltanton/lingti-bot/internal/logger"
	"github.com/pltanton/lingti-bot/internal/mcp"
	"github.com/spf13/cobra"
)

var (
	logLevel         string
	autoApprove      bool
	disableFileTools bool
)

var rootCmd = &cobra.Command{
	Use:   "lingti-bot",
	Short: "MCP server for system resources",
	Long: `lingti-bot is an MCP (Model Context Protocol) server that exposes
computer system resources to AI assistants.

It provides tools for:
  - File operations (read, write, list, search)
  - Shell command execution
  - System information (CPU, memory, disk)
  - Process management
  - Network information`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Parse and set log level
		level, err := logger.ParseLevel(logLevel)
		if err != nil {
			return err
		}
		logger.SetLevel(level)
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logLevel, "log", "info",
		"Log level: trace, debug, info, warn, error, fatal, panic")
	rootCmd.PersistentFlags().BoolVarP(&autoApprove, "yes", "y", false,
		"Automatically approve all operations without prompting (skip security checks)")
	rootCmd.PersistentFlags().BoolVar(&disableFileTools, "no-files", false,
		"Disable all file operation tools")
}

// IsAutoApprove returns true if auto-approve mode is enabled globally
func IsAutoApprove() bool {
	return autoApprove
}

// loadAllowedPaths returns security allowed_paths from config file.
func loadAllowedPaths() []string {
	if cfg, err := config.Load(); err == nil {
		return cfg.Security.AllowedPaths
	}
	return nil
}

// loadDisableFileTools returns true if file tools are disabled via flag or config.
func loadDisableFileTools() bool {
	if disableFileTools {
		return true
	}
	if cfg, err := config.Load(); err == nil {
		return cfg.Security.DisableFileTools
	}
	return false
}

// loadSecurityOptions returns MCP security options from config file.
func loadSecurityOptions() mcp.SecurityOptions {
	cfg, err := config.Load()
	if err != nil {
		return mcp.SecurityOptions{}
	}
	return mcp.SecurityOptions{
		AllowedPaths:     cfg.Security.AllowedPaths,
		DisableFileTools: cfg.Security.DisableFileTools,
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
