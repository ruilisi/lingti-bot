package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pltanton/lingti-bot/internal/agent"
	"github.com/pltanton/lingti-bot/internal/router"
	"github.com/pltanton/lingti-bot/internal/voice"
	"github.com/spf13/cobra"
)

var (
	recordDuration  int
	speakResponse   bool
	voiceName       string
	voiceLanguage   string
)

var voiceCmd = &cobra.Command{
	Use:   "voice",
	Short: "Voice input mode - speak to the AI",
	Long: `Start voice input mode for hands-free AI interaction.

This mode allows you to:
  - Press Enter to start recording
  - Speak your message
  - Recording stops after the specified duration
  - AI processes your speech and responds
  - Optionally speaks the response back

Example:
  lingti-bot voice                    # Default 5 second recording
  lingti-bot voice -d 10              # 10 second recording
  lingti-bot voice --speak            # Speak responses aloud
  lingti-bot voice --provider openai  # Use OpenAI Whisper

Environment variables:
  - VOICE_PROVIDER: STT/TTS provider (system, openai)
  - VOICE_API_KEY: API key for cloud providers
  - AI_API_KEY: API Key for the AI provider`,
	Run: runVoice,
}

func init() {
	rootCmd.AddCommand(voiceCmd)

	voiceCmd.Flags().IntVarP(&recordDuration, "duration", "d", 5, "Recording duration in seconds")
	voiceCmd.Flags().BoolVarP(&speakResponse, "speak", "s", false, "Speak AI responses aloud")
	voiceCmd.Flags().StringVar(&voiceName, "voice-name", "", "Voice name for TTS")
	voiceCmd.Flags().StringVarP(&voiceLanguage, "language", "l", "zh", "Language for speech recognition (default: zh)")
	voiceCmd.Flags().StringVar(&voiceProvider, "provider", "", "Voice provider: system, openai (or VOICE_PROVIDER env)")
	voiceCmd.Flags().StringVar(&voiceAPIKey, "voice-api-key", "", "Voice API key (or VOICE_API_KEY env)")
	voiceCmd.Flags().StringVar(&aiProvider, "ai-provider", "", "AI provider: claude, deepseek, kimi (or AI_PROVIDER env)")
	voiceCmd.Flags().StringVar(&aiAPIKey, "api-key", "", "AI API Key (or AI_API_KEY env)")
	voiceCmd.Flags().StringVar(&aiBaseURL, "base-url", "", "AI API base URL (or AI_BASE_URL env)")
	voiceCmd.Flags().StringVar(&aiModel, "model", "", "Model name (or AI_MODEL env)")
}

func runVoice(cmd *cobra.Command, args []string) {
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
		Provider: aiProvider,
		APIKey:   aiAPIKey,
		BaseURL:  aiBaseURL,
		Model:    aiModel,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating agent: %v\n", err)
		os.Exit(1)
	}

	// Check and download whisper model if using system provider
	if voiceProvider == "" || voiceProvider == "system" {
		if voice.FindWhisperModel() == "" {
			fmt.Println("📦 Whisper model not found. Downloading base model (141MB)...")
			if err := voice.DownloadWhisperModel("base"); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to download model: %v\n", err)
				fmt.Fprintln(os.Stderr, "Run 'lingti-bot setup' for manual installation instructions")
				os.Exit(1)
			}
			fmt.Println("✅ Model downloaded successfully")
		}
	}

	// Create voice recorder/transcriber
	recorder, err := voice.NewRecorder(voice.RecorderConfig{
		Provider: voiceProvider,
		APIKey:   voiceAPIKey,
		Language: voiceLanguage,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating voice recorder: %v\n", err)
		fmt.Fprintln(os.Stderr, "Run 'lingti-bot setup' for installation instructions")
		os.Exit(1)
	}

	// Create TTS speaker if speak mode is enabled
	var speaker *voice.Speaker
	if speakResponse {
		speaker, err = voice.NewSpeaker(voice.SpeakerConfig{
			Provider: voiceProvider,
			APIKey:   voiceAPIKey,
			Voice:    voiceName,
		})
		if err != nil {
			log.Printf("Warning: Failed to create speaker: %v (responses will be text only)", err)
			speakResponse = false
		}
	}

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
		os.Exit(0)
	}()

	// Print instructions
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Voice Input Mode                        ║")
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Provider: %-47s ║\n", voiceProvider)
	fmt.Printf("║  Recording duration: %d seconds                             ║\n", recordDuration)
	if speakResponse {
		fmt.Println("║  Response: Voice + Text                                    ║")
	} else {
		fmt.Println("║  Response: Text only                                       ║")
	}
	fmt.Println("╠════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Press [Enter] to start recording                          ║")
	fmt.Println("║  Type 'quit' or 'exit' to stop                             ║")
	fmt.Println("║  Press Ctrl+C to exit                                      ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	sessionID := fmt.Sprintf("voice-%d", time.Now().Unix())

	for {
		fmt.Print("Press [Enter] to speak (or type a message): ")

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())

		// Check for exit commands
		if input == "quit" || input == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		var text string

		if input == "" {
			// Record voice
			fmt.Printf("🎤 Recording for %d seconds... (speak now)\n", recordDuration)

			audio, err := recorder.Record(ctx, time.Duration(recordDuration)*time.Second)
			if err != nil {
				fmt.Printf("❌ Recording failed: %v\n", err)
				continue
			}

			fmt.Println("⏳ Transcribing...")

			text, err = recorder.Transcribe(ctx, audio)
			if err != nil {
				fmt.Printf("❌ Transcription failed: %v\n", err)
				continue
			}

			if text == "" {
				fmt.Println("❌ No speech detected, try again")
				continue
			}

			fmt.Printf("📝 You said: %s\n", text)
		} else {
			// Use typed input
			text = input
		}

		fmt.Println("🤔 Thinking...")

		// Send to AI agent
		msg := router.Message{
			ID:        fmt.Sprintf("voice-%d", time.Now().UnixNano()),
			Platform:  "voice",
			ChannelID: sessionID,
			UserID:    "local-user",
			Username:  "You",
			Text:      text,
			Metadata:  map[string]string{"input_type": "voice"},
		}

		response, err := aiAgent.HandleMessage(ctx, msg)
		if err != nil {
			fmt.Printf("❌ AI error: %v\n", err)
			continue
		}

		// Print response
		fmt.Println()
		fmt.Println("🤖 Assistant:")
		fmt.Println(response.Text)
		fmt.Println()

		// Speak response if enabled
		if speakResponse && speaker != nil {
			fmt.Println("🔊 Speaking...")
			if err := speaker.Speak(ctx, response.Text); err != nil {
				log.Printf("Warning: TTS failed: %v", err)
			}
		}
	}
}
