# sola-bot 前端缺失功能补全计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在管理后台前端补全 5 个后端已实现但前端无入口的功能：群组解绑、禁言操作、统计时间范围筛选、Bot 全局配置、群组详情增加解绑入口。

**Architecture:** 纯前端修改，不涉及后端代码。每个 Task 仅修改对应的 `.vue` 文件或新增 API 客户端函数。已有后端 API，只需对接。

**Tech Stack:** Vue 3, TypeScript, Element Plus, Axios（通过项目内 `@/api/http` 封装）

**项目路径:** `C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot`

**实施前必读：**
1. 先读 `web/src/api/http.ts` 了解 `request()` 函数的调用方式
2. 先读 `web/src/api/admin.ts` 了解 `createBan`/`createMute` 等已有函数的调用模式
3. 先读 `web/src/types/api.ts` 了解已有类型定义

---

## 文件修改清单

| 文件 | 操作 | 任务 |
|------|------|------|
| `web/src/api/chats.ts` | 新增函数 | Task 1 |
| `web/src/views/ChatsView.vue` | 修改（加解绑按钮） | Task 1 |
| `web/src/api/admin.ts` | 确认/新增 mute 函数 | Task 2 |
| `web/src/views/BansView.vue` | 修改（加禁言区块） | Task 2 |
| `web/src/views/StatsView.vue` | 修改（加日期范围选择器） | Task 3 |
| `web/src/api/bots.ts` 或 `web/src/api/botConfig.ts` | 新增文件 | Task 4 |
| `web/src/views/BotsView.vue` | 修改（加配置表单） | Task 4 |
| `web/src/router/index.ts` | 可能需要确认路由 | Task 4 |

---

## Task 1：群组解绑功能

**背景：** 后端已有 `DELETE /api/v1/chats/:chat_id/bind`（见 `internal/api/router.go`），但 `ChatsView.vue` 的群组操作列和详情抽屉中没有解绑按钮，已绑定的群组无法在前端删除。

**API 调用：**
```
DELETE /api/v1/chats/{chat_id}/bind
成功：HTTP 200（无 body 或 {ok: true}）
```

**Files:**
- Modify: `web/src/api/chats.ts`
- Modify: `web/src/views/ChatsView.vue`

- [ ] **Step 1: 在 chats.ts 中新增 unbindChat 函数**

读取 `web/src/api/chats.ts` 确认现有函数结构，然后在文件末尾添加：
```typescript
export function unbindChat(chatId: number | string): Promise<void> {
  return request<void>(`/chats/${chatId}/bind`, { method: "DELETE" });
}
```

- [ ] **Step 2: 在 ChatsView.vue 中导入 unbindChat 并添加 unbind 函数**

读取 `web/src/views/ChatsView.vue`，在 import 行找到：
```typescript
import { bindChat, fetchChats } from "@/api/chats";
```
修改为：
```typescript
import { bindChat, fetchChats, unbindChat } from "@/api/chats";
```

在 `<script setup>` 中，在 `submitBind` 函数后添加：
```typescript
const unbindingId = ref<string | number>();

async function submitUnbind(chat: ChatRecord): Promise<void> {
  try {
    await ElMessageBox.confirm(
      `确认解绑群组「${chat.title}」？解绑后该群组的所有配置数据将保留，但机器人将停止在该群响应。`,
      "确认解绑",
      {
        type: "warning",
        confirmButtonText: "确认解绑",
        cancelButtonText: "取消",
      },
    );
  } catch {
    return;
  }
  const chatId = chat.chat_id ?? chat.id;
  unbindingId.value = chatId;
  try {
    await unbindChat(chatId);
    ElMessage.success("群组已解绑");
    detailVisible.value = false;
    await loadChats();
  } catch (error) {
    ElMessage.error(errorMessage(error));
  } finally {
    unbindingId.value = undefined;
  }
}
```

确认 `ElMessageBox` 已在 import 中，如没有则添加：
```typescript
import { ElMessage, ElMessageBox } from "element-plus";
```

- [ ] **Step 3: 在操作列中添加解绑按钮**

