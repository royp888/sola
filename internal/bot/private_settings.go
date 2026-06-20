package bot

import (
	"fmt"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// ─── 进群验证 ────────────────────────────────────────────────────────────────

func (a *App) showPrivateVerifyPanel(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	if a.services.Admin == nil {
		return respondText(b, ctx, "群组配置服务尚未接入。", privateConsoleMarkup(chat))
	}
	cfg, err := a.services.Admin.GetConfig(scope.Context, chat.ChatID)
	if err != nil {
		return respondText(b, ctx, "获取配置失败："+err.Error(), privateConsoleMarkup(chat))
	}
	vt := cfg.VerifyType
	if vt == "" {
		vt = "button"
	}
	diff := cfg.VerifyDifficulty
	if diff == "" {
		diff = "medium"
	}
	text := strings.Join([]string{
		"✅ 进群验证",
		"━━━━━━━━━━",
		fmt.Sprintf("目标：%s", chatTitle(chat)),
		fmt.Sprintf("状态：%s", boolLabel(cfg.VerifyEnabled, "已开启 ✅", "已关闭 ❌")),
		fmt.Sprintf("验证类型：%s", verifyTypeLabel(vt)),
		fmt.Sprintf("超时：%d 秒", cfg.VerifyTimeout),
		fmt.Sprintf("难度：%s", diff),
	}, "\n")
	return respondText(b, ctx, text, privateVerifyMarkup(chat, cfg.VerifyEnabled))
}

func (a *App) toggleVerifyFromPrivate(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	if a.services.Admin == nil {
		return respondText(b, ctx, "群组配置服务尚未接入。", privateConsoleMarkup(chat))
	}
	cfg, err := a.services.Admin.ToggleVerify(scope.Context, chat.ChatID)
	if err != nil {
		return respondText(b, ctx, "切换失败："+err.Error(), privateConsoleMarkup(chat))
	}
	_ = answerCallback(b, ctx, "入群验证 "+boolLabel(cfg.VerifyEnabled, "已开启", "已关闭"))
	return a.showPrivateVerifyPanel(b, ctx, chat)
}

func (a *App) setVerifyTypeFromPrivate(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding, vtype string) error {
	scope := requestScope(ctx)
	if a.services.Admin == nil {
		return respondText(b, ctx, "群组配置服务尚未接入。", privateConsoleMarkup(chat))
	}
	validTypes := map[string]bool{
		"button": true, "captcha": true, "math": true,
		"multi_choice": true, "poll": true, "turnstile": true,
	}
	if !validTypes[vtype] {
		return answerCallback(b, ctx, "验证类型无效")
	}
	if _, err := a.services.Admin.UpdateConfig(scope.Context, chat.ChatID, ChatAdminConfigPatch{VerifyType: &vtype}); err != nil {
		return respondText(b, ctx, "设置失败："+err.Error(), privateConsoleMarkup(chat))
	}
	_ = answerCallback(b, ctx, "验证类型已更新为："+verifyTypeLabel(vtype))
	return a.showPrivateVerifyPanel(b, ctx, chat)
}

func privateVerifyMarkup(chat ChatBinding, enabled bool) *gotgbot.SendMessageOpts {
	toggleText := "❌ 关闭验证"
	if !enabled {
		toggleText = "✅ 开启验证"
	}
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: toggleText, CallbackData: CallbackData("private", "verify_toggle")},
		},
		{
			{Text: "🔘 按钮点击", CallbackData: CallbackData("private", "verify_set_type", "button")},
			{Text: "🔢 数字验证码", CallbackData: CallbackData("private", "verify_set_type", "captcha")},
		},
		{
			{Text: "➗ 数学题", CallbackData: CallbackData("private", "verify_set_type", "math")},
			{Text: "☑️ 多选题", CallbackData: CallbackData("private", "verify_set_type", "multi_choice")},
		},
		{
			{Text: "📊 原生投票", CallbackData: CallbackData("private", "verify_set_type", "poll")},
			{Text: "☁️ Turnstile", CallbackData: CallbackData("private", "verify_set_type", "turnstile")},
		},
		{
			{Text: "🔙 返回工作台", CallbackData: CallbackData("private", "console")},
		},
	}}}
}

