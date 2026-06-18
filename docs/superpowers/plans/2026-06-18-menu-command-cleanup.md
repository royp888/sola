# 菜单与命令优化 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 补全 BotFather 命令建议列表、补全 `/help` 管理员文本里缺失的命令、修复群组面板"私聊工作台"按钮（改为直接打开私聊 URL）、更新私聊群管中心说明文案。

**Architecture:** 
- `cmd/bot/main.go` 里的 `groupAdminCommands` 切片补全缺失命令项，Telegram 客户端才会在 `/` 输入时弹出提示
- `internal/bot/menu.go` 补全 `groupAdminHelpText` 并修改 `groupMarkup(botUsername)` 签名，把"私聊工作台"改成 URL 按钮
- `internal/bot/bind.go` 同步更新 `groupMarkup` 的所有调用点
- `internal/bot/private.go` 更新 `showPrivateAdminCenter` 文案

**Tech Stack:** Go 1.25、gotgbot/v2、现有 bot handler 模式

**依赖说明：** Task 1 里的 `set_verify_type` 条目依赖 `2026-06-18-bot-settings-commands.md` 的 Task 1 已完成（即 `/set_verify_type` 命令已注册）。如果那个计划还没执行，先跳过该条目，等合并后统一补上。

---

## 文件改动清单

| 文件 | 操作 |
|------|------|
| `cmd/bot/main.go` | 补全 `groupAdminCommands` 切片 |
| `internal/bot/menu.go` | 补全 `groupAdminHelpText`；修改 `groupMarkup(botUsername string)`；更新 `showGroupMenu` 调用 |
| `internal/bot/bind.go` | 更新 `groupMarkup(b.User.Username)` 三处调用 |
| `internal/bot/private.go` | 更新 `showPrivateAdminCenter` 文案 |

---

### Task 1: 补全 BotFather 命令列表

**Files:**
- Modify: `cmd/bot/main.go`

当前 `groupAdminCommands` 缺少：`set_welcome`、`set_warn_limit`、`set_verify_type`（需 bot-settings-commands 计划先完成）、`violations`、`warns`、`unwarn`、`verify_stats`、`setrules`（已在列表但 rules 相关的已有）。

- [ ] **Step 1: 替换 `registerBotCommands` 里的 `groupAdminCommands` 定义**

找到 `groupAdminCommands := append(append([]gotgbot.BotCommand{}, groupMemberCommands...), []gotgbot.BotCommand{` 这行，把整个追加列表替换为：

```go
groupAdminCommands := append(append([]gotgbot.BotCommand{}, groupMemberCommands...), []gotgbot.BotCommand{
    {Command: "ban", Description: "封禁成员（回复消息或接用户ID）"},
    {Command: "unban", Description: "解封 /unban 用户ID"},
    {Command: "mute", Description: "禁言 /mute 30m"},
    {Command: "unmute", Description: "解除禁言"},
    {Command: "kick", Description: "踢出成员"},
    {Command: "warn", Description: "警告成员"},
    {Command: "unwarn", Description: "清除警告（回复目标消息）"},
    {Command: "warns", Description: "查看成员警告记录"},
    {Command: "manage", Description: "打开成员管理面板"},
    {Command: "purge", Description: "批量删消息 /purge 或回复+/purge"},
    {Command: "del", Description: "删除回复的消息"},
    {Command: "promote", Description: "提升为管理员"},
    {Command: "demote", Description: "撤销管理员权限"},
    {Command: "set_title", Description: "设置管理员头衔"},
    {Command: "report", Description: "举报消息通知管理员"},
    {Command: "ban_ghosts", Description: "清理注销账号"},
    {Command: "bans", Description: "查看封禁记录"},
    {Command: "violations", Description: "查看违规记录"},
    {Command: "setrules", Description: "设置群规"},
    {Command: "clearrules", Description: "清除群规"},
    {Command: "rules", Description: "查看群规"},
    {Command: "publish", Description: "立即发布内容"},
    {Command: "posts", Description: "查看定时任务"},
    {Command: "adminconfig", Description: "查看群组配置"},
    {Command: "set_welcome", Description: "设置欢迎语（支持 {name}）"},
    {Command: "set_warn_limit", Description: "设置警告上限 /set_warn_limit 3"},
    {Command: "set_verify_type", Description: "更改验证类型（button/captcha/math 等）"},
    {Command: "verify_toggle", Description: "开关入群验证"},
    {Command: "verify_stats", Description: "查看入群验证统计"},
    {Command: "keywords", Description: "查看关键词规则"},
    {Command: "invites", Description: "管理邀请链接"},
    {Command: "levels", Description: "查看等级规则"},
}...)
```

