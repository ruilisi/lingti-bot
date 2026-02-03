# lingti-bot (灵小缇)

> 🚀 **更适合中国宝宝体质的 AI Bot，让 AI Bot 接入更简单**

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Website](https://img.shields.io/badge/官网-cli.lingti.com-blue?style=flat)](https://cli.lingti.com/bot)

**灵缇**是一个集 **MCP Server**、**多平台消息网关**、**丰富工具集**、**智能对话**于一体的 AI Bot 平台。各大平台秒接入，兼具 [OpenClaw](docs/openclaw-reference.md) 式灵活接入。

> **为什么叫"灵缇"？** 灵缇犬（Greyhound）是世界上跑得最快的犬，以敏捷、忠诚著称。灵缇 bot 同样敏捷高效，是你忠实的 AI 助手。

## 功能概览

| 模块 | 说明 | 特点 |
|------|------|------|
| **MCP Server** | 标准 MCP 协议服务器 | 兼容 Claude Desktop、Cursor 等所有 MCP 客户端 |
| **多平台消息网关** | 企业消息平台集成 | Slack、飞书一键接入，支持云中继 |
| **MCP 工具集** | 40+ 本地系统工具 | 文件、Shell、系统、网络、日历、音乐等 |
| **智能对话** | 多轮对话与记忆 | 上下文记忆、多 AI 后端（Claude/Kimi/DeepSeek） |

## Sponsors

- **[灵缇游戏加速](https://game.lingti.com)** - PC/Mac/iOS/Android 全平台游戏加速、热点加速、AI 及学术资源定向加速，And More
- **[灵缇路由](https://router.lingti.com)** - 您的路由管家、网游电竞专家

## lingti-cli 生态

**lingti-bot** 是 **lingti-cli** 五位一体平台的核心开源组件。

我们正在打造 **AI 时代开发者与知识工作者的终极效率平台**：

| 模块 | 定位 | 说明 |
|------|------|------|
| **CLI** | 操控总台 | 统一入口，如同操作系统的引导程序 |
| **Net** | 全球网络 | 跨洲 200Mbps 加速，畅享全球 AI 服务 |
| **Token** | 数字员工 | Token 即代码，代码即生产力 |
| **Bot** | 助理管理 | 数字员工接入与管理，简单到极致 ← *本项目* |
| **Code** | 开发环境 | Terminal 回归舞台中央，极致输入效率 |

> **为什么是 cli.lingti.com/bot 而不是 bot.lingti.com？**
>
> 因为 Bot 是 CLI 生态的一部分。IDE 正在消亡，纯粹的 Terminal 界面正在回归。未来的生产力工具，将围绕 CLI 重新构建。

**加入我们：** 无论您是追求极致效率的顶尖开发者、关注 AI 时代生产力变革的投资人，还是想成为 Sponsor，欢迎联系：jiefeng@ruc.edu.cn / jiefeng.hopkins@gmail.com

---

```
┌─────────────────────────────────────────────────────────────────┐
│                         lingti-bot                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────────┐   │
│  │  MCP Server  │      │   Message    │      │    Agent     │   │
│  │   (stdio)    │      │   Gateway    │      │   (Claude)   │   │
│  └──────┬───────┘      └──────┬───────┘      └──────┬───────┘   │
│         │                     │                     │            │
│         └─────────────────────┴─────────────────────┘            │
│                               │                                  │
│                               ▼                                  │
│                    ┌─────────────────────┐                       │
│                    │     MCP Tools       │                       │
│                    │  ┌───────┐ ┌─────┐  │                       │
│                    │  │ Files │ │Shell│  │                       │
│                    │  │System │ │ Net │  │                       │
│                    │  │Process│ │ Cal │  │                       │
│                    │  └───────┘ └─────┘  │                       │
│                    └─────────────────────┘                       │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
         │                     │
         ▼                     ▼
┌─────────────────┐    ┌───────────────┐
│ Claude Desktop  │    │  Slack/飞书   │
│ Cursor / 其他   │    │  消息平台      │
└─────────────────┘    └───────────────┘
```

---

## MCP Server

灵缇作为标准 MCP (Model Context Protocol) 服务器，让任何支持 MCP 的 AI 客户端都能访问本地系统资源。

### 支持的客户端

- **Claude Desktop** - Anthropic 官方桌面客户端
- **Cursor** - AI 代码编辑器
- **其他 MCP 客户端** - 任何实现 MCP 协议的应用

### 快速配置

**Claude Desktop** (`~/Library/Application Support/Claude/claude_desktop_config.json`)：

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

**Cursor** (`.cursor/mcp.json`)：

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

就这么简单！重启客户端后，AI 助手即可使用所有 lingti-bot 提供的工具。

### 特点

- **无需额外配置** - 一个二进制文件，两行配置
- **无需数据库** - 无外部依赖
- **无需 Docker** - 单一静态二进制
- **无需云服务** - 完全本地运行

---

## 多平台消息网关

灵缇支持多种企业消息平台，让你的团队在熟悉的工具中直接与 AI 对话。

### 支持的平台

| 平台 | 协议 | 状态 |
|------|------|------|
| **Slack** | Socket Mode | ✅ 已支持 |
| **飞书/Lark** | WebSocket | ✅ 已支持 |
| **云中继** | WebSocket | ✅ 已支持 |
| **钉钉** | - | 🚧 开发中 |
| **企业微信** | - | 🚧 开发中 |

### 一键接入

灵缇提供 **1 分钟内一键接入**方式，无需复杂配置：

```bash
# 设置 API 密钥
export ANTHROPIC_API_KEY="sk-ant-your-api-key"

# Slack 一键接入
export SLACK_BOT_TOKEN="xoxb-..."
export SLACK_APP_TOKEN="xapp-..."

# 飞书一键接入
export FEISHU_APP_ID="cli_..."
export FEISHU_APP_SECRET="..."

# 启动网关
./lingti-bot router
```

### 多 AI 后端

支持多种 AI 服务，按需切换：

| AI 服务 | 环境变量 |
|---------|----------|
| **Claude** (Anthropic) | `ANTHROPIC_API_KEY` |
| **Kimi** (月之暗面) | `KIMI_API_KEY` |
| **DeepSeek** | `DEEPSEEK_API_KEY` |

### 详细文档

- [Slack 集成指南](docs/slack-integration.md) - 完整的 Slack 应用配置教程
- [飞书集成指南](docs/feishu-integration.md) - 飞书/Lark 应用配置教程

---

## MCP 工具集

灵缇提供 **40+ MCP 工具**，覆盖日常工作的方方面面。

### 工具分类

| 分类 | 工具数 | 说明 |
|------|--------|------|
| 文件操作 | 9 | 读写、搜索、整理、废纸篓 |
| Shell 命令 | 2 | 命令执行、路径查找 |
| 系统信息 | 4 | CPU、内存、磁盘、环境变量 |
| 进程管理 | 3 | 列表、详情、终止 |
| 网络工具 | 4 | 接口、连接、Ping、DNS |
| 日历 (macOS) | 6 | 查看、创建、搜索、删除 |
| 提醒事项 (macOS) | 4 | 待办事项管理 |
| 备忘录 (macOS) | 5 | 笔记管理 |
| 天气 | 2 | 当前天气、预报 |
| 网页搜索 | 2 | DuckDuckGo 搜索、网页获取 |
| 剪贴板 | 2 | 读写剪贴板 |
| 截图 | 1 | 屏幕截图 |
| 系统通知 | 1 | 发送通知 |
| 音乐控制 (macOS) | 7 | 播放、暂停、切歌、音量 |

### 文件操作

| 工具 | 功能 |
|------|------|
| `file_read` | 读取文件内容 |
| `file_write` | 写入文件内容 |
| `file_list` | 列出目录内容 |
| `file_search` | 按模式搜索文件 |
| `file_info` | 获取文件详细信息 |
| `file_list_old` | 列出长时间未修改的文件 |
| `file_delete_old` | 删除长时间未修改的文件 |
| `file_delete_list` | 批量删除指定文件 |
| `file_trash` | 移动文件到废纸篓（macOS） |

### Shell 命令

| 工具 | 功能 |
|------|------|
| `shell_execute` | 执行 Shell 命令 |
| `shell_which` | 查找可执行文件路径 |

### 系统信息

| 工具 | 功能 |
|------|------|
| `system_info` | 获取系统信息（CPU、内存、OS） |
| `disk_usage` | 获取磁盘使用情况 |
| `env_get` | 获取环境变量 |
| `env_list` | 列出所有环境变量 |

### 进程管理

| 工具 | 功能 |
|------|------|
| `process_list` | 列出运行中的进程 |
| `process_info` | 获取进程详细信息 |
| `process_kill` | 终止进程 |

### 网络工具

| 工具 | 功能 |
|------|------|
| `network_interfaces` | 列出网络接口 |
| `network_connections` | 列出活动网络连接 |
| `network_ping` | TCP 连接测试 |
| `network_dns_lookup` | DNS 查询 |

### 日历（macOS）

| 工具 | 功能 |
|------|------|
| `calendar_today` | 获取今日日程 |
| `calendar_list_events` | 列出未来事件 |
| `calendar_create_event` | 创建日历事件 |
| `calendar_search` | 搜索日历事件 |
| `calendar_delete_event` | 删除日历事件 |
| `calendar_list_calendars` | 列出所有日历 |

### 提醒事项（macOS）

| 工具 | 功能 |
|------|------|
| `reminders_today` | 获取今日待办事项 |
| `reminders_add` | 添加新提醒 |
| `reminders_complete` | 标记提醒为已完成 |
| `reminders_delete` | 删除提醒 |

### 备忘录（macOS）

| 工具 | 功能 |
|------|------|
| `notes_list_folders` | 列出备忘录文件夹 |
| `notes_list` | 列出备忘录 |
| `notes_read` | 读取备忘录内容 |
| `notes_create` | 创建新备忘录 |
| `notes_search` | 搜索备忘录 |

### 天气

| 工具 | 功能 |
|------|------|
| `weather_current` | 获取当前天气 |
| `weather_forecast` | 获取天气预报 |

### 网页搜索

| 工具 | 功能 |
|------|------|
| `web_search` | DuckDuckGo 搜索 |
| `web_fetch` | 获取网页内容 |

### 剪贴板

| 工具 | 功能 |
|------|------|
| `clipboard_read` | 读取剪贴板内容 |
| `clipboard_write` | 写入剪贴板 |

### 系统通知

| 工具 | 功能 |
|------|------|
| `notification_send` | 发送系统通知 |

### 截图

| 工具 | 功能 |
|------|------|
| `screenshot` | 截取屏幕截图 |

### 音乐控制（macOS）

| 工具 | 功能 |
|------|------|
| `music_play` | 播放音乐 |
| `music_pause` | 暂停音乐 |
| `music_next` | 下一首 |
| `music_previous` | 上一首 |
| `music_now_playing` | 获取当前播放信息 |
| `music_volume` | 设置音量 |
| `music_search` | 搜索并播放音乐 |

### 其他

| 工具 | 功能 |
|------|------|
| `open_url` | 在浏览器中打开 URL |

---

## 智能对话

灵缇支持**多轮对话记忆**，能够记住之前的对话内容，实现连续自然的交流体验。

### 工作原理

- 每个用户在每个频道有独立的对话上下文
- 自动保存最近 **50 条消息**
- 对话 **60 分钟**无活动后自动过期
- 支持跨多轮对话的上下文理解

### 使用示例

```
用户：我叫小明，今年25岁
AI：你好小明！很高兴认识你。

用户：我叫什么名字？
AI：你叫小明。

用户：我多大了？
AI：你今年25岁。

用户：帮我创建一个日程，标题就用我的名字
AI：好的，我帮你创建了一个标题为"小明"的日程。
```

### 对话管理命令

| 命令 | 说明 |
|------|------|
| `/new` | 开始新对话，清除历史记忆 |
| `/reset` | 同上 |
| `/clear` | 同上 |
| `新对话` | 中文命令，开始新对话 |
| `清除历史` | 中文命令，清除对话历史 |

> **提示**：当你想让 AI "忘记"之前的内容重新开始时，只需发送 `/new` 即可。

---

## 快速开始

### 安装

```bash
# 克隆仓库
git clone https://github.com/ruilisi/lingti-bot.git
cd lingti-bot

# 编译
make build

# 或者编译到 dist 目录
make darwin-arm64  # Apple Silicon Mac
make darwin-amd64  # Intel Mac
make linux-amd64   # Linux x64
```

### 使用方式

**方式一：MCP Server 模式**

配置 Claude Desktop 或 Cursor，详见 [MCP Server](#mcp-server) 章节。

**方式二：消息网关模式**

连接 Slack、飞书等平台，详见 [多平台消息网关](#多平台消息网关) 章节。

---

## 使用示例

配置完成后，你可以让 AI 助手执行以下操作：

### 日历与日程

```
"今天有什么日程安排？"
"这周有哪些会议？"
"帮我创建一个明天下午3点的会议，标题是'产品评审'"
"搜索所有包含'周报'的日程"
```

### 文件操作

```
"列出桌面上的所有文件"
"读取 ~/Documents/notes.txt 的内容"
"桌面上超过30天没动过的文件有哪些？"
"帮我把这些旧文件移到废纸篓"
```

### 系统与进程

```
"我的电脑配置是什么？"
"现在 CPU 占用多少？"
"Chrome 占用了多少内存？"
"结束 PID 1234 的进程"
```

### 网络与搜索

```
"我的 IP 地址是什么？"
"帮我搜索一下最新的 AI 新闻"
"查询 github.com 的 DNS"
```

### 音乐控制

```
"播放音乐"
"下一首"
"音量调到 50%"
"播放周杰伦的歌"
```

### 组合任务

```
"查看今天的日程，然后检查天气，最后列出待办事项"
"帮我整理桌面：列出超过60天的旧文件，然后移到废纸篓"
"搜索最近的科技新闻，整理成备忘录"
```

---

## 为什么选择 lingti-bot？

### 极简集成

与其他方案相比，lingti-bot 的集成极其简单：

- **vs OpenClaw** - 提供同样灵活的软件接入方式，同时支持一键接入
- **vs 自建方案** - 无需从零开发，开箱即用
- **vs SaaS 方案** - 完全本地运行，数据不上云

### 单一二进制

```bash
# 编译
make build

# 即可使用
./dist/lingti-bot serve
```

无需 Docker，无需数据库，无需云服务。

### 本地优先

所有功能都在本地运行，数据不会上传到云端。你的文件、日历、进程信息都安全地保留在本地。

### 跨平台支持

核心功能支持 macOS、Linux、Windows。macOS 用户可享受日历、提醒事项、备忘录、音乐控制等原生功能。

---

## 项目结构

```
lingti-bot/
├── main.go                 # 程序入口
├── Makefile                # 构建脚本
├── go.mod                  # Go 模块定义
│
├── cmd/                    # 命令行接口
│   ├── root.go             # 根命令
│   ├── serve.go            # MCP 服务器命令
│   ├── service.go          # 系统服务管理
│   └── version.go          # 版本信息
│
├── internal/
│   ├── mcp/
│   │   └── server.go       # MCP 服务器实现
│   │
│   ├── tools/              # MCP 工具实现
│   │   ├── filesystem.go   # 文件读写、列表、搜索
│   │   ├── shell.go        # Shell 命令执行
│   │   ├── system.go       # 系统信息、磁盘、环境变量
│   │   ├── process.go      # 进程列表、信息、终止
│   │   ├── network.go      # 网络接口、连接、DNS
│   │   ├── calendar.go     # macOS 日历集成
│   │   ├── filemanager.go  # 文件整理（清理旧文件）
│   │   ├── reminders.go    # macOS 提醒事项
│   │   ├── notes.go        # macOS 备忘录
│   │   ├── weather.go      # 天气查询（wttr.in）
│   │   ├── websearch.go    # 网页搜索和获取
│   │   ├── clipboard.go    # 剪贴板读写
│   │   ├── notification.go # 系统通知
│   │   ├── screenshot.go   # 屏幕截图
│   │   └── music.go        # 音乐控制（Spotify/Apple Music）
│   │
│   ├── router/
│   │   └── router.go       # 多平台消息路由器
│   │
│   ├── platforms/          # 消息平台集成
│   │   ├── slack/
│   │   │   └── slack.go    # Slack Socket Mode
│   │   └── feishu/
│   │       └── feishu.go   # 飞书 WebSocket
│   │
│   ├── agent/
│   │   ├── tools.go        # Agent 工具执行
│   │   └── memory.go       # 会话记忆
│   │
│   └── service/
│       └── manager.go      # 系统服务管理
│
└── docs/                   # 文档
    ├── slack-integration.md    # Slack 集成指南
    ├── feishu-integration.md   # 飞书集成指南
    └── openclaw-reference.md   # 架构参考
```

---

## Make 目标

```bash
# 开发
make build          # 编译当前平台
make run            # 本地运行
make test           # 运行测试
make fmt            # 格式化代码
make lint           # 代码检查
make clean          # 清理构建产物
make version        # 显示版本

# 跨平台编译
make darwin-arm64   # macOS Apple Silicon
make darwin-amd64   # macOS Intel
make darwin-universal # macOS 通用二进制
make linux-amd64    # Linux x64
make linux-arm64    # Linux ARM64
make linux-all      # 所有 Linux 平台
make all            # 所有平台

# 服务管理
make install        # 安装为系统服务
make uninstall      # 卸载系统服务
make start          # 启动服务
make stop           # 停止服务
make status         # 查看服务状态

# macOS 签名
make codesign       # 代码签名（需要开发者证书）
```

---

## 环境变量

| 变量 | 说明 | 必需 |
|------|------|------|
| `ANTHROPIC_API_KEY` | Anthropic API 密钥 | 路由器模式必需 |
| `ANTHROPIC_BASE_URL` | 自定义 API 地址 | 可选 |
| `ANTHROPIC_MODEL` | 使用的模型 | 可选 |
| `SLACK_BOT_TOKEN` | Slack Bot Token (`xoxb-...`) | Slack 集成必需 |
| `SLACK_APP_TOKEN` | Slack App Token (`xapp-...`) | Slack 集成必需 |
| `FEISHU_APP_ID` | 飞书 App ID | 飞书集成必需 |
| `FEISHU_APP_SECRET` | 飞书 App Secret | 飞书集成必需 |

---

## 安全注意事项

- lingti-bot 提供对本地系统的访问能力，请在可信环境中使用
- Shell 命令执行有基本的危险命令过滤，但仍需谨慎
- API 密钥等敏感信息请使用环境变量，不要提交到版本控制
- 生产环境建议使用专用服务账号运行

---

## 依赖

- [mcp-go](https://github.com/mark3labs/mcp-go) - MCP 协议 Go 实现
- [cobra](https://github.com/spf13/cobra) - CLI 框架
- [gopsutil](https://github.com/shirou/gopsutil) - 系统信息
- [slack-go](https://github.com/slack-go/slack) - Slack SDK
- [oapi-sdk-go](https://github.com/larksuite/oapi-sdk-go) - 飞书/Lark SDK
- [go-anthropic](https://github.com/liushuangls/go-anthropic) - Anthropic API 客户端

---

## 许可证

MIT License

---

## 贡献

欢迎提交 Issue 和 Pull Request！

---

## 开发环境

本项目完全在 **[lingti-code](https://cli.lingti.com/code)** 环境中编写完成。

### 关于 lingti-code

[lingti-code](https://github.com/ruilisi/lingti-code) 是一个一体化的 AI 就绪开发环境平台，基于 **Tmux + Neovim + Zsh** 构建，支持 macOS、Ubuntu 和 Docker 部署。

**核心组件：**

- **Shell** - ZSH + Prezto 框架，100+ 常用别名和函数，fasd 智能导航
- **Editor** - Neovim + SpaceVim 发行版，LSP 集成，GitHub Copilot 支持
- **Terminal** - Tmux 终端复用，vim 风格键绑定，会话管理
- **版本控制** - Git 最佳实践配置，丰富的 Git 别名
- **开发工具** - asdf 版本管理器，ctags，IRB/Pry 增强

**AI 集成：**

- Claude Code CLI 配置，支持项目感知的 CLAUDE.md 文件
- 自定义状态栏显示 Token 用量
- 预配置 LSP 插件（Python basedpyright、Go gopls）

**一键安装：**

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/lingti/lingti-code/master/install.sh)"
```

更多信息请访问：[官网](https://cli.lingti.com/code) | [GitHub](https://github.com/ruilisi/lingti-code)

---

**灵缇** - 你的敏捷 AI 助手 🐕
