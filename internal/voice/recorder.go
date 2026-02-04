package voice

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// Recorder handles voice recording and transcription
type Recorder struct {
	provider   Provider
	sampleRate int
	language   string
}

// RecorderConfig holds recorder configuration
type RecorderConfig struct {
	Provider   string // "system", "openai"
	APIKey     string // API key for cloud providers
	SampleRate int    // Sample rate in Hz (default: 16000)
	Language   string // Language code for STT (default: "zh")
}

// NewRecorder creates a new voice recorder
func NewRecorder(cfg RecorderConfig) (*Recorder, error) {
	var provider Provider
	var err error

	switch cfg.Provider {
	case "openai":
		provider, err = NewOpenAIProvider(cfg.APIKey)
	case "system", "":
		provider = NewSystemProvider()
	default:
		return nil, fmt.Errorf("unknown voice provider: %s", cfg.Provider)
	}

	if err != nil {
		return nil, err
	}

	sampleRate := cfg.SampleRate
	if sampleRate == 0 {
		sampleRate = 16000
	}

	language := cfg.Language
	if language == "" {
		language = "zh" // Default to Chinese
	}

	return &Recorder{
		provider:   provider,
		sampleRate: sampleRate,
		language:   language,
	}, nil
}

// Record records audio from the microphone for the specified duration
func (r *Recorder) Record(ctx context.Context, duration time.Duration) ([]byte, error) {
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("recording-%d.wav", time.Now().UnixNano()))
	defer os.Remove(tmpFile)

	var cmd *exec.Cmd
	durationSecs := int(duration.Seconds())

	switch runtime.GOOS {
	case "darwin":
		// Check if sox is installed (provides 'rec' command)
		if _, err := exec.LookPath("rec"); err == nil {
			// Use sox's rec command
			cmd = exec.CommandContext(ctx, "rec",
				"-q",                            // Quiet mode
				"-r", fmt.Sprintf("%d", r.sampleRate), // Sample rate
				"-c", "1",                       // Mono
				"-b", "16",                      // 16-bit
				tmpFile,                         // Output file
				"trim", "0", fmt.Sprintf("%d", durationSecs), // Duration
			)
		} else if _, err := exec.LookPath("ffmpeg"); err == nil {
			// Fallback to ffmpeg with AVFoundation
			cmd = exec.CommandContext(ctx, "ffmpeg",
				"-f", "avfoundation",
				"-i", ":0", // Default audio input
				"-t", fmt.Sprintf("%d", durationSecs),
				"-ar", fmt.Sprintf("%d", r.sampleRate),
				"-ac", "1",
				"-y", // Overwrite
				tmpFile,
			)
		} else {
			return nil, fmt.Errorf("no audio recorder found.\n  Install: brew install sox\n  Or run: lingti-bot setup --all")
		}

	case "linux":
		// Try arecord first (ALSA)
		if _, err := exec.LookPath("arecord"); err == nil {
			cmd = exec.CommandContext(ctx, "arecord",
				"-q",
				"-r", fmt.Sprintf("%d", r.sampleRate),
				"-c", "1",
				"-f", "S16_LE",
				"-d", fmt.Sprintf("%d", durationSecs),
				tmpFile,
			)
		} else if _, err := exec.LookPath("rec"); err == nil {
			// Fallback to sox
			cmd = exec.CommandContext(ctx, "rec",
				"-q",
				"-r", fmt.Sprintf("%d", r.sampleRate),
				"-c", "1",
				"-b", "16",
				tmpFile,
				"trim", "0", fmt.Sprintf("%d", durationSecs),
			)
		} else {
			return nil, fmt.Errorf("no audio recorder found.\n  Install: sudo apt install alsa-utils sox\n  Or run: lingti-bot setup --all")
		}

	case "windows":
		// Use ffmpeg on Windows
		if _, err := exec.LookPath("ffmpeg"); err == nil {
			cmd = exec.CommandContext(ctx, "ffmpeg",
				"-f", "dshow",
				"-i", "audio=Microphone", // May need adjustment
				"-t", fmt.Sprintf("%d", durationSecs),
				"-ar", fmt.Sprintf("%d", r.sampleRate),
				"-ac", "1",
				"-y",
				tmpFile,
			)
		} else {
			return nil, fmt.Errorf("ffmpeg required for Windows recording.\n  Install: winget install ffmpeg\n  Or run: lingti-bot setup --all")
		}

	default:
		return nil, fmt.Errorf("audio recording not supported on %s", runtime.GOOS)
	}

	// Run the recording command
	cmd.Stderr = nil // Suppress stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("recording failed: %w", err)
	}

	// Read the recorded audio
	return os.ReadFile(tmpFile)
}

// Transcribe converts audio to text
func (r *Recorder) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return r.provider.SpeechToText(ctx, audio, STTOptions{Language: r.language})
}

// RecordAndTranscribe records audio and transcribes it in one step
func (r *Recorder) RecordAndTranscribe(ctx context.Context, duration time.Duration) (string, error) {
	audio, err := r.Record(ctx, duration)
	if err != nil {
		return "", err
	}

	return r.Transcribe(ctx, audio)
}

// ProviderName returns the name of the underlying provider
func (r *Recorder) ProviderName() string {
	return r.provider.Name()
}
