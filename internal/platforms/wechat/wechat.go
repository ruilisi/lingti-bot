package wechat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	tokenURL      = "https://api.weixin.qq.com/cgi-bin/token"
	uploadMediaURL = "https://api.weixin.qq.com/cgi-bin/media/upload"
	customSendURL  = "https://api.weixin.qq.com/cgi-bin/message/custom/send"
)

// Client is a lightweight WeChat Official Account API client for media upload/send.
type Client struct {
	appID       string
	appSecret   string
	accessToken string
	tokenExpiry time.Time
	tokenMu     sync.RWMutex
	httpClient  *http.Client
}

// NewClient creates a new WeChat OA API client.
func NewClient(appID, appSecret string) *Client {
	return &Client{
		appID:     appID,
		appSecret: appSecret,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// GetToken returns a valid access token, refreshing if needed.
func (c *Client) GetToken() (string, error) {
	c.tokenMu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		token := c.accessToken
		c.tokenMu.RUnlock()
		return token, nil
	}
	c.tokenMu.RUnlock()

	return c.refreshToken()
}

func (c *Client) refreshToken() (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Double-check after acquiring write lock
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	url := fmt.Sprintf("%s?grant_type=client_credential&appid=%s&secret=%s",
		tokenURL, c.appID, c.appSecret)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to request access token: %w", err)
	}
	defer resp.Body.Close()

	var result tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("WeChat token error %d: %s", result.ErrCode, result.ErrMsg)
	}

	c.accessToken = result.AccessToken
	// Refresh 5 minutes before expiry
	c.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn-300) * time.Second)

	log.Printf("[WeChat] Access token refreshed, expires in %ds", result.ExpiresIn)
	return c.accessToken, nil
}

type uploadResponse struct {
	Type      string `json:"type"`
	MediaID   string `json:"media_id"`
	CreatedAt int64  `json:"created_at"`
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
}

// UploadMedia uploads a file to WeChat temporary media storage.
// mediaType: "image", "voice", "video", "thumb"
func (c *Client) UploadMedia(filePath, mediaType string) (string, error) {
	token, err := c.GetToken()
	if err != nil {
		return "", err
	}

	body, contentType, err := buildMultipartBody(filePath)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s?access_token=%s&type=%s", uploadMediaURL, token, mediaType)
	resp, err := c.httpClient.Post(url, contentType, body)
	if err != nil {
		return "", fmt.Errorf("failed to upload media: %w", err)
	}
	defer resp.Body.Close()

	var result uploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode upload response: %w", err)
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("WeChat upload error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return result.MediaID, nil
}

// SendImage sends an image message via the customer service API.
func (c *Client) SendImage(openID, mediaID string) error {
	return c.sendCustomMessage(openID, "image", map[string]any{
		"media_id": mediaID,
	})
}

// SendVoice sends a voice message via the customer service API.
func (c *Client) SendVoice(openID, mediaID string) error {
	return c.sendCustomMessage(openID, "voice", map[string]any{
		"media_id": mediaID,
	})
}

// SendVideo sends a video message via the customer service API.
func (c *Client) SendVideo(openID, mediaID, title, description string) error {
	return c.sendCustomMessage(openID, "video", map[string]any{
		"media_id":    mediaID,
		"title":       title,
		"description": description,
	})
}

func (c *Client) sendCustomMessage(openID, msgType string, content map[string]any) error {
	token, err := c.GetToken()
	if err != nil {
		return err
	}

	msg := map[string]any{
		"touser":  openID,
		"msgtype": msgType,
		msgType:   content,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	url := fmt.Sprintf("%s?access_token=%s", customSendURL, token)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to send custom message: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode send response: %w", err)
	}

	if result.ErrCode != 0 {
		return fmt.Errorf("WeChat send error %d: %s", result.ErrCode, result.ErrMsg)
	}

	return nil
}

// buildMultipartBody creates a multipart/form-data body with the file in a "media" field.
func buildMultipartBody(filePath string) (*bytes.Buffer, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("media", filepath.Base(filePath))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, "", fmt.Errorf("failed to copy file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return body, writer.FormDataContentType(), nil
}