- [ ] **Step 2: 编译验证**

```powershell
cd "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot"
go build ./...
```

预期：无输出。

- [ ] **Step 3: 提交**

```powershell
git add cmd/bot/main.go
git commit -m "feat(bot): 补全 BotFather 管理员命令提示列表"
```

---

### Task 2: 补全 `/help` 管理员文本

**Files:**
- Modify: `internal/bot/menu.go`

当前 `groupAdminHelpText` 缺少 `/purge`、`/del`、`/ban_ghosts`、`/report`、`/set_title`、`/verify_stats`、`/warns`/`/unwarn`（只用一行合并写了）。

- [ ] **Step 1: 替换 `groupAdminHelpText` 函数**

```go
func groupAdminHelpText() string {
	return strings.Join([]string{
		"管理员命令",
		"",
		"-- 成员管理 --",
		"回复消息 /manage 打开按钮管理面板",
		"/ban 原因 — 封禁（回复消息或 /ban 用户ID 原因）",
		"/unban 用户ID — 解封",
		"/mute 30m — 禁言（30m/2h/1d）",
		"/unmute — 解除禁言",
		"/kick — 踢出",
		"/warn 原因 — 警告  /unwarn — 清除  /warns — 查看记录",
		"/purge — 批量删消息（回复起点消息）",
		"/del — 删除回复的那条消息",
		"/ban_ghosts — 清理已注销账号",
		"/report — 举报（回复目标消息）",
		"/set_title 头衔 — 给自己或管理员设置头衔",
		"/bans — 封禁记录  /violations — 违规记录",
		"",
		"-- 积分 --",
		"回复 /points 10 — 给用户加减积分",
		"/points_config — 查看积分配置",
		"/points_toggle — 开关积分系统",
		"",
		"-- 运营 --",
		"/publish 内容 — 立即发布",
		"/posts — 定时任务列表",
		"/post_create — 创建定时提醒/循环发布",
		"",
		"-- 入群验证 --",
		"/verify_toggle — 开关入群验证",
		"/set_verify_type 类型 — 更改验证类型（button/captcha/multi_choice/poll/math/turnstile）",
		"/verify_stats — 查看验证通过/失败统计",
		"",
		"-- 设置 --",
		"/adminconfig — 查看当前群组配置",
		"/set_welcome 文本 — 设置欢迎语（支持 {name}）",
		"/set_warn_limit 数字 — 设置警告上限（默认 3）",
		"/rules — 查看群规  /setrules — 设置  /clearrules — 清除",
		"/keywords — 关键词规则",
		"/invites — 邀请链接管理",
		"/levels — 等级规则",
	}, "\n")
}
```

- [ ] **Step 2: 编译验证**

```powershell
go build ./...
```

预期：无输出。

- [ ] **Step 3: 提交**

```powershell
git add internal/bot/menu.go
git commit -m "docs(bot): 补全 /help 管理员命令列表"
```

---

### Task 3: 修复群组面板"私聊工作台"按钮

**Files:**
- Modify: `internal/bot/menu.go`
- Modify: `internal/bot/bind.go`

当前 `groupMarkup()` 里的"私聊工作台"按钮触发 `private:home` callback，但这个 callback 是在群聊里发起的，所有私聊逻辑（选择目标群、切换工作台）都在群聊里执行，体验混乱。