找到表格操作列（约第 84 行）：
```html
<el-table-column label="操作" width="250" fixed="right">
  <template #default="{ row }">
    <el-button size="small" type="primary" @click="goUsers(row)">进入成员台</el-button>
    <el-button size="small" @click="goConfig(row)">配置</el-button>
    <el-button size="small" @click="goLogs(row)">日志</el-button>
  </template>
</el-table-column>
```
修改为（宽度从 250 改为 320，加解绑按钮）：
```html
<el-table-column label="操作" width="320" fixed="right">
  <template #default="{ row }">
    <el-button size="small" type="primary" @click="goUsers(row)">进入成员台</el-button>
    <el-button size="small" @click="goConfig(row)">配置</el-button>
    <el-button size="small" @click="goLogs(row)">日志</el-button>
    <el-button size="small" type="danger" :loading="unbindingId === (row.chat_id ?? row.id)" @click="submitUnbind(row)">解绑</el-button>
  </template>
</el-table-column>
```

- [ ] **Step 4: 在群组详情抽屉中也添加解绑按钮**

找到 `detail-actions` 区块（约第 119 行）：
```html
<div class="detail-actions">
  <el-button type="primary" @click="goUsers(currentChat)">进入成员台</el-button>
  <el-button @click="goConfig(currentChat)">群组设置</el-button>
  <el-button @click="goLogs(currentChat)">积分日志</el-button>
</div>
```
修改为：
```html
<div class="detail-actions">
  <el-button type="primary" @click="goUsers(currentChat)">进入成员台</el-button>
  <el-button @click="goConfig(currentChat)">群组设置</el-button>
  <el-button @click="goLogs(currentChat)">积分日志</el-button>
  <el-button type="danger" :loading="unbindingId === (currentChat.chat_id ?? currentChat.id)" @click="submitUnbind(currentChat)">解绑群组</el-button>
</div>
```

- [ ] **Step 5: 手动测试**

1. 打开群组会话页，确认每行操作列末尾有"解绑"按钮（红色）
2. 点击解绑，确认弹出确认对话框
3. 点取消，确认群组未被解绑
4. 点击任意群组名进入详情抽屉，确认底部有"解绑群组"按钮

- [ ] **Step 6: Commit**

```powershell
git add web/src/api/chats.ts web/src/views/ChatsView.vue
git commit -m "feat(web): add unbind chat button to ChatsView table and detail drawer"
```

---

## Task 2：BansView 添加禁言 / 解除禁言操作

**背景：** 后端有 `POST /api/admin/mute` 和 `POST /api/admin/unmute`（见 `internal/api/router.go`），`UsersView.vue` 中已有禁言操作，但 `BansView.vue` 没有禁言入口。在 BansView 的"后台操作"区块补充禁言表单。

**先读 `web/src/api/admin.ts` 确认 `createMute`/`createUnmute` 函数是否存在，以及入参格式。**

**Files:**
- Modify: `web/src/views/BansView.vue`

- [ ] **Step 1: 读取 admin.ts 确认 mute 函数**

```powershell
Select-String -Path "web\src\api\admin.ts" -Pattern "mute|Mute" -Context 3
```

如果 `createMute` 已存在，记录其签名。如果不存在，在 `admin.ts` 末尾添加：
```typescript
export function createMute(payload: {
  chat_id: string | number;
  user_id: string | number;
  duration_seconds?: number;
  reason?: string;
}): Promise<void> {
  return request<void>("/admin/mute", { method: "POST", body: payload });
}

export function createUnmute(payload: {
  chat_id: string | number;
  user_id: string | number;
}): Promise<void> {
  return request<void>("/admin/unmute", { method: "POST", body: payload });
}
```

- [ ] **Step 2: 在 BansView.vue 导入 mute 函数**

找到 import 行：
```typescript
import { createBan, deleteBan, fetchBans, fetchWarns } from "@/api/admin";
```
修改为（加入 createMute, createUnmute）：
```typescript
import { createBan, createMute, createUnmute, deleteBan, fetchBans, fetchWarns } from "@/api/admin";
```

- [ ] **Step 3: 在 script setup 中添加禁言表单状态**

在 `banForm` reactive 定义后面添加：
```typescript
const muteForm = reactive({ user_id: "", duration_minutes: 60, reason: "" });
const muteSaving = ref(false);
```

