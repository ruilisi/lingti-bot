package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Provider defines a speech provider interface
type Provider interface {
	Name() string
	TextToSpeech(ctx context.Context, text string, opts TTSOptions) ([]byte, error)
	SpeechToText(ctx context.Context, audio []byte, opts STTOptions) (string, error)
}

// TTSOptions contains text-to-speech options
type TTSOptions struct {
	Voice    string  // Voice name/ID
	Language string  // Language code (e.g., "en-US")
	Speed    float64 // Speech rate (1.0 = normal)
	Pitch    float64 // Voice pitch (1.0 = normal)
	Format   string  // Output format (wav, mp3, etc.)
}

// STTOptions contains speech-to-text options
type STTOptions struct {
	Language string // Language code
	Model    string // Model name if applicable
}

// TalkMode represents an active talk/voice session
type TalkMode struct {
	provider       Provider
	messageHandler func(text string) (string, error)
	listening      bool
	speaking       bool
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	wakeWord       string
	continuousMode bool
}

// Config holds voice configuration
type Config struct {
	Provider       string // "system", "openai", "elevenlabs"
	APIKey         string // API key for cloud providers
	WakeWord       string // Wake word for activation (e.g., "hey lingti")
	ContinuousMode bool   // Keep listening after response
	DefaultVoice   string // Default voice for TTS
	DefaultLang    string // Default language
}

// NewTalkMode creates a new talk mode session
func NewTalkMode(cfg Config, handler func(text string) (string, error)) (*TalkMode, error) {
	var provider Provider
	var err error

	switch cfg.Provider {
	case "openai":
		provider, err = NewOpenAIProvider(cfg.APIKey)
	case "elevenlabs":
		provider, err = NewElevenLabsProvider(cfg.APIKey)
	default:
		provider = NewSystemProvider()
	}

	if err != nil {
		return nil, err
	}

	return &TalkMode{
		provider:       provider,
		messageHandler: handler,
		wakeWord:       cfg.WakeWord,
		continuousMode: cfg.ContinuousMode,
	}, nil
}

// Start begins the talk mode session
func (t *TalkMode) Start(ctx context.Context) error {
	t.mu.Lock()
	t.ctx, t.cancel = context.WithCancel(ctx)
	t.listening = true
	t.mu.Unlock()

	log.Printf("[Voice] Talk mode started with provider: %s", t.provider.Name())

	// Start listening loop
	go t.listenLoop()

	return nil
}

// Stop ends the talk mode session
func (t *TalkMode) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.listening = false
	if t.cancel != nil {
		t.cancel()
	}
}

// IsListening returns whether talk mode is active
func (t *TalkMode) IsListening() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.listening
}

// IsSpeaking returns whether TTS is playing
func (t *TalkMode) IsSpeaking() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.speaking
}

// Speak converts text to speech and plays it
func (t *TalkMode) Speak(ctx context.Context, text string) error {
	t.mu.Lock()
	t.speaking = true
	t.mu.Unlock()
	defer func() {
		t.mu.Lock()
		t.speaking = false
		t.mu.Unlock()
	}()

	audio, err := t.provider.TextToSpeech(ctx, text, TTSOptions{
		Format: "wav",
	})
	if err != nil {
		return err
	}

	return playAudio(audio)
}

// ProcessAudio processes audio input and returns the response
func (t *TalkMode) ProcessAudio(ctx context.Context, audio []byte) (string, error) {
	// Convert speech to text
	text, err := t.provider.SpeechToText(ctx, audio, STTOptions{})
	if err != nil {
		return "", fmt.Errorf("speech-to-text failed: %w", err)
	}

	if text == "" {
		return "", nil
	}

	log.Printf("[Voice] Recognized: %s", text)

	// Check for wake word if configured
	if t.wakeWord != "" && !strings.Contains(strings.ToLower(text), strings.ToLower(t.wakeWord)) {
		return "", nil
	}

	// Process through message handler
	if t.messageHandler == nil {
		return "", fmt.Errorf("no message handler configured")
	}

	response, err := t.messageHandler(text)
	if err != nil {
		return "", fmt.Errorf("message handler failed: %w", err)
	}

	// Speak the response
	if err := t.Speak(ctx, response); err != nil {
		log.Printf("[Voice] TTS failed: %v", err)
	}

	return response, nil
}

