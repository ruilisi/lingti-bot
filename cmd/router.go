package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pltanton/lingti-bot/internal/agent"
	"github.com/pltanton/lingti-bot/internal/platforms/slack"
	"github.com/pltanton/lingti-bot/internal/router"
	"github.com/spf13/cobra"
)

var (
	slackBotToken string
	slackAppToken string
	claudeAPIKey  string
	claudeBaseURL string
	claudeModel   string
)

var routerCmd = &cobra.Command{
	Use:   "router",
	Short: "Start the message router",
	Long: `Start the message router to receive messages from various platforms
(Slack, Telegram, Discord, etc.) and respond using AI.

Required environment variables or flags:
  - SLACK_BOT_TOKEN: Slack Bot Token (xoxb-...)
  - SLACK_APP_TOKEN: Slack App Token (xapp-...)
  - ANTHROPIC_API_KEY: Claude API Key
  - ANTHROPIC_BASE_URL: Custom API base URL (optional)`,
	Run: runRouter,
}

func init() {
	rootCmd.AddCommand(routerCmd)

	routerCmd.Flags().StringVar(&slackBotToken, "slack-bot-token", "", "Slack Bot Token (or SLACK_BOT_TOKEN env)")
	routerCmd.Flags().StringVar(&slackAppToken, "slack-app-token", "", "Slack App Token (or SLACK_APP_TOKEN env)")
	routerCmd.Flags().StringVar(&claudeAPIKey, "api-key", "", "Claude API Key (or ANTHROPIC_API_KEY env)")
	routerCmd.Flags().StringVar(&claudeBaseURL, "base-url", "", "Custom API base URL (or ANTHROPIC_BASE_URL env)")
	routerCmd.Flags().StringVar(&claudeModel, "model", "", "Claude model to use (or ANTHROPIC_MODEL env)")
}

func runRouter(cmd *cobra.Command, args []string) {
	// Get tokens from flags or environment
	if slackBotToken == "" {
		slackBotToken = os.Getenv("SLACK_BOT_TOKEN")
	}
	if slackAppToken == "" {
		slackAppToken = os.Getenv("SLACK_APP_TOKEN")
	}
	if claudeAPIKey == "" {
		claudeAPIKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if claudeBaseURL == "" {
		claudeBaseURL = os.Getenv("ANTHROPIC_BASE_URL")
	}
	if claudeModel == "" {
		claudeModel = os.Getenv("ANTHROPIC_MODEL")
	}

	// Validate required tokens
	if claudeAPIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: ANTHROPIC_API_KEY is required")
		os.Exit(1)
	}

	// Create the AI agent
	aiAgent, err := agent.New(agent.Config{
		APIKey:  claudeAPIKey,
		BaseURL: claudeBaseURL,
		Model:   claudeModel,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating agent: %v\n", err)
		os.Exit(1)
	}

	// Create the router with the agent as message handler
	r := router.New(aiAgent.HandleMessage)

	// Register Slack if tokens are provided
	if slackBotToken != "" && slackAppToken != "" {
		slackPlatform, err := slack.New(slack.Config{
			BotToken: slackBotToken,
			AppToken: slackAppToken,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Slack platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(slackPlatform)
	} else {
		log.Println("Slack tokens not provided, skipping Slack integration")
	}

	// Start the router
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := r.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting router: %v\n", err)
		os.Exit(1)
	}

	log.Println("Router started. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	r.Stop()
}
