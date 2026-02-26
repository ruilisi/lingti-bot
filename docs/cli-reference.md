# Command Line Reference

Complete command line reference for lingti-bot.

## Table of Contents

- [Global Options](#global-options)
- [Commands](#commands)
  - [serve](#serve) - Start MCP server
  - [doctor](#doctor) - Check system health
  - [router](#router) - Start message router
  - [gateway](#gateway) - Start WebSocket gateway
  - [voice](#voice) - Voice input mode
  - [talk](#talk) - Continuous voice mode
  - [skills](#skills) - Manage modular skills
  - [setup](#setup) - Setup dependencies
  - [version](#version) - Show version
- [Environment Variables](#environment-variables)
- [AI Providers](#ai-providers)
- [Examples](#examples)

---

## Global Options

These options are available for all commands:

| Flag | Default | Description |
|------|---------|-------------|
| `-h, --help` | | Show help for any command |
| `--version` | | Show version information |
| `--log` | `info` | Log level: silent, info, verbose, very-verbose |

### Log Levels

| Level | Description |
|-------|-------------|
| `silent` | Minimal output, errors only |
| `info` | Default level, shows command execution |
| `verbose` | Shows more details including command results |
| `very-verbose` | Debug mode with all details (websocket ping/pong, etc.) |

**Examples:**

```bash
# Silent mode - minimal output
lingti-bot router --log silent

# Verbose mode - show command results
lingti-bot talk --log verbose

# Debug mode - show all details
lingti-bot gateway --log very-verbose
```

---

## Commands

### serve

Start the MCP (Model Context Protocol) server for integration with Claude Desktop, Cursor, and other MCP clients.

```bash
lingti-bot serve [flags]
```

**Flags:**

| Flag | Env Var | Description |
|------|---------|-------------|
| (none) | | Runs in stdio mode for MCP protocol |

**Example:**

```bash
# Start MCP server (typically called by MCP clients)
lingti-bot serve
```

**Configuration for Claude Desktop** (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "lingti-bot": {
      "command": "/path/to/lingti-bot",
      "args": ["serve"]
    }
  }
}
```

---

### doctor

Run diagnostic checks on configuration, credentials, connectivity, and required tools.

```bash
lingti-bot doctor
```

**Checks performed:**

| Check | Description |
|-------|-------------|
| Config file | `~/.lingti.yaml` exists and parses correctly |
| AI API key | Set via env var or config, valid format |
| AI connectivity | Test connection to AI provider API |
| Platform credentials | List which platforms have tokens configured |
| Required binaries | `gh`, `chrome`, `claude` available in PATH |
| Browser CDP | Connection to Chrome DevTools if `cdp_url` is set |
| MCP servers | External MCP server commands/URLs reachable |
| Temp directory | `/tmp` is writable |

**Output:** Colored checklist with ✓/✗ per check. Exit code 1 if any check fails.

**Example:**

```
lingti-bot doctor
=================
OS: darwin/arm64, Go: go1.24.0

Checks:
  ✓ Config file (~/.lingti.yaml) — loaded
  ✓ AI API key — set (sk-ant-a..., provider: claude)
  ✓ AI provider connectivity — reachable (HTTP 200)
  ✓ Platform credentials — wecom, telegram
  ✓ Binary: gh — found
  ✗ Binary: chrome — not found in PATH
  ✓ Binary: claude — found
  ✓ Temp directory — writable

7 passed, 1 failed
```

---

### router

Start the message router for multi-platform messaging (Slack, Telegram, Discord, Feishu). See [Router vs Relay](router-vs-relay.md) for when to use router vs relay.

```bash
lingti-bot router [flags]
```

**Flags:**

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--provider` | `AI_PROVIDER` | `claude` | AI provider: claude, deepseek, kimi |
| `--api-key` | `AI_API_KEY` | | AI API key (required) |
| `--base-url` | `AI_BASE_URL` | | Custom AI API base URL |
| `--model` | `AI_MODEL` | auto | Model name |
| `--instructions` | | | Path to custom instructions file (appended to system prompt) |
| `--slack-bot-token` | `SLACK_BOT_TOKEN` | | Slack bot token (xoxb-...) |
| `--slack-app-token` | `SLACK_APP_TOKEN` | | Slack app token (xapp-...) |
| `--telegram-token` | `TELEGRAM_BOT_TOKEN` | | Telegram bot token |
| `--discord-token` | `DISCORD_BOT_TOKEN` | | Discord bot token |
| `--feishu-app-id` | `FEISHU_APP_ID` | | Feishu app ID |
| `--feishu-app-secret` | `FEISHU_APP_SECRET` | | Feishu app secret |
| `--voice-stt-provider` | `VOICE_STT_PROVIDER` | | Voice STT provider for voice messages |
| `--voice-stt-api-key` | `VOICE_STT_API_KEY` | | Voice STT API key |

**Examples:**

```bash
# Slack only
lingti-bot router \
  --provider claude \
  --api-key sk-ant-xxx \
  --slack-bot-token xoxb-xxx \
  --slack-app-token xapp-xxx

# Telegram only
lingti-bot router \
  --provider claude \
  --api-key sk-ant-xxx \
  --telegram-token 123456:ABC-xxx

# Multiple platforms
lingti-bot router \
  --provider claude \
  --api-key sk-ant-xxx \
  --slack-bot-token xoxb-xxx \
  --slack-app-token xapp-xxx \
  --telegram-token 123456:ABC-xxx \
  --discord-token xxx

# With voice message transcription (Telegram)
lingti-bot router \
  --provider claude \
  --api-key sk-ant-xxx \
  --telegram-token 123456:ABC-xxx \
  --voice-stt-provider openai \
  --voice-stt-api-key sk-xxx

# Using environment variables
export AI_PROVIDER=claude
export AI_API_KEY=sk-ant-xxx
export TELEGRAM_BOT_TOKEN=123456:ABC-xxx
lingti-bot router
```

---

### gateway

Start the WebSocket gateway for real-time AI interaction from custom clients. See [docs/gateway.md](gateway.md) for the full protocol reference.

```bash
lingti-bot gateway [flags]
```

**Flags:**

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--addr` | `GATEWAY_ADDR` | `:18789` | Gateway listen address |
| `--auth-token` | `GATEWAY_AUTH_TOKEN` | | Single authentication token |
| `--auth-tokens` | `GATEWAY_AUTH_TOKENS` | | Comma-separated list of tokens (multiple admins) |
| `--provider` | `AI_PROVIDER` | `claude` | AI provider: claude, deepseek, kimi |
| `--api-key` | `AI_API_KEY` | | AI API key (required) |
| `--base-url` | `AI_BASE_URL` | | Custom AI API base URL |
| `--model` | `AI_MODEL` | auto | Model name |

**Auth tokens** control who can connect to the gateway. When no token is configured, all connections are accepted. When one or more tokens are set, a client must send a valid token in its `auth` message before it can send chat or command messages. Any token in the list grants full access — there is no per-token permission distinction.

**Examples:**

```bash
# Basic gateway (no authentication)
lingti-bot gateway \
  --provider claude \
  --api-key sk-ant-xxx

# Single admin token
lingti-bot gateway \
  --addr :8080 \
  --auth-token my-secret-token \
  --provider claude \
  --api-key sk-ant-xxx

# Multiple admins — each with their own token
lingti-bot gateway \
  --auth-tokens "token-alice,token-bob,token-charlie" \
  --provider claude \
  --api-key sk-ant-xxx

# Via environment variables
GATEWAY_AUTH_TOKENS="token-alice,token-bob" lingti-bot gateway --api-key sk-ant-xxx

# With custom base URL (proxy)
lingti-bot gateway \
  --provider claude \
  --api-key sk-ant-xxx \
  --base-url https://my-proxy.com/v1
```

**WebSocket Protocol:**

Connect to `ws://localhost:18789/ws` and send JSON messages:

```json
// Send chat message
{"type": "chat", "payload": {"text": "Hello", "session_id": "optional"}}

// Receive response
{"type": "response", "payload": {"text": "Hi there!", "done": true}}
```

**HTTP Endpoints:**

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Health check |
| `GET /status` | Gateway status and client count |
| `GET /ws` | WebSocket upgrade |

---

### relay

Connect to the lingti-bot cloud relay service — no public server needed. See [Router vs Relay](router-vs-relay.md) for when to use relay vs router.

```bash
lingti-bot relay [flags]
```

**Supported platforms:** feishu, slack, wechat, wecom

> Note: DingTalk uses Stream Mode (built-in serverless), so use `router` instead of `relay`.

**Flags:**

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--platform` | `RELAY_PLATFORM` | | Platform: feishu, slack, wechat, wecom (required) |
| `--user-id` | `RELAY_USER_ID` | | Your user ID from /whoami (required, except wecom) |
| `--provider` | `AI_PROVIDER` | `claude` | AI provider: claude, deepseek, kimi |
| `--api-key` | `AI_API_KEY` | | AI API key (required) |
| `--server` | `RELAY_SERVER_URL` | wss://bot.lingti.com/ws | WebSocket server URL |

WeCom-specific flags (when platform=wecom):

| Flag | Env Var | Description |
|------|---------|-------------|
| `--wecom-corp-id` | `WECOM_CORP_ID` | Corp ID |
| `--wecom-agent-id` | `WECOM_AGENT_ID` | Agent ID |
| `--wecom-secret` | `WECOM_SECRET` | Secret |
| `--wecom-token` | `WECOM_TOKEN` | Callback Token |
| `--wecom-aes-key` | `WECOM_AES_KEY` | Encoding AES Key |

**Examples:**

```bash
# Connect to WeCom cloud relay
lingti-bot relay --platform wecom \
  --wecom-corp-id CORP_ID \
  --wecom-agent-id AGENT_ID \
  --wecom-secret SECRET \
  --wecom-token TOKEN \
  --wecom-aes-key AES_KEY \
  --provider deepseek \
  --api-key sk-xxx

# Connect to Feishu relay
lingti-bot relay --platform feishu --user-id your-id --provider claude --api-key sk-ant-xxx
```

---

### voice

Interactive voice input mode - press Enter to record, speak, and get AI responses.

```bash
lingti-bot voice [flags]
```

**Flags:**

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `-d, --duration` | | `5` | Recording duration in seconds |
| `-s, --speak` | | `false` | Speak AI responses aloud (TTS) |
| `-l, --language` | | `zh` | Language for speech recognition (zh, en, ja, etc.) |
| `--provider` | `VOICE_PROVIDER` | `system` | Voice provider: system, openai |
| `--voice-api-key` | `VOICE_API_KEY` | | Voice API key (for openai provider) |
| `--voice-name` | | | Voice name for TTS |
| `--ai-provider` | `AI_PROVIDER` | `claude` | AI provider: claude, deepseek, kimi |
| `--api-key` | `AI_API_KEY` | | AI API key (required) |
| `--base-url` | `AI_BASE_URL` | | Custom AI API base URL |
| `--model` | `AI_MODEL` | auto | Model name |

**Examples:**

```bash
# Basic voice input (5 second recording, Chinese by default)
lingti-bot voice \
  --ai-provider claude \
  --api-key sk-ant-xxx

# English voice input
lingti-bot voice \
  --language en \
  --ai-provider claude \
  --api-key sk-ant-xxx

# Longer recording with spoken responses
lingti-bot voice \
  --duration 10 \
  --speak \
  --ai-provider claude \
  --api-key sk-ant-xxx

# Using OpenAI Whisper for better transcription
lingti-bot voice \
  --provider openai \
  --voice-api-key sk-xxx \
  --ai-provider claude \
  --api-key sk-ant-xxx

# Using Kimi with custom base URL
lingti-bot voice \
  --ai-provider kimi \
  --api-key sk-xxx \
  --base-url https://api.moonshot.cn/v1 \
  --model moonshot-v1-8k
```

**How it works:**

1. Press `Enter` to start recording (or type a message)
2. Speak your message (default 5 seconds)
3. Audio is transcribed to text
4. AI processes and responds
5. Response is displayed (and spoken if `--speak` enabled)

**Requirements:**

Run `lingti-bot setup` to check and install dependencies:

| Platform | Audio Recording | STT Engine |
|----------|-----------------|------------|
| macOS | sox or ffmpeg | whisper-cpp |
| Linux | alsa-utils or sox | openai-whisper |
| Windows | ffmpeg | openai-whisper |

---

### talk

Continuous voice conversation mode with optional wake word.

```bash
lingti-bot talk [flags]
```

**Flags:**

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--voice-provider` | `VOICE_PROVIDER` | `system` | Voice provider: system, openai, elevenlabs |
| `--voice-api-key` | `VOICE_API_KEY` | | Voice API key |
| `--wake-word` | `WAKE_WORD` | | Wake word for activation (e.g., "hey lingti") |
| `--continuous` | | `false` | Keep listening after each response |
| `--brief` | | `true` | Brief voice mode: print full text, speak only "已完成" |
| `--voice` | | | Default voice name for TTS |
| `--provider` | `AI_PROVIDER` | `claude` | AI provider: claude, deepseek, kimi |
| `--api-key` | `AI_API_KEY` | | AI API key (required) |
| `--base-url` | `AI_BASE_URL` | | Custom AI API base URL |
| `--model` | `AI_MODEL` | auto | Model name |

**Examples:**

```bash
# Basic talk mode (brief voice: print full text, speak "已完成")
lingti-bot talk \
  --provider claude \
  --api-key sk-ant-xxx

# Full voice output (speak entire response)
lingti-bot talk \
  --brief=false \
  --provider claude \
  --api-key sk-ant-xxx

# Continuous mode with wake word
lingti-bot talk \
  --continuous \
  --wake-word "hey lingti" \
  --provider claude \
  --api-key sk-ant-xxx

# Using ElevenLabs for high-quality TTS
lingti-bot talk \
  --voice-provider elevenlabs \
  --voice-api-key xxx \
  --provider claude \
  --api-key sk-ant-xxx
```

---

### setup

Check and install voice dependencies (audio tools, whisper, models).

```bash
lingti-bot setup [flags]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--all` | Install all dependencies |
| `-c, --component` | Component to setup: whisper, audio, all |

**Examples:**

```bash
# Check installation status
lingti-bot setup

# Install all dependencies
lingti-bot setup --all

# Download whisper model only
lingti-bot setup -c whisper

# Install audio tools only
lingti-bot setup -c audio
```

**Output example:**

```
╔════════════════════════════════════════════════════════════╗
║              Lingti Bot - Voice Setup                      ║
║              Platform: darwin/arm64                        ║
╚════════════════════════════════════════════════════════════╝

📋 Checking dependencies...

🎤 Audio Recording:
   ✅ sox (rec) - installed

🗣️ Speech-to-Text (Whisper):
   ✅ whisper-cli - installed

📦 Whisper Model:
   ✅ Model found: /Users/xxx/.local/share/whisper/ggml-base.bin

🔊 Audio Playback:
   ✅ afplay - installed (built-in)

────────────────────────────────────────────────────────────────
✅ All dependencies are installed! Run: lingti-bot voice
```

---

### version

Show version information.

```bash
lingti-bot version
```

---

## Environment Variables

### AI Provider Configuration

| Variable | Description | Example |
|----------|-------------|---------|
| `AI_PROVIDER` | AI provider name | `claude`, `deepseek`, `kimi` |
| `AI_API_KEY` | API key for AI provider | `sk-ant-xxx` |
| `AI_BASE_URL` | Custom API base URL | `https://api.anthropic.com` |
| `AI_MODEL` | Model name | `claude-sonnet-4-20250514` |

**Legacy variables (also supported):**

| Variable | Maps to |
|----------|---------|
| `ANTHROPIC_API_KEY` | `AI_API_KEY` |
| `ANTHROPIC_BASE_URL` | `AI_BASE_URL` |
| `ANTHROPIC_MODEL` | `AI_MODEL` |

### Platform Tokens

| Variable | Description |
|----------|-------------|
| `SLACK_BOT_TOKEN` | Slack bot token (xoxb-...) |
| `SLACK_APP_TOKEN` | Slack app token (xapp-...) |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token |
| `DISCORD_BOT_TOKEN` | Discord bot token |
| `FEISHU_APP_ID` | Feishu app ID |
| `FEISHU_APP_SECRET` | Feishu app secret |

### Voice Configuration

| Variable | Description |
|----------|-------------|
| `VOICE_PROVIDER` | Voice STT/TTS provider: system, openai, elevenlabs |
| `VOICE_API_KEY` | API key for voice provider |
| `VOICE_STT_PROVIDER` | STT-only provider (for router voice messages) |
| `VOICE_STT_API_KEY` | STT-only API key |
| `WHISPER_MODEL` | Custom path to whisper model file |
| `OPENAI_API_KEY` | OpenAI API key (fallback for voice) |
| `ELEVENLABS_API_KEY` | ElevenLabs API key (fallback for voice) |

### Gateway Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `GATEWAY_ADDR` | Gateway listen address | `:18789` |
| `GATEWAY_AUTH_TOKEN` | Single authentication token | |
| `GATEWAY_AUTH_TOKENS` | Comma-separated list of tokens (multiple admins) | |

---

## AI Providers

### Claude (Anthropic)

```bash
export AI_PROVIDER=claude
export AI_API_KEY=sk-ant-api03-xxx
export AI_MODEL=claude-sonnet-4-20250514  # optional
```

**Available models:**

| Model | Description |
|-------|-------------|
| `claude-sonnet-4-20250514` | Default, balanced performance |
| `claude-opus-4-20250514` | Most capable |
| `claude-haiku-3-20250307` | Fastest, most economical |

### DeepSeek

```bash
export AI_PROVIDER=deepseek
export AI_API_KEY=sk-xxx
export AI_BASE_URL=https://api.deepseek.com  # optional
export AI_MODEL=deepseek-chat  # optional
```

### Kimi (Moonshot)

```bash
export AI_PROVIDER=kimi
export AI_API_KEY=sk-xxx
export AI_BASE_URL=https://api.moonshot.cn/v1  # optional
export AI_MODEL=moonshot-v1-8k  # optional
```

**Available models:**

| Model | Context Window |
|-------|---------------|
| `moonshot-v1-8k` | 8K tokens |
| `moonshot-v1-32k` | 32K tokens |
| `moonshot-v1-128k` | 128K tokens |

---

## Examples

### Quick Start with Claude

```bash
# Set API key
export AI_API_KEY=sk-ant-xxx

# Voice input mode
lingti-bot voice

# Or with Telegram bot
export TELEGRAM_BOT_TOKEN=123456:ABC-xxx
lingti-bot router
```

### Using a Proxy

```bash
lingti-bot voice \
  --ai-provider claude \
  --api-key sk-ant-xxx \
  --base-url https://my-proxy.example.com/v1
```

### Multi-Platform Bot

```bash
export AI_API_KEY=sk-ant-xxx
export SLACK_BOT_TOKEN=xoxb-xxx
export SLACK_APP_TOKEN=xapp-xxx
export TELEGRAM_BOT_TOKEN=123456:ABC-xxx
export DISCORD_BOT_TOKEN=xxx

lingti-bot router
```

### Voice with OpenAI Whisper

```bash
lingti-bot voice \
  --provider openai \
  --voice-api-key $OPENAI_API_KEY \
  --ai-provider claude \
  --api-key $ANTHROPIC_API_KEY \
  --speak
```

### Docker Deployment

```bash
# Build the image
docker build -t lingti-bot .

# Run with environment variables
docker run -e AI_API_KEY=sk-xxx -e TELEGRAM_BOT_TOKEN=xxx lingti-bot

# Using docker-compose (recommended)
AI_API_KEY=sk-xxx docker compose up -d

# Mount config file for overrides and advanced settings
docker run -v ~/.lingti.yaml:/root/.lingti.yaml:ro \
  -e AI_API_KEY=sk-xxx lingti-bot router
```

### Docker / Headless Server

For servers without audio hardware, use the gateway or router commands:

```bash
# WebSocket gateway for custom clients
lingti-bot gateway --addr :18789 --api-key $AI_API_KEY

# Message router for chat platforms
lingti-bot router --api-key $AI_API_KEY --telegram-token $TELEGRAM_BOT_TOKEN
```

---

## See Also

- [Slack Integration Guide](slack-integration.md)
- [Feishu Integration Guide](feishu-integration.md)
- [OpenClaw Reference](openclaw-reference.md)