- [ ] **Step 4: 添加 submitMute 和 submitUnmute 函数**

在 `submitBan` 函数后面添加：
```typescript
async function submitMute(): Promise<void> {
  if (!selectedChatId.value || !muteForm.user_id) {
    ElMessage.warning("请先选择群组和成员");
    return;
  }
  try {
    await ElMessageBox.confirm(
      `确认禁言用户 ${muteForm.user_id} ${muteForm.duration_minutes} 分钟？`,
      "确认禁言",
      { type: "warning", confirmButtonText: "确认禁言", cancelButtonText: "取消" },
    );
  } catch {
    return;
  }
  muteSaving.value = true;
  try {
    await createMute({
      chat_id: selectedChatId.value,
      user_id: muteForm.user_id,
      duration_seconds: muteForm.duration_minutes * 60,
      reason: muteForm.reason,
    });
    ElMessage.success("禁言已生效");
    muteForm.user_id = "";
    muteForm.reason = "";
  } catch {
    ElMessage.error("服务暂时不可用");
  } finally {
    muteSaving.value = false;
  }
}
```

- [ ] **Step 5: 在模板中添加禁言表单区块**

在"后台操作" `PanelSection` 的"提交封禁"按钮之后，添加一个禁言表单区块（在同一个 PanelSection 内，用分割线分隔）：

找到：
```html
<el-button type="danger" :loading="saving" @click="submitBan">提交封禁</el-button>
```
在其后添加：
```html
<el-divider />
<el-form-item label="禁言成员">
  <UserSelect v-model="muteForm.user_id" :chat-id="selectedChatId" />
</el-form-item>
<el-form-item label="禁言时长（分钟）">
  <el-input-number v-model="muteForm.duration_minutes" :min="1" :max="43200" controls-position="right" style="width:100%" />
</el-form-item>
<el-form-item label="原因">
  <el-input v-model="muteForm.reason" type="textarea" :rows="2" />
</el-form-item>
<el-button type="warning" :loading="muteSaving" @click="submitMute">提交禁言</el-button>
```

- [ ] **Step 6: 手动测试**

1. 打开封禁与警告页，选择群组
2. 在"后台操作"面板中应看到分割线下方的禁言区块
3. 选择用户、填写时长，点"提交禁言"，应弹出确认框
4. 确认后应显示"禁言已生效"

- [ ] **Step 7: Commit**

```powershell
git add web/src/api/admin.ts web/src/views/BansView.vue
git commit -m "feat(web): add mute/unmute form to BansView"
```

---

## Task 3：统计页添加自定义日期范围选择

**背景：** `StatsView.vue` 已有"近 7/30/90 天"预设范围选择（`range` ref），`fetchStats(range)` 内部转为 `from`/`to` 日期后请求 API。但没有自定义任意区间的能力。

**修改方案：** 在现有 `el-select` 末尾加一个"自定义"选项，选中时显示 `el-date-picker`。`stats.ts` 的 `fetchStats` 函数改为支持传入自定义日期字符串（`custom:YYYY-MM-DD:YYYY-MM-DD` 格式）或沿用预设字符串（向后兼容）。

**Files:**
- Modify: `web/src/api/stats.ts`
- Modify: `web/src/views/StatsView.vue`

- [ ] **Step 1: 修改 stats.ts 的 fetchStats 和 resolveRange 函数支持自定义范围**

找到（第 30 行）：
```typescript
export function fetchStats(range: string): Promise<StatsOverview> {
  const { from, to } = resolveRange(range);
```
改为：
```typescript
export function fetchStats(range: string, customFrom?: string, customTo?: string): Promise<StatsOverview> {
  const { from, to } = range === "custom" && customFrom && customTo
    ? { from: customFrom, to: customTo }
    : resolveRange(range);
```

- [ ] **Step 2: 修改 StatsView.vue 的 range ref 和 loadStats 调用**

找到（约第 96 行）：
```typescript
const range = ref("7d");
```
在其后添加：
```typescript
const customRange = ref<[string, string] | null>(null);
```

