package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/pltanton/lingti-bot/internal/agent"
	"github.com/pltanton/lingti-bot/internal/browser"
	"github.com/pltanton/lingti-bot/internal/config"
	cronpkg "github.com/pltanton/lingti-bot/internal/cron"
	"github.com/pltanton/lingti-bot/internal/logger"
	"github.com/pltanton/lingti-bot/internal/platforms/dingtalk"
	"github.com/pltanton/lingti-bot/internal/platforms/discord"
	"github.com/pltanton/lingti-bot/internal/platforms/feishu"
	"github.com/pltanton/lingti-bot/internal/platforms/googlechat"
	"github.com/pltanton/lingti-bot/internal/platforms/imessage"
	"github.com/pltanton/lingti-bot/internal/platforms/slack"
	"github.com/pltanton/lingti-bot/internal/platforms/telegram"
	"github.com/pltanton/lingti-bot/internal/platforms/wecom"
	"github.com/pltanton/lingti-bot/internal/platforms/line"
	"github.com/pltanton/lingti-bot/internal/platforms/mattermost"
	"github.com/pltanton/lingti-bot/internal/platforms/nextcloud"
	"github.com/pltanton/lingti-bot/internal/platforms/nostr"
	signalplatform "github.com/pltanton/lingti-bot/internal/platforms/signal"
	"github.com/pltanton/lingti-bot/internal/platforms/zalo"
	"github.com/pltanton/lingti-bot/internal/platforms/twitch"
	"github.com/pltanton/lingti-bot/internal/platforms/matrix"
	"github.com/pltanton/lingti-bot/internal/platforms/teams"
	"github.com/pltanton/lingti-bot/internal/platforms/whatsapp"
	"github.com/pltanton/lingti-bot/internal/router"
	"github.com/pltanton/lingti-bot/internal/voice"
	"github.com/spf13/cobra"
)

