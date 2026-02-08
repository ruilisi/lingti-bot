package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pltanton/lingti-bot/internal/agent"
	"github.com/pltanton/lingti-bot/internal/logger"
	"github.com/pltanton/lingti-bot/internal/platforms/dingtalk"
	"github.com/pltanton/lingti-bot/internal/platforms/discord"
	"github.com/pltanton/lingti-bot/internal/platforms/feishu"
	"github.com/pltanton/lingti-bot/internal/platforms/slack"
	"github.com/pltanton/lingti-bot/internal/platforms/telegram"
	"github.com/pltanton/lingti-bot/internal/platforms/wecom"
	"github.com/pltanton/lingti-bot/internal/router"
	"github.com/pltanton/lingti-bot/internal/voice"
	"github.com/spf13/cobra"
)

var (
	slackBotToken      string
	slackAppToken      string
	feishuAppID        string
	feishuAppSecret    string
	telegramToken      string
	discordToken       string
	wecomCorpID        string
	wecomAgentID       string
	wecomSecret        string
	wecomToken         string
	wecomAESKey        string
	wecomPort          int
	dingtalkClientID   string
	dingtalkClientSecret string
	aiProvider         string
	aiAPIKey           string
	aiBaseURL          string
	aiModel            string
	voiceSTTProvider   string
	voiceSTTAPIKey     string
)

var routerCmd = &cobra.Command{
	Use:   "router",
	Short: "Start the message router",
	Long: `Start the message router to receive messages from various platforms
(Slack, Telegram, Discord, Feishu, DingTalk) and respond using AI.

Supported platforms:
  - Slack: SLACK_BOT_TOKEN + SLACK_APP_TOKEN
  - Telegram: TELEGRAM_BOT_TOKEN (supports voice messages with VOICE_STT_PROVIDER)
  - Discord: DISCORD_BOT_TOKEN
  - Feishu: FEISHU_APP_ID + FEISHU_APP_SECRET
  - DingTalk: DINGTALK_CLIENT_ID + DINGTALK_CLIENT_SECRET
  - WeCom: WECOM_CORP_ID + WECOM_AGENT_ID + WECOM_SECRET + WECOM_TOKEN + WECOM_AES_KEY

Voice message transcription (optional):
  - VOICE_STT_PROVIDER: system, openai (default: system)
  - VOICE_STT_API_KEY: API key for cloud STT provider

Required environment variables or flags:
  - AI_PROVIDER: AI provider (claude, deepseek, kimi, qwen) default: claude
  - AI_API_KEY: API Key for the AI provider
  - AI_BASE_URL: Custom API base URL (optional)
  - AI_MODEL: Model name (optional)`,
	Run: runRouter,
}