找到 `loadStats` 函数（调用 `fetchStats(range.value)` 的地方），修改为：
```typescript
const response = await fetchStats(
  range.value,
  customRange.value?.[0],
  customRange.value?.[1],
);
```

- [ ] **Step 3: 在模板中给范围选择器添加"自定义"选项，并条件显示日期选择器**

找到（第 9-13 行）：
```html
<el-select v-model="range" class="select" @change="loadStats">
  <el-option label="近 7 天" value="7d" />
  <el-option label="近 30 天" value="30d" />
  <el-option label="近 90 天" value="90d" />
</el-select>
```
修改为：
```html
<el-select v-model="range" class="select" @change="() => { customRange = null; loadStats(); }">
  <el-option label="近 7 天" value="7d" />
  <el-option label="近 30 天" value="30d" />
  <el-option label="近 90 天" value="90d" />
  <el-option label="自定义" value="custom" />
</el-select>
<el-date-picker
  v-if="range === 'custom'"
  v-model="customRange"
  type="daterange"
  range-separator="至"
  start-placeholder="开始日期"
  end-placeholder="结束日期"
  value-format="YYYY-MM-DD"
  @change="loadStats"
/>
```

- [ ] **Step 4: 手动测试**

1. 打开数据分析页，确认下拉有"自定义"选项
2. 选"自定义"后，旁边出现日期区间选择器
3. 选好日期后，图表和统计数字重新加载
4. 切回"近 7 天"，日期选择器消失，数据正常刷新

- [ ] **Step 5: Commit**

```powershell
git add web/src/api/stats.ts web/src/views/StatsView.vue
git commit -m "feat(web): add custom date range picker to StatsView"
```

---

## Task 4：BotsView 添加 Bot 全局配置表单

**背景：** 后端有 `GET /api/v1/bot/config` 和 `PUT /api/v1/bot/config`，支持 8 个字段：`enabled`（bool）、`default_language`（string）、`time_zone`（string）、`auto_delete_enabled`（bool）、`auto_delete_after_secs`（int）、`allow_forwarded_posts`（bool）、`enable_stats_tracking`（bool）、`enable_points`（bool）。

**先读 `web/src/views/BotsView.vue` 了解现有结构，然后在同一页面添加全局配置表单 PanelSection。**

**Files:**
- Create: `web/src/api/botConfig.ts`
- Modify: `web/src/views/BotsView.vue`

- [ ] **Step 1: 读取 BotsView.vue 现有结构**

```powershell
Get-Content "web\src\views\BotsView.vue" | Select-Object -First 60
```

- [ ] **Step 2: 创建 botConfig.ts API 客户端**

新建文件 `web/src/api/botConfig.ts`：
```typescript
import { request } from "@/api/http";

export interface BotConfig {
  enabled: boolean;
  default_language: string;
  time_zone: string;
  auto_delete_enabled: boolean;
  auto_delete_after_secs: number;
  allow_forwarded_posts: boolean;
  enable_stats_tracking: boolean;
  enable_points: boolean;
}

export function fetchBotConfig(): Promise<BotConfig> {
  return request<BotConfig>("/bot/config");
}

export function updateBotConfig(payload: Partial<BotConfig>): Promise<BotConfig> {
  return request<BotConfig>("/bot/config", { method: "PUT", body: payload });
}
```

- [ ] **Step 3: 在 BotsView.vue 中导入并添加配置表单状态**

在 `<script setup>` 中导入：
```typescript
import { fetchBotConfig, updateBotConfig } from "@/api/botConfig";
import type { BotConfig } from "@/api/botConfig";
```

添加 ref 和 reactive：
```typescript
const configLoading = ref(false);
const configSaving = ref(false);
const config = reactive<BotConfig>({
  enabled: true,
  default_language: "zh",
  time_zone: "Asia/Shanghai",
  auto_delete_enabled: false,
  auto_delete_after_secs: 0,
  allow_forwarded_posts: true,
  enable_stats_tracking: true,
  enable_points: true,
});

async function loadBotConfig(): Promise<void> {
  configLoading.value = true;
  try {
    const result = await fetchBotConfig();
    Object.assign(config, result);
  } catch {
    ElMessage.error("获取 Bot 配置失败");
  } finally {
    configLoading.value = false;
  }
}

async function saveBotConfig(): Promise<void> {
  configSaving.value = true;
  try {
    const result = await updateBotConfig({ ...config });
    Object.assign(config, result);
    ElMessage.success("Bot 全局配置已保存");
  } catch {
    ElMessage.error("保存失败，请重试");
  } finally {
    configSaving.value = false;
  }
}
```