func verifyTypeLabel(vt string) string {
	switch vt {
	case "button":
		return "按钮点击"
	case "captcha":
		return "数字验证码"
	case "math":
		return "数学题"
	case "multi_choice":
		return "多选题"
	case "poll":
		return "原生投票"
	case "turnstile":
		return "Cloudflare Turnstile"
	default:
		return vt
	}
}

// ─── 进群欢迎 ────────────────────────────────────────────────────────────────

func (a *App) showPrivateWelcomePanel(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	if a.services.Admin == nil {
		return respondText(b, ctx, "群组配置服务尚未接入。", privateConsoleMarkup(chat))
	}
	cfg, err := a.services.Admin.GetConfig(scope.Context, chat.ChatID)
	if err != nil {
		return respondText(b, ctx, "获取配置失败："+err.Error(), privateConsoleMarkup(chat))
	}
	welcome := strings.TrimSpace(cfg.WelcomeText)
	if welcome == "" {
		welcome = "（未设置）"
	}
	text := strings.Join([]string{
		"👋 进群欢迎",
		"━━━━━━━━━━",
		fmt.Sprintf("目标：%s", chatTitle(chat)),
		"",
		"当前欢迎语：",
		welcome,
		"",
		"支持 {name} 占位符（替换为新成员名称）。",
	}, "\n")
	return respondText(b, ctx, text, privateWelcomeMarkup())
}

func (a *App) startSetWelcomeWizard(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	state := &ConversationState{
		Type:   "set_welcome",
		Step:   1,
		ChatID: chat.ChatID,
		Data:   map[string]any{},
	}
	if err := a.setConversation(scope.Context, scope.Actor.ID, state); err != nil {
		return err
	}
	return respondText(b, ctx,
		"请发送新的欢迎语文本：\n\n支持 {name} 占位符，例如：\n欢迎 {name} 加入！\n\n发送 /cancel 取消。",
		&gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.ReplyKeyboardRemove{RemoveKeyboard: true}})
}

func (a *App) handleSetWelcomeStep(b *gotgbot.Bot, ctx *ext.Context, state *ConversationState) error {
	scope := requestScope(ctx)
	a.clearConversation(scope.Context, scope.Actor.ID)
	if ctx.Message == nil || strings.TrimSpace(ctx.Message.Text) == "" {
		return sendText(b, ctx, "欢迎语不能为空，操作已取消。", nil)
	}
	text := strings.TrimSpace(ctx.Message.Text)
	if a.services.Admin == nil {
		return sendText(b, ctx, "服务未接入。", nil)
	}
	if _, err := a.services.Admin.UpdateConfig(scope.Context, state.ChatID, ChatAdminConfigPatch{WelcomeText: &text}); err != nil {
		return sendText(b, ctx, "更新失败："+err.Error(), nil)
	}
	chats, _ := a.listPrivateManagedChats(ctx)
	for _, c := range chats {
		if c.ChatID == state.ChatID {
			return a.showPrivateWelcomePanel(b, ctx, c)
		}
	}
	return sendText(b, ctx, "✅ 欢迎语已更新。", nil)
}

func privateWelcomeMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "✏️ 修改欢迎语", CallbackData: CallbackData("private", "welcome_edit")},
		},
		{
			{Text: "🔙 返回工作台", CallbackData: CallbackData("private", "console")},
		},
	}}}
}

// ─── 关键词过滤 ──────────────────────────────────────────────────────────────

func (a *App) showPrivateKeywordsPanel(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	if a.services.KeywordFilter == nil {
		return respondText(b, ctx, "关键词过滤服务尚未接入。", privateConsoleMarkup(chat))
	}
	list, err := a.services.KeywordFilter.ListKeywords(scope.Context, chat.ChatID)
	if err != nil {
		return respondText(b, ctx, "获取失败："+err.Error(), privateConsoleMarkup(chat))
	}
	if strings.TrimSpace(list) == "" {
		list = "（暂无关键词）"
	}
	text := strings.Join([]string{
		"🚫 关键词过滤",
		"━━━━━━━━━━",
		fmt.Sprintf("目标：%s", chatTitle(chat)),
		"",
		list,
		"",
		"在群内发送命令添加或删除：",
		"/add_keyword 关键词",
		"/del_keyword 关键词",
	}, "\n")
	return respondText(b, ctx, text, privateBackConsoleMarkup())
}

// ─── 自动回复 ────────────────────────────────────────────────────────────────

