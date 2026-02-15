package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pltanton/lingti-bot/internal/agent"
	"github.com/pltanton/lingti-bot/internal/logger"
	"github.com/pltanton/lingti-bot/internal/router"
	"github.com/pltanton/lingti-bot/internal/voice"
	"github.com/spf13/cobra"
)

var (
	voiceProvider   string
	voiceAPIKey     string
	wakeWord        string
	continuousMode  bool
	defaultVoice    string
	briefVoice      bool
)

var talkCmd = &cobra.Command{
	Use:   "talk",
	Short: "Start voice/talk mode",
	Long: `Start voice/talk mode for hands-free AI interaction.

Talk mode allows you to interact with the AI assistant using voice commands.
It supports multiple speech providers:
  - system: Uses native OS speech (macOS say/dictation, Linux espeak)
  - openai: Uses OpenAI's TTS and Whisper APIs
  - elevenlabs: Uses ElevenLabs for high-quality TTS

Environment variables:
  - VOICE_PROVIDER: Speech provider (system, openai, elevenlabs)
  - VOICE_API_KEY: API key for cloud providers
  - WAKE_WORD: Wake word for activation (e.g., "hey lingti")
  - AI_API_KEY: API Key for the AI provider`,
	Run: runTalk,
}

func init() {
	rootCmd.AddCommand(talkCmd)

	talkCmd.Flags().StringVar(&voiceProvider, "voice-provider", "", "Voice provider: system, openai, elevenlabs (or VOICE_PROVIDER env)")
	talkCmd.Flags().StringVar(&voiceAPIKey, "voice-api-key", "", "Voice API key (or VOICE_API_KEY env)")
	talkCmd.Flags().StringVar(&wakeWord, "wake-word", "", "Wake word for activation (or WAKE_WORD env)")
	talkCmd.Flags().BoolVar(&continuousMode, "continuous", false, "Keep listening after each response")
	talkCmd.Flags().BoolVar(&briefVoice, "brief", true, "Brief voice mode: print full text, speak only notification")
	talkCmd.Flags().StringVar(&defaultVoice, "voice", "", "Default voice name")
	talkCmd.Flags().StringVar(&aiProvider, "provider", "", "AI provider: claude, deepseek, kimi, qwen (or AI_PROVIDER env)")
	talkCmd.Flags().StringVar(&aiAPIKey, "api-key", "", "AI API Key (or AI_API_KEY env)")
	talkCmd.Flags().StringVar(&aiBaseURL, "base-url", "", "AI API base URL (or AI_BASE_URL env)")
	talkCmd.Flags().StringVar(&aiModel, "model", "", "Model name (or AI_MODEL env)")
}

func runTalk(cmd *cobra.Command, args []string) {
	// Get config from flags or environment
	if voiceProvider == "" {
		voiceProvider = os.Getenv("VOICE_PROVIDER")
		if voiceProvider == "" {
			voiceProvider = "system"
		}
	}
	if voiceAPIKey == "" {
		voiceAPIKey = os.Getenv("VOICE_API_KEY")
		if voiceAPIKey == "" && voiceProvider == "openai" {
			voiceAPIKey = os.Getenv("OPENAI_API_KEY")
		}
		if voiceAPIKey == "" && voiceProvider == "elevenlabs" {
			voiceAPIKey = os.Getenv("ELEVENLABS_API_KEY")
		}
	}
	if wakeWord == "" {
		wakeWord = os.Getenv("WAKE_WORD")
	}
	if aiProvider == "" {
		aiProvider = os.Getenv("AI_PROVIDER")
	}
	if aiAPIKey == "" {
		aiAPIKey = os.Getenv("AI_API_KEY")
		if aiAPIKey == "" {
			aiAPIKey = os.Getenv("ANTHROPIC_API_KEY")
		}
	}
	if aiModel == "" {
		aiModel = os.Getenv("AI_MODEL")
		if aiModel == "" {
			aiModel = os.Getenv("ANTHROPIC_MODEL")
		}
	}
	if aiBaseURL == "" {
		aiBaseURL = os.Getenv("AI_BASE_URL")
		if aiBaseURL == "" {
			aiBaseURL = os.Getenv("ANTHROPIC_BASE_URL")
		}
	}

	if aiAPIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: AI_API_KEY is required")
		os.Exit(1)
	}

	// Create the AI agent
	aiAgent, err := agent.New(agent.Config{
		Provider:     aiProvider,
		APIKey:       aiAPIKey,
		BaseURL:      aiBaseURL,
		Model:        aiModel,
		AllowedPaths:     loadAllowedPaths(),
		DisableFileTools: loadDisableFileTools(),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating agent: %v\n", err)
		os.Exit(1)
	}

	// Create message handler for voice mode
	messageHandler := func(text string) (string, error) {
		msg := router.Message{
			ID:        "voice-session",
			Platform:  "voice",
			ChannelID: "local",
			UserID:    "local-user",
			Username:  "You",
			Text:      text,
			Metadata:  map[string]string{},
		}

		response, err := aiAgent.HandleMessage(context.Background(), msg)
		if err != nil {
			return "", err
		}

		return response.Text, nil
	}

	// Create talk mode
	talkMode, err := voice.NewTalkMode(voice.Config{
		Provider:       voiceProvider,
		APIKey:         voiceAPIKey,
		WakeWord:       wakeWord,
		ContinuousMode: continuousMode,
		DefaultVoice:   defaultVoice,
		BriefVoice:     briefVoice,
	}, messageHandler)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating talk mode: %v\n", err)
		os.Exit(1)
	}

	// Start talk mode
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := talkMode.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting talk mode: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Talk mode started (provider: %s)", voiceProvider)
	if wakeWord != "" {
		logger.Info("Wake word: %s", wakeWord)
	}
	if continuousMode {
		logger.Info("Continuous mode enabled")
	}
	logger.Info("Press Ctrl+C to stop.")

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down...")
	talkMode.Stop()
}
