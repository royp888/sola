package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"

	"github.com/dabowin/sola/internal/bootstrap"
	botapp "github.com/dabowin/sola/internal/bot"
	"github.com/dabowin/sola/internal/config"
	"github.com/dabowin/sola/internal/service"
	"github.com/dabowin/sola/internal/web"
	"github.com/dabowin/sola/internal/worker"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	botServices := service.NewBundle(nil, nil).BotServices()

	resources, err := bootstrap.New(ctx, "")
	if err != nil {
		log.Printf("bootstrap resources unavailable; starting bot without database/redis: %v", sanitizeTokenError(err, cfg.Bot.Token))
	} else {
		defer resources.Close(context.Background())
		cfg = resources.Config
		botServices = resources.BotServices()
	}

	token := strings.TrimSpace(cfg.Bot.Token)
	if token == "" {
		return fmt.Errorf("bot token is required: set SOLA_BOT_TOKEN or bot.token")
	}

	requestOpts := &gotgbot.RequestOpts{Timeout: 30 * time.Second}
	tgBot, err := gotgbot.NewBot(token, &gotgbot.BotOpts{
		BotClient: &gotgbot.BaseBotClient{
			Client:             http.Client{Timeout: 35 * time.Second},
			DefaultRequestOpts: requestOpts,
		},
		RequestOpts: requestOpts,
	})
	if err != nil {
		return fmt.Errorf("create telegram bot: %w", sanitizeTokenError(err, token))
	}

	me, err := tgBot.GetMe(nil)
	if err != nil {
		return fmt.Errorf("getMe: %w", sanitizeTokenError(err, token))
	}
	log.Printf("telegram bot connected: id=%d username=@%s", me.Id, me.Username)
	if err := registerBotCommands(ctx, tgBot); err != nil {
		log.Printf("register bot commands failed: %v", sanitizeTokenError(err, token))
	} else {
		log.Printf("telegram bot commands registered")
	}

	dispatcher := ext.NewDispatcher(&ext.DispatcherOpts{
		Error: func(_ *gotgbot.Bot, _ *ext.Context, err error) ext.DispatcherAction {
			log.Printf("telegram handler error: %v", err)
			return ext.DispatcherActionNoop
		},
		UnhandledErrFunc: func(err error) {
			log.Printf("telegram dispatcher error: %v", err)
		},
	})

	app := botapp.New(botServices, botapp.Options{
		DefaultLocale:         cfg.Bot.DefaultLocale,
		MiniAppURL:            cfg.Bot.MiniAppURL,
		TurnstileVerifySecret: cfg.Turnstile.VerifySecret,
		Features:              botapp.NewFeatures(cfg.Bot.DisabledFeatures),
	})
	app.Register(dispatcher)

	updater := ext.NewUpdater(dispatcher, &ext.UpdaterOpts{
		UnhandledErrFunc: func(err error) {
			log.Printf("telegram polling error: %v", sanitizeTokenError(err, token))
		},
	})

	if mode := strings.TrimSpace(cfg.Bot.Mode); mode != "" && !strings.EqualFold(mode, "polling") {
		log.Printf("bot.mode=%q configured; cmd/bot is starting polling", mode)
	}

	if resources != nil {
		go func() {
			runner := worker.New(resources.Config, resources.Store, resources.Logger)
			if err := runner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				resources.Logger.Error("worker exited with error", zap.Error(err))
			}
		}()
	}

	// Start Mini App HTTP server for Turnstile verification (if configured).
	if cfg.App.HTTPAddr != "" && cfg.Turnstile.SiteKey != "" {
		mux := http.NewServeMux()
		handler := web.NewTurnstileHandler(web.TurnstileConfig{
			SiteKey:      cfg.Turnstile.SiteKey,
			SecretKey:    cfg.Turnstile.SecretKey,
			VerifySecret: cfg.Turnstile.VerifySecret,
			BotToken:     token,
		})
		handler.Register(mux)
		srv := &http.Server{
			Addr:         cfg.App.HTTPAddr,
			Handler:      mux,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
		}
		go func() {
			log.Printf("mini app HTTP server listening on %s", cfg.App.HTTPAddr)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("mini app HTTP server error: %v", err)
			}
		}()
		go func() {
			<-ctx.Done()
			shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(shutCtx)
		}()
	}

	if err := updater.StartPolling(tgBot, &ext.PollingOpts{
		EnableWebhookDeletion: true,
		GetUpdatesOpts: &gotgbot.GetUpdatesOpts{
			Timeout: 30,
			RequestOpts: &gotgbot.RequestOpts{
				Timeout: 45 * time.Second,
			},
			AllowedUpdates: []string{
				"message",
				"callback_query",
				"chat_member",
				"chat_join_request",
				"my_chat_member",
				"poll_answer",
			},
		},
	}); err != nil {
		return fmt.Errorf("start polling: %w", sanitizeTokenError(err, token))
	}
	log.Printf("telegram polling started")

	<-ctx.Done()
	log.Printf("shutdown signal received")

	stopCtx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- updater.Stop()
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("stop updater: %w", sanitizeTokenError(err, token))
		}
	case <-stopCtx.Done():
		return fmt.Errorf("stop updater: %w", stopCtx.Err())
	}

	return nil
}

func sanitizeTokenError(err error, token string) error {
	if err == nil || token == "" {
		return err
	}
	return fmt.Errorf("%s", strings.ReplaceAll(err.Error(), token, "<redacted>"))
}

func registerBotCommands(ctx context.Context, tgBot *gotgbot.Bot) error {
	// 群内只暴露成员可用命令，管理操作统一通过私聊机器人面板完成
	groupCommands := []gotgbot.BotCommand{
		{Command: "start", Description: "开始菜单"},
		{Command: "help", Description: "帮助"},
		{Command: "html", Description: "HTML 格式说明"},
		{Command: "sign", Description: "每日签到"},
		{Command: "points", Description: "查看积分"},
		{Command: "rank", Description: "积分排行榜"},
		{Command: "stat", Description: "今日活跃统计"},
		{Command: "stat_week", Description: "周活跃统计"},
		{Command: "stats", Description: "自定义查询活跃统计"},
		{Command: "lottery", Description: "群组中正在进行的抽奖"},
		{Command: "rules", Description: "查看群规"},
		{Command: "info", Description: "查看会话信息"},
	}
	privateCommands := []gotgbot.BotCommand{
		{Command: "start", Description: "打开管理工作台"},
		{Command: "help", Description: "查看私聊功能说明"},
		{Command: "html", Description: "HTML 格式参考"},
		{Command: "info", Description: "查看会话信息"},
		{Command: "cancel", Description: "取消当前操作"},
	}

	var errs []error
	// 成员和管理员在群内看到的命令相同（管理操作通过私聊面板）
	if _, err := tgBot.SetMyCommandsWithContext(ctx, groupCommands, &gotgbot.SetMyCommandsOpts{Scope: gotgbot.BotCommandScopeAllGroupChats{}}); err != nil {
		errs = append(errs, err)
	}
	if _, err := tgBot.SetMyCommandsWithContext(ctx, groupCommands, &gotgbot.SetMyCommandsOpts{Scope: gotgbot.BotCommandScopeAllChatAdministrators{}}); err != nil {
		errs = append(errs, err)
	}
	if _, err := tgBot.SetMyCommandsWithContext(ctx, privateCommands, &gotgbot.SetMyCommandsOpts{Scope: gotgbot.BotCommandScopeAllPrivateChats{}}); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