var (
	slackBotToken        string
	slackAppToken        string
	feishuAppID          string
	feishuAppSecret      string
	telegramToken        string
	discordToken         string
	wecomCorpID          string
	wecomAgentID         string
	wecomSecret          string
	wecomToken           string
	wecomAESKey          string
	wecomPort            int
	dingtalkClientID     string
	dingtalkClientSecret string
	lineChannelSecret    string
	lineChannelToken     string
	teamsAppID           string
	teamsAppPassword     string
	teamsTenantID        string
	matrixHomeserverURL  string
	matrixUserID         string
	matrixAccessToken    string
	googlechatProjectID       string
	googlechatCredentialsFile string
	mattermostServerURL  string
	mattermostToken      string
	mattermostTeamName   string
	blueBubblesURL       string
	blueBubblesPassword  string
	signalAPIURL         string
	signalPhoneNumber    string
	twitchToken          string
	twitchChannel        string
	twitchBotName        string
	nostrPrivateKey      string
	nostrRelays          string
	zaloAppID            string
	zaloSecretKey        string
	zaloAccessToken      string
	nextcloudServerURL   string
	nextcloudUsername    string
	nextcloudPassword    string
	nextcloudRoomToken   string
	whatsappPhoneID      string
	whatsappAccessToken  string
	whatsappVerifyToken  string
	aiProvider           string
	aiAPIKey             string
	aiBaseURL            string
	aiModel              string
	aiInstructions       string
	voiceSTTProvider     string
	voiceSTTAPIKey       string
	browserDebugDir      string
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
	routerCmd.Flags().StringVar(&lineChannelSecret, "line-channel-secret", "", "LINE Channel Secret (or LINE_CHANNEL_SECRET env)")
	routerCmd.Flags().StringVar(&lineChannelToken, "line-channel-token", "", "LINE Channel Token (or LINE_CHANNEL_TOKEN env)")
	routerCmd.Flags().StringVar(&teamsAppID, "teams-app-id", "", "Teams App ID (or TEAMS_APP_ID env)")
	routerCmd.Flags().StringVar(&teamsAppPassword, "teams-app-password", "", "Teams App Password (or TEAMS_APP_PASSWORD env)")
	routerCmd.Flags().StringVar(&teamsTenantID, "teams-tenant-id", "", "Teams Tenant ID (or TEAMS_TENANT_ID env)")
	routerCmd.Flags().StringVar(&matrixHomeserverURL, "matrix-homeserver-url", "", "Matrix Homeserver URL (or MATRIX_HOMESERVER_URL env)")
	routerCmd.Flags().StringVar(&matrixUserID, "matrix-user-id", "", "Matrix User ID (or MATRIX_USER_ID env)")
	routerCmd.Flags().StringVar(&matrixAccessToken, "matrix-access-token", "", "Matrix Access Token (or MATRIX_ACCESS_TOKEN env)")
	routerCmd.Flags().StringVar(&googlechatProjectID, "googlechat-project-id", "", "Google Chat Project ID (or GOOGLE_CHAT_PROJECT_ID env)")
	routerCmd.Flags().StringVar(&googlechatCredentialsFile, "googlechat-credentials-file", "", "Google Chat Credentials File (or GOOGLE_CHAT_CREDENTIALS_FILE env)")
	routerCmd.Flags().StringVar(&mattermostServerURL, "mattermost-server-url", "", "Mattermost Server URL (or MATTERMOST_SERVER_URL env)")
	routerCmd.Flags().StringVar(&mattermostToken, "mattermost-token", "", "Mattermost Token (or MATTERMOST_TOKEN env)")
	routerCmd.Flags().StringVar(&mattermostTeamName, "mattermost-team-name", "", "Mattermost Team Name (or MATTERMOST_TEAM_NAME env)")
	routerCmd.Flags().StringVar(&blueBubblesURL, "bluebubbles-url", "", "BlueBubbles Server URL (or BLUEBUBBLES_URL env)")
	routerCmd.Flags().StringVar(&blueBubblesPassword, "bluebubbles-password", "", "BlueBubbles Password (or BLUEBUBBLES_PASSWORD env)")
	routerCmd.Flags().StringVar(&signalAPIURL, "signal-api-url", "", "Signal API URL (or SIGNAL_API_URL env)")
	routerCmd.Flags().StringVar(&signalPhoneNumber, "signal-phone-number", "", "Signal Phone Number (or SIGNAL_PHONE_NUMBER env)")
	routerCmd.Flags().StringVar(&twitchToken, "twitch-token", "", "Twitch OAuth Token (or TWITCH_TOKEN env)")
	routerCmd.Flags().StringVar(&twitchChannel, "twitch-channel", "", "Twitch Channel (or TWITCH_CHANNEL env)")
	routerCmd.Flags().StringVar(&twitchBotName, "twitch-bot-name", "", "Twitch Bot Name (or TWITCH_BOT_NAME env)")
	routerCmd.Flags().StringVar(&nostrPrivateKey, "nostr-private-key", "", "NOSTR Private Key (or NOSTR_PRIVATE_KEY env)")
	routerCmd.Flags().StringVar(&nostrRelays, "nostr-relays", "", "NOSTR Relay URLs (or NOSTR_RELAYS env)")
	routerCmd.Flags().StringVar(&zaloAppID, "zalo-app-id", "", "Zalo App ID (or ZALO_APP_ID env)")
	routerCmd.Flags().StringVar(&zaloSecretKey, "zalo-secret-key", "", "Zalo Secret Key (or ZALO_SECRET_KEY env)")
	routerCmd.Flags().StringVar(&zaloAccessToken, "zalo-access-token", "", "Zalo Access Token (or ZALO_ACCESS_TOKEN env)")
	routerCmd.Flags().StringVar(&nextcloudServerURL, "nextcloud-server-url", "", "Nextcloud Server URL (or NEXTCLOUD_SERVER_URL env)")
	routerCmd.Flags().StringVar(&nextcloudUsername, "nextcloud-username", "", "Nextcloud Username (or NEXTCLOUD_USERNAME env)")
	routerCmd.Flags().StringVar(&nextcloudPassword, "nextcloud-password", "", "Nextcloud Password (or NEXTCLOUD_PASSWORD env)")
	routerCmd.Flags().StringVar(&nextcloudRoomToken, "nextcloud-room-token", "", "Nextcloud Room Token (or NEXTCLOUD_ROOM_TOKEN env)")
	routerCmd.Flags().StringVar(&whatsappPhoneID, "whatsapp-phone-id", "", "WhatsApp Phone Number ID (or WHATSAPP_PHONE_NUMBER_ID env)")
	routerCmd.Flags().StringVar(&whatsappAccessToken, "whatsapp-access-token", "", "WhatsApp Access Token (or WHATSAPP_ACCESS_TOKEN env)")
	routerCmd.Flags().StringVar(&whatsappVerifyToken, "whatsapp-verify-token", "", "WhatsApp Verify Token (or WHATSAPP_VERIFY_TOKEN env)")
	routerCmd.Flags().StringVar(&aiProvider, "provider", "", "AI provider: claude, deepseek, kimi, qwen (or AI_PROVIDER env)")
	routerCmd.Flags().StringVar(&aiAPIKey, "api-key", "", "AI API Key (or AI_API_KEY env)")
	routerCmd.Flags().StringVar(&aiBaseURL, "base-url", "", "Custom API base URL (or AI_BASE_URL env)")
	routerCmd.Flags().StringVar(&aiModel, "model", "", "Model name (or AI_MODEL env)")
	routerCmd.Flags().StringVar(&aiInstructions, "instructions", "", "Path to custom instructions file appended to system prompt")
	routerCmd.Flags().StringVar(&voiceSTTProvider, "voice-stt-provider", "", "Voice STT provider: system, openai (or VOICE_STT_PROVIDER env)")
	routerCmd.Flags().StringVar(&voiceSTTAPIKey, "voice-stt-api-key", "", "Voice STT API key (or VOICE_STT_API_KEY env)")
	routerCmd.Flags().StringVar(&browserDebugDir, "debug-dir", "", "Directory for debug screenshots (or BROWSER_DEBUG_DIR env, default: /tmp/lingti-bot on Unix)")
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
	if nextcloudServerURL == "" {
		nextcloudServerURL = os.Getenv("NEXTCLOUD_SERVER_URL")
	}
	if nextcloudUsername == "" {
		nextcloudUsername = os.Getenv("NEXTCLOUD_USERNAME")
	}
	if nextcloudPassword == "" {
		nextcloudPassword = os.Getenv("NEXTCLOUD_PASSWORD")
	}
	if nextcloudRoomToken == "" {
		nextcloudRoomToken = os.Getenv("NEXTCLOUD_ROOM_TOKEN")
	}
	if zaloAppID == "" {
		zaloAppID = os.Getenv("ZALO_APP_ID")
	}
	if zaloSecretKey == "" {
		zaloSecretKey = os.Getenv("ZALO_SECRET_KEY")
	}
	if zaloAccessToken == "" {
		zaloAccessToken = os.Getenv("ZALO_ACCESS_TOKEN")
	}
	if nostrPrivateKey == "" {
		nostrPrivateKey = os.Getenv("NOSTR_PRIVATE_KEY")
	}
	if nostrRelays == "" {
		nostrRelays = os.Getenv("NOSTR_RELAYS")
	}
	if twitchToken == "" {
		twitchToken = os.Getenv("TWITCH_TOKEN")
	}
	if twitchChannel == "" {
		twitchChannel = os.Getenv("TWITCH_CHANNEL")
	}
	if twitchBotName == "" {
		twitchBotName = os.Getenv("TWITCH_BOT_NAME")
	}
	if signalAPIURL == "" {
		signalAPIURL = os.Getenv("SIGNAL_API_URL")
	}
	if signalPhoneNumber == "" {
		signalPhoneNumber = os.Getenv("SIGNAL_PHONE_NUMBER")
	}
	if blueBubblesURL == "" {
		blueBubblesURL = os.Getenv("BLUEBUBBLES_URL")
	}
	if blueBubblesPassword == "" {
		blueBubblesPassword = os.Getenv("BLUEBUBBLES_PASSWORD")
	}
	if mattermostServerURL == "" {
		mattermostServerURL = os.Getenv("MATTERMOST_SERVER_URL")
	}
	if mattermostToken == "" {
		mattermostToken = os.Getenv("MATTERMOST_TOKEN")
	}
	if mattermostTeamName == "" {
		mattermostTeamName = os.Getenv("MATTERMOST_TEAM_NAME")
	}
	if googlechatProjectID == "" {
		googlechatProjectID = os.Getenv("GOOGLE_CHAT_PROJECT_ID")
	}
	if googlechatCredentialsFile == "" {
		googlechatCredentialsFile = os.Getenv("GOOGLE_CHAT_CREDENTIALS_FILE")
	}
	if matrixHomeserverURL == "" {
		matrixHomeserverURL = os.Getenv("MATRIX_HOMESERVER_URL")
	}
	if matrixUserID == "" {
		matrixUserID = os.Getenv("MATRIX_USER_ID")
	}
	if matrixAccessToken == "" {
		matrixAccessToken = os.Getenv("MATRIX_ACCESS_TOKEN")
	}
	if teamsAppID == "" {
		teamsAppID = os.Getenv("TEAMS_APP_ID")
	}
	if teamsAppPassword == "" {
		teamsAppPassword = os.Getenv("TEAMS_APP_PASSWORD")
	}
	if teamsTenantID == "" {
		teamsTenantID = os.Getenv("TEAMS_TENANT_ID")
	}
	if lineChannelSecret == "" {
		lineChannelSecret = os.Getenv("LINE_CHANNEL_SECRET")
	}
	if lineChannelToken == "" {
		lineChannelToken = os.Getenv("LINE_CHANNEL_TOKEN")
	}
	if whatsappPhoneID == "" {
		whatsappPhoneID = os.Getenv("WHATSAPP_PHONE_NUMBER_ID")
	}
	if whatsappAccessToken == "" {
		whatsappAccessToken = os.Getenv("WHATSAPP_ACCESS_TOKEN")
	}
	if whatsappVerifyToken == "" {
		whatsappVerifyToken = os.Getenv("WHATSAPP_VERIFY_TOKEN")
	}
	if aiProvider == "" {
		aiProvider = os.Getenv("AI_PROVIDER")
	}
	if aiAPIKey == "" {
		aiAPIKey = os.Getenv("AI_API_KEY")
		// Fallback: ANTHROPIC_OAUTH_TOKEN (setup token) > ANTHROPIC_API_KEY
		if aiAPIKey == "" {
			aiAPIKey = os.Getenv("ANTHROPIC_OAUTH_TOKEN")
		}
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

	// Fallback to saved config file
	if savedCfg, err := config.Load(); err == nil {
		if aiProvider == "" {
			aiProvider = savedCfg.AI.Provider
		}
		if aiAPIKey == "" {
			aiAPIKey = savedCfg.AI.APIKey
		}
		if aiBaseURL == "" {
			aiBaseURL = savedCfg.AI.BaseURL
		}
		if aiModel == "" {
			aiModel = savedCfg.AI.Model
		}
		if slackBotToken == "" {
			slackBotToken = savedCfg.Platforms.Slack.BotToken
		}
		if slackAppToken == "" {
			slackAppToken = savedCfg.Platforms.Slack.AppToken
		}
		if feishuAppID == "" {
			feishuAppID = savedCfg.Platforms.Feishu.AppID
		}
		if feishuAppSecret == "" {
			feishuAppSecret = savedCfg.Platforms.Feishu.AppSecret
		}
		if telegramToken == "" {
			telegramToken = savedCfg.Platforms.Telegram.Token
		}
		if discordToken == "" {
			discordToken = savedCfg.Platforms.Discord.Token
		}
		if wecomCorpID == "" {
			wecomCorpID = savedCfg.Platforms.WeCom.CorpID
		}
		if wecomAgentID == "" {
			wecomAgentID = savedCfg.Platforms.WeCom.AgentID
		}
		if wecomSecret == "" {
			wecomSecret = savedCfg.Platforms.WeCom.Secret
		}
		if wecomToken == "" {
			wecomToken = savedCfg.Platforms.WeCom.Token
		}
		if wecomAESKey == "" {
			wecomAESKey = savedCfg.Platforms.WeCom.AESKey
		}
		if wecomPort == 0 && savedCfg.Platforms.WeCom.CallbackPort != 0 {
			wecomPort = savedCfg.Platforms.WeCom.CallbackPort
		}
		if dingtalkClientID == "" {
			dingtalkClientID = savedCfg.Platforms.DingTalk.ClientID
		}
		if dingtalkClientSecret == "" {
			dingtalkClientSecret = savedCfg.Platforms.DingTalk.ClientSecret
		}
		if nextcloudServerURL == "" {
			nextcloudServerURL = savedCfg.Platforms.Nextcloud.ServerURL
		}
		if nextcloudUsername == "" {
			nextcloudUsername = savedCfg.Platforms.Nextcloud.Username
		}
		if nextcloudPassword == "" {
			nextcloudPassword = savedCfg.Platforms.Nextcloud.Password
		}
		if nextcloudRoomToken == "" {
			nextcloudRoomToken = savedCfg.Platforms.Nextcloud.RoomToken
		}
		if zaloAppID == "" {
			zaloAppID = savedCfg.Platforms.Zalo.AppID
		}
		if zaloSecretKey == "" {
			zaloSecretKey = savedCfg.Platforms.Zalo.SecretKey
		}
		if zaloAccessToken == "" {
			zaloAccessToken = savedCfg.Platforms.Zalo.AccessToken
		}
		if nostrPrivateKey == "" {
			nostrPrivateKey = savedCfg.Platforms.NOSTR.PrivateKey
		}
		if nostrRelays == "" {
			nostrRelays = savedCfg.Platforms.NOSTR.Relays
		}
		if twitchToken == "" {
			twitchToken = savedCfg.Platforms.Twitch.Token
		}
		if twitchChannel == "" {
			twitchChannel = savedCfg.Platforms.Twitch.Channel
		}
		if twitchBotName == "" {
			twitchBotName = savedCfg.Platforms.Twitch.BotName
		}
		if signalAPIURL == "" {
			signalAPIURL = savedCfg.Platforms.Signal.APIURL
		}
		if signalPhoneNumber == "" {
			signalPhoneNumber = savedCfg.Platforms.Signal.PhoneNumber
		}
		if blueBubblesURL == "" {
			blueBubblesURL = savedCfg.Platforms.IMessage.BlueBubblesURL
		}
		if blueBubblesPassword == "" {
			blueBubblesPassword = savedCfg.Platforms.IMessage.BlueBubblesPassword
		}
		if mattermostServerURL == "" {
			mattermostServerURL = savedCfg.Platforms.Mattermost.ServerURL
		}
		if mattermostToken == "" {
			mattermostToken = savedCfg.Platforms.Mattermost.Token
		}
		if mattermostTeamName == "" {
			mattermostTeamName = savedCfg.Platforms.Mattermost.TeamName
		}
		if googlechatProjectID == "" {
			googlechatProjectID = savedCfg.Platforms.GoogleChat.ProjectID
		}
		if googlechatCredentialsFile == "" {
			googlechatCredentialsFile = savedCfg.Platforms.GoogleChat.CredentialsFile
		}
		if matrixHomeserverURL == "" {
			matrixHomeserverURL = savedCfg.Platforms.Matrix.HomeserverURL
		}
		if matrixUserID == "" {
			matrixUserID = savedCfg.Platforms.Matrix.UserID
		}
		if matrixAccessToken == "" {
			matrixAccessToken = savedCfg.Platforms.Matrix.AccessToken
		}
		if teamsAppID == "" {
			teamsAppID = savedCfg.Platforms.Teams.AppID
		}
		if teamsAppPassword == "" {
			teamsAppPassword = savedCfg.Platforms.Teams.AppPassword
		}
		if teamsTenantID == "" {
			teamsTenantID = savedCfg.Platforms.Teams.TenantID
		}
		if lineChannelSecret == "" {
			lineChannelSecret = savedCfg.Platforms.LINE.ChannelSecret
		}
		if lineChannelToken == "" {
			lineChannelToken = savedCfg.Platforms.LINE.ChannelToken
		}
		if whatsappPhoneID == "" {
			whatsappPhoneID = savedCfg.Platforms.WhatsApp.PhoneNumberID
		}
		if whatsappAccessToken == "" {
			whatsappAccessToken = savedCfg.Platforms.WhatsApp.AccessToken
		}
		if whatsappVerifyToken == "" {
			whatsappVerifyToken = savedCfg.Platforms.WhatsApp.VerifyToken
		}
	}

	// Check if debug mode is enabled (log level or env var)
	debugEnabled := logger.IsDebug()
	if !debugEnabled {
		if os.Getenv("BROWSER_DEBUG") == "1" || os.Getenv("BROWSER_DEBUG") == "true" {
			debugEnabled = true
		}
	}
	if browserDebugDir == "" {
		browserDebugDir = os.Getenv("BROWSER_DEBUG_DIR")
	}

	// Validate required tokens
	if aiAPIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: AI_API_KEY is required")
		os.Exit(1)
	}

	// Load custom instructions if specified
	var customInstructions string
	if aiInstructions != "" {
		data, err := os.ReadFile(aiInstructions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading instructions file: %v\n", err)
			os.Exit(1)
		}
		customInstructions = string(data)
		logger.Info("Loaded custom instructions from %s (%d bytes)", aiInstructions, len(data))
	}

	// Create the AI agent
	aiAgent, err := agent.New(agent.Config{
		Provider:           aiProvider,
		APIKey:             aiAPIKey,
		BaseURL:            aiBaseURL,
		Model:              aiModel,
		AutoApprove:        IsAutoApprove(),
		CustomInstructions: customInstructions,
		AllowedPaths:       loadAllowedPaths(),
		DisableFileTools:   loadDisableFileTools(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating agent: %v\n", err)
		os.Exit(1)
	}

	// Configure browser debug mode if enabled
	if debugEnabled {
		// Import is needed at the top: "github.com/pltanton/lingti-bot/internal/browser"
		if err := setupBrowserDebug(browserDebugDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to setup browser debug: %v\n", err)
		} else {
			logger.Info("Browser debug mode enabled, screenshots will be saved to: %s", browserDebugDir)
		}
	}

	// Create the router with the agent as message handler
	r := router.New(aiAgent.HandleMessage)

	// Initialize cron scheduler
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}
	cronPath := filepath.Join(homeDir, ".lingti.db")
	cronStore, err := cronpkg.NewStore(cronPath)
	if err != nil {
		logger.Error("Failed to open cron store: %v", err)
		os.Exit(1)
	}
	cronNotifier := agent.NewRouterCronNotifier(r)
	cronScheduler := cronpkg.NewScheduler(cronStore, aiAgent, aiAgent, cronNotifier)
	aiAgent.SetCronScheduler(cronScheduler)
	if err := cronScheduler.Start(); err != nil {
		logger.Warn("Failed to start cron scheduler: %v", err)
	}

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
				logger.Warn("Failed to create voice transcriber: %v", err)
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

	// Register Nextcloud Talk if tokens are provided
	if nextcloudServerURL != "" && nextcloudUsername != "" && nextcloudPassword != "" && nextcloudRoomToken != "" {
		nextcloudPlatform, err := nextcloud.New(nextcloud.Config{
			ServerURL: nextcloudServerURL,
			Username:  nextcloudUsername,
			Password:  nextcloudPassword,
			RoomToken: nextcloudRoomToken,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Nextcloud Talk platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(nextcloudPlatform)
	} else {
		logger.Info("Nextcloud Talk tokens not provided, skipping Nextcloud Talk integration")
	}

	// Register Zalo if tokens are provided
	if zaloAppID != "" && zaloAccessToken != "" {
		zaloPlatform, err := zalo.New(zalo.Config{
			AppID:       zaloAppID,
			SecretKey:   zaloSecretKey,
			AccessToken: zaloAccessToken,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Zalo platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(zaloPlatform)
	} else {
		logger.Info("Zalo tokens not provided, skipping Zalo integration")
	}

	// Register NOSTR if tokens are provided
	if nostrPrivateKey != "" && nostrRelays != "" {
		nostrPlatform, err := nostr.New(nostr.Config{
			PrivateKey: nostrPrivateKey,
			Relays:     nostrRelays,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating NOSTR platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(nostrPlatform)
	} else {
		logger.Info("NOSTR tokens not provided, skipping NOSTR integration")
	}

	// Register Twitch if tokens are provided
	if twitchToken != "" && twitchChannel != "" && twitchBotName != "" {
		twitchPlatform, err := twitch.New(twitch.Config{
			Token:   twitchToken,
			Channel: twitchChannel,
			BotName: twitchBotName,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Twitch platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(twitchPlatform)
	} else {
		logger.Info("Twitch tokens not provided, skipping Twitch integration")
	}

	// Register Signal if tokens are provided
	if signalAPIURL != "" && signalPhoneNumber != "" {
		signalPlatform, err := signalplatform.New(signalplatform.Config{
			APIURL:      signalAPIURL,
			PhoneNumber: signalPhoneNumber,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Signal platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(signalPlatform)
	} else {
		logger.Info("Signal tokens not provided, skipping Signal integration")
	}

	// Register iMessage if tokens are provided
	if blueBubblesURL != "" && blueBubblesPassword != "" {
		imessagePlatform, err := imessage.New(imessage.Config{
			BlueBubblesURL:      blueBubblesURL,
			BlueBubblesPassword: blueBubblesPassword,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating iMessage platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(imessagePlatform)
	} else {
		logger.Info("iMessage tokens not provided, skipping iMessage integration")
	}

	// Register Mattermost if tokens are provided
	if mattermostServerURL != "" && mattermostToken != "" {
		mattermostPlatform, err := mattermost.New(mattermost.Config{
			ServerURL: mattermostServerURL,
			Token:     mattermostToken,
			TeamName:  mattermostTeamName,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Mattermost platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(mattermostPlatform)
	} else {
		logger.Info("Mattermost tokens not provided, skipping Mattermost integration")
	}

	// Register Google Chat if tokens are provided
	if googlechatProjectID != "" {
		googlechatPlatform, err := googlechat.New(googlechat.Config{
			ProjectID:       googlechatProjectID,
			CredentialsFile: googlechatCredentialsFile,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Google Chat platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(googlechatPlatform)
	} else {
		logger.Info("Google Chat tokens not provided, skipping Google Chat integration")
	}

	// Register Matrix if tokens are provided
	if matrixHomeserverURL != "" && matrixAccessToken != "" {
		matrixPlatform, err := matrix.New(matrix.Config{
			HomeserverURL: matrixHomeserverURL,
			UserID:        matrixUserID,
			AccessToken:   matrixAccessToken,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Matrix platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(matrixPlatform)
	} else {
		logger.Info("Matrix tokens not provided, skipping Matrix integration")
	}

	// Register Teams if tokens are provided
	if teamsAppID != "" && teamsAppPassword != "" {
		teamsPlatform, err := teams.New(teams.Config{
			AppID:       teamsAppID,
			AppPassword: teamsAppPassword,
			TenantID:    teamsTenantID,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Teams platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(teamsPlatform)
	} else {
		logger.Info("Teams tokens not provided, skipping Teams integration")
	}

	// Register LINE if tokens are provided
	if lineChannelSecret != "" && lineChannelToken != "" {
		linePlatform, err := line.New(line.Config{
			ChannelSecret: lineChannelSecret,
			ChannelToken:  lineChannelToken,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating LINE platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(linePlatform)
	} else {
		logger.Info("LINE tokens not provided, skipping LINE integration")
	}

	// Register WhatsApp if tokens are provided
	if whatsappPhoneID != "" && whatsappAccessToken != "" {
		whatsappPlatform, err := whatsapp.New(whatsapp.Config{
			PhoneNumberID: whatsappPhoneID,
			AccessToken:   whatsappAccessToken,
			VerifyToken:   whatsappVerifyToken,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating WhatsApp platform: %v\n", err)
			os.Exit(1)
		}
		r.Register(whatsappPlatform)
	} else {
		logger.Info("WhatsApp tokens not provided, skipping WhatsApp integration")
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
	cronScheduler.Stop()
	r.Stop()
}

func setupBrowserDebug(debugDir string) error {
	// Use default debug directory if not specified
	if debugDir == "" {
		debugDir = filepath.Join(os.TempDir(), "lingti-bot")
	}

	// Create debug directory
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return fmt.Errorf("failed to create debug directory: %w", err)
	}

	// Configure browser instance
	b := browser.Instance()
	b.EnableDebug(debugDir)

	return nil
}
