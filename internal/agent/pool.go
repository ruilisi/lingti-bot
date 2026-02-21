package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/pltanton/lingti-bot/internal/config"
	"github.com/pltanton/lingti-bot/internal/logger"
	"github.com/pltanton/lingti-bot/internal/router"
)

// AgentPool manages multiple agents for per-platform/channel model overrides.
// When no overrides match, it uses the default agent.
type AgentPool struct {
	defaultAgent *Agent
	baseCfg      Config
	fullCfg      *config.Config
	agents       map[string]*Agent
	mu           sync.RWMutex
}

// NewAgentPool creates a pool with a default agent and config for overrides.
func NewAgentPool(defaultAgent *Agent, baseCfg Config, fullCfg *config.Config) *AgentPool {
	return &AgentPool{
		defaultAgent: defaultAgent,
		baseCfg:      baseCfg,
		fullCfg:      fullCfg,
		agents:       make(map[string]*Agent),
	}
}

// DefaultAgent returns the default agent (for setting cron scheduler, etc.)
func (p *AgentPool) DefaultAgent() *Agent {
	return p.defaultAgent
}

// HandleMessage resolves the right agent for the message and delegates.
func (p *AgentPool) HandleMessage(ctx context.Context, msg router.Message) (router.Response, error) {
	if p.fullCfg == nil || len(p.fullCfg.AI.Overrides) == 0 {
		return p.defaultAgent.HandleMessage(ctx, msg)
	}

	platform := msg.Platform
	if ap, ok := msg.Metadata["actual_platform"]; ok && ap != "" {
		platform = ap
	}
	resolved := p.fullCfg.ResolveAI(platform, msg.ChannelID)
	// If resolved config matches default, use default agent
	if resolved.Provider == p.fullCfg.AI.Provider &&
		resolved.APIKey == p.fullCfg.AI.APIKey &&
		resolved.Model == p.fullCfg.AI.Model {
		return p.defaultAgent.HandleMessage(ctx, msg)
	}

	agent := p.getOrCreate(resolved)
	if agent == nil {
		return p.defaultAgent.HandleMessage(ctx, msg)
	}
	return agent.HandleMessage(ctx, msg)
}

func (p *AgentPool) getOrCreate(aiCfg config.AIConfig) *Agent {
	key := fmt.Sprintf("%s:%s:%s", aiCfg.Provider, aiCfg.APIKey, aiCfg.Model)

	p.mu.RLock()
	if a, ok := p.agents[key]; ok {
		p.mu.RUnlock()
		return a
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check
	if a, ok := p.agents[key]; ok {
		return a
	}

	cfg := p.baseCfg
	cfg.Provider = aiCfg.Provider
	cfg.APIKey = aiCfg.APIKey
	cfg.BaseURL = aiCfg.BaseURL
	cfg.Model = aiCfg.Model

	a, err := New(cfg)
	if err != nil {
		logger.Error("[AgentPool] Failed to create agent for %s/%s: %v", aiCfg.Provider, aiCfg.Model, err)
		return nil
	}

	// Share cron scheduler from default agent
	if p.defaultAgent.cronScheduler != nil {
		a.SetCronScheduler(p.defaultAgent.cronScheduler)
	}

	logger.Info("[AgentPool] Created agent for provider=%s model=%s", aiCfg.Provider, aiCfg.Model)
	p.agents[key] = a
	return a
}
