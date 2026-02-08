package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pltanton/lingti-bot/internal/agent"
	"github.com/pltanton/lingti-bot/internal/platforms/relay"
	"github.com/pltanton/lingti-bot/internal/router"
	"github.com/spf13/cobra"
)

var (
	relayUserID     string
	relayPlatform   string
	relayServerURL  string
	relayWebhookURL string
	relayAIProvider string
	relayAPIKey     string
	relayBaseURL    string
	relayModel      string
	// WeCom credentials for cloud relay
	relayWeComCorpID  string
	relayWeComAgentID string
	relayWeComSecret  string
	relayWeComToken   string
	relayWeComAESKey  string
)

var relayCmd = &cobra.Command{
	Use:   "relay",
	Short: "Connect to the cloud relay service",
	Long: `Connect to the lingti-bot cloud relay service to process messages
using your local AI API key.

This allows you to use the official lingti-bot service on Feishu/Slack/WeChat
without registering your own bot application.

User Flow (Feishu/Slack/WeChat):
  1. Follow the official lingti-bot on Feishu/Slack/WeChat
  2. Send /whoami to get your user ID
  3. Run: lingti-bot relay --user-id <ID> --platform feishu
  4. Messages are processed locally with your AI API key
  5. Responses are sent back through the relay service

WeCom Cloud Relay:
  For WeCom, no user-id is needed - just provide your credentials.
  This command handles both callback verification AND message processing.

  lingti-bot relay --platform wecom \
    --wecom-corp-id YOUR_CORP_ID \
    --wecom-agent-id YOUR_AGENT_ID \
    --wecom-secret YOUR_SECRET \
    --wecom-token YOUR_TOKEN \
    --wecom-aes-key YOUR_AES_KEY \
    --provider deepseek \
    --api-key YOUR_API_KEY

  1. Run this command first
  2. Configure callback URL in WeCom: https://bot.lingti.com/wecom
  3. Save config in WeCom - verification will succeed automatically
  4. Messages will be processed with your AI provider

Required:
  --user-id     Your user ID from /whoami (not needed for WeCom)
  --platform    Platform type: feishu, slack, wechat, or wecom
  --api-key     AI API key (or AI_API_KEY env)

WeCom Required (when platform=wecom):
  --wecom-corp-id    WeCom Corp ID (or WECOM_CORP_ID env)
  --wecom-agent-id   WeCom Agent ID (or WECOM_AGENT_ID env)
  --wecom-secret     WeCom Secret (or WECOM_SECRET env)
  --wecom-token      WeCom Callback Token (or WECOM_TOKEN env)
  --wecom-aes-key    WeCom Encoding AES Key (or WECOM_AES_KEY env)

Environment variables:
  RELAY_USER_ID        Alternative to --user-id
  RELAY_PLATFORM       Alternative to --platform
  RELAY_SERVER_URL     Custom WebSocket server URL
  RELAY_WEBHOOK_URL    Custom webhook URL
  AI_PROVIDER          AI provider: claude, deepseek, kimi, qwen (default: claude)
  AI_API_KEY           AI API key
  AI_BASE_URL          Custom API base URL
  AI_MODEL             Model name`,
	Run: runRelay,
}

func init() {
	rootCmd.AddCommand(relayCmd)

	relayCmd.Flags().StringVar(&relayUserID, "user-id", "", "User ID from /whoami (required, or RELAY_USER_ID env)")
	relayCmd.Flags().StringVar(&relayPlatform, "platform", "", "Platform: feishu, slack, wechat, or wecom (required, or RELAY_PLATFORM env)")
	relayCmd.Flags().StringVar(&relayServerURL, "server", "", "WebSocket URL (default: wss://bot.lingti.com/ws, or RELAY_SERVER_URL env)")
	relayCmd.Flags().StringVar(&relayWebhookURL, "webhook", "", "Webhook URL (default: https://bot.lingti.com/webhook, or RELAY_WEBHOOK_URL env)")
	relayCmd.Flags().StringVar(&relayAIProvider, "provider", "", "AI provider: claude, deepseek, kimi, qwen (or AI_PROVIDER env)")
	relayCmd.Flags().StringVar(&relayAPIKey, "api-key", "", "AI API key (or AI_API_KEY env)")
	relayCmd.Flags().StringVar(&relayBaseURL, "base-url", "", "Custom API base URL (or AI_BASE_URL env)")
	relayCmd.Flags().StringVar(&relayModel, "model", "", "Model name (or AI_MODEL env)")

	// WeCom credentials for cloud relay
	relayCmd.Flags().StringVar(&relayWeComCorpID, "wecom-corp-id", "", "WeCom Corp ID (or WECOM_CORP_ID env)")
	relayCmd.Flags().StringVar(&relayWeComAgentID, "wecom-agent-id", "", "WeCom Agent ID (or WECOM_AGENT_ID env)")
	relayCmd.Flags().StringVar(&relayWeComSecret, "wecom-secret", "", "WeCom Secret (or WECOM_SECRET env)")
	relayCmd.Flags().StringVar(&relayWeComToken, "wecom-token", "", "WeCom Callback Token (or WECOM_TOKEN env)")
	relayCmd.Flags().StringVar(&relayWeComAESKey, "wecom-aes-key", "", "WeCom Encoding AES Key (or WECOM_AES_KEY env)")
}