func (a *App) showPrivateAutoreplyPanel(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	if a.services.AutoReply == nil {
		return respondText(b, ctx, "自动回复服务尚未接入。", privateConsoleMarkup(chat))
	}
	replies, err := a.services.AutoReply.ListForChat(scope.Context, chat.ChatID)
	if err != nil {
		return respondText(b, ctx, "获取失败："+err.Error(), privateConsoleMarkup(chat))
	}
	lines := []string{
		"🤖 自动回复",
		"━━━━━━━━━━",
		fmt.Sprintf("目标：%s", chatTitle(chat)),
		"",
	}
	if len(replies) == 0 {
		lines = append(lines, "（暂无自动回复规则）")
	} else {
		for i, r := range replies {
			status := "✅"
			if !r.Enabled {
				status = "❌"
			}
			lines = append(lines, fmt.Sprintf("%d. %s [%s] %s → %s",
				i+1, status, r.MatchType,
				truncateRunes(r.Keyword, 15),
				truncateRunes(r.ReplyText, 20)))
		}
	}
	lines = append(lines, "", "在群内发送命令添加或删除：", "/add_reply 触发词 | 回复内容", "/del_reply 触发词")
	return respondText(b, ctx, strings.Join(lines, "\n"), privateBackConsoleMarkup())
}

// ─── 群规 ────────────────────────────────────────────────────────────────────

func (a *App) showPrivateRulesPanel(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	if a.services.Admin == nil {
		return respondText(b, ctx, "群组配置服务尚未接入。", privateConsoleMarkup(chat))
	}
	cfg, err := a.services.Admin.GetConfig(scope.Context, chat.ChatID)
	if err != nil {
		return respondText(b, ctx, "获取配置失败："+err.Error(), privateConsoleMarkup(chat))
	}
	rules := strings.TrimSpace(cfg.RulesText)
	if rules == "" {
		rules = "（未设置群规）"
	}
	text := strings.Join([]string{
		"📋 群规",
		"━━━━━━━━━━",
		fmt.Sprintf("目标：%s", chatTitle(chat)),
		"",
		rules,
	}, "\n")
	return respondText(b, ctx, text, privateRulesMarkup())
}

func (a *App) startSetRulesWizard(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	state := &ConversationState{
		Type:   "set_rules",
		Step:   1,
		ChatID: chat.ChatID,
		Data:   map[string]any{},
	}
	if err := a.setConversation(scope.Context, scope.Actor.ID, state); err != nil {
		return err
	}
	return respondText(b, ctx,
		"请发送新的群规内容：\n\n发送 /cancel 取消。",
		&gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.ReplyKeyboardRemove{RemoveKeyboard: true}})
}

func (a *App) handleSetRulesStep(b *gotgbot.Bot, ctx *ext.Context, state *ConversationState) error {
	scope := requestScope(ctx)
	a.clearConversation(scope.Context, scope.Actor.ID)
	if ctx.Message == nil || strings.TrimSpace(ctx.Message.Text) == "" {
		return sendText(b, ctx, "群规内容不能为空，操作已取消。", nil)
	}
	text := strings.TrimSpace(ctx.Message.Text)
	if a.services.Admin == nil {
		return sendText(b, ctx, "服务未接入。", nil)
	}
	if _, err := a.services.Admin.UpdateConfig(scope.Context, state.ChatID, ChatAdminConfigPatch{RulesText: &text}); err != nil {
		return sendText(b, ctx, "更新失败："+err.Error(), nil)
	}
	chats, _ := a.listPrivateManagedChats(ctx)
	for _, c := range chats {
		if c.ChatID == state.ChatID {
			return a.showPrivateRulesPanel(b, ctx, c)
		}
	}
	return sendText(b, ctx, "✅ 群规已更新。", nil)
}

func privateRulesMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{
			{Text: "✏️ 修改群规", CallbackData: CallbackData("private", "rules_edit")},
		},
		{
			{Text: "🔙 返回工作台", CallbackData: CallbackData("private", "console")},
		},
	}}}
}

// ─── 等级规则 ────────────────────────────────────────────────────────────────