在 `onMounted` 中调用（或在现有 onMounted 后追加）：
```typescript
onMounted(loadBotConfig);
```

- [ ] **Step 4: 在模板中添加 Bot 全局配置 PanelSection**

在现有机器人列表 PanelSection 之后添加：
```html
<PanelSection title="Bot 全局配置" description="接口：GET/PUT /api/v1/bot/config。修改后立即生效。">
  <template #actions>
    <el-button :icon="Refresh" :loading="configLoading" @click="loadBotConfig">刷新</el-button>
    <el-button type="primary" :loading="configSaving" @click="saveBotConfig">保存</el-button>
  </template>
  <el-form v-loading="configLoading" label-position="top" class="config-form">
    <div class="switch-row">
      <div>
        <strong>Bot 总开关</strong>
        <span>关闭后机器人停止响应所有群组消息</span>
      </div>
      <el-switch v-model="config.enabled" />
    </div>
    <div class="switch-row">
      <div>
        <strong>积分系统</strong>
        <span>全局开关，关闭后所有群组积分功能停用</span>
      </div>
      <el-switch v-model="config.enable_points" />
    </div>
    <div class="switch-row">
      <div>
        <strong>统计追踪</strong>
        <span>开启后记录消息量、活跃度等统计数据</span>
      </div>
      <el-switch v-model="config.enable_stats_tracking" />
    </div>
    <div class="switch-row">
      <div>
        <strong>允许转发消息计分</strong>
        <span>关闭后转发的消息不计入积分</span>
      </div>
      <el-switch v-model="config.allow_forwarded_posts" />
    </div>
    <div class="switch-row">
      <div>
        <strong>自动删除消息</strong>
        <span>开启后 Bot 发送的消息会在指定时间后自动删除</span>
      </div>
      <el-switch v-model="config.auto_delete_enabled" />
    </div>
    <el-form-item v-if="config.auto_delete_enabled" label="自动删除延迟（秒）">
      <el-input-number v-model="config.auto_delete_after_secs" :min="10" :max="86400" controls-position="right" style="width:200px" />
    </el-form-item>
    <el-form-item label="默认语言">
      <el-select v-model="config.default_language" style="width:200px">
        <el-option label="中文" value="zh" />
        <el-option label="English" value="en" />
      </el-select>
    </el-form-item>
    <el-form-item label="时区">
      <el-input v-model="config.time_zone" placeholder="Asia/Shanghai" style="width:200px" />
    </el-form-item>
  </el-form>
</PanelSection>
```

确认 `Refresh` icon 已在 import 中（如 `import { Cpu, Refresh } from "@element-plus/icons-vue"`）。

- [ ] **Step 5: 手动测试**

1. 打开机器人管理页，确认页面底部出现"Bot 全局配置"区块
2. 数据正常加载（显示当前配置值）
3. 修改"积分系统"开关，点保存，成功提示显示

- [ ] **Step 6: Commit**

```powershell
git add web/src/api/botConfig.ts web/src/views/BotsView.vue
git commit -m "feat(web): add Bot global config form to BotsView"
```

---

## 最终验证

- [ ] **TypeScript 类型检查**

```powershell
cd "C:\Users\Administrator\Desktop\新建文件夹 (5)\TG群管机器人\sola-bot\web"
npx vue-tsc --noEmit
```
期望：无类型错误。

- [ ] **前端构建**

```powershell
npm run build
```
期望：Build 成功。

- [ ] **功能清单确认**

| 功能 | 验证方式 | 状态 |
|------|----------|------|
| 群组解绑 | ChatsView 操作列有"解绑"按钮，点击有确认框 | - |
| 禁言操作 | BansView 后台操作面板有禁言区块 | - |
| 统计日期筛选 | StatsView 顶部有日期范围选择器 | - |
| Bot 全局配置 | BotsView 有配置表单，读写正常 | - |
