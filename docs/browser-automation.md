# 浏览器自动化

> 基于 go-rod 的纯 Go 浏览器自动化，采用 **快照-操作（Snapshot-then-Act）** 模式

## 概述

lingti-bot 提供 12 个浏览器自动化 MCP 工具，可以程序化控制 Chrome/Brave/Edge 等 Chromium 内核浏览器。设计灵感来自 [OpenClaw](../docs/openclaw-reference.md) 的浏览器工具，但完全使用 Go 实现，无需 Node.js 或 Playwright。

### 核心设计：快照-操作模式

```
1. browser_snapshot  →  获取页面无障碍树，每个可交互元素分配数字编号(ref)
2. browser_click/type →  使用编号操作对应元素
3. 页面变化后        →  重新 snapshot 获取新编号
```

**示例流程：**

```
[1] textbox "用户名"
[2] textbox "密码"
[3] button "登录"
[4] link "忘记密码？"
```

→ `browser_type ref=1 text="admin"` → `browser_type ref=2 text="password123"` → `browser_click ref=3`

### 为什么用快照而不是 CSS 选择器？

- **稳定性** — 无障碍树不受 CSS 类名变化影响
- **AI 友好** — AI 模型可以直接理解角色和名称
- **简洁** — 用数字编号代替冗长的选择器路径

## 前置条件

需要安装 Chromium 内核浏览器（Chrome、Brave、Edge 或 Chromium）。lingti-bot 会自动检测已安装的浏览器。

**自动检测顺序：** Chrome → Brave → Edge → Chromium → Chrome Canary

如需指定浏览器路径，在启动时传入 `executable_path` 参数。

## 快速开始

```
"打开百度首页"                    → browser_navigate url="https://www.baidu.com"
"看看页面上有什么"                → browser_snapshot
"点击搜索框并输入 lingti-bot"    → browser_type ref=1 text="lingti-bot"
"点击搜索按钮"                    → browser_click ref=3
"截图保存"                        → browser_screenshot
```

## 工具参考

### 生命周期

| 工具 | 说明 | 参数 |
|------|------|------|
| `browser_start` | 启动浏览器 | `headless`(bool), `url`(string), `executable_path`(string) |
| `browser_stop` | 关闭浏览器 | 无 |
| `browser_status` | 查看浏览器状态 | 无 |

### 导航与快照

| 工具 | 说明 | 参数 |
|------|------|------|
| `browser_navigate` | 导航到指定 URL | `url`(必需) |
| `browser_snapshot` | 获取页面无障碍快照 | 无 |
| `browser_screenshot` | 截取页面截图 | `path`(string), `full_page`(bool) |

### 元素操作

| 工具 | 说明 | 参数 |
|------|------|------|
| `browser_click` | 点击元素 | `ref`(必需, number) |
| `browser_type` | 向元素输入文本 | `ref`(必需, number), `text`(必需), `submit`(bool) |
| `browser_press` | 按下键盘按键 | `key`(必需, 如 "Enter", "Tab", "Escape") |

### 标签页管理

| 工具 | 说明 | 参数 |
|------|------|------|
| `browser_tabs` | 列出所有标签页 | 无 |
| `browser_tab_open` | 打开新标签页 | `url`(string) |
| `browser_tab_close` | 关闭标签页 | `target_id`(string) |

## 快照格式详解

`browser_snapshot` 返回页面的无障碍树，格式如下：

```
[1] link "首页"
[2] link "新闻"
[3] textbox "搜索"
[4] button "百度一下"
[5] link "更多"
  [6] link "地图"
  [7] link "视频"
```

**规则：**
- 每个可交互元素分配一个数字编号 `[ref]`
- 显示元素的角色（role）和名称（name）
- 缩进表示层级关系
- 忽略不可见和装饰性元素
- **编号在页面导航后失效**，需要重新执行 `browser_snapshot`

## 配置

### 无头模式（Headless）

默认以无头模式启动（无可见窗口）。设置 `headless=false` 可显示浏览器窗口：

```
browser_start headless=false
```

### 自定义浏览器路径

```
browser_start executable_path="/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"
```

### 数据目录

浏览器数据（Cookie、缓存等）默认存储在 `~/.lingti-bot/browser/`，与个人浏览器完全隔离。

## 使用示例

### 网页信息采集

```
1. browser_navigate url="https://news.example.com"
2. browser_snapshot
   → 看到 [1] heading "今日头条" [2] link "科技新闻" ...
3. browser_click ref=2
4. browser_snapshot
   → 看到新页面内容
5. browser_screenshot path="/tmp/news.png"
```

### 表单填写

```
1. browser_navigate url="https://example.com/register"
2. browser_snapshot
   → [1] textbox "邮箱" [2] textbox "密码" [3] textbox "确认密码" [4] button "注册"
3. browser_type ref=1 text="user@example.com"
4. browser_type ref=2 text="mypassword"
5. browser_type ref=3 text="mypassword"
6. browser_click ref=4
```

### 多标签页操作

```
1. browser_start
2. browser_navigate url="https://example.com"
3. browser_tab_open url="https://news.example.com"
4. browser_tabs
   → 列出所有打开的标签页及其 target_id
5. browser_tab_close target_id="xxx"
```

### 键盘快捷键

```
1. browser_snapshot
2. browser_click ref=5        # 点击输入框
3. browser_type ref=5 text="搜索内容"
4. browser_press key="Enter"  # 按回车提交
```

## 技术架构

```
MCP Tool (internal/tools/browser.go)
    ↓
Browser Manager (internal/browser/browser.go)   ← 管理浏览器生命周期
    ↓
Snapshot Engine (internal/browser/snapshot.go)   ← 无障碍树 → ref 映射
Action Engine  (internal/browser/actions.go)     ← ref → DOM 元素 → 交互
    ↓
go-rod/rod (CDP)
    ↓
Chrome / Brave / Edge
```

**关键组件：**

- **Browser Manager** — 单例模式管理浏览器实例，支持启动、停止、连接已有浏览器
- **Snapshot Engine** — 调用 CDP `Accessibility.getFullAXTree` 获取完整无障碍树，为可交互元素分配 ref 编号
- **Action Engine** — 通过 ref 查找对应的 `BackendDOMNodeID`，使用 `DOM.resolveNode` 解析为可操作的 DOM 元素

## 故障排除

### 找不到浏览器

```
错误：no chrome executable found
```

安装 Chrome、Brave 或 Edge，或使用 `executable_path` 参数指定路径。

### 快照为空

页面可能还在加载中。`browser_navigate` 会等待页面加载完成，但动态内容可能需要额外等待。尝试再次执行 `browser_snapshot`。

### ref 无效

```
错误：ref 5 not found in snapshot
```

页面内容已变化，ref 已失效。重新执行 `browser_snapshot` 获取新编号。

### Linux 无头模式问题

如果在 Linux 服务器（无 GUI）上运行，确保使用 `headless=true`（默认）并安装必要的依赖：

```bash
# Debian/Ubuntu
apt-get install -y chromium-browser
```
