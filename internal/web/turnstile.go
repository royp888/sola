package web

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const cfSiteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// TurnstileConfig holds credentials needed for Turnstile verification.
type TurnstileConfig struct {
	SiteKey      string // public, shown in browser widget
	SecretKey    string // used to verify CF tokens server-side
	VerifySecret string // HMAC secret for signing Mini App URLs (must match bot/verify.go)
	BotToken     string // used to approve/decline join requests via Telegram API
}

// TurnstileHandler serves the Turnstile Mini App page and verification API.
type TurnstileHandler struct {
	cfg TurnstileConfig
}

// NewTurnstileHandler creates a TurnstileHandler.
func NewTurnstileHandler(cfg TurnstileConfig) *TurnstileHandler {
	return &TurnstileHandler{cfg: cfg}
}

// Register wires routes onto mux.
func (h *TurnstileHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/verify/turnstile/config", h.handleConfig)
	mux.HandleFunc("/api/verify/turnstile", h.handleVerify)
	mux.HandleFunc("/", h.handlePage)
}

// handleConfig returns the public Turnstile site key for the Mini App.
func (h *TurnstileHandler) handleConfig(w http.ResponseWriter, r *http.Request) {
	if h.cfg.SiteKey == "" {
		jsonResp(w, http.StatusServiceUnavailable, map[string]any{"error": "turnstile not configured"})
		return
	}
	jsonResp(w, http.StatusOK, map[string]any{"site_key": h.cfg.SiteKey})
}

// handleVerify receives the Turnstile token from the Mini App after the user passes.
func (h *TurnstileHandler) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ChatID  int64  `json:"chat_id"`
		UserID  int64  `json:"user_id"`
		Sig     string `json:"sig"`
		Exp     int64  `json:"exp"`
		CFToken string `json:"cf_token"`
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
	if err := json.Unmarshal(body, &req); err != nil {
		jsonResp(w, http.StatusBadRequest, map[string]any{"ok": false, "message": "invalid request body"})
		return
	}

	if h.cfg.VerifySecret == "" {
		jsonResp(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "message": "turnstile not configured"})
		return
	}

	if time.Now().Unix() > req.Exp {
		jsonResp(w, http.StatusUnprocessableEntity, map[string]any{"ok": false, "message": "verification link expired"})
		return
	}

	// Verify HMAC — must match turnstileBotHMAC in internal/bot/verify.go
	expected := computeHMAC(h.cfg.VerifySecret, req.ChatID, req.UserID, req.Exp)
	if !hmac.Equal([]byte(req.Sig), []byte(expected)) {
		jsonResp(w, http.StatusForbidden, map[string]any{"ok": false, "message": "invalid signature"})
		return
	}

	if h.cfg.SecretKey == "" {
		jsonResp(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "message": "turnstile secret key not configured"})
		return
	}
	cfOK, err := verifyCFToken(r.Context(), h.cfg.SecretKey, req.CFToken)
	if err != nil {
		log.Printf("turnstile: CF verify error: %v", err)
		jsonResp(w, http.StatusBadGateway, map[string]any{"ok": false, "message": "failed to verify captcha"})
		return
	}

	if !cfOK {
		if h.cfg.BotToken != "" {
			_ = telegramJoinAction(r.Context(), h.cfg.BotToken, req.ChatID, req.UserID, false)
		}
		jsonResp(w, http.StatusOK, map[string]any{"ok": false, "message": "captcha verification failed"})
		return
	}

	if h.cfg.BotToken != "" {
		if err := telegramJoinAction(r.Context(), h.cfg.BotToken, req.ChatID, req.UserID, true); err != nil {
			log.Printf("turnstile: approve join request failed: %v", err)
			jsonResp(w, http.StatusBadGateway, map[string]any{"ok": false, "message": "failed to approve join request"})
			return
		}
	}
	jsonResp(w, http.StatusOK, map[string]any{"ok": true, "message": "verified"})
}

// handlePage serves the standalone Turnstile verification Mini App HTML page.
func (h *TurnstileHandler) handlePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Allow Telegram clients (web.telegram.org, t.me, desktop) to embed this page as a Mini App.
	w.Header().Set("Content-Security-Policy", "frame-ancestors 'self' https://web.telegram.org https://t.me https://*.telegram.org")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, verifyPageHTML)
}

// computeHMAC mirrors turnstileBotHMAC in internal/bot/verify.go.
// Message format: "chatID|userID|exp"
func computeHMAC(secret string, chatID, userID, exp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d|%d|%d", chatID, userID, exp)
	return hex.EncodeToString(mac.Sum(nil))
}

func verifyCFToken(ctx context.Context, secret, token string) (bool, error) {
	form := url.Values{"secret": {secret}, "response": {token}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfSiteverifyURL,
		strings.NewReader(form.Encode()))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return false, err
	}
	return result.Success, nil
}

