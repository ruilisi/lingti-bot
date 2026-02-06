# lingti-bot (灵小缇)

> 🚀 **[更适合中国宝宝体质](docs/为何更适合中国宝宝体质.md)的 AI Bot，让 AI Bot 接入更简单**

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Website](https://img.shields.io/badge/官网-cli.lingti.com-blue?style=flat)](https://cli.lingti.com/bot)

**灵小缇** 是一个集 **MCP Server**、**多平台消息网关**、**丰富工具集**、**智能对话**、**语音交互**于一体的 AI Bot 平台。微信、飞书等平台秒接入，兼具 [OpenClaw](docs/openclaw-reference.md) 式灵活接入。查看 [开发路线图](docs/roadmap.md) 了解更多功能规划。

> **为什么叫"灵小缇"？** 灵缇犬（Greyhound）是世界上跑得最快的犬，以敏捷、忠诚著称。灵小缇同样敏捷高效，是你忠实的 AI 助手。

<table>
<tr>
<td width="33%"><img src="docs/images/demo-chat-1.png" alt="智能助手" /></td>
<td width="33%"><img src="docs/images/demo-chat-2.png" alt="文件管理" /></td>
<td width="33%"><img src="docs/images/demo-chat-3.png" alt="信息搜索" /></td>
</tr>
<tr>
<td align="center"><sub>💬 智能对话</sub></td>
<td align="center"><sub>📁 文件管理</sub></td>
<td align="center"><sub>🔍 信息搜索</sub></td>
</tr>
</table>

<details>
<summary>📺 <b>后台运行演示</b> — <code>make && dist/lingti-bot router</code></summary>
<br>
<img src="docs/images/demo-terminal.png" alt="Terminal Demo" />
<p><sub>克隆代码后直接编译运行，配合 DeepSeek 模型，实时处理钉钉消息</sub></p>
</details>

## 安装

```bash
curl -fsSL https://cli.lingti.com/install.sh | bash -s -- --bot
```

## 为什么选择 lingti-bot？

### lingti-bot vs OpenClaw

|  | **lingti-bot** | **OpenClaw** |
|--|----------------|--------------|
| **语言** | 纯 Go 实现 | Node.js |
| **运行依赖** | 无（单一二进制） | 需要 Node.js 运行时 |
| **分发方式** | 单个可执行文件，复制即用 | npm 安装，依赖 node_modules |
| **嵌入式设备** | ✅ 可轻松部署到 ARM/MIPS 等小型设备 | ❌ 需要 Node.js 环境 |
| **安装大小** | ~15MB 单文件 | 100MB+ (含 node_modules) |
| **输出风格** | 纯文本，无彩色 | 彩色输出 |
| **设计哲学** | 极简主义，够用就好 | 功能丰富，灵活优先 |
| **中国平台** | 原生支持飞书/企微/钉钉 | 需自行集成 |
| **云中继** | ✅ 免自建服务器，秒级接入微信/企微 | ❌ 需自建 Web 服务 |

> 详细功能对比请参考：[OpenClaw vs lingti-bot 技术特性对比](docs/openclaw-feature-comparison.md)

**为什么选择纯 Go + 纯文本输出？**

> *"Simplicity is the ultimate sophistication."* — Leonardo da Vinci

lingti-bot 将**简洁性**作为最高设计原则：

1. **零依赖部署** — 单一二进制，`scp` 到任何机器即可运行，无需安装 Node.js、Python 或其他运行时
2. **嵌入式友好** — 可编译到 ARM、MIPS 等架构，轻松部署到树莓派、路由器、NAS 等小型设备
3. **纯文本输出** — 不使用彩色终端输出，避免引入额外的渲染库或终端兼容性问题
4. **代码克制** — 每一行代码都有明确的存在理由，拒绝过度设计
5. **云中继加持** — 无需自建 Web 服务器，通过云中继秒级完成微信公众号、企业微信的回调验证，Bot 即刻上线

```bash
# 克隆即编译，编译即运行
git clone https://github.com/ruilisi/lingti-bot.git
cd lingti-bot && make
./dist/lingti-bot router --provider deepseek --api-key sk-xxx
```

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

**支持的目标平台：**

| 平台 | 架构 | 编译命令 |
|------|------|----------|
| macOS | ARM64 (Apple Silicon) | `make darwin-arm64` |
| macOS | AMD64 (Intel) | `make darwin-amd64` |
| Linux | AMD64 | `make linux-amd64` |
| Linux | ARM64 | `make linux-arm64` |
| Linux | ARMv7 (树莓派等) | `make linux-arm` |
| Windows | AMD64 | `make windows-amd64` |

## 功能概览

### MCP Server — 标准协议，无缝集成

灵小缇实现了完整的 [MCP (Model Context Protocol)](https://modelcontextprotocol.io/) 协议，让任何支持 MCP 的 AI 客户端都能访问本地系统资源。

| 客户端 | 状态 | 说明 |
|--------|------|------|
| Claude Desktop | ✅ | Anthropic 官方桌面客户端 |
| Cursor | ✅ | AI 代码编辑器 |
| Windsurf | ✅ | Codeium 的 AI IDE |
| 其他 MCP 客户端 | ✅ | 任何实现 MCP 协议的应用 |

**特点：** 无需额外配置、无需数据库、无需 Docker、无需云服务，单一二进制文件即可运行。

### 多平台消息网关 — 企业 IM 秒接入

支持国内外主流企业消息平台，让团队在熟悉的工具中直接与 AI 对话。

| 平台 | 协议 | 接入方式 | 状态 |
|------|------|----------|------|
| **企业微信** | 回调 API | 云中继 / 自建 | ✅ |
| **飞书/Lark** | WebSocket | 一键接入 | ✅ |
| **微信公众号** | 云中继 | 10秒接入 | ✅ |
| **Slack** | Socket Mode | 一键接入 | ✅ |
| **Telegram** | Bot API | 一键接入 | ✅ |
| **Discord** | Gateway | 一键接入 | ✅ |
| **钉钉** | Stream Mode | 一键接入 | ✅ |

**云中继优势：** 无需公网服务器、无需域名备案、无需 HTTPS 证书、无需防火墙配置，5 分钟完成接入。

### MCP 工具集 — 60+ 本地系统工具

覆盖日常工作的方方面面，让 AI 成为你的全能助手。

| 分类 | 工具数 | 功能 |
|------|--------|------|
| **文件操作** | 9 | 读写、搜索、整理、批量删除、废纸篓 |
| **Shell 命令** | 2 | 命令执行、路径查找 |
| **系统信息** | 4 | CPU、内存、磁盘、环境变量 |
| **进程管理** | 3 | 列表、详情、终止 |
| **网络工具** | 4 | 接口、连接、Ping、DNS |
| **日历** | 6 | 查看、创建、搜索、删除日程 (macOS) |
| **提醒事项** | 5 | 列表、添加、完成、删除 (macOS) |
| **备忘录** | 6 | 文件夹、列表、读取、创建、搜索、删除 (macOS) |
| **天气** | 2 | 当前天气、多日预报 |
| **网页搜索** | 2 | DuckDuckGo 搜索、网页内容获取 |
| **剪贴板** | 2 | 读写剪贴板 |
| **截图** | 1 | 屏幕截图 |
| **系统通知** | 1 | 发送桌面通知 |
| **音乐控制** | 7 | 播放、暂停、切歌、音量、搜索 (macOS) |
| **Git** | 4 | 状态、日志、差异、分支 |
| **GitHub** | 6 | PR 列表/详情、Issue 管理、仓库信息 |

### 智能对话 — 多轮记忆，自然交流

支持多轮对话记忆，能够记住之前的对话内容，实现连续自然的交流体验。

| 特性 | 说明 |
|------|------|
| **上下文记忆** | 每个用户独立的对话上下文，最近 50 条消息 |
| **自动过期** | 对话 60 分钟无活动后自动清除 |
| **多 AI 后端** | Claude、DeepSeek、Kimi、MiniMax 按需切换 |
| **对话管理** | `/new`、`/reset`、`新对话` 命令重置对话 |

### 语音交互 — 解放双手，畅快对话

支持语音输入和语音输出，实现真正的免提 AI 交互体验。

| 命令 | 说明 |
|------|------|
| `lingti-bot voice` | 按 Enter 录音，AI 处理后返回文字/语音响应 |
| `lingti-bot talk` | 持续监听模式，支持唤醒词激活 |

| 语音引擎 | 说明 |
|----------|------|
| **system** | 系统原生（macOS say/whisper-cpp，Linux espeak） |
| **openai** | OpenAI TTS + Whisper API |
| **elevenlabs** | ElevenLabs 高品质 TTS |

**特点：** 本地语音识别（whisper-cpp）、多语言支持、唤醒词激活、连续对话模式。

### 功能速览表

| 模块 | 说明 | 特点 |
|------|------|------|
| **MCP Server** | 标准 MCP 协议服务器 | 兼容 Claude Desktop、Cursor、Windsurf 等所有 MCP 客户端 |
| **多平台消息网关** | 消息平台集成 | 微信公众号、企业微信、Slack、飞书一键接入，支持云中继 |
| **MCP 工具集** | 60+ 本地系统工具 | 文件、Shell、系统、网络、日历、Git、GitHub 等全覆盖 |
| **智能对话** | 多轮对话与记忆 | 上下文记忆、多 AI 后端（Claude/DeepSeek/Kimi/MiniMax） |
| **语音交互** | 语音输入/输出 | 本地 whisper-cpp、OpenAI、ElevenLabs 多引擎支持 |

## 云中继：零门槛接入企业消息平台

> **告别公网服务器、告别复杂配置，让 AI Bot 接入像配置 Wi-Fi 一样简单**

传统接入企业微信等平台需要：公网服务器 → 域名备案 → HTTPS 证书 → 防火墙配置 → 回调服务开发...

**lingti-bot 云中继** 将这一切简化为 3 步：

```bash
# 步骤 1: 安装
curl -fsSL https://cli.lingti.com/install.sh | bash -s -- --bot

# 步骤 2: 配置企业可信IP（应用管理 → 找到应用 → 企业可信IP → 添加 106.52.166.51）

# 步骤 3: 一条命令搞定验证和消息处理
lingti-bot relay --platform wecom \
  --wecom-corp-id ... --wecom-token ... --wecom-aes-key ... \
  --provider deepseek --api-key sk-xxx

# 然后去企业微信后台配置回调 URL: https://bot.lingti.com/wecom
```

**工作原理：**

```
企业微信(用户消息) --> bot.lingti.com(云中继) --WebSocket--> lingti-bot(本地AI处理)
```

**优势对比：**

| | 传统方案 | 云中继方案 |
|---|---|---|
| 公网服务器 | ✅ 需要 | ❌ 不需要 |
| 域名/备案 | ✅ 需要 | ❌ 不需要 |
| HTTPS证书 | ✅ 需要 | ❌ 不需要 |
| 回调服务开发 | ✅ 需要 | ❌ 不需要 |
| 接入时间 | 数天 | **5分钟** |
| AI处理位置 | 服务器 | **本地** |
| 数据安全 | 云端存储 | **本地处理** |

> 详细对比请参考：[lingti-bot vs OpenClaw：简化 AI 集成的努力](docs/vs-openclaw-integration.md)

### 微信公众号一键接入

微信搜索公众号「**灵缇小秘**」，关注后发送任意消息获取接入教程，10秒将lingti-bot接入微信。
详细教程请参考：[微信公众号接入指南](docs/wechat-integration.md)
### 飞书接入

- 飞书商店应用正在上架流程中，目前可通过自建应用实现绑定。教程请参考：[飞书集成指南](https://github.com/ruilisi/lingti-bot/blob/master/docs/feishu-integration.md)

### 企业微信接入

通过**云中继模式**，无需公网服务器即可接入企业微信：

```bash
# 1. 先去企业微信后台配置企业可信IP
#    应用管理 → 找到应用 → 企业可信IP → 添加: 106.52.166.51

# 2. 一条命令搞定验证和消息处理
lingti-bot relay --platform wecom \
  --wecom-corp-id YOUR_CORP_ID \
  --wecom-agent-id YOUR_AGENT_ID \
  --wecom-secret YOUR_SECRET \
  --wecom-token YOUR_TOKEN \
  --wecom-aes-key YOUR_AES_KEY \
  --provider deepseek \
  --api-key YOUR_API_KEY

# 3. 去企业微信后台配置回调 URL: https://bot.lingti.com/wecom
#    保存配置后验证自动完成，消息立即可以处理
```

详细教程请参考：[企业微信集成指南](docs/wecom-integration.md)

### 钉钉接入

使用 **Stream 模式**，无需公网服务器即可接入钉钉机器人：

```bash
# 一条命令搞定
lingti-bot router \
  --dingtalk-client-id YOUR_APP_KEY \
  --dingtalk-client-secret YOUR_APP_SECRET \
  --provider deepseek \
  --api-key YOUR_API_KEY
```

**配置步骤：**
1. 登录 [钉钉开放平台](https://open.dingtalk.com/)，创建企业内部应用
2. 在应用详情页获取 AppKey (ClientID) 和 AppSecret (ClientSecret)
3. 开启机器人功能，配置消息接收模式为 **Stream 模式**
4. 运行上述命令即可

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

**联系我们 / 加入我们**

<table>
  <tr>
    <th align="center" width="56%">邮件联系</th>
    <th align="center" width="44%">扫码加群</th>
  </tr>
  <tr>
    <td width="56%">
      无论您是追求极致效率的顶尖开发者、关注 AI 时代生产力变革的投资人，还是想成为 Sponsor，
      欢迎联系：
      <code>jiefeng@ruc.edu.cn</code>
      /
      <code>jiefeng.hopkins@gmail.com</code>
    </td>
    <td width="44%" align="center">
      <img src="https://lingti-1302055788.cos.ap-guangzhou.myqcloud.com/contact_me_qr-2.png" alt="扫码加群" width="230" />
    </td>
  </tr>
</table>

---

```
                              lingti-bot
    +---------------+    +---------------+    +---------------+
    |  MCP Server   |    |   Message     |    |    Agent      |
    |   (stdio)     |    |   Gateway     |    |   (Claude)    |
    +-------+-------+    +-------+-------+    +-------+-------+
            |                    |                    |
            +--------------------+--------------------+
                                 |
                                 v
                       +-------------------+
                       |    MCP Tools      |
                       | Files, Shell, Net |
                       | System, Calendar  |
                       +-------------------+
                                 |
            +--------------------+--------------------+
            |                                         |
            v                                         v
    +---------------+                       +------------------+
    | Claude Desktop|                       | Slack / Feishu   |
    | Cursor, etc.  |                       | Messaging Apps   |
    +---------------+                       +------------------+
```

---

## MCP Server

灵小缇作为标准 MCP (Model Context Protocol) 服务器，让任何支持 MCP 的 AI 客户端都能访问本地系统资源。

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

灵小缇支持多种企业消息平台，让你的团队在熟悉的工具中直接与 AI 对话。

### 支持的平台

| 平台 | 协议 | 状态 |
|------|------|------|
| **Slack** | Socket Mode | ✅ 已支持 |
| **飞书/Lark** | WebSocket | ✅ 已支持 |
| **Telegram** | Bot API | ✅ 已支持 |
| **Discord** | Gateway | ✅ 已支持 |
| **云中继** | WebSocket | ✅ 已支持 |
| **钉钉** | Stream Mode | ✅ 已支持 |
| **企业微信** | 回调 API | ✅ 已支持 |

### 一键接入

灵小缇提供 **1 分钟内一键接入**方式，无需复杂配置：

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
| **MiniMax** | `ANTHROPIC_API_KEY`、`ANTHROPIC_BASE_URL` |

### 详细文档

- [命令行参考](docs/cli-reference.md) - 完整的命令行使用文档
- [Slack 集成指南](docs/slack-integration.md) - 完整的 Slack 应用配置教程
- [飞书集成指南](docs/feishu-integration.md) - 飞书/Lark 应用配置教程
- [企业微信集成指南](docs/wecom-integration.md) - 企业微信应用配置教程
- [OpenClaw 技术特性对比](docs/openclaw-feature-comparison.md) - 详细功能差异分析

---

## MCP 工具集

灵小缇提供 **60+ MCP 工具**，覆盖日常工作的方方面面。

### 工具分类

| 分类 | 工具数 | 说明 |
|------|--------|------|
| 文件操作 | 9 | 读写、搜索、整理、废纸篓 |
| Shell 命令 | 2 | 命令执行、路径查找 |
| 系统信息 | 4 | CPU、内存、磁盘、环境变量 |
| 进程管理 | 3 | 列表、详情、终止 |
| 网络工具 | 4 | 接口、连接、Ping、DNS |
| 日历 (macOS) | 6 | 查看、创建、搜索、删除 |
| 提醒事项 (macOS) | 5 | 列表、添加、完成、删除 |
| 备忘录 (macOS) | 6 | 文件夹、列表、读取、创建、搜索、删除 |
| 天气 | 2 | 当前天气、预报 |
| 网页搜索 | 2 | DuckDuckGo 搜索、网页获取 |
| 剪贴板 | 2 | 读写剪贴板 |
| 截图 | 1 | 屏幕截图 |
| 系统通知 | 1 | 发送通知 |
| 音乐控制 (macOS) | 7 | 播放、暂停、切歌、音量 |
| Git | 4 | 状态、日志、差异、分支 |
| GitHub | 6 | PR、Issue、仓库信息 |

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
| `reminders_list_lists` | 列出所有提醒列表 |

### 备忘录（macOS）

| 工具 | 功能 |
|------|------|
| `notes_list_folders` | 列出备忘录文件夹 |
| `notes_list` | 列出备忘录 |
| `notes_read` | 读取备忘录内容 |
| `notes_create` | 创建新备忘录 |
| `notes_search` | 搜索备忘录 |
| `notes_delete` | 删除备忘录 |

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

### Git

| 工具 | 功能 |
|------|------|
| `git_status` | 查看仓库状态 |
| `git_log` | 查看提交日志 |
| `git_diff` | 查看文件差异 |
| `git_branch` | 查看分支信息 |

### GitHub

| 工具 | 功能 |
|------|------|
| `github_pr_list` | 列出 Pull Requests |
| `github_pr_view` | 查看 PR 详情 |
| `github_issue_list` | 列出 Issues |
| `github_issue_view` | 查看 Issue 详情 |
| `github_issue_create` | 创建新 Issue |
| `github_repo_view` | 查看仓库信息 |

### 其他

| 工具 | 功能 |
|------|------|
| `open_url` | 在浏览器中打开 URL |

---

## 智能对话

灵小缇支持**多轮对话记忆**，能够记住之前的对话内容，实现连续自然的交流体验。

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

## 语音交互

灵小缇支持**语音输入和语音输出**，让你可以完全通过语音与 AI 交互，解放双手。

### 两种模式

| 模式 | 命令 | 说明 |
|------|------|------|
| **Voice 模式** | `lingti-bot voice` | 按 Enter 开始录音，录音结束后 AI 处理并响应 |
| **Talk 模式** | `lingti-bot talk` | 持续监听，支持唤醒词激活，连续对话 |

### 语音引擎

| 引擎 | STT（语音转文字） | TTS（文字转语音） | 说明 |
|------|------------------|------------------|------|
| **system** | whisper-cpp | macOS say / Linux espeak | 本地处理，无需联网 |
| **openai** | Whisper API | OpenAI TTS | 云端处理，效果好 |
| **elevenlabs** | - | ElevenLabs API | 高品质语音合成 |

### 快速开始

```bash
# Voice 模式（按 Enter 录音）
lingti-bot voice --api-key sk-xxx

# 指定录音时长和语言
lingti-bot voice -d 10 -l zh --api-key sk-xxx

# 启用语音回复
lingti-bot voice --speak --api-key sk-xxx

# Talk 模式（持续监听）
lingti-bot talk --api-key sk-xxx

# 使用 OpenAI 语音引擎
lingti-bot voice --provider openai --voice-api-key sk-xxx --api-key sk-xxx
```

### 环境变量

| 变量 | 说明 |
|------|------|
| `VOICE_PROVIDER` | 语音引擎：system、openai、elevenlabs |
| `VOICE_API_KEY` | 语音 API 密钥（OpenAI 或 ElevenLabs） |
| `WHISPER_MODEL` | whisper-cpp 模型路径 |
| `WAKE_WORD` | 唤醒词（如 "hey lingti"） |

> **提示**：首次使用 system 引擎时会自动下载 whisper-cpp 模型（约 141MB）。

---

## 快速开始

### 其他安装方式

**从源码编译**

```bash
git clone https://github.com/ruilisi/lingti-bot.git
cd lingti-bot
make build  # 或: make darwin-arm64 / make linux-amd64
```

**手动下载**

前往 [GitHub Releases](https://github.com/ruilisi/lingti-bot/releases) 下载对应平台的二进制文件。

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
| `DINGTALK_CLIENT_ID` | 钉钉 AppKey | 钉钉集成必需 |
| `DINGTALK_CLIENT_SECRET` | 钉钉 AppSecret | 钉钉集成必需 |

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

**灵小缇** - 你的敏捷 AI 助手 🐕
