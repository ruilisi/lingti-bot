package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pltanton/lingti-bot/internal/debug"
	"github.com/pltanton/lingti-bot/internal/router"
)

const (
	DefaultServerURL  = "wss://bot.lingti.com/ws"
	DefaultWebhookURL = "https://bot.lingti.com/webhook"
	ClientVersion     = "1.0.0"

	pingInterval      = 15 * time.Second
	writeTimeout      = 10 * time.Second
	readTimeout       = 60 * time.Second
	initialRetryDelay = 5 * time.Second
	maxRetryDelay     = 5 * time.Minute
)

// Config holds relay configuration
type Config struct {
	UserID     string // From /whoami
	Platform   string // "feishu" or "slack"
	ServerURL  string // WebSocket URL (default: wss://bot.lingti.com/ws)
	WebhookURL string // Webhook URL (default: https://bot.lingti.com/webhook)
	AIProvider string // AI provider name (e.g., "claude", "deepseek")
	AIModel    string // AI model name
}

// Platform implements router.Platform for cloud relay
type Platform struct {
	config         Config
	conn           *websocket.Conn
	connMu         sync.Mutex
	sessionID      string
	messageHandler func(msg router.Message)
	httpClient     *http.Client
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// Protocol message types

// AuthMessage is sent on WebSocket connect
type AuthMessage struct {
	Type          string `json:"type"`
	UserID        string `json:"user_id"`
	Platform      string `json:"platform"`
	ClientVersion string `json:"client_version"`
	AIProvider    string `json:"ai_provider,omitempty"`
	AIModel       string `json:"ai_model,omitempty"`
}

// AuthResult is the response to authentication
type AuthResult struct {
	Type      string `json:"type"`
	Success   bool   `json:"success"`
	SessionID string `json:"session_id"`
	Error     string `json:"error,omitempty"`
}

// IncomingMessage is a message from the server
type IncomingMessage struct {
	Type      string            `json:"type"`
	ID        string            `json:"id"`
	Platform  string            `json:"platform"`
	ChannelID string            `json:"channel_id"`
	UserID    string            `json:"user_id"`
	Username  string            `json:"username"`
	Text      string            `json:"text"`
	ThreadID  string            `json:"thread_id"`
	Metadata  map[string]string `json:"metadata"`
}

// OutgoingResponse is sent via webhook
type OutgoingResponse struct {
	Type      string `json:"type"`
	MessageID string `json:"message_id"`
	Platform  string `json:"platform"`
	ChannelID string `json:"channel_id"`
	Text      string `json:"text"`
}

// PingPong for heartbeat
type PingPong struct {
	Type string `json:"type"`
}

// New creates a new relay platform
func New(cfg Config) (*Platform, error) {
	if cfg.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if cfg.Platform == "" {
		return nil, fmt.Errorf("platform is required")
	}
	if cfg.Platform != "feishu" && cfg.Platform != "slack" {
		return nil, fmt.Errorf("platform must be 'feishu' or 'slack'")
	}

	if cfg.ServerURL == "" {
		cfg.ServerURL = DefaultServerURL
	}
	if cfg.WebhookURL == "" {
		cfg.WebhookURL = DefaultWebhookURL
	}

	return &Platform{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "relay"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins the relay connection
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	// Initial connection
	if err := p.connect(); err != nil {
		return fmt.Errorf("initial connection failed: %w", err)
	}

	// Start read loop
	p.wg.Add(1)
	go p.readLoop()

	// Start heartbeat
	p.wg.Add(1)
	go p.heartbeat()

	log.Printf("[Relay] Connected to %s as user %s (%s)", p.config.ServerURL, p.config.UserID, p.config.Platform)
	return nil
}

// Stop shuts down the relay connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}

	p.connMu.Lock()
	if p.conn != nil {
		p.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		p.conn.Close()
	}
	p.connMu.Unlock()

	p.wg.Wait()
	return nil
}

// Send sends a response via webhook
func (p *Platform) Send(ctx context.Context, channelID string, resp router.Response) error {
	outgoing := OutgoingResponse{
		Type:      "response",
		MessageID: resp.Metadata["message_id"],
		Platform:  p.config.Platform,
		ChannelID: channelID,
		Text:      resp.Text,
	}

	body, err := json.Marshal(outgoing)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Session-ID", p.sessionID)
	req.Header.Set("X-User-ID", p.config.UserID)

	httpResp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", httpResp.StatusCode)
	}

	return nil
}