修复方案：把按钮改为 URL 按钮，直接跳到机器人私聊。`groupMarkup` 接收 `botUsername string` 参数，调用方传入 `b.User.Username`。

- [ ] **Step 1: 修改 `groupMarkup` 函数签名和按钮**

找到 `func groupMarkup() *gotgbot.SendMessageOpts` 并完整替换为：

```go
func groupMarkup(botUsername string) *gotgbot.SendMessageOpts {
	privateURL := "https://t.me/" + botUsername
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "💎 我的积分", CallbackData: CallbackData("points", "menu")},
			{Text: "🎁 抽奖大厅", CallbackData: CallbackData("lottery", "active")},
		},
		{
			{Text: "📣 发布中心", CallbackData: CallbackData("publish", "quick")},
			{Text: "⚙️ 群组配置", CallbackData: CallbackData("admin", "config")},
		},
		{
			{Text: "🛡 群管中心", CallbackData: CallbackData("admin", "moderation")},
			{Text: "🔍 关键词", CallbackData: CallbackData("admin", "keywords")},
		},
		{
			{Text: "🏅 等级规则", CallbackData: CallbackData("admin", "levels")},
			{Text: "🔗 邀请链接", CallbackData: CallbackData("admin", "invites")},
		},
		{
			{Text: "📱 私聊工作台", Url: privateURL},
		},
	}}}
}
```

- [ ] **Step 2: 更新 `showGroupMenu` 调用**

找到 `menu.go` 里 `showGroupMenu` 函数末尾的 `return respondText(b, ctx, ..., groupMarkup())`，改为：

```go
return respondText(b, ctx, strings.Join(lines, "\n"), groupMarkup(b.User.Username))
```

- [ ] **Step 3: 更新 `bind.go` 里的三处调用**

在 `bind.go` 里全局搜索 `groupMarkup()`（共 3 处），全部改为 `groupMarkup(b.User.Username)`：

```go
// 第 1 处（绑定失败）
return respondText(b, ctx, fmt.Sprintf("绑定失败：%s", err.Error()), groupMarkup(b.User.Username))

// 第 2 处（绑定成功）
return respondText(b, ctx, formatBotAdminStatus(scope.Chat, status)+"\n\n绑定成功：现在你可以用 Telegram 账号登录后台管理这个群。", groupMarkup(b.User.Username))

// 第 3 处（非管理员提示）
return respondText(b, ctx, formatBotAdminStatus(scope.Chat, status), groupMarkup(b.User.Username))
```

- [ ] **Step 4: 搜索确认没有遗漏的 `groupMarkup()` 调用**

```powershell
Select-String -Path "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\internal\bot\*.go" -Pattern "groupMarkup\(\)"
```

预期：无输出（即没有遗漏的旧调用）。

- [ ] **Step 5: 编译验证**

```powershell
go build ./...
```

预期：无输出。

- [ ] **Step 6: 提交**

```powershell
git add internal/bot/menu.go internal/bot/bind.go
git commit -m "fix(bot): 群组面板私聊工作台按钮改为 URL 直达私聊"
```

---

### Task 4: 更新私聊群管中心文案

**Files:**
- Modify: `internal/bot/private.go`

当前 `showPrivateAdminCenter` 写的是"入群验证、欢迎语、警告上限请在后台或群内命令里配置"，现在这些命令都已存在，直接列出来更有用。

- [ ] **Step 1: 替换 `showPrivateAdminCenter` 函数体里的 lines 定义**

找到 `showPrivateAdminCenter` 函数，把其中的 `lines` 定义替换为：

```go
lines := []string{
    "🛡 群管中心",
    "",
    fmt.Sprintf("目标：%s", chatTitle(chat)),
    "群组里可直接使用的命令：",
    "  /ban /unban /mute /unmute /kick — 成员管控",
    "  /warn /unwarn /warns — 警告管理",
    "  /purge /del — 消息清理",
    "",
    "群内设置命令：",
    "  /set_welcome 文本 — 设置欢迎语",
    "  /set_warn_limit 数字 — 警告上限",
    "  /set_verify_type 类型 — 验证类型",
    "  /verify_toggle — 开关入群验证",
    "  /adminconfig — 查看完整配置",
}
```