func init() {
	rootCmd.AddCommand(routerCmd)

	routerCmd.Flags().StringVar(&slackBotToken, "slack-bot-token", "", "Slack Bot Token (or SLACK_BOT_TOKEN env)")
	routerCmd.Flags().StringVar(&slackAppToken, "slack-app-token", "", "Slack App Token (or SLACK_APP_TOKEN env)")
	routerCmd.Flags().StringVar(&feishuAppID, "feishu-app-id", "", "Feishu App ID (or FEISHU_APP_ID env)")
	routerCmd.Flags().StringVar(&feishuAppSecret, "feishu-app-secret", "", "Feishu App Secret (or FEISHU_APP_SECRET env)")
	routerCmd.Flags().StringVar(&telegramToken, "telegram-token", "", "Telegram Bot Token (or TELEGRAM_BOT_TOKEN env)")
	routerCmd.Flags().StringVar(&discordToken, "discord-token", "", "Discord Bot Token (or DISCORD_BOT_TOKEN env)")
	routerCmd.Flags().StringVar(&wecomCorpID, "wecom-corp-id", "", "WeCom Corp ID (or WECOM_CORP_ID env)")
	routerCmd.Flags().StringVar(&wecomAgentID, "wecom-agent-id", "", "WeCom Agent ID (or WECOM_AGENT_ID env)")
	routerCmd.Flags().StringVar(&wecomSecret, "wecom-secret", "", "WeCom Secret (or WECOM_SECRET env)")
	routerCmd.Flags().StringVar(&wecomToken, "wecom-token", "", "WeCom Callback Token (or WECOM_TOKEN env)")
	routerCmd.Flags().StringVar(&wecomAESKey, "wecom-aes-key", "", "WeCom EncodingAESKey (or WECOM_AES_KEY env)")
	routerCmd.Flags().IntVar(&wecomPort, "wecom-port", 0, "WeCom Callback Port (or WECOM_PORT env, default: 8080)")
	routerCmd.Flags().StringVar(&dingtalkClientID, "dingtalk-client-id", "", "DingTalk AppKey (or DINGTALK_CLIENT_ID env)")
	routerCmd.Flags().StringVar(&dingtalkClientSecret, "dingtalk-client-secret", "", "DingTalk AppSecret (or DINGTALK_CLIENT_SECRET env)")
	routerCmd.Flags().StringVar(&aiProvider, "provider", "", "AI provider: claude, deepseek, kimi, qwen (or AI_PROVIDER env)")
	routerCmd.Flags().StringVar(&aiAPIKey, "api-key", "", "AI API Key (or AI_API_KEY env)")
	routerCmd.Flags().StringVar(&aiBaseURL, "base-url", "", "Custom API base URL (or AI_BASE_URL env)")
	routerCmd.Flags().StringVar(&aiModel, "model", "", "Model name (or AI_MODEL env)")
	routerCmd.Flags().StringVar(&voiceSTTProvider, "voice-stt-provider", "", "Voice STT provider: system, openai (or VOICE_STT_PROVIDER env)")
	routerCmd.Flags().StringVar(&voiceSTTAPIKey, "voice-stt-api-key", "", "Voice STT API key (or VOICE_STT_API_KEY env)")
}

