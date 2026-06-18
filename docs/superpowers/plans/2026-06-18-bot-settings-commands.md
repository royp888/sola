# Bot 设置命令 + 群名自动同步 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增 `/set_verify_type` 命令让管理员在群内直接切换验证类型；新增群名自动同步（收到 `new_chat_title` 服务消息时更新 DB）；补全 `/help` 里的设置命令说明。

**Architecture:** 全部改动在 `internal/bot/` 层。`/set_verify_type` 跟已有的 `/set_welcome`、`/set_warn_limit` 完全相同的模式：校验权限 → 解析参数 → 调 `AdminService.UpdateConfig`。群名同步用自定义 filter 注册 `message.NewChatTitle` 类型的 handler，调已有的 `ChatBindingService.Bind()`（GORM 用 Assign+FirstOrCreate，nil 指针字段不覆盖）。

**Tech Stack:** Go 1.25、gotgbot/v2、GORM、internal/bot/ 现有模式

---

## 文件改动清单

| 文件 | 操作 |
|------|------|
| `internal/bot/admin.go` | 新增 `handleSetVerifyType`；在 `registerAdminHandlers` 里注册 |
| `internal/bot/bind.go` | 新增 `isNewChatTitle` filter + `handleNewChatTitle` handler |
| `internal/bot/menu.go` | 在 `registerCoreHandlers` 注册 title handler；补全 `groupAdminHelpText` |

---

### Task 1: 新增 `/set_verify_type` 命令

**Files:**
- Modify: `internal/bot/admin.go`

- [ ] **Step 1: 在 `registerAdminHandlers` 里注册新命令**

在 `admin.go` 的 `registerAdminHandlers` 函数末尾（`a.registerRulesHandlers(d)` 之前）添加一行：

```go
func (a *App) registerAdminHandlers(d *ext.Dispatcher) {
	d.AddHandler(handlers.NewCommand("adminconfig", a.wrap(a.handleAdminConfig, a.RateLimit("cmd:adminconfig", 1))))
	d.AddHandler(handlers.NewCommand("set_welcome", a.wrap(a.handleSetWelcome, a.RateLimit("cmd:set_welcome", 1))))
	d.AddHandler(handlers.NewCommand("set_warn_limit", a.wrap(a.handleSetWarnLimit, a.RateLimit("cmd:set_warn_limit", 1))))
	d.AddHandler(handlers.NewCommand("set_verify_type", a.wrap(a.handleSetVerifyType, a.RateLimit("cmd:set_verify_type", 1))))
	d.AddHandler(handlers.NewCommand("set_level", a.wrap(a.handleSetLevel, a.RateLimit("cmd:set_level", 1))))
	d.AddHandler(handlers.NewCommand("levels", a.wrap(a.handleLevels, a.RateLimit("cmd:levels", 1))))
	d.AddHandler(handlers.NewCommand("add_level", a.wrap(a.handleAddLevel, a.RateLimit("cmd:add_level", 1))))
	d.AddHandler(handlers.NewCommand("del_level", a.wrap(a.handleDelLevel, a.RateLimit("cmd:del_level", 1))))
	a.registerRulesHandlers(d)
}
```

- [ ] **Step 2: 新增 `handleSetVerifyType` 函数**

在 `admin.go` 里 `handleSetWarnLimit` 函数结束后（`}` 之后）插入：

```go
var validVerifyTypes = map[string]bool{
	"button":       true,
	"captcha":      true,
	"multi_choice": true,
	"poll":         true,
	"math":         true,
	"turnstile":    true,
}

func (a *App) handleSetVerifyType(b *gotgbot.Bot, ctx *ext.Context) error {
	scope := requestScope(ctx)
	if err := a.requireTelegramManager(b, ctx); err != nil {
		return err
	}
	if a.services.Admin == nil {
		return sendText(b, ctx, "群组配置服务尚未接入。", nil)
	}
	args := commandArgs(ctx)
	if len(args) < 1 {
		return sendText(b, ctx, "用法：/set_verify_type <类型>\n可选类型：button captcha multi_choice poll math turnstile", nil)
	}
	vtype := strings.ToLower(strings.TrimSpace(args[0]))
	if !validVerifyTypes[vtype] {
		return sendText(b, ctx, "未知验证类型，可选：button captcha multi_choice poll math turnstile", nil)
	}
	cfg, err := a.services.Admin.UpdateConfig(scope.Context, scope.Chat.ID, ChatAdminConfigPatch{VerifyType: &vtype})
	if err != nil {
		return err
	}
	return sendText(b, ctx, "验证类型已更新。\n"+formatAdminConfig(cfg), nil)
}
```

