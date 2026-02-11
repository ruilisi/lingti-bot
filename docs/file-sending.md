# 文件发送指南

lingti-bot 支持通过自然语言在聊天平台中传输本地文件。用户只需对 AI 说"把桌面上的 xxx 文件发给我"，即可接收文件。

## 平台支持

| 平台 | 图片 | 语音 | 视频 | 任意文件 | 额外配置 |
|------|------|------|------|---------|---------|
| 企业微信 (WeCom) | ✅ | ✅ | ✅ | ✅ | 无需额外配置 |
| 微信公众号 | ✅ | ✅ | ✅ | ⚠️ 文本预览 | 需配置 AppID + AppSecret |
| 飞书 / Slack | — | — | — | — | 暂不支持 |

- **企业微信**：支持所有文件类型，包括文档、压缩包等任意格式
- **微信公众号**：支持图片/语音/视频直接发送；不支持的文件类型（如 .md、.pdf、.docx）会以文本预览形式发送（截取前 500 字）

## 企业微信 (WeCom)

企业微信通过 WeCom API 发送文件，relay 模式下自动启用（已配置 Corp ID + Secret）。

```bash
lingti-bot relay --platform wecom \
  --wecom-corp-id YOUR_CORP_ID \
  --wecom-agent-id YOUR_AGENT_ID \
  --wecom-secret YOUR_SECRET \
  --wecom-token YOUR_TOKEN \
  --wecom-aes-key YOUR_AES_KEY \
  --provider deepseek \
  --api-key YOUR_API_KEY
```

支持的媒体类型：`image`、`voice`、`video`、`file`（任意文件）。

详细配置：[企业微信集成指南](wecom-integration.md)

## 微信公众号

### 为什么需要额外配置？

微信公众号有两种消息发送方式：

1. **被动回复（Passive Reply）**— 用户发消息后，公众号在 5 秒内直接返回响应。这是 lingti-bot 默认使用的文本回复方式，通过云中继网关 `bot.lingti.com` 转发，**不需要你自己的公众号凭据**。
2. **主动发送（Customer Service API / 客服接口）**— 通过调用微信 API 主动向用户推送消息。这是发送图片、语音、视频等媒体文件的**唯一方式**，因为被动回复不支持上传和发送媒体素材。

文件发送使用的是**方式 2（主动发送）**，因此必须提供你自己公众号（或服务号）的 `WECHAT_APP_ID` 和 `WECHAT_APP_SECRET`，用于获取 access_token 来调用微信 API。

> **总结**：纯文本对话不需要任何公众号凭据；发送文件/图片/语音/视频则必须配置 AppID + AppSecret。

### 两个限制

**限制 1：必须提供自己的公众号/服务号凭据**

你需要一个已认证的微信公众号或服务号，并提供其 AppID 和 AppSecret。这是因为：
- 客服接口需要用你的公众号身份调用微信 API
- access_token 是基于你的 AppID + AppSecret 生成的
- 灵缇小秘公众号的凭据无法用于向你的关注者发送消息（微信的安全机制要求消息发送方和接收方属于同一公众号）

**限制 2：仅支持图片/语音/视频，不支持任意文件**

微信公众号客服接口[仅支持以下媒体类型](https://developers.weixin.qq.com/doc/offiaccount/Message_Management/Service_Center_messages.html)：

| 文件类型 | 扩展名 | 发送方式 | 说明 |
|---------|--------|---------|------|
| 图片 | .jpg .jpeg .png .gif .bmp | 图片消息 | 直接发送，用户可查看/保存 |
| 语音 | .amr .mp3 .speex | 语音消息 | 直接发送 |
| 视频 | .mp4 | 视频消息 | 直接发送 |
| 其他 | .md .txt .pdf .docx 等 | ⚠️ 文本预览 | 截取前 500 字以文本消息发送 |

与企业微信不同，**公众号 API 没有通用的 `file` 类型**。企业微信的应用消息接口支持发送任意文件附件（`msgtype: file`），但公众号客服接口不提供此能力，这是微信平台本身的限制。

对于不支持的文件类型（如 .md、.pdf、.docx），lingti-bot 会读取文件内容并以文本消息形式发送预览。由于微信文本消息有字数限制，内容会被截取至前 500 字并标注"内容过长，已截断"。

### 配置步骤

1. 登录[微信公众平台](https://mp.weixin.qq.com/)
2. 在「设置与开发」→「基本配置」中获取 **AppID** 和 **AppSecret**
3. 启动 relay 时传入这两个参数：

```bash
lingti-bot relay \
  --user-id YOUR_USER_ID \
  --platform wechat \
  --wechat-app-id YOUR_APP_ID \
  --wechat-app-secret YOUR_APP_SECRET \
  --provider deepseek \
  --api-key YOUR_API_KEY
```

也可使用环境变量：

```bash
export WECHAT_APP_ID="wx1234567890abcdef"
export WECHAT_APP_SECRET="your-app-secret"

lingti-bot relay --user-id YOUR_USER_ID --platform wechat --api-key YOUR_API_KEY
```

### 不配置 AppID/AppSecret 时

如果未配置 `--wechat-app-id` 和 `--wechat-app-secret`，当 AI 尝试发送文件时会收到错误：

```
media API not available for file sending
```

文本消息不受影响，仅文件发送需要这两个参数。

详细配置：[微信公众号接入指南](wechat-integration.md)

## 工作原理

1. 用户发送类似"把桌面上的 a.png 发给我"的消息
2. AI 调用 `file_send` 工具，指定文件路径和媒体类型
3. relay 客户端根据平台类型选择发送方式：
   - **企业微信**：调用 WeCom 临时素材上传 API → 发送应用消息
   - **微信公众号**：调用公众号临时素材上传 API → 通过客服接口发送
   - **微信公众号（不支持的文件类型）**：读取文件内容 → 以文本消息发送预览

## 配置参数

| 参数 | 环境变量 | 平台 | 说明 |
|------|---------|------|------|
| `--wechat-app-id` | `WECHAT_APP_ID` | 微信公众号 | 公众号 AppID |
| `--wechat-app-secret` | `WECHAT_APP_SECRET` | 微信公众号 | 公众号 AppSecret |
| `--wecom-corp-id` | `WECOM_CORP_ID` | 企业微信 | 企业 ID |
| `--wecom-secret` | `WECOM_SECRET` | 企业微信 | 应用 Secret |