// listenLoop continuously listens for voice input
func (t *TalkMode) listenLoop() {
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			if !t.IsListening() || t.IsSpeaking() {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Record audio (platform-specific)
			audio, err := recordAudio(t.ctx, 5*time.Second)
			if err != nil {
				log.Printf("[Voice] Recording failed: %v", err)
				continue
			}

			if len(audio) > 0 {
				_, err := t.ProcessAudio(t.ctx, audio)
				if err != nil {
					log.Printf("[Voice] Processing failed: %v", err)
				}

				// If not in continuous mode, stop after one interaction
				if !t.continuousMode {
					t.Stop()
					return
				}
			}
		}
	}
}

// SystemProvider uses system TTS/STT capabilities
type SystemProvider struct{}

// NewSystemProvider creates a system provider
func NewSystemProvider() *SystemProvider {
	return &SystemProvider{}
}

// Name returns the provider name
func (p *SystemProvider) Name() string {
	return "system"
}

// TextToSpeech uses system TTS
func (p *SystemProvider) TextToSpeech(ctx context.Context, text string, opts TTSOptions) ([]byte, error) {
	switch runtime.GOOS {
	case "darwin":
		return p.macTTS(ctx, text, opts)
	case "linux":
		return p.linuxTTS(ctx, text, opts)
	default:
		return nil, fmt.Errorf("system TTS not supported on %s", runtime.GOOS)
	}
}

// SpeechToText uses system STT
func (p *SystemProvider) SpeechToText(ctx context.Context, audio []byte, opts STTOptions) (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return p.macSTT(ctx, audio, opts)
	default:
		return "", fmt.Errorf("system STT not supported on %s", runtime.GOOS)
	}
}

// macTTS uses macOS say command
func (p *SystemProvider) macTTS(ctx context.Context, text string, opts TTSOptions) ([]byte, error) {
	// Create temp file for output
	tmpFile, err := os.CreateTemp("", "tts-*.aiff")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	args := []string{"-o", tmpFile.Name()}

	if opts.Voice != "" {
		args = append(args, "-v", opts.Voice)
	}

	if opts.Speed != 0 {
		rate := int(175 * opts.Speed) // 175 wpm is default
		args = append(args, "-r", fmt.Sprintf("%d", rate))
	}

	args = append(args, text)

	cmd := exec.CommandContext(ctx, "say", args...)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("say command failed: %w", err)
	}

	return os.ReadFile(tmpFile.Name())
}

// linuxTTS uses espeak
func (p *SystemProvider) linuxTTS(ctx context.Context, text string, opts TTSOptions) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "tts-*.wav")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	args := []string{"-w", tmpFile.Name()}

	if opts.Voice != "" {
		args = append(args, "-v", opts.Voice)
	}

	if opts.Speed != 0 {
		speed := int(175 * opts.Speed)
		args = append(args, "-s", fmt.Sprintf("%d", speed))
	}

	args = append(args, text)

	cmd := exec.CommandContext(ctx, "espeak", args...)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("espeak command failed: %w", err)
	}

	return os.ReadFile(tmpFile.Name())
}

// macSTT uses whisper.cpp for speech-to-text
func (p *SystemProvider) macSTT(ctx context.Context, audio []byte, opts STTOptions) (string, error) {
	// Save audio to temp file
	tmpFile, err := os.CreateTemp("", "stt-*.wav")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(audio); err != nil {
		return "", err
	}
	tmpFile.Close()

	// Find whisper binary - try multiple names
	var whisperPath string
	for _, name := range []string{"whisper-cli", "whisper", "whisper-cpp"} {
		if path, err := exec.LookPath(name); err == nil {
			whisperPath = path
			break
		}
	}

	if whisperPath == "" {
		return "", fmt.Errorf("no STT engine available (install whisper-cpp: brew install whisper-cpp)")
	}

	// Find model file
	modelPath := FindWhisperModel()
	if modelPath == "" {
		return "", fmt.Errorf("whisper model not found (download from https://huggingface.co/ggerganov/whisper.cpp)")
	}

	// Build whisper-cli arguments
	args := []string{"-m", modelPath, "-f", tmpFile.Name(), "--no-prints", "-nt"}

	// Add language option (default to Chinese)
	lang := opts.Language
	if lang == "" {
		lang = "zh" // Default to Chinese
	}
	args = append(args, "-l", lang)

	// Run whisper-cli
	cmd := exec.CommandContext(ctx, whisperPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("whisper failed: %w\n%s", err, output)
	}

	return strings.TrimSpace(string(output)), nil
}