func runRouter(cmd *cobra.Command, args []string) {
	// Get tokens from flags or environment
	if slackBotToken == "" {
		slackBotToken = os.Getenv("SLACK_BOT_TOKEN")
	}
	if slackAppToken == "" {
		slackAppToken = os.Getenv("SLACK_APP_TOKEN")
	}
	if feishuAppID == "" {
		feishuAppID = os.Getenv("FEISHU_APP_ID")
	}
	if feishuAppSecret == "" {
		feishuAppSecret = os.Getenv("FEISHU_APP_SECRET")
	}
	if telegramToken == "" {
		telegramToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	}
	if discordToken == "" {
		discordToken = os.Getenv("DISCORD_BOT_TOKEN")
	}
	if wecomCorpID == "" {
		wecomCorpID = os.Getenv("WECOM_CORP_ID")
	}
	if wecomAgentID == "" {
		wecomAgentID = os.Getenv("WECOM_AGENT_ID")
	}
	if wecomSecret == "" {
		wecomSecret = os.Getenv("WECOM_SECRET")
	}
	if wecomToken == "" {
		wecomToken = os.Getenv("WECOM_TOKEN")
	}
	if wecomAESKey == "" {
		wecomAESKey = os.Getenv("WECOM_AES_KEY")
	}
	if wecomPort == 0 {
		if port := os.Getenv("WECOM_PORT"); port != "" {
			fmt.Sscanf(port, "%d", &wecomPort)
		}
	}
	if dingtalkClientID == "" {
		dingtalkClientID = os.Getenv("DINGTALK_CLIENT_ID")
	}
	if dingtalkClientSecret == "" {
		dingtalkClientSecret = os.Getenv("DINGTALK_CLIENT_SECRET")
	}
	if aiProvider == "" {
		aiProvider = os.Getenv("AI_PROVIDER")
	}
	if aiAPIKey == "" {
		aiAPIKey = os.Getenv("AI_API_KEY")
		// Fallback to legacy env var
		if aiAPIKey == "" {
			aiAPIKey = os.Getenv("ANTHROPIC_API_KEY")
		}
	}
	if aiBaseURL == "" {
		aiBaseURL = os.Getenv("AI_BASE_URL")
		if aiBaseURL == "" {
			aiBaseURL = os.Getenv("ANTHROPIC_BASE_URL")
		}
	}
	if aiModel == "" {
		aiModel = os.Getenv("AI_MODEL")
		if aiModel == "" {
			aiModel = os.Getenv("ANTHROPIC_MODEL")
		}
	}
	if voiceSTTProvider == "" {
		voiceSTTProvider = os.Getenv("VOICE_STT_PROVIDER")
	}
	if voiceSTTAPIKey == "" {
		voiceSTTAPIKey = os.Getenv("VOICE_STT_API_KEY")
		if voiceSTTAPIKey == "" && voiceSTTProvider == "openai" {
			voiceSTTAPIKey = os.Getenv("OPENAI_API_KEY")
		}
	}

	// Validate required tokens
	if aiAPIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: AI_API_KEY is required")
		os.Exit(1)
	}

	// Create the AI agent
	aiAgent, err := agent.New(agent.Config{
		Provider: aiProvider,
		APIKey:   aiAPIKey,
		BaseURL:  aiBaseURL,
		Model:    aiModel,
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
		logger.Info("Slack tokens not provided, skipping Slack integration")
	}

	// Register Feishu if tokens are provided
	if feishuAppID != "" && feishuAppSecret != "" {
		feishuPlatform, err := feishu.New(feishu.Config{
			AppID:     feishuAppID,
			AppSecret: feishuAppSecret,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Feishu platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(feishuPlatform)
	} else {
		logger.Info("Feishu tokens not provided, skipping Feishu integration")
	}

	// Register Telegram if token is provided
	if telegramToken != "" {
		// Create voice transcriber if STT provider is configured
		var transcriber *voice.Transcriber
		if voiceSTTProvider != "" {
			var err error
			transcriber, err = voice.NewTranscriber(voice.TranscriberConfig{
				Provider: voiceSTTProvider,
				APIKey:   voiceSTTAPIKey,
			})
			if err != nil {
				logger.Info("Warning: Failed to create voice transcriber: %v", err)
			} else {
				logger.Info("Voice transcription enabled (provider: %s)", voiceSTTProvider)
			}
		}

		telegramPlatform, err := telegram.New(telegram.Config{
			Token:       telegramToken,
			Transcriber: transcriber,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Telegram platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(telegramPlatform)
	} else {
		logger.Info("Telegram token not provided, skipping Telegram integration")
	}

	// Register Discord if token is provided
	if discordToken != "" {
		discordPlatform, err := discord.New(discord.Config{
			Token: discordToken,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Discord platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(discordPlatform)
	} else {
		logger.Info("Discord token not provided, skipping Discord integration")
	}

	// Register WeCom if tokens are provided
	if wecomCorpID != "" && wecomAgentID != "" && wecomSecret != "" && wecomToken != "" && wecomAESKey != "" {
		wecomPlatform, err := wecom.New(wecom.Config{
			CorpID:         wecomCorpID,
			AgentID:        wecomAgentID,
			Secret:         wecomSecret,
			Token:          wecomToken,
			EncodingAESKey: wecomAESKey,
			CallbackPort:   wecomPort,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating WeCom platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(wecomPlatform)
	} else {
		logger.Info("WeCom tokens not provided, skipping WeCom integration")
	}

	// Register DingTalk if tokens are provided
	if dingtalkClientID != "" && dingtalkClientSecret != "" {
		dingtalkPlatform, err := dingtalk.New(dingtalk.Config{
			ClientID:     dingtalkClientID,
			ClientSecret: dingtalkClientSecret,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating DingTalk platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(dingtalkPlatform)
	} else {
		logger.Info("DingTalk tokens not provided, skipping DingTalk integration")
	}

	// Start the router
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := r.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting router: %v\n", err)
		os.Exit(1)
	}

	providerName := aiProvider
	if providerName == "" {
		providerName = "claude"
	}
	modelName := aiModel
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
	logger.Info("Router started. AI Provider: %s, Model: %s", providerName, modelName)
	logger.Info("Press Ctrl+C to stop.")

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down...")
	r.Stop()
}