// connect establishes WebSocket connection and authenticates
func (p *Platform) connect() error {
	debug.Log("Connecting to %s", p.config.ServerURL)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, resp, err := dialer.DialContext(p.ctx, p.config.ServerURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	debug.Log("WebSocket connected, status: %s", resp.Status)

	// Send authentication
	authMsg := AuthMessage{
		Type:          "auth",
		UserID:        p.config.UserID,
		Platform:      p.config.Platform,
		ClientVersion: ClientVersion,
		AIProvider:    p.config.AIProvider,
		AIModel:       p.config.AIModel,
	}

	debug.Log("Sending auth message")
	conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := conn.WriteJSON(authMsg); err != nil {
		conn.Close()
		return fmt.Errorf("failed to send auth: %w", err)
	}

	// Wait for auth response
	debug.Log("Waiting for auth response")
	conn.SetReadDeadline(time.Now().Add(readTimeout))
	var authResult AuthResult
	if err := conn.ReadJSON(&authResult); err != nil {
		conn.Close()
		return fmt.Errorf("failed to read auth response: %w", err)
	}
	debug.Log("Auth response: success=%v, session=%s", authResult.Success, authResult.SessionID)

	if authResult.Type != "auth_result" {
		conn.Close()
		return fmt.Errorf("unexpected response type: %s", authResult.Type)
	}

	if !authResult.Success {
		conn.Close()
		return fmt.Errorf("authentication failed: %s", authResult.Error)
	}

	// Set up pong handler to reset read deadline
	conn.SetPongHandler(func(appData string) error {
		debug.Log("Received pong")
		conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})

	// Set up ping handler
	conn.SetPingHandler(func(appData string) error {
		debug.Log("Received ping from server")
		conn.SetReadDeadline(time.Now().Add(readTimeout))
		conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		return conn.WriteMessage(websocket.PongMessage, []byte(appData))
	})

	p.connMu.Lock()
	p.conn = conn
	p.sessionID = authResult.SessionID
	p.connMu.Unlock()

	log.Printf("[Relay] Authenticated, session: %s", p.sessionID)
	return nil
}

// readLoop handles incoming WebSocket messages
func (p *Platform) readLoop() {
	defer p.wg.Done()

	retryDelay := initialRetryDelay

	for {
		select {
		case <-p.ctx.Done():
			debug.Log("Context done, exiting readLoop")
			return
		default:
		}

		p.connMu.Lock()
		conn := p.conn
		p.connMu.Unlock()

		if conn == nil {
			debug.Log("No connection, reconnecting")
			p.reconnect(&retryDelay)
			continue
		}

		debug.Log("Waiting for message (timeout: %v)", readTimeout)
		conn.SetReadDeadline(time.Now().Add(readTimeout))
		msgType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("[Relay] Connection closed normally")
				return
			}

			// Check for close error with reason (e.g., duplicate connection)
			if closeErr, ok := err.(*websocket.CloseError); ok {
				// Policy violation means another client connected with same user-id
				if closeErr.Code == websocket.ClosePolicyViolation {
					log.Printf("[Relay] Disconnected: %s", closeErr.Text)
					log.Printf("[Relay] Exiting - please ensure only one client is running per user-id")
					os.Exit(1)
				}
				if closeErr.Text != "" {
					log.Printf("[Relay] Connection closed by server: %s", closeErr.Text)
				} else {
					log.Printf("[Relay] Connection closed with code %d", closeErr.Code)
				}
			} else {
				debug.Log("Read error (msgType=%d): %v", msgType, err)
				log.Printf("[Relay] Read error: %v", err)
			}
			p.connMu.Lock()
			if p.conn != nil {
				p.conn.Close()
				p.conn = nil
			}
			p.connMu.Unlock()

			p.reconnect(&retryDelay)
			continue
		}

		debug.Log("Received message (type=%d, len=%d)", msgType, len(message))

		// Reset retry delay on successful read
		retryDelay = initialRetryDelay

		// Parse message type
		var jsonMsg struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(message, &jsonMsg); err != nil {
			debug.Log("Failed to parse JSON: %v, raw: %s", err, string(message))
			log.Printf("[Relay] Failed to parse message type: %v", err)
			continue
		}

		debug.Log("Message type: %s", jsonMsg.Type)

		switch jsonMsg.Type {
		case "ping":
			debug.Log("Received app-level ping, sending pong")
			p.sendPong()
		case "pong":
			debug.Log("Received app-level pong")
		case "message":
			debug.Log("Received message, handling")
			p.handleMessage(message)
		default:
			log.Printf("[Relay] Unknown message type: %s", jsonMsg.Type)
		}
	}
}