- [ ] **Step 3: 编译验证**

```powershell
cd "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot"
go build ./...
```

预期：无输出（编译通过）。

- [ ] **Step 4: 提交**

```powershell
git add internal/bot/admin.go
git commit -m "feat(bot): add /set_verify_type command"
```

---

### Task 2: 群名自动同步

**Files:**
- Modify: `internal/bot/bind.go`
- Modify: `internal/bot/menu.go`

背景：`ChatBindingService.Bind()` 内部用 GORM 的 `Assign+FirstOrCreate`。`Username`/`InviteLink`/`Description` 是 `*string`，传 `nil` 时 GORM 不覆盖现有值；`OwnerUserID` 也只在 `owner != nil` 时才写。所以只传 `ChatID`、`ChatType`、`Title` 就能安全更新群名而不动其他字段。

- [ ] **Step 1: 在 `bind.go` 新增 filter 和 handler**

在 `bind.go` 文件末尾追加以下代码（紧接现有函数之后）：

```go
// isNewChatTitle 是 gotgbot message filter，匹配群名变更服务消息。
func isNewChatTitle(msg *gotgbot.Message) bool {
	return msg != nil && msg.NewChatTitle != ""
}

// handleNewChatTitle 在群名变更时自动更新 DB 里存储的标题。
// 只更新 Title 字段；不改变群主归属或其他绑定信息。
// 必须返回 ext.ContinueGroups，否则会阻断后续 handler。
func (a *App) handleNewChatTitle(b *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.Message == nil || ctx.Message.NewChatTitle == "" {
		return ext.ContinueGroups
	}
	if a.services.ChatBindings == nil {
		return ext.ContinueGroups
	}
	scope := requestScope(ctx)
	if scope.Chat.ID == 0 {
		return ext.ContinueGroups
	}
	_, _ = a.services.ChatBindings.Bind(scope.Context, api.ChatBindingRequest{
		ChatID:   scope.Chat.ID,
		ChatType: scope.Chat.Type,
		Title:    ctx.Message.NewChatTitle,
	})
	return ext.ContinueGroups
}
```

确认 `bind.go` 顶部已有 import `"github.com/PaulSonOfLars/gotgbot/v2/ext"` 和 `"github.com/dabowin/sola/internal/api"`。如果已有则不需要新增。

- [ ] **Step 2: 在 `menu.go` 的 `registerCoreHandlers` 里注册**

找到 `registerCoreHandlers` 函数，在 `a.registerSedHandlers(d)` 这行**之前**插入 handler 注册：

```go
func (a *App) registerCoreHandlers(d *ext.Dispatcher) {
	d.AddHandler(handlers.NewCommand("start", a.wrap(a.handleStart, a.RateLimit("cmd:start", 1))))
	d.AddHandler(handlers.NewCommand("menu", a.wrap(a.handleStart, a.RateLimit("cmd:menu", 1))))
	d.AddHandler(handlers.NewCommand("help", a.wrap(a.handleHelp, a.RateLimit("cmd:help", 1))))
	d.AddHandler(handlers.NewCommand("info", a.wrap(a.handleInfo, a.RateLimit("cmd:info", 1))))
	d.AddHandler(handlers.NewCommand("html", a.wrap(a.handleHTMLHelp, a.RateLimit("cmd:html", 1))))
	d.AddHandler(handlers.NewCommand("bind", a.wrap(a.handleBind, a.RateLimit("cmd:bind", 1))))
	d.AddHandler(handlers.NewCommand("check_admin", a.wrap(a.handleCheckAdmin, a.RateLimit("cmd:check_admin", 1))))
	d.AddHandler(handlers.NewCommand("cancel", a.wrap(a.handleCancel, a.RateLimit("cmd:cancel", 1))))
	d.AddHandler(handlers.NewMessage(isNewChatTitle, a.handleNewChatTitle))
	d.AddHandler(handlers.NewMessage(message.All, a.handleChineseCommand))
	a.registerSedHandlers(d)
	d.AddHandler(handlers.NewCallback(callbackquery.Prefix(CallbackPrefix+":"), a.wrap(a.router.Handle, a.RateLimit("callback", 1))))
}
```

注意：`isNewChatTitle` 这行必须在 `message.All` 之前注册，否则 `message.All` handler 会先匹配并可能干扰执行顺序。

