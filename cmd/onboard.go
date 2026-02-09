package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/pltanton/lingti-bot/internal/config"
	"github.com/spf13/cobra"
)

var (
	onboardProvider  string
	onboardAPIKey    string
	onboardBaseURL   string
	onboardModel     string
	onboardPlatform  string
	onboardMode      string
	onboardReset     bool
	// WeCom
	onboardWeComCorpID  string
	onboardWeComAgentID string
	onboardWeComSecret  string
	onboardWeComToken   string
	onboardWeComAESKey  string
	// Slack
	onboardSlackBotToken string
	onboardSlackAppToken string
	// Telegram
	onboardTelegramToken string
	// Discord
	onboardDiscordToken string
	// Feishu
	onboardFeishuAppID     string
	onboardFeishuAppSecret string
	// DingTalk
	onboardDingTalkClientID     string
	onboardDingTalkClientSecret string
)

var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Interactive setup wizard for first-time configuration",
	Long: `Interactive setup wizard that saves AI provider and platform credentials
to a config file. After running onboard once, you can use 'relay' or 'router'
without passing any flags.

Usage:
  lingti-bot onboard              # Interactive wizard
  lingti-bot onboard --reset      # Clear config and start fresh
  lingti-bot onboard --provider deepseek --api-key sk-xxx  # Non-interactive

Config saved to:
  macOS: ~/Library/Preferences/Lingti/bot.yaml
  Linux: ~/.config/lingti/bot.yaml`,
	Run: runOnboard,
}

func init() {
	rootCmd.AddCommand(onboardCmd)

	onboardCmd.Flags().StringVar(&onboardProvider, "provider", "", "AI provider: deepseek, qwen, claude, kimi")
	onboardCmd.Flags().StringVar(&onboardAPIKey, "api-key", "", "AI API key")
	onboardCmd.Flags().StringVar(&onboardBaseURL, "base-url", "", "Custom API base URL")
	onboardCmd.Flags().StringVar(&onboardModel, "model", "", "Model name")
	onboardCmd.Flags().StringVar(&onboardPlatform, "platform", "", "Platform: wecom, dingtalk, feishu, slack, telegram, discord")
	onboardCmd.Flags().StringVar(&onboardMode, "mode", "", "Connection mode: relay, router")
	onboardCmd.Flags().BoolVar(&onboardReset, "reset", false, "Clear existing config and start fresh")

	// WeCom
	onboardCmd.Flags().StringVar(&onboardWeComCorpID, "wecom-corp-id", "", "WeCom Corp ID")
	onboardCmd.Flags().StringVar(&onboardWeComAgentID, "wecom-agent-id", "", "WeCom Agent ID")
	onboardCmd.Flags().StringVar(&onboardWeComSecret, "wecom-secret", "", "WeCom Secret")
	onboardCmd.Flags().StringVar(&onboardWeComToken, "wecom-token", "", "WeCom Callback Token")
	onboardCmd.Flags().StringVar(&onboardWeComAESKey, "wecom-aes-key", "", "WeCom EncodingAESKey")
	// Slack
	onboardCmd.Flags().StringVar(&onboardSlackBotToken, "slack-bot-token", "", "Slack Bot Token")
	onboardCmd.Flags().StringVar(&onboardSlackAppToken, "slack-app-token", "", "Slack App Token")
	// Telegram
	onboardCmd.Flags().StringVar(&onboardTelegramToken, "telegram-token", "", "Telegram Bot Token")
	// Discord
	onboardCmd.Flags().StringVar(&onboardDiscordToken, "discord-token", "", "Discord Bot Token")
	// Feishu
	onboardCmd.Flags().StringVar(&onboardFeishuAppID, "feishu-app-id", "", "Feishu App ID")
	onboardCmd.Flags().StringVar(&onboardFeishuAppSecret, "feishu-app-secret", "", "Feishu App Secret")
	// DingTalk
	onboardCmd.Flags().StringVar(&onboardDingTalkClientID, "dingtalk-client-id", "", "DingTalk AppKey")
	onboardCmd.Flags().StringVar(&onboardDingTalkClientSecret, "dingtalk-client-secret", "", "DingTalk AppSecret")
}

var scanner *bufio.Scanner

func initScanner() {
	scanner = bufio.NewScanner(os.Stdin)
}

func promptText(prompt string, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s (default: %s):\n  > ", prompt, defaultVal)
	} else {
		fmt.Printf("  %s:\n  > ", prompt)
	}
	scanner.Scan()
	val := strings.TrimSpace(scanner.Text())
	if val == "" {
		return defaultVal
	}
	return val
}

