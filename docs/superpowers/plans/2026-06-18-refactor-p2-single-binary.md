# Refactor Phase 2：合并为单 Binary

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Prerequisite:** Phase 1（删除 Web API 层）必须已完成且编译通过。

**Goal:** 把 `cmd/worker/` 合并进 `cmd/bot/`，Worker 作为 goroutine 在 Bot 进程内运行，最终只有一个 binary、一个 Docker 容器，部署和维护大幅简化。同时把 Redis 改为可选（单人运营不需要强依赖）。

**Architecture:** `worker.New(cfg, store, logger).Run(ctx)` 现在在独立进程里阻塞运行，只需在 `cmd/bot/main.go` 里起一个 `go runner.Run(ctx)` goroutine 即可，worker 的错误通过 channel 上报但不阻止 bot 继续运行。

**Tech Stack:** Go 1.25，gotgbot/v2，现有 worker 包

---

## 文件改动清单

| 操作 | 目标 |
|------|------|
| 修改 | `cmd/bot/main.go` — 嵌入 worker goroutine |
| 删除 | `cmd/worker/` 整个目录 |
| 修改 | `internal/bootstrap/bootstrap.go` — Redis 变可选，不再 Fatal |
| 修改 | `docker-compose.yml` — 删除 worker service |

---

### Task 1: 在 cmd/bot/main.go 里嵌入 worker

**Files:**
- Modify: `cmd/bot/main.go`

- [ ] **Step 1: 查看当前 cmd/bot/main.go 内容**

```powershell
Get-Content "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\cmd\bot\main.go"
```

找到 `bootstrap.New(ctx, "")` 之后创建 bot/dispatcher 的位置，以及最后 `updater.StartPolling` 或 `updater.StartWebhook` 的调用。

- [ ] **Step 2: 在 import 块加入 worker 包**

在 `cmd/bot/main.go` 的 import 块里，追加：

```go
"github.com/dabowin/sola/internal/worker"
```

- [ ] **Step 3: 在 bot 启动后、阻塞等待前插入 worker goroutine**

找到类似 `updater.StartPolling(...)` 或 `updater.Idle()` 的行（bot 开始接收消息的地方），在它**之前**插入：

```go
// 在 bot 进程内启动 worker（定时发帖、任务调度）
go func() {
    runner := worker.New(resources.Config, resources.Store, resources.Logger)
    if err := runner.Run(ctx); err != nil && err != context.Canceled {
        resources.Logger.Error("worker exited with error", zap.Error(err))
    }
}()
```

确认顶部 import 已有 `"go.uber.org/zap"`（通常已有）。

完整的 main.go 关键部分示例（只示范新增的那段，其余不变）：

```go
func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    resources, err := bootstrap.New(ctx, "")
    if err != nil {
        log.Fatal(err)
    }
    defer resources.Close(context.Background())

    // ... 初始化 bot、dispatcher、updater ...

    // 嵌入 worker goroutine
    go func() {
        runner := worker.New(resources.Config, resources.Store, resources.Logger)
        if err := runner.Run(ctx); err != nil && err != context.Canceled {
            resources.Logger.Error("worker exited with error", zap.Error(err))
        }
    }()

    // bot 阻塞运行（StartPolling 或 Idle）
    _ = updater.StartPolling(b, &ext.PollingOpts{...})
    updater.Idle()
}
```

- [ ] **Step 4: 编译验证**

```powershell
cd "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版"
go build ./cmd/bot/...
```

预期：无输出。

- [ ] **Step 5: 提交**

```powershell
git add cmd/bot/main.go
git commit -m "feat(bot): 嵌入 worker goroutine，两进程合并为一"
```

---

### Task 2: 删除 cmd/worker/

- [ ] **Step 1: 删除目录**

```powershell
Remove-Item -Recurse -Force "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\cmd\worker"
```

- [ ] **Step 2: 编译整个项目**

```powershell
go build ./...
```

预期：无输出（`cmd/worker` 消失，`cmd/bot` 依然编译通过）。

- [ ] **Step 3: 提交**

```powershell
git add -A
git commit -m "chore: 删除 cmd/worker，统一由 cmd/bot 运行"
```

---

### Task 3: Redis 变可选

