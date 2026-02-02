package agent

import (
	"sync"
	"time"

	"github.com/liushuangls/go-anthropic/v2"
)

// ConversationMemory stores conversation history per user/channel
type ConversationMemory struct {
	conversations map[string]*Conversation
	mu            sync.RWMutex
	maxMessages   int           // Max messages to keep per conversation
	ttl           time.Duration // Time to live for conversations
}

// Conversation holds messages for a single conversation
type Conversation struct {
	Messages  []anthropic.Message
	UpdatedAt time.Time
}

// NewMemory creates a new conversation memory store
func NewMemory(maxMessages int, ttl time.Duration) *ConversationMemory {
	if maxMessages <= 0 {
		maxMessages = 20 // Default: keep last 20 messages
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute // Default: 30 minutes
	}

	m := &ConversationMemory{
		conversations: make(map[string]*Conversation),
		maxMessages:   maxMessages,
		ttl:           ttl,
	}

	// Start cleanup goroutine
	go m.cleanup()

	return m
}

// GetHistory returns the conversation history for a key (user+channel)
func (m *ConversationMemory) GetHistory(key string) []anthropic.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conv, ok := m.conversations[key]
	if !ok {
		return nil
	}

	// Check if expired
	if time.Since(conv.UpdatedAt) > m.ttl {
		return nil
	}

	// Return a copy
	messages := make([]anthropic.Message, len(conv.Messages))
	copy(messages, conv.Messages)
	return messages
}

// AddMessage adds a message to the conversation history
func (m *ConversationMemory) AddMessage(key string, msg anthropic.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conv, ok := m.conversations[key]
	if !ok {
		conv = &Conversation{
			Messages: make([]anthropic.Message, 0),
		}
		m.conversations[key] = conv
	}

	conv.Messages = append(conv.Messages, msg)
	conv.UpdatedAt = time.Now()

	// Trim if exceeds max
	if len(conv.Messages) > m.maxMessages {
		// Keep the last maxMessages, but always keep pairs (user+assistant)
		startIdx := len(conv.Messages) - m.maxMessages
		if startIdx%2 != 0 {
			startIdx++ // Ensure we start with a user message
		}
		conv.Messages = conv.Messages[startIdx:]
	}
}

// AddExchange adds both user and assistant messages
func (m *ConversationMemory) AddExchange(key string, userMsg, assistantMsg anthropic.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conv, ok := m.conversations[key]
	if !ok {
		conv = &Conversation{
			Messages: make([]anthropic.Message, 0),
		}
		m.conversations[key] = conv
	}

	conv.Messages = append(conv.Messages, userMsg, assistantMsg)
	conv.UpdatedAt = time.Now()

	// Trim if exceeds max
	if len(conv.Messages) > m.maxMessages {
		startIdx := len(conv.Messages) - m.maxMessages
		if startIdx%2 != 0 {
			startIdx++
		}
		conv.Messages = conv.Messages[startIdx:]
	}
}

// Clear clears the conversation history for a key
func (m *ConversationMemory) Clear(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.conversations, key)
}

// ClearAll clears all conversation histories
func (m *ConversationMemory) ClearAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conversations = make(map[string]*Conversation)
}

// cleanup periodically removes expired conversations
func (m *ConversationMemory) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, conv := range m.conversations {
			if now.Sub(conv.UpdatedAt) > m.ttl {
				delete(m.conversations, key)
			}
		}
		m.mu.Unlock()
	}
}

// ConversationKey generates a unique key for a conversation
func ConversationKey(platform, channelID, userID string) string {
	// Use channel+user for unique conversations
	// This means each user has their own context per channel
	return platform + ":" + channelID + ":" + userID
}