完整函数变为：

```go
func (a *App) showPrivateAdminCenter(b *gotgbot.Bot, ctx *ext.Context, chat api.ChatBinding) error {
	scope := requestScope(ctx)
	lines := []string{
		"🛡 群管中心",
		"",
		fmt.Sprintf("目标：%s", chatTitle(chat)),
		"群组里可直接使用的命令：",
		"  /ban /unban /mute /unmute /kick — 成员管控",
		"  /warn /unwarn /warns — 警告管理",
		"  /purge /del — 消息清理",
		"",
		"群内设置命令：",
		"  /set_welcome 文本 — 设置欢迎语",
		"  /set_warn_limit 数字 — 警告上限",
		"  /set_verify_type 类型 — 验证类型",
		"  /verify_toggle — 开关入群验证",
		"  /adminconfig — 查看完整配置",
	}
	if a.services.TelegramAccess != nil {
		status, err := a.services.TelegramAccess.CheckBotAdmin(scope.Context, b, chat.ChatID)
		if err == nil {
			lines = append(lines, "", fmt.Sprintf("Bot 管理状态：%s", status.Status))
		}
	}
	return respondText(b, ctx, strings.Join(lines, "\n"), privateConsoleMarkup(chat))
}
```

- [ ] **Step 2: 编译验证**

```powershell
go build ./...
```

预期：无输出。

- [ ] **Step 3: 提交**

```powershell
git add internal/bot/private.go
git commit -m "docs(bot): 更新私聊群管中心说明文案，列出具体命令"
```

---

### Task 5: 推送并发布 v1.0.6

> 注意：本计划与 `2026-06-18-bot-settings-commands.md` 独立，可以先执行本计划再执行那份，也可以反过来。v1.0.5 对应 bot-settings-commands，v1.0.6 对应本计划。如果两份计划同时跑，先确认 v1.0.5 的提交都已 push，再执行本步骤。

- [ ] **Step 1: 推送**

```powershell
git push origin main
```

- [ ] **Step 2: 打 tag**

```powershell
git tag v1.0.6
git push origin v1.0.6
```

- [ ] **Step 3: 创建 GitHub Release**

```powershell
gh release create v1.0.6 --title "v1.0.6" --notes "## 菜单与命令优化

- **BotFather 命令列表补全**：`/set_welcome`、`/set_warn_limit`、`/set_verify_type`、`/violations`、`/warns`、`/unwarn`、`/verify_stats` 等命令现在在 Telegram 输入 `/` 时会正确弹出提示
- **`/help` 管理员版补全**：新增 `/purge`、`/del`、`/ban_ghosts`、`/report`、`/set_title`、`/verify_stats` 等遗漏命令的说明，并按模块重新分组（成员管理 / 积分 / 运营 / 入群验证 / 设置）
- **群组面板私聊按钮修复**：「私聊工作台」按钮由 callback 改为 URL 直达私聊，避免私聊逻辑在群聊上下文里错误触发
- **私聊群管中心文案更新**：直接列出 `/set_welcome`、`/set_warn_limit`、`/set_verify_type` 等命令，不再笼统说「请在后台配置」"
```

- [ ] **Step 4: 更新 README changelog**

在 `README.md` 更新日志最顶部插入：

```
- **2026-06-18** v1.0.6 — 补全 BotFather 命令提示列表；补全 /help 管理员说明；修复群组面板私聊按钮；更新私聊群管中心文案
```

在 `README.en.md` 更新日志最顶部插入：

```
- **2026-06-18** v1.0.6 — Complete BotFather command hint list; fill in missing /help admin commands; fix group panel private-chat button (URL instead of callback); update private admin center copy
```

```powershell
git add README.md README.en.md
git commit -m "docs: 更新 v1.0.6 更新日志"
git push origin main
```
