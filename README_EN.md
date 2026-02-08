English | [中文](./README.md)

---

# lingti-bot (Lingti) 🐕⚡

> 🐕⚡ **Minimal · Efficient · Compile Once Run Anywhere · Lightning Integration** AI Bot

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Website](https://img.shields.io/badge/Website-cli.lingti.com-blue?style=flat)](https://cli.lingti.com/bot)

**Lingti Bot** is an all-in-one AI Bot platform featuring **MCP Server**, **multi-platform messaging gateway**, **rich toolset**, **intelligent conversation**, and **voice interaction**.

**Core Advantages:**
- 🚀 **Zero Dependency** — Single 30MB binary, no Node.js/Python runtime needed, just `scp` and run
- ☁️ **Cloud Relay** — No public server, domain registration, or HTTPS certificate needed, 5-min setup for WeCom/WeChat
- 🤖 **Browser Automation** — Built-in CDP protocol control, snapshot-then-act pattern, no Puppeteer/Playwright installation
- 🛠️ **75+ MCP Tools** — Covers files, Shell, system, network, calendar, Git, GitHub, and more
- 🌏 **China Platform Native** — DingTalk, Feishu, WeCom, WeChat Official Account ready out-of-box
- 🔌 **Embedded Friendly** — Compile to ARM/MIPS, easy deployment to Raspberry Pi, routers, NAS
- 🧠 **Multi-AI Backend** — Integrated with Claude, DeepSeek, Kimi, MiniMax, Gemini, switchable on demand

Supports DingTalk, Feishu, WeCom, WeChat Official Account, Slack, Telegram, Discord and more. Either **5-minute cloud relay** or [OpenClaw](docs/openclaw-reference.md)-style **self-hosted deployment**. Check [Roadmap](docs/roadmap.md) for more features.

> 🐕⚡ **Why "Lingti"?** Lingti (灵缇) means Greyhound in Chinese - the fastest dog in the world, known for agility and loyalty. Lingti Bot is equally agile and efficient, your faithful AI assistant.

## Installation

```bash
curl -fsSL https://cli.lingti.com/install.sh | bash -s -- --bot
```

## Examples

### Intelligent Chat, File Management, Information Retrieval
<table>
<tr>
<td width="33%"><img src="docs/images/demo-chat-1.png" alt="Smart Assistant" /></td>
<td width="33%"><img src="docs/images/demo-chat-2.png" alt="WeCom File Transfer" /></td>
<td width="33%"><img src="docs/images/demo-chat-3.png" alt="Information Search" /></td>
</tr>
<tr>
<td align="center"><sub>💬 Smart Chat</sub></td>
<td align="center"><sub>📁 WeCom File Transfer</sub></td>
<td align="center"><sub>🔍 Information Search</sub></td>
</tr>
</table>

<summary>📺 <b>Background Running Demo</b> — <code>make && dist/lingti-bot router</code></summary>
<br>
<img src="docs/images/demo-terminal.png" alt="Terminal Demo" />
<p><sub>Clone, compile and run directly, paired with DeepSeek model, processing DingTalk messages in real-time</sub></p>

## Why lingti-bot?

**Single Binary, Zero Dependencies, Local-First**

Unlike traditional Bot frameworks that require Docker, databases, or complex runtime environments, lingti-bot achieves the ultimate in simplicity:

1. **Zero Dependencies** — One 30MB binary file, no external dependencies, `scp` to any server and run
2. **Embedded Friendly** — Pure Go implementation, compilable to ARM/MIPS, deployable to Raspberry Pi, routers, NAS
3. **Plain Text Output** — No colored terminal output, avoiding extra rendering libraries or terminal compatibility issues
4. **Code Restraint** — Every line of code has a clear reason to exist, rejecting over-design
5. **Cloud Relay Boost** — No need for self-hosted web server, cloud relay completes WeChat Official Account and WeCom callback verification in seconds, Bot goes live immediately

```bash
# Clone, compile, run
git clone https://github.com/ruilisi/lingti-bot.git
cd lingti-bot && make
./dist/lingti-bot router --provider deepseek --api-key sk-xxx
```

### Single Binary

```bash
# Compile
make build

# Ready to use
./dist/lingti-bot serve
```

No Docker, no database, no cloud service required.

### Local-First

All functions run locally, data is not uploaded to the cloud. Your files, calendar, and process information are safely kept locally.

### Cross-Platform Support

Core functions support macOS, Linux, Windows. macOS users can enjoy native calendar, reminders, notes, music control and more.

**Supported Platforms:**

| Platform | Architecture | Build Command |
|----------|--------------|---------------|
| macOS | ARM64 (Apple Silicon) | `make darwin-arm64` |
| macOS | AMD64 (Intel) | `make darwin-amd64` |
| Linux | AMD64 | `make linux-amd64` |
| Linux | ARM64 | `make linux-arm64` |
| Linux | MIPS | `make linux-mips` |
| Windows | AMD64 | `make windows-amd64` |

## Architecture

### MCP Server — Claude Desktop Native Integration

lingti-bot is primarily an **MCP (Model Context Protocol) Server** that provides rich local tools for Claude Desktop and other MCP-compatible clients.

**Quick Start:**
1. Install lingti-bot: `curl -fsSL https://cli.lingti.com/install.sh | bash -s -- --bot`
2. Configure Claude Desktop MCP: `~/.config/Claude/claude_desktop_config.json`
   ```json
   {
     "mcpServers": {
       "lingti-bot": {
         "command": "lingti-bot",
         "args": ["serve"]
       }
     }
   }
   ```
3. Restart Claude Desktop, you can now use 75+ local tools

### Multi-Platform Gateway — Message Router

In addition to MCP mode, lingti-bot can also run as a **message router**, connecting multiple messaging platforms simultaneously.

**Supported Platforms:**

| Platform | Connection Method | Setup Time | Status |
|----------|-------------------|------------|--------|
| **WeCom** | Callback API | Cloud Relay / Self-hosted | ✅ |
| **Feishu/Lark** | WebSocket | One-click | ✅ |
| **WeChat Official** | Cloud Relay | 10 seconds | ✅ |
| **Slack** | Socket Mode | One-click | ✅ |
| **Telegram** | Bot API | One-click | ✅ |
| **Discord** | Gateway | One-click | ✅ |
| **DingTalk** | Stream Mode | One-click | ✅ |

**Cloud Relay Advantage:** No public server, no domain registration, no HTTPS certificate, no firewall configuration, 5 minutes to complete integration.

### MCP Toolset — 75+ Local System Tools

Covers all aspects of daily work, making AI your all-around assistant.

| Category | Tools | Features |
|----------|-------|----------|
| **File Operations** | 9 | Read/write, search, organize, batch delete, trash |
| **Shell Commands** | 2 | Command execution, path finding |
| **System Info** | 4 | CPU, memory, disk, environment variables |
| **Process Management** | 3 | List, details, terminate |
| **Network Tools** | 4 | Interfaces, connections, Ping, DNS |
| **Calendar** | 6 | View, create, search, delete events (macOS) |
| **Browser Automation** | 12 | Snapshot, click, input, screenshot, tab management |
| **Scheduled Tasks** | 5 | Create, list, delete, pause, resume cron jobs |

### Scheduled Tasks — Automate Your Workflow

Use standard Cron expressions to schedule periodic tasks for true unattended automation.

**Core Features:**
- 🕐 Support standard Cron expressions (minute, hour, day, month, weekday)
- 💾 Task persistence, auto-resume after restart
- 🔄 Pause/resume task execution
- 📊 Record execution status and error information
- 🛠️ Call any MCP tool

**Quick Examples:**

```bash
# Daily backup at 2 AM
cron_create(
  name="daily-backup",
  schedule="0 2 * * *",
  tool="shell_execute",
  arguments={"command": "tar -czf ~/backup-$(date +%Y%m%d).tar.gz ~/data"}
)

# Check disk space every 15 minutes
cron_create(
  name="disk-check",
  schedule="*/15 * * * *",
  tool="disk_usage",
  arguments={"path": "/"}
)

# Weekday morning standup reminder at 9 AM
cron_create(
  name="morning-standup",
  schedule="0 9 * * 1-5",
  tool="notification_send",
  arguments={"title": "Standup Reminder", "message": "Time for daily standup!"}
)
```

**Cron Expression Format:**

```
* * * * *
│ │ │ │ │
│ │ │ │ └─ Day of week (0-6, 0=Sunday)
│ │ │ └─── Month (1-12)
│ │ └───── Day of month (1-31)
│ └─────── Hour (0-23)
└───────── Minute (0-59)
```

**Common Examples:**
- `0 * * * *` - Every hour
- `*/15 * * * *` - Every 15 minutes
- `0 9 * * 1-5` - Weekdays at 9 AM
- `0 0 1 * *` - First day of every month
- `30 8-18 * * *` - Every hour from 8:30 to 18:30

Task configuration saved to `~/.lingti/crons.json`, auto-resume after MCP service restart.

### Multi-AI Backend

Support multiple AI services, switch on demand:

| AI Service | Environment Variable | Provider Parameter | Default Model |
|------------|---------------------|-------------------|---------------|
| **Claude** (Anthropic) | `ANTHROPIC_API_KEY` | `claude` / `anthropic` | claude-sonnet-4.5 |
| **Kimi** (Moonshot) | `KIMI_API_KEY` | `kimi` / `moonshot` | moonshot-v1-8k |
| **DeepSeek** | `DEEPSEEK_API_KEY` | `deepseek` | deepseek-chat |
| **Qwen** (Qianwen/通义千问) | `QWEN_API_KEY` | `qwen` / `qianwen` / `tongyi` | qwen-plus |

**Qwen Usage Example:**

```bash
# Using environment variable
export QWEN_API_KEY="sk-your-qwen-api-key"
lingti-bot router --provider qwen

# Using command line parameters
lingti-bot router \
  --provider qwen \
  --api-key "sk-your-qwen-api-key" \
  --model "qwen-plus"

# Available models: qwen-plus (recommended), qwen-turbo, qwen-max, qwen-long
```

Get Qwen API Key: Visit [Alibaba Cloud Bailian Platform](https://bailian.console.aliyun.com/) to create a DashScope API Key.

## Documentation

- [CLI Reference](docs/cli-reference.md) - Complete CLI documentation
- [Slack Integration Guide](docs/slack-integration.md) - Complete Slack app configuration tutorial
- [Feishu Integration Guide](docs/feishu-integration.md) - Feishu/Lark app configuration tutorial
- [WeCom Integration Guide](docs/wecom-integration.md) - WeCom app configuration tutorial
- [Browser Automation Guide](docs/browser-automation.md) - Snapshot-then-act browser control
- [OpenClaw Feature Comparison](docs/openclaw-feature-comparison.md) - Detailed feature difference analysis

## License

MIT License - see [LICENSE](LICENSE) file for details

## Contact

- Website: [cli.lingti.com](https://cli.lingti.com/bot)
- Email: `jiefeng@ruc.edu.cn` / `jiefeng.hopkins@gmail.com`
- GitHub: [github.com/ruilisi/lingti-bot](https://github.com/ruilisi/lingti-bot)