// FindWhisperModel searches for a whisper model file
func FindWhisperModel() string {
	homeDir, _ := os.UserHomeDir()

	// Common model locations
	searchPaths := []string{
		// User-specific locations
		filepath.Join(homeDir, ".local", "share", "whisper", "ggml-base.bin"),
		filepath.Join(homeDir, ".local", "share", "whisper", "ggml-small.bin"),
		filepath.Join(homeDir, ".local", "share", "whisper", "ggml-tiny.bin"),
		filepath.Join(homeDir, ".cache", "whisper", "ggml-base.bin"),
		filepath.Join(homeDir, ".cache", "whisper", "ggml-small.bin"),
		filepath.Join(homeDir, ".cache", "whisper", "ggml-tiny.bin"),
		// Homebrew locations
		"/opt/homebrew/share/whisper-cpp/ggml-base.bin",
		"/opt/homebrew/share/whisper-cpp/ggml-small.bin",
		"/opt/homebrew/share/whisper-cpp/ggml-tiny.bin",
		"/opt/homebrew/Cellar/whisper-cpp/1.8.3/share/whisper-cpp/for-tests-ggml-tiny.bin",
		"/usr/local/share/whisper-cpp/ggml-base.bin",
		// Linux locations
		"/usr/share/whisper-cpp/ggml-base.bin",
		"/usr/local/share/whisper/ggml-base.bin",
	}

	// Check WHISPER_MODEL env var first
	if modelPath := os.Getenv("WHISPER_MODEL"); modelPath != "" {
		if _, err := os.Stat(modelPath); err == nil {
			return modelPath
		}
	}

	// Search common paths
	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// GetWhisperModelDir returns the directory for whisper models
func GetWhisperModelDir() string {
	homeDir, _ := os.UserHomeDir()

	// Use platform-appropriate location
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(homeDir, "AppData", "Local", "whisper")
	default:
		return filepath.Join(homeDir, ".local", "share", "whisper")
	}
}

// DownloadWhisperModel downloads a whisper model
func DownloadWhisperModel(model string) error {
	// Model URLs from huggingface
	modelURLs := map[string]string{
		"tiny":   "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin",
		"base":   "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin",
		"small":  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin",
		"medium": "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.bin",
		"large":  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large.bin",
	}

	url, ok := modelURLs[model]
	if !ok {
		return fmt.Errorf("unknown model: %s (available: tiny, base, small, medium, large)", model)
	}

	// Create model directory
	modelDir := GetWhisperModelDir()
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	modelPath := filepath.Join(modelDir, fmt.Sprintf("ggml-%s.bin", model))

	// Check if already exists
	if _, err := os.Stat(modelPath); err == nil {
		return nil // Already downloaded
	}

	// Download the model
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Create temp file first
	tmpFile, err := os.CreateTemp(modelDir, "download-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Download with progress
	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("download failed: %w", err)
	}

	// Rename to final location
	if err := os.Rename(tmpPath, modelPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save model: %w", err)
	}

	return nil
}

// OpenAIProvider uses OpenAI's TTS/STT APIs
type OpenAIProvider struct {
	apiKey string
	client *http.Client
}

// NewOpenAIProvider creates an OpenAI provider
func NewOpenAIProvider(apiKey string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key required")
	}
	return &OpenAIProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// TextToSpeech uses OpenAI TTS API
