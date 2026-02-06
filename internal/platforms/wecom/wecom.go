package wecom

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/pltanton/lingti-bot/internal/logger"
	"github.com/pltanton/lingti-bot/internal/router"
)

const (
	tokenURL   = "https://qyapi.weixin.qq.com/cgi-bin/gettoken"
	sendMsgURL = "https://qyapi.weixin.qq.com/cgi-bin/message/send"
)

// Platform implements router.Platform for WeChat Work (企业微信)
type Platform struct {
	corpID         string
	agentID        string
	secret         string
	token          string
	encodingAESKey string

	msgCrypt       *MsgCrypt
	accessToken    string
	tokenExpiry    time.Time
	tokenMu        sync.RWMutex
	messageHandler func(msg router.Message)
	server         *http.Server
	ctx            context.Context
	cancel         context.CancelFunc
}

// Config holds WeChat Work configuration
type Config struct {
	CorpID         string // 企业ID
	AgentID        string // 应用AgentId
	Secret         string // 应用Secret
	Token          string // 回调Token
	EncodingAESKey string // 回调EncodingAESKey
	CallbackPort   int    // 回调服务端口 (default: 8080)
}

// New creates a new WeChat Work platform
func New(cfg Config) (*Platform, error) {
	if cfg.CorpID == "" || cfg.AgentID == "" || cfg.Secret == "" {
		return nil, fmt.Errorf("CorpID, AgentID, and Secret are required")
	}
	if cfg.Token == "" || cfg.EncodingAESKey == "" {
		return nil, fmt.Errorf("Token and EncodingAESKey are required for callback")
	}

	msgCrypt, err := NewMsgCrypt(cfg.Token, cfg.EncodingAESKey, cfg.CorpID)
	if err != nil {
		return nil, fmt.Errorf("failed to create message cryptographer: %w", err)
	}

	p := &Platform{
		corpID:         cfg.CorpID,
		agentID:        cfg.AgentID,
		secret:         cfg.Secret,
		token:          cfg.Token,
		encodingAESKey: cfg.EncodingAESKey,
		msgCrypt:       msgCrypt,
	}

	// Set up HTTP server for callbacks (skip if CallbackPort < 0, e.g. API-only mode)
	if cfg.CallbackPort >= 0 {
		port := cfg.CallbackPort
		if port == 0 {
			port = 8080
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/wecom/callback", p.handleCallback)
		p.server = &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		}
	}

	return p, nil
}

// Name returns the platform name
func (p *Platform) Name() string {
	return "wecom"
}

// SetMessageHandler sets the callback for incoming messages
func (p *Platform) SetMessageHandler(handler func(msg router.Message)) {
	p.messageHandler = handler
}

// Start begins listening for WeChat Work events
func (p *Platform) Start(ctx context.Context) error {
	p.ctx, p.cancel = context.WithCancel(ctx)

	// Get initial access token
	if err := p.refreshToken(); err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	// Start token refresh goroutine
	go p.tokenRefreshLoop()

	// Start HTTP server (if configured)
	if p.server != nil {
		go func() {
			logger.Info("[WeCom] Starting callback server on %s", p.server.Addr)
			if err := p.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("[WeCom] Server error: %v", err)
			}
		}()
	}

	logger.Info("[WeCom] Connected with CorpID: %s, AgentID: %s", p.corpID, p.agentID)
	return nil
}

// Stop shuts down the WeChat Work connection
func (p *Platform) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return p.server.Shutdown(ctx)
	}
	return nil
}

// Send sends a message to a WeChat Work user
func (p *Platform) Send(ctx context.Context, userID string, resp router.Response) error {
	// Send text message if present
	if resp.Text != "" {
		if err := p.sendTextMessage(userID, resp.Text); err != nil {
			return err
		}
	}

	// Send file attachments — notify user on per-file errors and continue
	var failCount int
	for _, file := range resp.Files {
		mediaType := file.MediaType
		if mediaType == "" {
			mediaType = "file"
		}

		name := file.Name
		if name == "" {
			name = filepath.Base(file.Path)
		}

		mediaID, err := p.UploadMedia(file.Path, mediaType)
		if err != nil {
			logger.Error("[WeCom] Failed to upload %s: %v", file.Path, err)
			_ = p.sendTextMessage(userID, fmt.Sprintf("[Error] Failed to send file \"%s\": %v", name, err))
			failCount++
			continue
		}

		if err := p.SendMediaMessage(userID, mediaID, mediaType); err != nil {
			logger.Error("[WeCom] Failed to send media %s: %v", file.Path, err)
			_ = p.sendTextMessage(userID, fmt.Sprintf("[Error] Failed to send file \"%s\": %v", name, err))
			failCount++
			continue
		}
	}

	if failCount > 0 {
		return fmt.Errorf("failed to send %d file(s)", failCount)
	}
	return nil
}

