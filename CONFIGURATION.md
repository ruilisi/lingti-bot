# 配置优先级

lingti-bot 采用三层配置解析机制，优先级从高到低：

```
命令行参数  >  环境变量  >  配置文件 (~/.lingti.yaml)
```

每个配置项按此顺序查找，找到即停止。这意味着：

- **命令行参数**始终优先，适合临时覆盖或运行多个实例
- **环境变量**适合 CI/CD 或容器化部署
- **配置文件**适合日常使用，通过 `lingti-bot onboard` 生成

## 示例

以 AI Provider 为例，解析顺序为：

| 优先级 | 来源 | 示例 |
|--------|------|------|
| 1 | `--provider deepseek` | 命令行参数 |
| 2 | `AI_PROVIDER=deepseek` | 环境变量 |
| 3 | `ai.provider: deepseek` | ~/.lingti.yaml |

```bash
# 配置文件中设置了 provider: qwen
# 环境变量设置了 AI_PROVIDER=deepseek
# 命令行指定了 --provider claude
# 最终使用: claude（命令行参数最高优先）
```

## 配置文件

默认路径：`~/.lingti.yaml`

通过交互式向导生成：

```bash
lingti-bot onboard
```

### 完整结构

```yaml
mode: relay  # "relay" 或 "router"

ai:
  provider: deepseek
  api_key: sk-xxx
  base_url: ""       # 自定义 API 地址（可选）
  model: ""          # 自定义模型名（可选，留空使用 provider 默认值）

relay:
  platform: wecom    # "feishu", "slack", "wechat", "wecom"
  user_id: ""        # 从 /whoami 获取（WeCom 不需要）

platforms:
  wecom:
    corp_id: ""
    agent_id: ""
    secret: ""
    token: ""
    aes_key: ""
  wechat:
    app_id: ""
    app_secret: ""
  feishu:
    app_id: ""
    app_secret: ""
  slack:
    bot_token: ""
    app_token: ""
  dingtalk:
    client_id: ""
    client_secret: ""
  telegram:
    token: ""
  discord:
    token: ""

browser:
  screen_size: fullscreen  # "fullscreen" 或 "宽x高"（如 "1024x768"），默认 fullscreen
```

## 环境变量

### AI 配置

| 环境变量 | 对应参数 | 说明 |
|----------|----------|------|
| `AI_PROVIDER` | `--provider` | AI 服务商 |
| `AI_API_KEY` | `--api-key` | API 密钥 |
| `AI_BASE_URL` | `--base-url` | 自定义 API 地址 |
| `AI_MODEL` | `--model` | 模型名称 |
| `ANTHROPIC_API_KEY` | `--api-key` | API 密钥（fallback） |
| `ANTHROPIC_BASE_URL` | `--base-url` | API 地址（fallback） |
| `ANTHROPIC_MODEL` | `--model` | 模型名称（fallback） |

### Relay 配置

| 环境变量 | 对应参数 | 说明 |
|----------|----------|------|
| `RELAY_USER_ID` | `--user-id` | 用户 ID |
| `RELAY_PLATFORM` | `--platform` | 平台类型 |
| `RELAY_SERVER_URL` | `--server` | WebSocket 服务器地址 |
| `RELAY_WEBHOOK_URL` | `--webhook` | Webhook 地址 |

### 平台凭证

| 环境变量 | 对应参数 | 说明 |
|----------|----------|------|
| `WECOM_CORP_ID` | `--wecom-corp-id` | 企业微信 Corp ID |
| `WECOM_AGENT_ID` | `--wecom-agent-id` | 企业微信 Agent ID |
| `WECOM_SECRET` | `--wecom-secret` | 企业微信 Secret |
| `WECOM_TOKEN` | `--wecom-token` | 企业微信回调 Token |
| `WECOM_AES_KEY` | `--wecom-aes-key` | 企业微信 AES Key |
| `WECHAT_APP_ID` | `--wechat-app-id` | 微信公众号 App ID |
| `WECHAT_APP_SECRET` | `--wechat-app-secret` | 微信公众号 App Secret |
| `SLACK_BOT_TOKEN` | - | Slack Bot Token |
| `SLACK_APP_TOKEN` | - | Slack App Token |
| `FEISHU_APP_ID` | - | 飞书 App ID |
| `FEISHU_APP_SECRET` | - | 飞书 App Secret |
| `DINGTALK_CLIENT_ID` | - | 钉钉 Client ID |
| `DINGTALK_CLIENT_SECRET` | - | 钉钉 Client Secret |

## 典型用法

### 日常使用：配置文件

```bash
lingti-bot onboard        # 首次配置
lingti-bot relay           # 之后无需任何参数
```

### 临时覆盖：命令行参数

```bash
# 配置文件用 deepseek，临时切换到 qwen 测试
lingti-bot relay --provider qwen --model qwen-plus
```

### 容器部署：环境变量

```bash
docker run -e AI_PROVIDER=deepseek -e AI_API_KEY=sk-xxx lingti-bot relay
```

### 多实例运行：命令行参数覆盖

```bash
# 实例 1: 企业微信
lingti-bot relay --platform wecom --provider deepseek --api-key sk-aaa

# 实例 2: 飞书（不同 provider）
lingti-bot relay --platform feishu --user-id xxx --provider claude --api-key sk-bbb
```