- [ ] **Step 3: 编译验证**

```powershell
cd "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot"
go build ./...
```

预期：无输出（编译通过）。

- [ ] **Step 4: 提交**

```powershell
git add internal/bot/bind.go internal/bot/menu.go
git commit -m "feat(bot): auto-sync chat title on new_chat_title service message"
```

---

### Task 3: 补全 `/help` 设置命令说明

**Files:**
- Modify: `internal/bot/menu.go`

当前 `groupAdminHelpText()` 的「设置」节里缺少 `/set_warn_limit` 和 `/set_verify_type`。

- [ ] **Step 1: 更新 `groupAdminHelpText` 函数**

找到 `groupAdminHelpText` 函数，将其中的「-- 设置 --」节替换为：

```go
"-- 设置 --",
"/adminconfig 查看当前群组配置",
"/set_welcome 文本 设置欢迎语（支持 {name}）",
"/set_warn_limit 数字 设置警告上限（默认 3）",
"/set_verify_type 类型 更改验证类型（button/captcha/multi_choice/poll/math/turnstile）",
"/verify_toggle 开关入群验证",
"/keywords 关键词规则",
"/invites 邀请链接管理",
"/levels 等级规则",
```

完整函数应为：

```go
func groupAdminHelpText() string {
	return strings.Join([]string{
		"管理员命令",
		"",
		"-- 成员管理 --",
		"回复消息 /manage 打开按钮管理面板",
		"/ban 原因 封禁（回复消息或 /ban 用户ID 原因）",
		"/unban 用户ID 解封",
		"/mute 30m 禁言（30m/2h/1d）",
		"/unmute 解除禁言",
		"/kick 踢出",
		"/warn 原因 警告  /unwarn 清除  /warns 查看",
		"/bans 封禁记录  /violations 违规记录",
		"",
		"-- 积分 --",
		"回复 /points 10 给用户加减积分",
		"/points_config 查看积分配置",
		"/points_toggle 开关积分系统",
		"",
		"-- 运营 --",
		"/publish 内容 立即发布",
		"/posts 定时任务列表",
		"/post_create 创建定时提醒/循环发布",
		"",
		"-- 设置 --",
		"/adminconfig 查看当前群组配置",
		"/set_welcome 文本 设置欢迎语（支持 {name}）",
		"/set_warn_limit 数字 设置警告上限（默认 3）",
		"/set_verify_type 类型 更改验证类型（button/captcha/multi_choice/poll/math/turnstile）",
		"/verify_toggle 开关入群验证",
		"/keywords 关键词规则",
		"/invites 邀请链接管理",
		"/levels 等级规则",
	}, "\n")
}
```

- [ ] **Step 2: 编译验证**

```powershell
cd "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot"
go build ./...
```

预期：无输出（编译通过）。

- [ ] **Step 3: 提交**

```powershell
git add internal/bot/menu.go
git commit -m "docs(bot): add set_warn_limit and set_verify_type to /help admin text"
```

---

### Task 4: 推送并发布 v1.0.5

- [ ] **Step 1: 推送**

```powershell
git push origin main
```

- [ ] **Step 2: 打 tag**

```powershell
git tag v1.0.5
git push origin v1.0.5
```

- [ ] **Step 3: 创建 GitHub Release**

```powershell
gh release create v1.0.5 --title "v1.0.5" --notes "## 新增功能

- **\`/set_verify_type\` 命令**：管理员可在群内直接切换验证类型（button/captcha/multi_choice/poll/math/turnstile），无需打开后台
- **群名自动同步**：群改名后 DB 里的标题自动更新，后台不再显示旧名称
- **补全 \`/help\` 命令列表**：新增 \`/set_warn_limit\` 和 \`/set_verify_type\` 的说明条目"
```

- [ ] **Step 4: 更新 README changelog**

在 `README.md` 的更新日志最顶部插入：

```
- **2026-06-18** v1.0.5 — 新增 /set_verify_type 命令（群内直接切换验证类型）、群名改变后自动同步到后台、补全 /help 设置命令说明
```

在 `README.en.md` 的更新日志最顶部插入：

```
- **2026-06-18** v1.0.5 — Add /set_verify_type command (change verify type in-group), auto-sync chat title when group is renamed, add missing settings commands to /help
```

```powershell
git add README.md README.en.md
git commit -m "docs: 更新 v1.0.5 更新日志"
git push origin main
```