func promptSelect(prompt string, options []string, defaultIdx int) int {
	fmt.Printf("\n  %s\n", prompt)
	for i, opt := range options {
		fmt.Printf("    %d. %s\n", i+1, opt)
	}
	fmt.Printf("  Choice [%d]: ", defaultIdx+1)
	scanner.Scan()
	val := strings.TrimSpace(scanner.Text())
	if val == "" {
		return defaultIdx
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 || n > len(options) {
		return defaultIdx
	}
	return n - 1
}

func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func runOnboard(cmd *cobra.Command, args []string) {
	if onboardReset {
		path := config.ConfigPath()
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error removing config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Config cleared: %s\n", path)
	}

	cfg, _ := config.Load()

	if hasOnboardFlags(cmd) {
		applyOnboardFlags(cfg)
	} else {
		initScanner()
		runInteractiveWizard(cfg)
	}

	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	printOnboardSummary(cfg)
}

func hasOnboardFlags(_ *cobra.Command) bool {
	return onboardProvider != "" || onboardAPIKey != "" || onboardPlatform != ""
}

func applyOnboardFlags(cfg *config.Config) {
	if onboardProvider != "" {
		cfg.AI.Provider = onboardProvider
	}
	if onboardAPIKey != "" {
		cfg.AI.APIKey = onboardAPIKey
	}
	if onboardBaseURL != "" {
		cfg.AI.BaseURL = onboardBaseURL
	}
	if onboardModel != "" {
		cfg.AI.Model = onboardModel
	}
	if onboardMode != "" {
		cfg.Mode = onboardMode
	}

	// Platform credentials
	switch onboardPlatform {
	case "wecom":
		if onboardWeComCorpID != "" {
			cfg.Platforms.WeCom.CorpID = onboardWeComCorpID
		}
		if onboardWeComAgentID != "" {
			cfg.Platforms.WeCom.AgentID = onboardWeComAgentID
		}
		if onboardWeComSecret != "" {
			cfg.Platforms.WeCom.Secret = onboardWeComSecret
		}
		if onboardWeComToken != "" {
			cfg.Platforms.WeCom.Token = onboardWeComToken
		}
		if onboardWeComAESKey != "" {
			cfg.Platforms.WeCom.AESKey = onboardWeComAESKey
		}
	case "slack":
		if onboardSlackBotToken != "" {
			cfg.Platforms.Slack.BotToken = onboardSlackBotToken
		}
		if onboardSlackAppToken != "" {
			cfg.Platforms.Slack.AppToken = onboardSlackAppToken
		}
	case "telegram":
		if onboardTelegramToken != "" {
			cfg.Platforms.Telegram.Token = onboardTelegramToken
		}
	case "discord":
		if onboardDiscordToken != "" {
			cfg.Platforms.Discord.Token = onboardDiscordToken
		}
	case "feishu":
		if onboardFeishuAppID != "" {
			cfg.Platforms.Feishu.AppID = onboardFeishuAppID
		}
		if onboardFeishuAppSecret != "" {
			cfg.Platforms.Feishu.AppSecret = onboardFeishuAppSecret
		}
	case "dingtalk":
		if onboardDingTalkClientID != "" {
			cfg.Platforms.DingTalk.ClientID = onboardDingTalkClientID
		}
		if onboardDingTalkClientSecret != "" {
			cfg.Platforms.DingTalk.ClientSecret = onboardDingTalkClientSecret
		}
	}
}

func runInteractiveWizard(cfg *config.Config) {
	fmt.Println()
	fmt.Println("  lingti-bot -- Interactive Setup")
	fmt.Println("  ───────────────────────────────────")

	// Show existing config if present
	if cfg.AI.Provider != "" {
		fmt.Printf("\n  Existing config found: %s / %s\n", cfg.AI.Provider, maskKey(cfg.AI.APIKey))
		idx := promptSelect("What would you like to do?", []string{
			"Update existing config",
			"Start fresh",
			"Keep and exit",
		}, 0)
		if idx == 2 {
			return
		}
		if idx == 1 {
			*cfg = *config.DefaultConfig()
		}
	}

	stepAIProvider(cfg)
	stepPlatform(cfg)
	stepConnectionMode(cfg)
}

type providerInfo struct {
	name     string
	label    string
	keyURL   string
	defModel string
}

var providers = []providerInfo{
	{"deepseek", "deepseek  (recommended)", "https://platform.deepseek.com/api_keys", "deepseek-chat"},
	{"qwen", "qwen      (tongyi qianwen)", "https://bailian.console.aliyun.com/", "qwen-plus"},
	{"claude", "claude    (Anthropic)", "https://console.anthropic.com/", "claude-sonnet-4-20250514"},
	{"kimi", "kimi      (Moonshot)", "https://platform.moonshot.cn/", "moonshot-v1-8k"},
}

// detectClaudeOAuthToken tries to find an existing Claude OAuth token from env vars or macOS Keychain.
func detectClaudeOAuthToken() string {
	// 1. Check ANTHROPIC_OAUTH_TOKEN env var
	if tok := os.Getenv("ANTHROPIC_OAUTH_TOKEN"); tok != "" && strings.HasPrefix(tok, "sk-ant-oat") {
		return tok
	}

	// 2. macOS Keychain: Claude Code stores credentials under "Claude Code-credentials"
	if runtime.GOOS == "darwin" {
		if tok := readClaudeKeychain(); tok != "" {
			return tok
		}
	}

	// 3. Check ANTHROPIC_API_KEY if it looks like an OAuth token
	if tok := os.Getenv("ANTHROPIC_API_KEY"); tok != "" && strings.HasPrefix(tok, "sk-ant-oat") {
		return tok
	}

	return ""
}

// detectClaudeAPIKey tries to find an existing Anthropic API key from env vars.
func detectClaudeAPIKey() string {
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" && strings.HasPrefix(key, "sk-ant-") && !strings.HasPrefix(key, "sk-ant-oat") {
		return key
	}
	return ""
}