func runRelay(cmd *cobra.Command, args []string) {
	// Get values from flags or environment
	if relayUserID == "" {
		relayUserID = os.Getenv("RELAY_USER_ID")
	}
	if relayPlatform == "" {
		relayPlatform = os.Getenv("RELAY_PLATFORM")
	}
	if relayServerURL == "" {
		relayServerURL = os.Getenv("RELAY_SERVER_URL")
	}
	if relayWebhookURL == "" {
		relayWebhookURL = os.Getenv("RELAY_WEBHOOK_URL")
	}
	if relayAIProvider == "" {
		relayAIProvider = os.Getenv("AI_PROVIDER")
	}
	if relayAPIKey == "" {
		relayAPIKey = os.Getenv("AI_API_KEY")
		// Fallback to legacy env var
		if relayAPIKey == "" {
			relayAPIKey = os.Getenv("ANTHROPIC_API_KEY")
		}
	}
	if relayBaseURL == "" {
		relayBaseURL = os.Getenv("AI_BASE_URL")
		if relayBaseURL == "" {
			relayBaseURL = os.Getenv("ANTHROPIC_BASE_URL")
		}
	}
	if relayModel == "" {
		relayModel = os.Getenv("AI_MODEL")
		if relayModel == "" {
			relayModel = os.Getenv("ANTHROPIC_MODEL")
		}
	}

	// Get WeCom credentials from flags or environment
	if relayWeComCorpID == "" {
		relayWeComCorpID = os.Getenv("WECOM_CORP_ID")
	}
	if relayWeComAgentID == "" {
		relayWeComAgentID = os.Getenv("WECOM_AGENT_ID")
	}
	if relayWeComSecret == "" {
		relayWeComSecret = os.Getenv("WECOM_SECRET")
	}
	if relayWeComToken == "" {
		relayWeComToken = os.Getenv("WECOM_TOKEN")
	}
	if relayWeComAESKey == "" {
		relayWeComAESKey = os.Getenv("WECOM_AES_KEY")
	}

	// Validate required parameters
	if relayPlatform == "" {
		fmt.Fprintln(os.Stderr, "Error: --platform is required (feishu, slack, wechat, or wecom)")
		os.Exit(1)
	}
	if relayPlatform != "feishu" && relayPlatform != "slack" && relayPlatform != "wechat" && relayPlatform != "wecom" {
		fmt.Fprintln(os.Stderr, "Error: --platform must be 'feishu', 'slack', 'wechat', or 'wecom'")
		os.Exit(1)
	}
	if relayAPIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: AI API key is required (--api-key or AI_API_KEY env)")
		os.Exit(1)
	}

	// For WeCom, user-id is optional - auto-generate from corp_id
	// For other platforms, user-id is required
	if relayUserID == "" {
		if relayPlatform == "wecom" && relayWeComCorpID != "" {
			relayUserID = "wecom-" + relayWeComCorpID
		} else if relayPlatform != "wecom" {
			fmt.Fprintln(os.Stderr, "Error: --user-id is required (get it from /whoami)")
			os.Exit(1)
		}
	}

	// Validate WeCom credentials when platform is wecom
	if relayPlatform == "wecom" {
		missing := []string{}
		if relayWeComCorpID == "" {
			missing = append(missing, "--wecom-corp-id")
		}
		if relayWeComAgentID == "" {
			missing = append(missing, "--wecom-agent-id")
		}
		if relayWeComSecret == "" {
			missing = append(missing, "--wecom-secret")
		}
		if relayWeComToken == "" {
			missing = append(missing, "--wecom-token")
		}
		if relayWeComAESKey == "" {
			missing = append(missing, "--wecom-aes-key")
		}
		if len(missing) > 0 {
			fmt.Fprintf(os.Stderr, "Error: WeCom credentials required for cloud relay: %v\n", missing)
			fmt.Fprintln(os.Stderr, "Configure callback URL in WeCom: https://bot.lingti.com/wecom")
			os.Exit(1)
		}
	}

	// Create the AI agent
	aiAgent, err := agent.New(agent.Config{
		Provider: relayAIProvider,
		APIKey:   relayAPIKey,
		BaseURL:  relayBaseURL,
		Model:    relayModel,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating agent: %v\n", err)
		os.Exit(1)
	}

	// Resolve provider and model names
	providerName := relayAIProvider
	if providerName == "" {
		providerName = "claude"
	}
	modelName := relayModel
	if modelName == "" {
		switch providerName {
		case "deepseek":
			modelName = "deepseek-chat"
		case "kimi", "moonshot":
			modelName = "moonshot-v1-8k"
		default:
			modelName = "claude-sonnet-4-20250514"
		}
	}

	// Create the router with the agent as message handler
	r := router.New(aiAgent.HandleMessage)

	// Create and register relay platform
	relayPlatformInstance, err := relay.New(relay.Config{
		UserID:       relayUserID,
		Platform:     relayPlatform,
		ServerURL:    relayServerURL,
		WebhookURL:   relayWebhookURL,
		AIProvider:   providerName,
		AIModel:      modelName,
		WeComCorpID:  relayWeComCorpID,
		WeComAgentID: relayWeComAgentID,
		WeComSecret:  relayWeComSecret,
		WeComToken:   relayWeComToken,
		WeComAESKey:  relayWeComAESKey,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating relay platform: %v\n", err)
		os.Exit(1)
	}
	r.Register(relayPlatformInstance)

	// Start the router
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := r.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting relay: %v\n", err)
		os.Exit(1)
	}

	log.Printf("Relay connected. User: %s, Platform: %s", relayUserID, relayPlatform)
	log.Printf("AI Provider: %s, Model: %s", providerName, modelName)
	log.Println("Press Ctrl+C to stop.")

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	r.Stop()
}