// sendTextMessage sends a text message to a user.
func (p *Platform) sendTextMessage(userID string, text string) error {
	token, err := p.getToken()
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	agentID, _ := strconv.Atoi(p.agentID)
	msg := map[string]any{
		"touser":  userID,
		"msgtype": "text",
		"agentid": agentID,
		"text": map[string]string{
			"content": text,
		},
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("%s?access_token=%s", sendMsgURL, token)
	httpResp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer httpResp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("API error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// handleCallback handles incoming callback requests from WeChat Work
func (p *Platform) handleCallback(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	msgSignature := query.Get("msg_signature")
	timestamp := query.Get("timestamp")
	nonce := query.Get("nonce")

	switch r.Method {
	case http.MethodGet:
		// URL verification
		echostr := query.Get("echostr")
		plaintext, err := p.msgCrypt.VerifyURL(msgSignature, timestamp, nonce, echostr)
		if err != nil {
			logger.Error("[WeCom] URL verification failed: %v", err)
			http.Error(w, "verification failed", http.StatusForbidden)
			return
		}
		w.Write([]byte(plaintext))

	case http.MethodPost:
		// Message handling
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("[WeCom] Failed to read body: %v", err)
			http.Error(w, "read failed", http.StatusBadRequest)
			return
		}

		var encryptedMsg EncryptedMsg
		if err := xml.Unmarshal(body, &encryptedMsg); err != nil {
			logger.Error("[WeCom] Failed to parse XML: %v", err)
			http.Error(w, "parse failed", http.StatusBadRequest)
			return
		}

		plaintext, err := p.msgCrypt.DecryptMsg(msgSignature, timestamp, nonce, &encryptedMsg)
		if err != nil {
			logger.Error("[WeCom] Failed to decrypt message: %v", err)
			http.Error(w, "decrypt failed", http.StatusBadRequest)
			return
		}

		// Parse the decrypted message
		p.processMessage(plaintext)

		// Return success (empty response)
		w.WriteHeader(http.StatusOK)
	}
}

// ReceivedMsg represents a received message from WeChat Work
type ReceivedMsg struct {
	XMLName      xml.Name `xml:"xml"`
	ToUserName   string   `xml:"ToUserName"`
	FromUserName string   `xml:"FromUserName"`
	CreateTime   int64    `xml:"CreateTime"`
	MsgType      string   `xml:"MsgType"`
	Content      string   `xml:"Content"`
	MsgId        string   `xml:"MsgId"`
	AgentID      string   `xml:"AgentID"`
	// Media fields
	MediaId  string `xml:"MediaId"`
	Format   string `xml:"Format"`   // Voice format (amr)
	PicUrl   string `xml:"PicUrl"`   // Image thumbnail URL
	FileName string `xml:"FileName"` // File name
	FileSize string `xml:"FileSize"` // File size
	// Event fields
	Event    string `xml:"Event"`
	EventKey string `xml:"EventKey"`
}

// processMessage processes the decrypted message
func (p *Platform) processMessage(plaintext []byte) {
	var msg ReceivedMsg
	if err := xml.Unmarshal(plaintext, &msg); err != nil {
		logger.Error("[WeCom] Failed to parse message: %v", err)
		return
	}

	// Skip event messages
	if msg.MsgType == "event" {
		logger.Debug("[WeCom] Ignoring event: %s", msg.Event)
		return
	}

	routerMsg := router.Message{
		ID:        msg.MsgId,
		Platform:  "wecom",
		ChannelID: msg.FromUserName,
		UserID:    msg.FromUserName,
		Username:  msg.FromUserName,
		Text:      msg.Content,
		Metadata: map[string]string{
			"agent_id": msg.AgentID,
			"msg_type": msg.MsgType,
		},
	}

	switch msg.MsgType {
	case "text":
		// Text is already set via msg.Content
	case "image":
		routerMsg.MediaID = msg.MediaId
		routerMsg.Text = "[图片]"
		routerMsg.Metadata["pic_url"] = msg.PicUrl
	case "voice":
		routerMsg.MediaID = msg.MediaId
		routerMsg.Text = "[语音]"
		routerMsg.Metadata["format"] = msg.Format
	case "video":
		routerMsg.MediaID = msg.MediaId
		routerMsg.Text = "[视频]"
	case "file":
		routerMsg.MediaID = msg.MediaId
		routerMsg.FileName = msg.FileName
		routerMsg.Text = "[文件] " + msg.FileName
		routerMsg.Metadata["file_size"] = msg.FileSize
	default:
		logger.Debug("[WeCom] Ignoring message type: %s", msg.MsgType)
		return
	}

	if p.messageHandler != nil {
		p.messageHandler(routerMsg)
	}
}

// Token management

type tokenResponse struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

func (p *Platform) refreshToken() error {
	url := fmt.Sprintf("%s?corpid=%s&corpsecret=%s", tokenURL, p.corpID, p.secret)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to request token: %w", err)
	}
	defer resp.Body.Close()

	var result tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("token API error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	p.tokenMu.Lock()
	p.accessToken = result.AccessToken
	// Refresh 5 minutes before expiry
	p.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)
	p.tokenMu.Unlock()

	logger.Debug("[WeCom] Access token refreshed, expires in %d seconds", result.ExpiresIn)
	return nil
}

func (p *Platform) getToken() (string, error) {
	p.tokenMu.RLock()
	token := p.accessToken
	expiry := p.tokenExpiry
	p.tokenMu.RUnlock()

	if time.Now().After(expiry) {
		if err := p.refreshToken(); err != nil {
			return "", err
		}
		p.tokenMu.RLock()
		token = p.accessToken
		p.tokenMu.RUnlock()
	}

	return token, nil
}

func (p *Platform) tokenRefreshLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			if err := p.refreshToken(); err != nil {
				logger.Error("[WeCom] Failed to refresh token: %v", err)
			}
		}
	}
}