func telegramJoinAction(ctx context.Context, botToken string, chatID, userID int64, approve bool) error {
	method := "approveChatJoinRequest"
	if !approve {
		method = "declineChatJoinRequest"
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/%s", botToken, method)
	body, _ := json.Marshal(map[string]any{"chat_id": chatID, "user_id": userID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("telegram %s: %d %s", method, resp.StatusCode, b)
	}
	return nil
}

func jsonResp(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// verifyPageHTML is a standalone Turnstile verification page served to Mini App users.
// It reads params from the URL hash fragment (#/verify?chat=X&user=Y&sig=Z&exp=T),
// fetches the site key from /api/verify/turnstile/config, and renders the CF widget.
const verifyPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1, maximum-scale=1, user-scalable=no">
  <meta name="color-scheme" content="light dark">
  <title>入群验证</title>
  <script src="https://telegram.org/js/telegram-web-app.js"></script>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px 16px;
      background: var(--tg-theme-bg-color, #ffffff);
      color: var(--tg-theme-text-color, #000000);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    .card {
      text-align: center;
      max-width: 380px;
      width: 100%;
    }
    .icon { font-size: 3rem; display: block; margin-bottom: 12px; }
    h2 { font-size: 1.2rem; margin-bottom: 8px; }
    p  { color: var(--tg-theme-hint-color, #6b7280); margin-bottom: 16px; font-size: 0.9rem; }
    #cf-widget { display: flex; justify-content: center; margin: 16px 0; min-height: 65px; }
    .spinner {
      width: 36px; height: 36px;
      border: 3px solid var(--tg-theme-hint-color, #d1d5db);
      border-top-color: var(--tg-theme-button-color, #2563eb);
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
      margin: 0 auto 16px;
    }
    @keyframes spin { to { transform: rotate(360deg); } }
    .success h2 { color: #16a34a; }
    .error-text  { color: #dc2626; font-size: 0.9rem; }
    .retry-btn {
      margin-top: 12px; padding: 8px 24px;
      background: var(--tg-theme-button-color, #2563eb);
      color: var(--tg-theme-button-text-color, #ffffff);
      border: none; border-radius: 8px;
      font-size: 0.95rem; cursor: pointer;
    }
    .retry-btn:hover { opacity: 0.9; }
  </style>
</head>
<body>
<div class="card" id="root">
  <div class="spinner"></div>
  <p>正在加载验证组件…</p>
</div>

<script>
(function () {
  'use strict';

  function getParams() {
    var hash = window.location.hash; // "#/verify?chat=X&user=Y&sig=Z&exp=T"
    var qi = hash.indexOf('?');
    if (qi === -1) return null;
    var p = new URLSearchParams(hash.slice(qi + 1));
    var chat = p.get('chat'), user = p.get('user'), sig = p.get('sig'), exp = p.get('exp');
    if (!chat || !user || !sig || !exp) return null;
    if (Date.now() / 1000 > Number(exp)) return null;
    return { chatId: Number(chat), userId: Number(user), sig: sig, exp: Number(exp) };
  }

  function render(html) {
    document.getElementById('root').innerHTML = html;
  }

  function showError(msg) {
    render('<span class="icon">&#10060;</span><h2>验证失败</h2>' +
           '<p class="error-text">' + (msg || '请重试或联系管理员。') + '</p>' +
           '<button class="retry-btn" onclick="location.reload()">重试</button>');
  }

  function showSuccess() {
    render('<div class="success"><span class="icon">&#9989;</span>' +
           '<h2>验证通过</h2>' +
           '<p>已批准您的入群申请，请返回 Telegram 群组。</p></div>');
    if (window.Telegram && window.Telegram.WebApp) {
      setTimeout(function() { window.Telegram.WebApp.close(); }, 1500);
    }
  }

  function onToken(token, params) {
    fetch('/api/verify/turnstile', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        chat_id: params.chatId,
        user_id: params.userId,
        sig: params.sig,
        exp: params.exp,
        cf_token: token
      })
    })
    .then(function(res) { return res.json(); })
    .then(function(data) {
      if (data.ok) {
        showSuccess();
      } else {
        showError(data.message || '验证失败，请重新申请入群。');
      }
    })
    .catch(function() { showError('网络错误，请稍后重试。'); });
  }

  function loadWidget(siteKey, params) {
    render('<h2>入群验证</h2><p>请完成下方人机验证以加入群组。</p>' +
           '<div id="cf-widget"></div>');
    var script = document.createElement('script');
    script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';
    script.async = true;
    script.defer = true;
    script.onload = function() {
      window.turnstile.render('#cf-widget', {
        sitekey: siteKey,
        callback: function(token) { onToken(token, params); },
        'error-callback': function() { showError('验证组件出错，请刷新重试。'); },
        'expired-callback': function() { window.turnstile && window.turnstile.reset(); }
      });
    };
    script.onerror = function() { showError('验证脚本加载失败，请检查网络。'); };
    document.head.appendChild(script);
  }

  function init() {
    var params = getParams();
    if (!params) {
      showError('链接无效或已过期，请重新向机器人申请入群。');
      return;
    }
    fetch('/api/verify/turnstile/config')
      .then(function(res) { return res.ok ? res.json() : Promise.reject(); })
      .then(function(data) {
        if (data.site_key) {
          loadWidget(data.site_key, params);
        } else {
          showError('验证服务未配置，请联系管理员。');
        }
      })
      .catch(function() { showError('无法连接验证服务，请稍后重试。'); });
  }

  init();
})();
</script>
</body>
</html>`
