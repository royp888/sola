# Refactor Phase 1：删除 Web API 层

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 删除整个 Web 管理后台（`cmd/api/`、`internal/api/`）及仅供 API 使用的服务，将项目精简为纯 Bot 架构。执行后 bot 功能不受任何影响，只是没有了 HTTP 管理面板。

**Architecture:** `internal/api/types.go` 里的 `ChatBindingRequest`/`ChatBinding` 被 `internal/bot/` 和 `internal/service/` 同时引用，必须先把这两个类型迁移到 `internal/bot/types.go`，再删除 `internal/api/`。其余所有 API 类型（JWT、Admin 登录、各种 Request/Response）是纯 web 专用，直接随包一起删除。

**Tech Stack:** Go 1.25，GORM，现有模块结构

---

## 文件改动清单

| 操作 | 目标 |
|------|------|
| 修改 | `internal/bot/types.go` — 追加 `ChatBindingRequest`、`ChatBinding` 两个结构体 |
| 修改 | `internal/bot/bind.go` — 更新 import，去掉 `internal/api`，直接用 bot 包内的类型 |
| 修改 | `internal/bot/private.go` — 更新 import，`api.ChatBinding` → `bot.ChatBinding` |
| 修改 | `internal/service/api.go` — 更新 import，`api.ChatBindingRequest`/`api.ChatBinding` → `bot.ChatBindingRequest`/`bot.ChatBinding` |
| 修改 | `internal/service/admin_api.go` — 去掉 `internal/api` import，改为 `internal/bot` |
| 修改 | `internal/service/bundle.go` — 删除 `Backup *BackupService` 字段和 `NewBackupService(st)` 行 |
| 删除 | `cmd/api/` 目录（整个） |
| 删除 | `internal/api/` 目录（整个） |
| 删除 | `internal/service/backup.go` |
| 删除 | `internal/service/telegram_auth.go` |

---

### Task 1: 把 ChatBindingRequest / ChatBinding 迁移到 bot 包

**Files:**
- Modify: `internal/bot/types.go`

- [ ] **Step 1: 在 `internal/bot/types.go` 末尾追加两个类型定义**

找到 `internal/bot/types.go` 文件末尾，追加以下代码（不要修改文件中已有的任何内容）：

```go
// ChatBindingRequest 是绑定/更新群组信息时使用的请求类型。
// 迁移自 internal/api/types.go，供 bot 层和 service 层共用。
type ChatBindingRequest struct {
	ChatID              int64
	ChatType            string
	Title               string
	Username            string
	InviteLink          string
	BoundBy             string
	Description         string
	OwnerTelegramUserID int64
	OwnerUsername       string
	OwnerDisplayName    string
}

// ChatBinding 是从数据库读出的群组绑定信息，供 bot 层展示使用。
type ChatBinding struct {
	ChatID      int64
	ChatType    string
	Title       string
	Username    string
	InviteLink  string
	BoundBy     string
	Description string
	OwnerUserID string
	BoundAt     time.Time
}
```

确认 `types.go` 顶部已有 `"time"` import（`BoundAt time.Time` 需要它）。如没有，追加到 import 块里。

- [ ] **Step 2: 检查 `ChatBindings` 接口是否已在 `types.go` 里引用 `api.ChatBinding`**

运行：
```powershell
Select-String -Path "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\internal\bot\types.go" -Pattern "api\."
```

如果有输出，说明接口定义里直接用了 `api.ChatBindingRequest` 或 `api.ChatBinding`，需要把对应接口方法里的类型也改为本包的类型（去掉 `api.` 前缀）。

- [ ] **Step 3: 暂不编译，Task 2 改完 import 再一起验证**

---

### Task 2: 更新 internal/bot/ 的 import

**Files:**
- Modify: `internal/bot/bind.go`
- Modify: `internal/bot/private.go`

- [ ] **Step 1: 检查 bot 层哪些文件 import 了 internal/api**

```powershell
Select-String -Path "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\internal\bot\*.go" -Pattern '"github.com/dabowin/sola/internal/api"'
```

记录所有输出的文件名。

- [ ] **Step 2: 对每个输出文件，执行以下操作**

a. 删除 `"github.com/dabowin/sola/internal/api"` 这行 import

b. 把文件中所有 `api.ChatBindingRequest{` 替换为 `ChatBindingRequest{`

c. 把文件中所有 `api.ChatBinding` 类型引用替换为 `ChatBinding`

d. 如果文件中有其他 `api.Xxx` 引用（非 ChatBinding 相关），先查清楚是什么类型，在下一步处理。

- [ ] **Step 3: 确认 bot 层无残留 api import**

```powershell
Select-String -Path "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\internal\bot\*.go" -Pattern '"github.com/dabowin/sola/internal/api"'
```

预期：无输出。

---

### Task 3: 更新 internal/service/ 的 import

**Files:**
- Modify: `internal/service/api.go`
- Modify: `internal/service/admin_api.go`
- Modify: `internal/service/audit.go`（如有 api import）
- Modify: `internal/service/telegram_auth.go`（准备删除，先确认 bundle.go 不引用它导出的函数）

- [ ] **Step 1: 检查 service 层哪些文件 import 了 internal/api**

```powershell
Select-String -Path "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\internal\service\*.go" -Pattern '"github.com/dabowin/sola/internal/api"'
```