func (a *App) showPrivateLevelsPanel(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	if a.services.Level == nil {
		return respondText(b, ctx, "等级服务尚未接入。", privateConsoleMarkup(chat))
	}
	list, err := a.services.Level.ListLevelRules(scope.Context, chat.ChatID)
	if err != nil {
		return respondText(b, ctx, "获取失败："+err.Error(), privateConsoleMarkup(chat))
	}
	if strings.TrimSpace(list) == "" {
		list = "（暂无等级规则）"
	}
	text := strings.Join([]string{
		"🏅 等级规则",
		"━━━━━━━━━━",
		fmt.Sprintf("目标：%s", chatTitle(chat)),
		"",
		list,
		"",
		"在群内发送命令添加或删除：",
		"/add_level 等级 名称 最低积分",
		"/del_level 等级",
	}, "\n")
	return respondText(b, ctx, text, privateBackConsoleMarkup())
}

// ─── 邀请链接 ────────────────────────────────────────────────────────────────

func (a *App) showPrivateInvitePanel(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	if a.services.InviteLink == nil {
		return respondText(b, ctx, "邀请链接服务尚未接入。", privateConsoleMarkup(chat))
	}
	list, err := a.services.InviteLink.ListForChat(scope.Context, chat.ChatID, 20)
	if err != nil {
		return respondText(b, ctx, "获取失败："+err.Error(), privateConsoleMarkup(chat))
	}
	content := strings.TrimSpace(list)
	if content == "" {
		content = "（暂无邀请链接记录）"
	}
	text := strings.Join([]string{
		"🔗 邀请链接",
		"━━━━━━━━━━",
		fmt.Sprintf("目标：%s", chatTitle(chat)),
		"",
		content,
		"",
		"在群内发送命令创建或删除：",
		"/invite_create 名称",
		"/invite_delete ID",
	}, "\n")
	return respondText(b, ctx, text, privateBackConsoleMarkup())
}

// ─── 群组统计 ────────────────────────────────────────────────────────────────

func (a *App) showPrivateGroupStatsPanel(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	chatResource := fmt.Sprintf("%d", chat.ChatID)
	lines := []string{
		"📊 群组统计",
		"━━━━━━━━━━",
		fmt.Sprintf("目标：%s", chatTitle(chat)),
		"",
	}
	if a.services.Points != nil {
		stats, err := a.services.Points.GetActivityStats(scope.Context, chat.ChatID, "today")
		if err == nil && strings.TrimSpace(stats) != "" {
			lines = append(lines, "今日活跃：", stats)
		}
	}
	return respondText(b, ctx, strings.Join(lines, "\n"), &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "📈 今日统计", CallbackData: CallbackData("points", "private_stats", chatResource, "today")},
				{Text: "📊 本周统计", CallbackData: CallbackData("points", "private_stats", chatResource, "week")},
			},
			{
				{Text: "🏆 积分总榜", CallbackData: CallbackData("points", "private_rank", chatResource, "all")},
				{Text: "🔄 刷新", CallbackData: CallbackData("private", "stats")},
			},
			{
				{Text: "🔙 返回工作台", CallbackData: CallbackData("private", "console")},
			},
		}},
	})
}

// ─── 群管 ────────────────────────────────────────────────────────────────────

func (a *App) showPrivateModerationPanel(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	lines := []string{
		"🛡 群管",
		"━━━━━━━━━━",
		fmt.Sprintf("目标：%s", chatTitle(chat)),
		"",
	}
	if a.services.TelegramAccess != nil {
		status, err := a.services.TelegramAccess.CheckBotAdmin(scope.Context, b, chat.ChatID)
		if err == nil {
			lines = append(lines,
				fmt.Sprintf("Bot 状态：%s", status.Status),
				fmt.Sprintf("删消息 %s  封禁 %s  邀请 %s",
					boolLabel(status.CanDeleteMessages, "✅", "❌"),
					boolLabel(status.CanRestrictMembers, "✅", "❌"),
					boolLabel(status.CanInviteUsers, "✅", "❌"),
				),
			)
		}
	}
	if a.services.Admin != nil {
		cfg, err := a.services.Admin.GetConfig(scope.Context, chat.ChatID)
		if err == nil {
			lines = append(lines, fmt.Sprintf("警告上限：%d 次", cfg.WarnLimit))
		}
	}
	lines = append(lines, "", "成员操作请在群内回复目标消息使用：", "/ban  /mute  /kick  /warn  /manage")
	return respondText(b, ctx, strings.Join(lines, "\n"), &gotgbot.SendMessageOpts{
		ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{Text: "📜 封禁记录", CallbackData: CallbackData("private", "bans")},
				{Text: "⚠️ 违规记录", CallbackData: CallbackData("private", "violations")},
			},
			{
				{Text: "🔑 检查权限", CallbackData: CallbackData("private", "check_bot_perm")},
			},
			{
				{Text: "🔙 返回工作台", CallbackData: CallbackData("private", "console")},
			},
		}},
	})
}