// handleMessage processes an incoming message
func (p *Platform) handleMessage(data []byte) {
	var msg IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("[Relay] Failed to parse message: %v", err)
		return
	}

	if p.messageHandler != nil {
		metadata := msg.Metadata
		if metadata == nil {
			metadata = make(map[string]string)
		}
		metadata["message_id"] = msg.ID

		p.messageHandler(router.Message{
			ID:        msg.ID,
			Platform:  "relay",
			ChannelID: msg.ChannelID,
			UserID:    msg.UserID,
			Username:  msg.Username,
			Text:      msg.Text,
			ThreadID:  msg.ThreadID,
			Metadata:  metadata,
		})
	}
}

// sendPong sends a pong response
func (p *Platform) sendPong() {
	p.connMu.Lock()
	defer p.connMu.Unlock()

	if p.conn == nil {
		return
	}

	pong := PingPong{Type: "pong"}
	p.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := p.conn.WriteJSON(pong); err != nil {
		log.Printf("[Relay] Failed to send pong: %v", err)
	}
}

// heartbeat sends periodic pings to keep connection alive
func (p *Platform) heartbeat() {
	defer p.wg.Done()

	debug.Log("Heartbeat started, waiting 500ms before first ping")

	// Short delay then send initial ping
	time.Sleep(500 * time.Millisecond)
	p.sendPing()

	// Send pings frequently to keep connection alive
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			debug.Log("Heartbeat stopped (context done)")
			return
		case <-ticker.C:
			p.sendPing()
		}
	}
}

func (p *Platform) sendPing() {
	p.connMu.Lock()
	conn := p.conn
	p.connMu.Unlock()

	if conn == nil {
		debug.Log("sendPing: no connection")
		return
	}

	debug.Log("Sending WebSocket ping")
	// Use WebSocket-level ping for better proxy/load balancer compatibility
	conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		debug.Log("sendPing error: %v", err)
		log.Printf("[Relay] Failed to send ping: %v", err)
	} else {
		debug.Log("Ping sent successfully")
	}
}

// reconnect attempts to reconnect with exponential backoff
func (p *Platform) reconnect(retryDelay *time.Duration) {
	select {
	case <-p.ctx.Done():
		return
	default:
	}

	log.Printf("[Relay] Reconnecting in %v...", *retryDelay)

	select {
	case <-p.ctx.Done():
		return
	case <-time.After(*retryDelay):
	}

	if err := p.connect(); err != nil {
		log.Printf("[Relay] Reconnection failed: %v", err)

		// Exponential backoff
		*retryDelay *= 2
		if *retryDelay > maxRetryDelay {
			*retryDelay = maxRetryDelay
		}
	} else {
		log.Printf("[Relay] Reconnected successfully")
		*retryDelay = initialRetryDelay
	}
}
