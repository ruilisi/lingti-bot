package cmd

import (
	"fmt"
	"os"

	"github.com/pltanton/lingti-bot/internal/logger"
	"github.com/spf13/cobra"
)

var (
	logLevel    string
	autoApprove bool
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
}

// IsAutoApprove returns true if auto-approve mode is enabled globally
func IsAutoApprove() bool {
	return autoApprove
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