func (a *App) showPrivateBansPanel(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	if a.services.Admin == nil {
		return respondText(b, ctx, "封禁记录服务尚未接入。", privateConsoleMarkup(chat))
	}
	bans, err := a.services.Admin.ListBans(scope.Context, chat.ChatID, 10)
	if err != nil {
		return respondText(b, ctx, "获取失败："+err.Error(), privateConsoleMarkup(chat))
	}
	var sb strings.Builder
	sb.WriteString("📜 封禁记录\n━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("目标：%s\n\n", chatTitle(chat)))
	if len(bans) == 0 {
		sb.WriteString("当前群暂无封禁记录。\n")
	} else {
		for i, ban := range bans {
			status := "封禁中"
			if ban.UnbannedAt != nil {
				status = "已解封"
			}
			reason := strings.TrimSpace(ban.Reason)
			if reason == "" {
				reason = "无"
			}
			sb.WriteString(fmt.Sprintf("%d. [%s] 用户 %d，原因：%s\n",
				i+1, status, ban.UserID, truncateRunes(reason, 28)))
		}
	}
	sb.WriteString("\n解封：/unban 用户ID（在群内使用）")
	return respondText(b, ctx, sb.String(), privateBackModerationMarkup())
}

func (a *App) showPrivateViolationsPanel(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	if a.services.Violations == nil {
		return respondText(b, ctx, "违规记录服务尚未接入。", privateConsoleMarkup(chat))
	}
	records, err := a.services.Violations.ListViolations(scope.Context, chat.ChatID, 0, 10, 0)
	if err != nil {
		return respondText(b, ctx, "获取失败："+err.Error(), privateConsoleMarkup(chat))
	}
	text := strings.Join([]string{
		"⚠️ 违规记录",
		"━━━━━━━━━━",
		fmt.Sprintf("目标：%s", chatTitle(chat)),
		"",
		formatViolationRecords(records),
	}, "\n")
	return respondText(b, ctx, text, privateBackModerationMarkup())
}

func (a *App) showPrivateCheckBotPermPanel(b *gotgbot.Bot, ctx *ext.Context, chat ChatBinding) error {
	scope := requestScope(ctx)
	lines := []string{
		"🔑 Bot 权限检查",
		"━━━━━━━━━━",
		fmt.Sprintf("目标：%s", chatTitle(chat)),
		"",
	}
	if a.services.TelegramAccess == nil {
		lines = append(lines, "权限检查服务尚未接入。")
	} else {
		status, err := a.services.TelegramAccess.CheckBotAdmin(scope.Context, b, chat.ChatID)
		if err != nil {
			lines = append(lines, "检查失败："+err.Error())
		} else if !status.IsAdmin {
			lines = append(lines, "❌ Bot 不是该群管理员，无法正常工作。", "", "请在群设置中将 Bot 设为管理员，并授予以下权限：", "• 删除消息", "• 封禁用户", "• 邀请用户")
		} else {
			lines = append(lines,
				"✅ Bot 是管理员",
				fmt.Sprintf("删除消息：%s", boolLabel(status.CanDeleteMessages, "✅", "❌")),
				fmt.Sprintf("封禁用户：%s", boolLabel(status.CanRestrictMembers, "✅", "❌")),
				fmt.Sprintf("邀请用户：%s", boolLabel(status.CanInviteUsers, "✅", "❌")),
				fmt.Sprintf("固定消息：%s", boolLabel(status.CanPinMessages, "✅", "❌")),
			)
		}
	}
	return respondText(b, ctx, strings.Join(lines, "\n"), privateBackModerationMarkup())
}

// ─── 通用 back 按钮 ──────────────────────────────────────────────────────────

func privateBackConsoleMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{{Text: "🔙 返回工作台", CallbackData: CallbackData("private", "console")}},
	}}}
}

func privateBackModerationMarkup() *gotgbot.SendMessageOpts {
	return &gotgbot.SendMessageOpts{ReplyMarkup: gotgbot.InlineKeyboardMarkup{InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
		{{Text: "🔙 返回群管", CallbackData: CallbackData("private", "moderation")}},
		{{Text: "🏠 返回工作台", CallbackData: CallbackData("private", "console")}},
	}}}
}