// readClaudeKeychain reads the Claude Code OAuth token from macOS Keychain.
func readClaudeKeychain() string {
	out, err := exec.Command("security", "find-generic-password", "-s", "Claude Code-credentials", "-w").Output()
	if err != nil {
		return ""
	}

	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(out, &creds); err != nil {
		return ""
	}

	tok := creds.ClaudeAiOauth.AccessToken
	if strings.HasPrefix(tok, "sk-ant-oat") {
		return tok
	}
	return ""
}

func stepAIProvider(cfg *config.Config) {
	fmt.Println("\n  Step 1/3: AI Provider")

	options := make([]string, len(providers))
	for i, p := range providers {
		options[i] = p.label
	}

	defIdx := 0
	for i, p := range providers {
		if p.name == cfg.AI.Provider {
			defIdx = i
			break
		}
	}

	idx := promptSelect("Select AI provider:", options, defIdx)
	p := providers[idx]
	cfg.AI.Provider = p.name

	if p.name == "claude" {
		// Auto-detect existing tokens to suggest as defaults
		detectedOAuth := detectClaudeOAuthToken()
		detectedAPIKey := detectClaudeAPIKey()

		// If existing config or detected token is OAuth, default to Setup Token auth
		defAuth := 0
		if detectedOAuth != "" || strings.HasPrefix(cfg.AI.APIKey, "sk-ant-oat") {
			defAuth = 1
		}

		authIdx := promptSelect("Auth method:", []string{
			"API Key       (from console.anthropic.com)",
			"Setup Token   (from 'claude setup-token', requires Claude subscription)",
		}, defAuth)
		if authIdx == 0 {
			defKey := cfg.AI.APIKey
			if defKey == "" && detectedAPIKey != "" {
				defKey = detectedAPIKey
				fmt.Printf("\n  Detected existing API key: %s\n", maskKey(defKey))
			}
			fmt.Printf("\n  Claude API Key (%s)\n", p.keyURL)
			cfg.AI.APIKey = promptText("API Key", defKey)
		} else {
			// Pick best default: prefer detected (freshest), fall back to config
			defToken := detectedOAuth
			if defToken == "" {
				defToken = cfg.AI.APIKey
			}

			if defToken != "" {
				fmt.Printf("\n  Detected existing Claude token: %s\n", maskKey(defToken))
				fmt.Println("  Press Enter to use it, or paste a different token.")
			} else {
				fmt.Println("\n  Run 'claude setup-token' in another terminal, then paste the token here.")
				fmt.Println("  (Requires Claude Code CLI and an active Claude subscription)")
			}
			cfg.AI.APIKey = promptText("Setup Token (sk-ant-oat01-...)", defToken)
			if cfg.AI.APIKey != "" && !strings.HasPrefix(cfg.AI.APIKey, "sk-ant-oat") {
				fmt.Println("  Warning: expected token starting with sk-ant-oat01-")
			}
		}
	} else {
		displayName := strings.ToUpper(p.name[:1]) + p.name[1:]
		fmt.Printf("\n  %s API Key (%s)\n", displayName, p.keyURL)
		cfg.AI.APIKey = promptText("API Key", cfg.AI.APIKey)
	}

	model := promptText("Model", p.defModel)
	cfg.AI.Model = model

	fmt.Printf("\n  > AI provider configured: %s / %s\n", cfg.AI.Provider, cfg.AI.Model)
}

