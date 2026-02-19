# 浏览器 AI 操作规则与技巧

> lingti-bot AI agent 在执行浏览器任务时遵循的所有规则、法则与最佳实践

---

## 核心法则：快照-操作（Snapshot-then-Act）

所有浏览器交互必须遵循以下模式：

```
1. Navigate   → 导航到目标网站（若已在该页面则跳过）
2. Snapshot   → browser_snapshot 获取页面元素及其编号
3. Act        → browser_click / browser_type / browser_press 操作元素
4. Re-snapshot → 页面变化后（点击、提交、跳转）立即重新 snapshot
```

**为什么用无障碍树而不是 CSS 选择器：**

| | 无障碍树（ref 编号） | CSS 选择器 |
|--|--|--|
| 稳定性 | 不受样式重构影响 | 类名变化即失效 |
| AI 可读性 | `[3] button "搜索"` 语义清晰 | 需理解 DOM 结构 |
| 通用性 | 标准 ARIA 规范，跨站点一致 | 每个网站各不相同 |

---

## 法则一：页面已打开时不重复导航

**规则：** 如果 browser 已在目标网站上（对话上下文中可以看到之前调用过 browser_navigate），直接从 `browser_snapshot` 开始，**不要**重新导航。

```
✅ 正确：
  用户：打开知乎
    → browser_navigate url="https://www.zhihu.com"
  用户：搜索 OpenClaw
    → browser_snapshot  （直接在当前知乎页面操作）
    → browser_type ref=N text="OpenClaw" submit=true

❌ 错误：
  用户：搜索 OpenClaw
    → browser_navigate url="https://www.zhihu.com"  ← 重复导航，浪费一轮
    → browser_snapshot
```

---

## 法则二：网站内搜索用搜索框，不用 web_search

**规则：** 当用户要求在某个已打开的网站上搜索内容时，必须使用页面的搜索输入框（`browser_type`），**绝不**调用 `web_search` 或 `web_fetch`。

```
✅ 正确：
  用户打开了知乎，然后说"搜索 OpenClaw"
    → browser_snapshot
    → 发现 [5] textbox "搜索"
    → browser_type ref=5 text="OpenClaw" submit=true

❌ 错误：
  用户打开了知乎，然后说"搜索 OpenClaw"
    → web_search query="site:zhihu.com OpenClaw"  ← 绕过了页面搜索栏
```

这条规则适用于所有平台：知乎、微博、小红书、B站、GitHub、Twitter 等。

---

## 法则三：不猜测、不构造 URL

**规则：** 不要通过拼接 URL 跳过 UI 交互步骤，必须模拟真实用户行为通过页面元素操作。

```
✅ 正确：
  → browser_navigate url="https://www.xiaohongshu.com"
  → browser_snapshot → 找到搜索框 → browser_type → 提交

❌ 错误：
  → browser_navigate url="https://www.xiaohongshu.com/search/OpenClaw"  ← 跳过 UI
```

**例外：** 已知的稳定入口 URL 可以直接使用（如登录页 `/login`、个人主页等），但搜索/过滤/分页参数不可猜测。

---

## 法则四：页面变化后立即重新 Snapshot

**规则：** 点击按钮、提交表单、跳转页面后，上一次 snapshot 的所有 ref 编号全部失效，必须重新执行 `browser_snapshot`。

```
browser_snapshot → [1]textbox "搜索" [2]button "提交" ...
browser_click ref=2   ← 页面跳转
browser_snapshot → ← 必须重新获取，旧 ref 已无效
```

---

## 法则五：遮罩/弹窗优先处理

**规则：** 如果点击元素时报错 "element covered by"（被遮挡），说明有弹窗或遮罩层覆盖。先用 `browser_execute_js` 关闭它，再重试。

**常用清理脚本：**

```javascript
// 关闭通用弹窗遮罩
document.querySelector('.modal-overlay')?.remove()
document.querySelector('.dialog-close-btn')?.click()

// 关闭知乎登录弹窗
document.querySelector('.Modal-closeButton')?.click()

// 关闭 Cookie 提示
document.querySelector('[class*="cookie"] button')?.click()

// 强制移除所有固定遮罩
document.querySelectorAll('[class*="mask"],[class*="overlay"],[class*="modal"]')
  .forEach(el => el.remove())
```

流程：

```
browser_click ref=N
→ 报错 "element covered by"
→ browser_execute_js script="document.querySelector('.Modal-closeButton')?.click()"
→ browser_snapshot  （重新获取 ref）
→ browser_click ref=N  （重试）
```

---

## 法则六：批量操作用 browser_click_all

**规则：** 当用户要求对"所有"内容执行点赞、关注、收藏等操作时，必须使用 `browser_click_all`，**不要**逐条从 snapshot 里找 ref 点击。

`browser_click_all` 会自动向下滚动并持续点击，直到不再出现新元素。

**国内主流平台常用选择器：**

| 操作 | 平台 | selector | skip_selector |
|------|------|----------|---------------|
| 点赞 | 通用 | `.like-wrapper` | `.like-wrapper.active, .like-wrapper.liked` |
| 收藏 | 通用 | `[class*='collect']` | `[class*='collect'].active` |
| 关注 | 通用 | `[class*='follow']` | `[class*='follow'].active` |

