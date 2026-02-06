package router

import (
	"context"
	"fmt"
	"sync"

	"github.com/pltanton/lingti-bot/internal/logger"
)

// Message represents an incoming message from any platform
type Message struct {
	ID        string
	Platform  string            // "slack", "telegram", "discord", etc.
	ChannelID string            // Channel/Chat ID
	UserID    string            // User who sent the message
	Username  string            // Human-readable username
	Text      string            // Message content
	ThreadID  string            // For threaded replies
	MediaID   string            // Media file ID (for file/image/voice/video messages)
	FileName  string            // Original filename (for file messages)
	Metadata  map[string]string // Platform-specific metadata
}

// FileAttachment represents a file to upload and send
type FileAttachment struct {
	Path      string // Local file path to upload and send
	Name      string // Display name (defaults to filepath.Base)
	MediaType string // "file", "image", "voice", "video" (default: "file")
}

// Response represents a response to send back
type Response struct {
	Text     string
	Files    []FileAttachment  // File attachments to send
	ThreadID string            // Reply in thread if set
	Metadata map[string]string // Platform-specific options
}

// Platform interface for messaging platforms
type Platform interface {
	Name() string
	Start(ctx context.Context) error
	Stop() error
	Send(ctx context.Context, channelID string, resp Response) error
	SetMessageHandler(handler func(msg Message))
}

// MessageHandler processes incoming messages and returns responses
type MessageHandler func(ctx context.Context, msg Message) (Response, error)

// Router manages multiple messaging platforms
type Router struct {
	platforms map[string]Platform
	handler   MessageHandler
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// New creates a new Router
func New(handler MessageHandler) *Router {
	return &Router{
		platforms: make(map[string]Platform),
		handler:   handler,
	}
}

// Register adds a platform to the router
func (r *Router) Register(platform Platform) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := platform.Name()
	r.platforms[name] = platform

	// Set up message handling for this platform
	platform.SetMessageHandler(func(msg Message) {
		go r.handleMessage(msg)
	})

	logger.Info("[Router] Registered platform: %s", name)
}

// handleMessage processes an incoming message
func (r *Router) handleMessage(msg Message) {
	ctx := context.Background()

	logger.Info("[Router] Message from %s/%s: %s", msg.Platform, msg.Username, msg.Text)

	// Call the message handler
	resp, err := r.handler(ctx, msg)
	if err != nil {
		logger.Error("[Router] Error handling message: %v", err)
		resp = Response{Text: "Sorry, I encountered an error processing your request."}
	}

	// Send response back to the platform
	r.mu.RLock()
	platform, ok := r.platforms[msg.Platform]
	r.mu.RUnlock()

	if ok && (resp.Text != "" || len(resp.Files) > 0) {
		if msg.ThreadID != "" {
			resp.ThreadID = msg.ThreadID
		}
		if err := platform.Send(ctx, msg.ChannelID, resp); err != nil {
			logger.Error("[Router] Error sending response: %v", err)
			// Try to notify the user about the error in chat
			errResp := Response{
				Text:     fmt.Sprintf("[Error] %v", err),
				ThreadID: resp.ThreadID,
			}
			if notifyErr := platform.Send(ctx, msg.ChannelID, errResp); notifyErr != nil {
				logger.Error("[Router] Failed to send error notification: %v", notifyErr)
			}
		}
	}
}

// Start begins listening on all registered platforms
func (r *Router) Start(ctx context.Context) error {
	r.ctx, r.cancel = context.WithCancel(ctx)

	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, platform := range r.platforms {
		logger.Info("[Router] Starting platform: %s", name)
		if err := platform.Start(r.ctx); err != nil {
			return err
		}
	}

	logger.Info("[Router] All platforms started")
	return nil
}

// Stop shuts down all platforms
func (r *Router) Stop() error {
	if r.cancel != nil {
		r.cancel()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, platform := range r.platforms {
		logger.Info("[Router] Stopping platform: %s", name)
		if err := platform.Stop(); err != nil {
			logger.Error("[Router] Error stopping %s: %v", name, err)
		}
	}

	return nil
}

// Wait blocks until the router is stopped
func (r *Router) Wait() {
	if r.ctx != nil {
		<-r.ctx.Done()
	}
}