type platformInfo struct {
	name  string
	label string
}

var platformOptions = []platformInfo{
	{"wecom", "wecom     (WeCom)"},
	{"dingtalk", "dingtalk  (DingTalk)"},
	{"feishu", "feishu    (Feishu/Lark)"},
	{"slack", "slack"},
	{"telegram", "telegram"},
	{"discord", "discord"},
	{"skip", "skip      (configure later)"},
}

func stepPlatform(cfg *config.Config) {
	fmt.Println("\n  Step 2/3: Platform")

	options := make([]string, len(platformOptions))
	for i, p := range platformOptions {
		options[i] = p.label
	}

	idx := promptSelect("Select messaging platform:", options, 0)
	platform := platformOptions[idx].name

	switch platform {
	case "wecom":
		stepWecom(cfg)
	case "dingtalk":
		stepDingTalk(cfg)
	case "feishu":
		stepFeishu(cfg)
	case "slack":
		stepSlack(cfg)
	case "telegram":
		stepTelegram(cfg)
	case "discord":
		stepDiscord(cfg)
	case "skip":
		fmt.Println("\n  > Platform configuration skipped")
	}
}

func stepWecom(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.WeCom.CorpID = promptText("WeCom Corp ID", cfg.Platforms.WeCom.CorpID)
	cfg.Platforms.WeCom.AgentID = promptText("WeCom Agent ID", cfg.Platforms.WeCom.AgentID)
	cfg.Platforms.WeCom.Secret = promptText("WeCom Secret", cfg.Platforms.WeCom.Secret)
	cfg.Platforms.WeCom.Token = promptText("WeCom Token", cfg.Platforms.WeCom.Token)
	cfg.Platforms.WeCom.AESKey = promptText("WeCom AES Key (EncodingAESKey)", cfg.Platforms.WeCom.AESKey)
	fmt.Println("\n  > WeCom configured")
}

func stepDingTalk(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.DingTalk.ClientID = promptText("DingTalk AppKey (ClientID)", cfg.Platforms.DingTalk.ClientID)
	cfg.Platforms.DingTalk.ClientSecret = promptText("DingTalk AppSecret (ClientSecret)", cfg.Platforms.DingTalk.ClientSecret)
	fmt.Println("\n  > DingTalk configured")
}

func stepFeishu(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Feishu.AppID = promptText("Feishu App ID", cfg.Platforms.Feishu.AppID)
	cfg.Platforms.Feishu.AppSecret = promptText("Feishu App Secret", cfg.Platforms.Feishu.AppSecret)
	fmt.Println("\n  > Feishu configured")
}

func stepSlack(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Slack.BotToken = promptText("Slack Bot Token (xoxb-...)", cfg.Platforms.Slack.BotToken)
	cfg.Platforms.Slack.AppToken = promptText("Slack App Token (xapp-...)", cfg.Platforms.Slack.AppToken)
	fmt.Println("\n  > Slack configured")
}

func stepTelegram(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Telegram.Token = promptText("Telegram Bot Token", cfg.Platforms.Telegram.Token)
	fmt.Println("\n  > Telegram configured")
}

func stepDiscord(cfg *config.Config) {
	fmt.Println()
	cfg.Platforms.Discord.Token = promptText("Discord Bot Token", cfg.Platforms.Discord.Token)
	fmt.Println("\n  > Discord configured")
}

func stepConnectionMode(cfg *config.Config) {
	fmt.Println("\n  Step 3/3: Connection Mode")

	defIdx := 0
	if cfg.Mode == "router" {
		defIdx = 1
	}

	idx := promptSelect("Select connection mode:", []string{
		"relay   (cloud relay, recommended, no public server needed)",
		"router  (self-hosted, requires public IP)",
	}, defIdx)

	if idx == 0 {
		cfg.Mode = "relay"
	} else {
		cfg.Mode = "router"
	}

	fmt.Printf("\n  > Connection mode: %s\n", cfg.Mode)
}

func printOnboardSummary(cfg *config.Config) {
	fmt.Println()
	fmt.Println("  ───────────────────────────────────")
	fmt.Printf("  > Configuration saved to %s\n", config.ConfigPath())
	fmt.Println()

	if cfg.Mode == "relay" {
		fmt.Println("  To start the bot, run:")
		fmt.Println("    lingti-bot relay")
	} else {
		fmt.Println("  To start the bot, run:")
		fmt.Println("    lingti-bot router")
	}
	fmt.Println()
	fmt.Println("  To reconfigure:")
	fmt.Println("    lingti-bot onboard")
	fmt.Println()
}