**调试流程（点击数为 0 时）：**

```javascript
// 用 JS 探查实际 DOM 结构
return Array.from(document.querySelectorAll('span,button'))
  .filter(e => e.children.length < 5)
  .slice(0, 10)
  .map(e => e.className + ' | ' + e.textContent.trim().slice(0, 15))
  .join('\n')
```

先试后查，不要未尝试就直接去检查 DOM。

---

## 法则七：连接模式与标签页行为

lingti-bot 有两种浏览器运行模式，行为不同：

| | **已连接模式**（用户已有 Chrome） | **独立模式**（bot 自己启动的 Chrome） |
|--|--|--|
| **browser_navigate** | 开新标签页，设为当前工作页 | 在现有标签页内导航 |
| **browser_snapshot/click/type** | 在上次 navigate 的标签页上操作 | 在现有标签页上操作 |
| **browser_stop** | 只断开连接，不关闭 Chrome | 关闭整个 Chrome |

**已连接模式配置（`~/.lingti.yaml`）：**

```yaml
browser:
  cdp_url: "127.0.0.1:9222"
```

Chrome 需以调试端口启动：

```bash
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome \
  --remote-debugging-port=9222 \
  --user-data-dir="$HOME/.lingti-bot/chrome-profile"
```

---

## 法则八：自动连接优先级

当 `browser_navigate` 被调用而浏览器未运行时，`EnsureRunning()` 按以下顺序决定使用哪个浏览器：

```
1. ~/.lingti.yaml 中配置的 cdp_url
2. 127.0.0.1:9222（well-known 默认调试端口，无需配置）
3. 启动新 Chrome 实例（fallback）
```

---

## 典型完整工作流

### 网站搜索

```
用户：打开知乎搜索 OpenClaw

→ browser_navigate url="https://www.zhihu.com"
→ browser_snapshot
   [1] link "关注"  [2] link "推荐"  [3] textbox "搜索"  [4] button "搜索"
→ browser_type ref=3 text="OpenClaw" submit=true
→ browser_snapshot  （搜索结果页，获取新 ref）
→ 返回结果摘要给用户
```

### 登录后操作

```
用户：登录 GitHub 查看我的 PR

→ browser_navigate url="https://github.com/login"
→ browser_snapshot
   [1] textbox "Username" [2] textbox "Password" [3] button "Sign in"
→ browser_type ref=1 text="your_username"
→ browser_type ref=2 text="your_password"
→ browser_click ref=3
→ browser_snapshot  （登录后页面）
→ browser_navigate url="https://github.com/pulls"
→ browser_snapshot
→ 提取并返回 PR 列表
```

### 表单填写

```
用户：帮我填写报名表

→ browser_navigate url="https://example.com/register"
→ browser_snapshot
   [1] textbox "姓名" [2] textbox "手机" [3] textbox "邮箱" [4] button "提交"
→ browser_type ref=1 text="张三"
→ browser_type ref=2 text="138xxxx1234"
→ browser_type ref=3 text="zhangsan@example.com"
→ browser_click ref=4
→ browser_snapshot  （确认页面）
→ browser_screenshot  （截图存档）
```

### 批量操作

```
用户：把我关注列表里所有人都取关

→ browser_navigate url="https://example.com/following"
→ browser_snapshot  （了解页面结构）
→ browser_click_all selector="[class*='unfollow']" delay_ms=500
→ 返回点击数量
```

### 多标签页并行

```
用户：同时打开知乎和微博，比较两个平台对同一话题的讨论

→ browser_navigate url="https://www.zhihu.com"
→ browser_type ref=N text="话题关键词" submit=true
→ browser_screenshot path="/tmp/zhihu.png"
→ browser_tab_open url="https://www.weibo.com"
→ browser_snapshot
→ browser_type ref=M text="话题关键词" submit=true
→ browser_screenshot path="/tmp/weibo.png"
→ 汇总两个平台的内容返回给用户
```

---

## 常见错误与纠正

| 错误行为 | 正确做法 |
|----------|----------|
| 页面已打开还重新导航 | 直接 `browser_snapshot` |
| 用 `web_search` 搜索网站内内容 | `browser_snapshot` → 找搜索框 → `browser_type` |
| 构造搜索 URL（`/search?q=xxx`） | 通过 UI 搜索框操作 |
| 点击后不重新 snapshot | 每次页面变化后必须 `browser_snapshot` |
| 批量点赞用逐条 `browser_click` | 改用 `browser_click_all` |
| 遮挡报错后直接放弃 | `browser_execute_js` 关闭弹窗后重试 |
| 用 `browser_start` 当第一步 | `browser_navigate` 自动启动，无需手动 start |

---

## ref 编号速查

- `[N] button "xxx"` → 可点击按钮
- `[N] textbox "xxx"` → 文本输入框（用 `browser_type`）
- `[N] link "xxx"` → 链接（用 `browser_click`）
- `[N] combobox "xxx"` → 下拉选择框
- `[N] checkbox "xxx"` → 复选框（用 `browser_click` 切换）
- `[N] heading "xxx"` → 标题（信息性，通常不可交互）

ref 仅在本次 snapshot 内有效。导航或页面刷新后必须重新 `browser_snapshot`。