func (p *OpenAIProvider) TextToSpeech(ctx context.Context, text string, opts TTSOptions) ([]byte, error) {
	voice := opts.Voice
	if voice == "" {
		voice = "alloy"
	}

	reqBody, _ := json.Marshal(map[string]any{
		"model":           "tts-1",
		"input":           text,
		"voice":           voice,
		"response_format": "mp3",
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/speech", bytes.NewReader(reqBody))
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI TTS failed: %s", body)
	}

	return io.ReadAll(resp.Body)
}

// SpeechToText uses OpenAI Whisper API
func (p *OpenAIProvider) SpeechToText(ctx context.Context, audio []byte, opts STTOptions) (string, error) {
	// Create multipart form
	var buf bytes.Buffer
	boundary := "----WebKitFormBoundary7MA4YWxkTrZu0gW"

	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString("Content-Disposition: form-data; name=\"file\"; filename=\"audio.wav\"\r\n")
	buf.WriteString("Content-Type: audio/wav\r\n\r\n")
	buf.Write(audio)
	buf.WriteString("\r\n")

	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString("Content-Disposition: form-data; name=\"model\"\r\n\r\n")
	buf.WriteString("whisper-1\r\n")

	if opts.Language != "" {
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Disposition: form-data; name=\"language\"\r\n\r\n")
		buf.WriteString(opts.Language + "\r\n")
	}

	buf.WriteString("--" + boundary + "--\r\n")

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/transcriptions", &buf)
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("OpenAI STT failed: %s", body)
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Text, nil
}

// ElevenLabsProvider uses ElevenLabs TTS API
type ElevenLabsProvider struct {
	apiKey string
	client *http.Client
}

// NewElevenLabsProvider creates an ElevenLabs provider
func NewElevenLabsProvider(apiKey string) (*ElevenLabsProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("ElevenLabs API key required")
	}
	return &ElevenLabsProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

// Name returns the provider name
func (p *ElevenLabsProvider) Name() string {
	return "elevenlabs"
}

// TextToSpeech uses ElevenLabs TTS API
func (p *ElevenLabsProvider) TextToSpeech(ctx context.Context, text string, opts TTSOptions) ([]byte, error) {
	voiceID := opts.Voice
	if voiceID == "" {
		voiceID = "21m00Tcm4TlvDq8ikWAM" // Default Rachel voice
	}

	reqBody, _ := json.Marshal(map[string]any{
		"text":     text,
		"model_id": "eleven_monolingual_v1",
		"voice_settings": map[string]float64{
			"stability":        0.5,
			"similarity_boost": 0.75,
		},
	})

	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s", voiceID)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	req.Header.Set("xi-api-key", p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ElevenLabs TTS failed: %s", body)
	}

	return io.ReadAll(resp.Body)
}

// SpeechToText - ElevenLabs doesn't have STT, use system fallback
func (p *ElevenLabsProvider) SpeechToText(ctx context.Context, audio []byte, opts STTOptions) (string, error) {
	return NewSystemProvider().SpeechToText(ctx, audio, opts)
}

// Helper functions

func playAudio(audio []byte) error {
	tmpFile, err := os.CreateTemp("", "audio-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(audio); err != nil {
		return err
	}
	tmpFile.Close()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("afplay", tmpFile.Name())
	case "linux":
		cmd = exec.Command("aplay", tmpFile.Name())
	default:
		return fmt.Errorf("audio playback not supported on %s", runtime.GOOS)
	}

	return cmd.Run()
}

func recordAudio(ctx context.Context, duration time.Duration) ([]byte, error) {
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("recording-%d.wav", time.Now().UnixNano()))
	defer os.Remove(tmpFile)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		// Use sox for recording
		cmd = exec.CommandContext(ctx, "rec", "-q", "-r", "16000", "-c", "1", tmpFile, "trim", "0", fmt.Sprintf("%d", int(duration.Seconds())))
	case "linux":
		cmd = exec.CommandContext(ctx, "arecord", "-q", "-r", "16000", "-c", "1", "-f", "S16_LE", "-d", fmt.Sprintf("%d", int(duration.Seconds())), tmpFile)
	default:
		return nil, fmt.Errorf("audio recording not supported on %s", runtime.GOOS)
	}

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("recording failed: %w", err)
	}

	return os.ReadFile(tmpFile)
}
