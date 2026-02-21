# Supported AI Providers / 支持的 AI 服务

lingti-bot 支持 **16 种 AI 服务**，涵盖国内外主流大模型平台及本地模型，按需切换。所有 provider 均通过 `--provider` 参数指定，也可在 `lingti-bot onboard` 交互式向导中选择。

lingti-bot supports **16 AI providers** covering mainstream LLM platforms globally plus local models. Select via `--provider` flag or the `lingti-bot onboard` interactive wizard.

## Provider List / 服务列表

| # | Provider | 名称 | Default Model / 默认模型 | API Key URL |
|---|----------|------|--------------------------|-------------|
| 1 | `deepseek` | DeepSeek (recommended / 推荐) | `deepseek-chat` | [platform.deepseek.com](https://platform.deepseek.com/api_keys) |
| 2 | `qwen` | Qwen / 通义千问 | `qwen-plus` | [bailian.console.aliyun.com](https://bailian.console.aliyun.com/) |
| 3 | `claude` | Claude (Anthropic) | `claude-sonnet-4-20250514` | [console.anthropic.com](https://console.anthropic.com/) |
| 4 | `kimi` | Kimi / Moonshot / 月之暗面 | `moonshot-v1-8k` | [platform.moonshot.cn](https://platform.moonshot.cn/) |
| 5 | `minimax` | MiniMax / 海螺 AI | `MiniMax-Text-01` | [platform.minimaxi.com](https://platform.minimaxi.com/) |
| 6 | `doubao` | Doubao / 豆包 (ByteDance) | `doubao-pro-32k` | [console.volcengine.com/ark](https://console.volcengine.com/ark) |
| 7 | `zhipu` | Zhipu / 智谱 GLM | `glm-4-flash` | [open.bigmodel.cn](https://open.bigmodel.cn/) |
| 8 | `openai` | OpenAI (GPT) | `gpt-4o` | [platform.openai.com](https://platform.openai.com/api-keys) |
| 9 | `gemini` | Gemini (Google) | `gemini-2.0-flash` | [aistudio.google.com](https://aistudio.google.com/apikey) |
| 10 | `yi` | Yi / 零一万物 (Lingyiwanwu) | `yi-large` | [platform.lingyiwanwu.com](https://platform.lingyiwanwu.com/) |
| 11 | `stepfun` | StepFun / 阶跃星辰 | `step-2-16k` | [platform.stepfun.com](https://platform.stepfun.com/) |
| 12 | `baichuan` | Baichuan / 百川智能 | `Baichuan4` | [platform.baichuan-ai.com](https://platform.baichuan-ai.com/) |
| 13 | `spark` | Spark / 讯飞星火 (iFlytek) | `generalv3.5` | [console.xfyun.cn](https://console.xfyun.cn/) |
| 14 | `siliconflow` | SiliconFlow / 硅基流动 (aggregator) | `Qwen/Qwen2.5-72B-Instruct` | [cloud.siliconflow.cn](https://cloud.siliconflow.cn/) |
| 15 | `grok` | Grok (xAI) | `grok-2-latest` | [console.x.ai](https://console.x.ai/) |
| 16 | `ollama` | Ollama (local / 本地) | `llama3.2` | No API key needed / 无需密钥 |

## Aliases / 别名

以下别名可以替代 `--provider` 值：

| Alias / 别名 | Maps to / 对应 |
|---------------|----------------|
| `anthropic` | `claude` |
| `moonshot` | `kimi` |
| `qianwen`, `tongyi` | `qwen` |
| `gpt`, `chatgpt` | `openai` |
| `glm`, `chatglm` | `zhipu` |
| `google` | `gemini` |
| `lingyiwanwu`, `wanwu` | `yi` |
| `bytedance`, `volcengine` | `doubao` |
| `iflytek`, `xunfei` | `spark` |
| `xai` | `grok` |
| `tencent`, `hungyuan` | `hunyuan` |

## Usage / 用法

```bash
# Interactive wizard / 交互式向导
lingti-bot onboard

# Command line / 命令行指定
lingti-bot relay --provider deepseek --api-key sk-xxx
lingti-bot router --provider openai --api-key sk-xxx --model gpt-4o

# Custom base URL / 自定义 API 地址
lingti-bot relay --provider siliconflow --api-key sk-xxx --base-url https://api.siliconflow.cn/v1

# Override default model / 覆盖默认模型
lingti-bot relay --provider qwen --api-key sk-xxx --model qwen-max
```

## Ollama (Local Models / 本地模型)

Ollama 在本地运行开源大模型，无需 API 密钥，默认监听 `http://localhost:11434`。

Ollama runs open-source LLMs locally with no API key required, listening on `http://localhost:11434` by default.

### Setup / 安装

```bash
# Install Ollama / 安装
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.com/install.sh | sh

# Pull a model / 拉取模型
ollama pull llama3.2

# Start the server (if not running as a service) / 启动服务
ollama serve

# Stop the server / 停止服务
# macOS (installed via brew or .dmg): quit from menu bar, or:
launchctl stop com.ollama.ollama
# Linux (systemd):
sudo systemctl stop ollama
# Foreground process: Ctrl+C
```

### Usage / 用法

```bash
# Default model (llama3.2) / 默认模型
lingti-bot relay --provider ollama

# Specify a model / 指定模型
lingti-bot relay --provider ollama --model mistral
lingti-bot relay --provider ollama --model qwen2.5:7b
lingti-bot relay --provider ollama --model deepseek-r1:8b

# Connect to a remote Ollama instance / 连接远程实例
lingti-bot relay --provider ollama --base-url http://192.168.1.100:11434/v1
```

### Available Models / 可用模型

Run `ollama list` to see installed models. Popular choices:

| Model | Size | Notes |
|-------|------|-------|
| `llama3.2` | 3B | Default, good general-purpose / 默认，通用 |
| `llama3.2:1b` | 1B | Lightweight / 轻量 |
| `mistral` | 7B | Strong reasoning / 推理能力强 |
| `qwen2.5:7b` | 7B | Good Chinese support / 中文支持好 |
| `deepseek-r1:8b` | 8B | Code & reasoning / 代码与推理 |

## Notes / 说明

- All non-Claude providers use the **OpenAI-compatible API** format, making it easy to add new providers.
- 除 Claude 外，所有 provider 均使用 **OpenAI 兼容 API** 格式，便于扩展。
- `siliconflow` is an aggregator platform that provides access to many open-source models (Qwen, DeepSeek, Llama, etc.) through a single API key.
- `siliconflow` 是一个聚合平台，通过一个 API Key 即可访问多种开源模型（Qwen、DeepSeek、Llama 等）。
- You can always override the default model with `--model` and the default API URL with `--base-url`.
- 可通过 `--model` 和 `--base-url` 覆盖默认模型和 API 地址。
