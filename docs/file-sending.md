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

> **重要**：微信公众号的文件发送需要额外配置 `--wechat-app-id` 和 `--wechat-app-secret`。仅配置 `--user-id` 和 `--platform wechat` 只能发送文本消息，不能发送文件。

微信公众号通过[客服接口](https://developers.weixin.qq.com/doc/offiaccount/Message_Management/Service_Center_messages.html)发送媒体文件。这要求公众号已获得「客服接口」权限，并使用公众号的 AppID 和 AppSecret 获取 access_token。

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

### 支持的文件类型

| 文件类型 | 扩展名 | 发送方式 |
|---------|--------|---------|
| 图片 | .jpg .jpeg .png .gif .bmp | 图片消息 |
| 语音 | .amr .mp3 .speex | 语音消息 |
| 视频 | .mp4 | 视频消息 |
| 其他 | .md .txt .pdf .docx 等 | 文本预览（截取前 500 字） |

> **注意**：微信公众号 API 不支持发送任意文件附件（与企业微信不同）。对于文档类文件，bot 会读取文件内容并以文本消息形式发送预览。

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
