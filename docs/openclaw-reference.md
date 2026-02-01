# OpenClaw Reference

OpenClaw is an open-source autonomous AI personal assistant created by Peter Steinberger in late 2025. This document serves as a reference for understanding its architecture and features, which can inform the design of lingti-bot.

## Overview

OpenClaw is a self-hosted agent runtime and message router that acts as a personal AI assistant running on your own machine. It went viral with 100k+ GitHub stars and 2M visitors in one week.

Originally released as "Clawdbot", then renamed "Moltbot" (after Anthropic's trademark request), and finally "OpenClaw" in early 2026.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    OpenClaw Runtime                      │
│                    (Node.js Service)                     │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │   Message    │  │   Agentic    │  │  Persistent  │   │
│  │   Router     │  │    Loop      │  │   Memory     │   │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘   │
│         │                 │                 │            │
└─────────┼─────────────────┼─────────────────┼────────────┘
          │                 │                 │
          ▼                 ▼                 ▼
┌─────────────────┐  ┌─────────────┐  ┌─────────────────┐
│ Chat Platforms  │  │  External   │  │  Local Machine  │
│ - WhatsApp      │  │  AI Models  │  │  - Files        │
│ - Telegram      │  │  - Claude   │  │  - Apps         │
│ - Discord       │  │  - GPT      │  │  - Calendar     │
│ - Slack         │  │  - etc      │  │  - etc          │
│ - iMessage      │  │             │  │                 │
│ - Signal        │  │             │  │                 │
│ - Teams         │  │             │  │                 │
└─────────────────┘  └─────────────┘  └─────────────────┘
```

## How It Works

1. **Self-hosted agent runtime** - Runs as a long-running Node.js service on your machine
2. **Multi-channel messaging** - Connects to WhatsApp, Telegram, Discord, Slack, iMessage, Signal, Teams, Matrix, Google Chat, etc.
3. **Agentic loop** - Executes tasks autonomously, not just conversational chat

## Key Features

| Feature | Description |
|---------|-------------|
| **Persistent memory** | Remembers context across sessions, learns patterns over time |
| **Local machine access** | Deep integration with your local apps and files |
| **Heartbeat** | Proactively wakes up and takes action without user prompting |
| **Multi-platform inbox** | One AI assistant across all messaging platforms |
| **Contextual understanding** | Learns relationships (e.g., recognizes work emails vs personal) |

## Core Capabilities

### Three Pillars

OpenClaw works because it does three simple things:

1. **Persistent memory across sessions** - Retains context between conversations
2. **Deep local machine access** - Unapologetic access to your apps and files
3. **Autonomous agentic loop** - Takes action, not just suggests steps

### Supported Platforms

**Messaging:**
- WhatsApp
- Telegram
- Discord
- Slack
- iMessage
- Signal
- Microsoft Teams
- Matrix
- Google Chat

**Productivity Apps:**
- Apple Notes
- Apple Reminders
- Things 3
- Notion
- Obsidian
- Trello
- GitHub

## Use Cases

### Developer & Technical Workflows
- Automate debugging and DevOps tasks
- Codebase management with GitHub integration
- Scheduled cron jobs and webhook triggers

### Personal Productivity
- Manage calendars and reminders
- Cross-app task management from a single conversation
- Research and information gathering

### Autonomous Actions
Notable example: When unable to make a restaurant reservation through OpenTable, OpenClaw obtained AI voice software and called the restaurant directly to secure the reservation.

## Security Considerations

OpenClaw explicitly trades security for capability. From their FAQ:

> "There is no 'perfectly secure' setup."

**Recommendations:**
- Best used on personal devices
- Not recommended for enterprise environments without proper guardrails
- Consider using a separate device for OpenClaw

## Relevance to lingti-bot

Features to consider adopting:

1. **Multi-channel support** - Currently lingti-bot uses MCP stdio, could expand to messaging platforms
2. **Persistent memory** - Add session memory across tool calls
3. **Heartbeat/proactive actions** - Scheduled tasks without user prompting
4. **Contextual learning** - Learn user patterns over time

## References

- [OpenClaw Official Website](https://openclaw.ai/)
- [OpenClaw Documentation](https://docs.openclaw.ai/)
- [GitHub Repository](https://github.com/openclaw/openclaw)
- [Wikipedia](https://en.wikipedia.org/wiki/OpenClaw)
- [DigitalOcean Guide](https://www.digitalocean.com/resources/articles/what-is-openclaw)
- [IBM Analysis](https://www.ibm.com/think/news/clawdbot-ai-agent-testing-limits-vertical-integration)