记录所有输出文件。

- [ ] **Step 2: 更新 `internal/service/api.go`**

把文件中：
- `api.ChatBindingRequest` → `bot.ChatBindingRequest`
- `api.ChatBinding` → `bot.ChatBinding`
- 如果有其他 `api.Xxx` 类型且被 bot 层使用，同样移到 bot 包

从 import 块中删除 `"github.com/dabowin/sola/internal/api"`（如果还有其他 api 类型引用，先处理完再删）。

- [ ] **Step 3: 更新 `internal/service/admin_api.go`**

这个文件实现了 `api.ChatAdminService` 接口供 web 用。现在 web 要删了，这个文件整体可以删除（Task 5 处理）。但要先确认 `bundle.go` 里是否有字段引用了这个文件提供的类型。

运行：
```powershell
Select-String -Path "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\internal\service\bundle.go" -Pattern "adminAPI|AdminAPI|NewAPIAdmin|NewAdminAPI"
```

- 如有输出：先在 bundle.go 中删除对应字段和初始化代码，再删 admin_api.go
- 如无输出：直接在 Task 5 删除

- [ ] **Step 4: 检查 `internal/service/telegram_auth.go`**

```powershell
Select-String -Path "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\internal\service\bundle.go" -Pattern "TelegramAuth|telegramAuth|NewTelegramAuth"
```

- 如有输出：在 bundle.go 删除对应字段和初始化，再删文件
- 如无输出：直接在 Task 5 删除

---

### Task 4: 清理 service/bundle.go 里的 API 专用字段

**Files:**
- Modify: `internal/service/bundle.go`

- [ ] **Step 1: 删除 `Backup *BackupService` 字段**

在 `Bundle` 结构体里找到 `Backup *BackupService` 这行，删除。

- [ ] **Step 2: 删除 `NewBackupService(st)` 初始化**

在 `NewBundleWithBotToken` 函数里找到 `Backup: NewBackupService(st),` 这行，删除。

- [ ] **Step 3: 确认 BotServices() 里没有 Backup**

检查 `BotServices()` 函数体，确认 `Backup` 不在返回值里（它本来就不应该在里面）。

---

### Task 5: 删除文件和目录

- [ ] **Step 1: 删除 cmd/api/ 整个目录**

```powershell
Remove-Item -Recurse -Force "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\cmd\api"
```

- [ ] **Step 2: 删除 internal/api/ 整个目录**

```powershell
Remove-Item -Recurse -Force "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\internal\api"
```

- [ ] **Step 3: 删除 API 专用 service 文件**

```powershell
Remove-Item -Force "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\internal\service\backup.go"
Remove-Item -Force "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\internal\service\telegram_auth.go"
Remove-Item -Force "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\internal\service\admin_api.go"
```

---

### Task 6: 编译验证 + 修复残余错误

- [ ] **Step 1: 编译整个项目**

```powershell
cd "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版"
go build ./...
```

预期：只有 `cmd/bot/` 和 `cmd/worker/` 的输出，无报错。

- [ ] **Step 2: 修复可能出现的编译错误**

常见错误及处理方式：

**错误：`undefined: api.SomeType`**
- 说明还有文件引用了未迁移的类型
- 在 `internal/bot/types.go` 或 `internal/service/` 里补充定义或删除引用

**错误：`cannot use bot.ChatBindingRequest as type api.ChatBindingRequest`**
- 说明 service 层的接口和实现类型不一致
- 检查 `internal/bot/types.go` 里 `ChatBindings` 接口的方法签名，确保用的是 `bot.ChatBindingRequest` 而非 `api.ChatBindingRequest`

**错误：`imported and not used: "github.com/dabowin/sola/internal/api"`**
- 找到对应文件，删除该 import 行

- [ ] **Step 3: 再次编译确认干净**

```powershell
go build ./...
```

预期：无输出。

- [ ] **Step 4: 运行现有测试**

```powershell
go test ./...
```

记录失败的测试，修复或在提交说明里注明已知跳过项。

---

### Task 7: 更新配置和部署文件

**Files:**
- Modify: `docker-compose.yml`（如存在）
- Modify: `README.md`

- [ ] **Step 1: 检查 docker-compose.yml**

```powershell
Get-Content "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\tg机器人重构版\docker-compose.yml" -ErrorAction SilentlyContinue
```

删除 `api:` service 段（通常包含 `cmd/api` 的 build 和端口映射），保留 `bot:` 和 `worker:`（worker 在 Phase 2 再处理）。

- [ ] **Step 2: 更新 .env.example**

检查是否有 `HTTP_ADDR`、`ADMIN_PASSWORD`、`ADMIN_PASSWORD_HASH`、`JWT_*` 等仅 API 使用的变量，注释掉或删除。

---

### Task 8: 安全扫描 + 提交

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

- [ ] **Step 2: 提交**

```powershell
git add -A
git commit -m "refactor: 删除 Web API 层，迁移 ChatBinding 类型到 bot 包

- 删除 cmd/api/、internal/api/ 整个包
- 删除 service/backup.go、telegram_auth.go、admin_api.go（API 专用）
- 将 ChatBindingRequest/ChatBinding 迁移到 internal/bot/types.go
- 清理 service/bundle.go 中的 Backup 字段
项目精简为纯 Bot 架构，无 HTTP 管理面板"
```