当前 `bootstrap.go` 在 Redis ping 失败时只打 Warn 然后把 `rdb` 置 nil，实际上已经是"可选"的了。但 `config.go` 里可能把 `Redis.Addr` 标为必填。

- [ ] **Step 1: 检查 config 里 Redis 是否必填**

```powershell
Select-String -Path "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\internal\config\config.go" -Pattern "Redis|redis" -Context 0,2
```

如果 Redis 字段有 `validate:"required"` 或类似标签，把它改为可选（删除 required 标签，或改为 `validate:"omitempty"`）。

- [ ] **Step 2: 检查 .env.example**

确认 `REDIS_ADDR` 注释掉或标为"可选"，让单人部署不填 Redis 时 bot 也能启动。

- [ ] **Step 3: 编译 + 测试**

```powershell
go build ./...
go test ./...
```

- [ ] **Step 4: 提交**

```powershell
git add internal/config/config.go
git commit -m "chore: Redis 改为可选，单人部署不需要 Redis"
```

---

### Task 4: 更新 docker-compose.yml

- [ ] **Step 1: 打开文件，找到 worker service 段**

```powershell
Get-Content "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\docker-compose.yml"
```

- [ ] **Step 2: 删除 worker service**

找到类似以下内容并删除：

```yaml
  worker:
    build: .
    command: /app/worker
    env_file: .env
    restart: unless-stopped
    depends_on:
      - postgres
```

- [ ] **Step 3: 确认 bot service 的 command 是 /app/bot 而非 /app/api**

```yaml
  bot:
    build: .
    command: /app/bot   # ← 确认这里
    env_file: .env
    restart: unless-stopped
```

- [ ] **Step 4: 提交**

```powershell
git add docker-compose.yml
git commit -m "chore: docker-compose 删除 worker service，部署只需要 bot"
```

---

### Task 5: 安全扫描 + 打 tag

- [ ] **Step 1: 安全扫描**

```powershell
$patterns = @('\d{8,10}:[A-Za-z0-9_-]{35}', 'BOT_TOKEN\s*=\s*\S+', 'password\s*=\s*\S{6,}')
Get-ChildItem -Path "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版" -Recurse -File |
  Where-Object { $_.Extension -match '\.(go|md|yaml|yml|env|json|txt)$' -and $_.FullName -notmatch '(node_modules|\.git|vendor)' } |
  ForEach-Object {
    $content = Get-Content $_.FullName -Raw -ErrorAction SilentlyContinue
    foreach ($p in $patterns) {
      if ($content -match $p) { Write-Host "FOUND: $($_.FullName)" }
    }
  }
```

预期：无输出。

- [ ] **Step 2: 推送**

```powershell
git push origin main
```

- [ ] **Step 3: 打 tag**

```powershell
git tag v2.0.0
git push origin v2.0.0
```

（大版本号 v2，因为这是破坏性重构：删除了 web 管理面板，部署方式变更）

- [ ] **Step 4: GitHub Release**

```powershell
gh release create v2.0.0 --title "v2.0.0 — 重构为纯 Bot 架构" --notes "## 重大变更（Breaking Changes）

- **删除 Web 管理面板**：不再提供 HTTP 管理 API 和 Vue 前端，所有配置改为通过 Bot 命令完成
- **合并为单进程**：bot 和 worker 合并为一个 binary（\`/app/bot\`），部署只需一个容器
- **Redis 变为可选**：不配置 Redis 时 Bot 仍可正常运行（仅限速功能降级）

## 迁移指南

如果你在用 Docker Compose：
1. 删除 docker-compose.yml 里的 \`api:\` 和 \`worker:\` service
2. 只保留 \`bot:\` service
3. 可以删除 \`.env\` 里的 \`HTTP_ADDR\`、\`ADMIN_PASSWORD\`、\`JWT_*\` 等变量"
```

- [ ] **Step 5: 更新 README.md / README.en.md**

在更新日志最顶部插入：

```
- **2026-06-18** v2.0.0 — 重构为纯 Bot 架构：删除 Web 管理面板，合并 bot+worker 为单进程，Redis 变为可选
```

```
- **2026-06-18** v2.0.0 — Refactor to pure-bot architecture: remove web admin panel, merge bot+worker into single binary, Redis is now optional
```

```powershell
git add README.md README.en.md
git commit -m "docs: 更新 v2.0.0 更新日志"
git push origin main
```
