# xxyCodingAgents × gstack 集成进度与实施路线

> 更新时间：2026-05-02
> 适用范围：`/Volumes/Elements/code/codingagents/xxyCodingAgents`
> 说明：这是当前 gstack 集成工作的主文档。Step 1/2/3/4 的实现进度、验收标准和文件落点都以本文为准。

---

## 1. 集成目标

我们引入 gstack，不是把整个项目照搬进 xxyCodingAgents，而是复用它已经验证过的几个关键能力：

1. 浏览器 QA 执行层：复用 gstack `browse` 的 daemon + CLI 能力。
2. 开发流程语义层：复用 gstack 的 `review / qa / ship / retro` 工作流思想。
3. Prompt 注入层：借鉴 gstack 的阶段模板、上下文组织和信任边界。
4. Learnings 层：兼容 gstack 的 `learnings.jsonl` 存储与检索模式。

明确不直接照搬的部分：

1. 不直接把 gstack 的 `SKILL.md` 原样喂给 agent。
2. 不把 `HostConfig` 当成 `AgentSpec`。
3. 不在 MVP 阶段做 centralized browse daemon。
4. 不在当前阶段引入 gstack 的完整模板编译系统。

---

## 2. 当前基线

截至 2026-05-02，xxyCodingAgents 已具备的基础能力：

1. 调度内核：`scheduler + watchdog + orchestrator + reconciler` 已接通。
2. 数据持久化：SQLite、Repository、迁移、13 张核心表已可用。
3. 终端执行：tmux session 管理、终端输出捕获、WebSocket 终端页面已可用。
4. 资源管理：资源采样、降载、驱逐、恢复基础能力已存在。
5. Git 工作区：workspace 与 git manager 已接入 orchestrator 基础流程。

与 gstack 集成直接相关的当前状态：

| 能力 | 当前状态 | 说明 |
|------|----------|------|
| 任务完成检测 | 已完成 | 调度器可解析 `[TASK_COMPLETED]` / `[TASK_FAILED]` |
| checkpoint 恢复链 | 已完成 | 恢复流程已重走 launcher/runtime 路径 |
| `TaskSpec.TimeoutSeconds` 强制 | 已完成 | `started_at` 已真正落库，超时可触发 |
| `AgentSpec` 激活 | 已完成 | `resolveAgentKind()` 和 `AdapterRegistry` 已接通 |
| Prompt 进入 runtime | 已完成 | launcher script + prompt file 已打通 |
| gstack browse | 已完成（基础版 + smoke） | workspace 级 daemon、env 注入、QA 并发约束已接通；已补可重复 smoke 脚本 |
| PromptEngine | 已完成（Step 3） | 已落地 `internal/prompt/engine.go` + `configs/prompts/*.yaml`，`scheduler.buildPrompt()` 默认走模板引擎，失败回退 legacy |
| gstack learnings 兼容 | 进行中（Step 4） | 已落地 `internal/learning` JSONL 读写/检索并接入 Prompt Layer 3，同时失败链路可自动写入 learning |

---

## 3. 总体路线

采用“先把 runtime 接缝做对，再逐层接入 gstack 能力”的路线：

| Step | 目标 | 状态 |
|------|------|------|
| Step 1 | Prompt file + launcher script + runtime 闭环 | 已完成 |
| Step 2 | workspace 级 gstack browse CLI 接入 | 已完成（基础版 + smoke） |
| Step 3 | PromptEngine + phase YAML 模板 | 已完成 |
| Step 4 | trust boundary + gstack learnings JSONL 兼容 | 进行中 |
| Step 5 | 七阶段工作流与 Gate 机制 | 设计中 |

---

## 4. Step 1：runtime 基础设施落地

### 4.1 目标

把“自然语言 prompt / shell command”可靠地送进 agent runtime，并且不依赖脆弱的 `echo | cli` 或多次 `tmux send-keys export ...`。

### 4.2 当前实现

1. `claude-code` 和 `generic-shell` 走不同命令路径。
2. 调度器只生成 launcher 脚本路径。
3. runtime adapter 只负责在 tmux 中执行 `bash /abs/path/to/run.sh`。
4. prompt 和 env 在 launcher 生成阶段烘进脚本，不再在 adapter 层拼接。

### 4.3 代码落点

| 文件 | 作用 |
|------|------|
| [internal/agentlauncher/launcher.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/agentlauncher/launcher.go) | 生成 prompt file 和 launcher script |
| [internal/runtime/adapter.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/runtime/adapter.go) | `StartRequest.Command` 改为 launcher 绝对路径 |
| [internal/runtime/adapters.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/runtime/adapters.go) | tmux 单次执行 launcher |
| [internal/config/config.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/config/config.go) | `agent_runtime.base_dir` / `browse_cli_path` 配置 |
| [internal/scheduler/scheduler.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/scheduler/scheduler.go) | `buildLauncher()`、任务完成检测、超时检测、恢复链 |
| [internal/scheduler/watchdog.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/scheduler/watchdog.go) | 死亡 agent 恢复与清理链 |
| [internal/orchestrator/orchestrator.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/orchestrator/orchestrator.go) | `completed_at` 持久化 |

### 4.4 Step 1 已解决的问题

1. `generic-shell` 不再错误消费 prompt。
2. `claude -p` 不再通过 `echo` 管道传入多行 prompt。
3. tmux 不再依赖多次 `send-keys` 的顺序和 quoting。
4. `recoverFromCheckpoint()` 不再绕过 launcher 路径。
5. `started_at` / `completed_at` 现在真正持久化。
6. prompt / launcher 产物在完成、失败、超时、恢复失败时可清理。

### 4.5 当前数据流

```text
Scheduler.launchAgent(task)
  -> resolveAgentKind(task)
  -> buildPrompt(task) or resolveCommand(task)
  -> agentlauncher.Build(...)
  -> prompt.md + run.sh
  -> runtime.Start(Command=/abs/path/run.sh)
  -> tmux send-keys "bash /abs/path/run.sh"
```

### 4.6 验收状态

1. `go test ./...` 通过
2. `go vet ./...` 通过
3. 恢复链、超时链、清理链已闭环

---

## 5. Step 2：workspace 级 gstack browse 集成

当前状态：**已完成（含 wrapper + smoke 脚本）**

### 5.1 目标

为 `qa` / `browser-qa` 任务提供浏览器自动化能力，但不引入多 run 共享状态的 centralized daemon。

### 5.2 采用方案

采用 workspace 级 browse daemon：

1. 每个 workspace 使用自己的 `BROWSE_STATE_FILE`。
2. 由 `browse` CLI 的 `ensureServer()` 自动拉起 daemon。
3. xxyCodingAgents 不直接管理 root token 生命周期。
4. QA agent 直接通过 `browse` CLI 与 daemon 交互。

### 5.3 为什么不先做 centralized daemon

1. 当前平台还没有多租户或弱信任 worker 模型。
2. 多 run 共享一个 browser daemon 会引入 tab 状态互踩。
3. Step 2 重点是让 QA 能力先跑通，不是做浏览器资源池。

### 5.4 已完成的实现

#### 新增文件

- [internal/runtime/browse.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/runtime/browse.go)

职责：

1. `EnsureDaemon(ctx)`：确保 workspace 级 daemon 已启动。
2. `ReadState()`：读取 `.gstack/browse.json`。
3. `IsHealthy(ctx, state)`：调用 `/health`。
4. `BuildEnv()`：生成 `BROWSE_STATE_FILE`、`PATH`、`BROWSE_CLI_PATH`。

#### 已修改文件

| 文件 | 已完成内容 |
|------|-----------|
| [internal/scheduler/scheduler.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/scheduler/scheduler.go) | `buildEnv()` 接入 browse manager；QA 任务附加 browse 指令说明；同 workspace QA 并发约束 |
| [internal/config/config.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/config/config.go) | 支持 `browse_cli_path` 默认值、绝对路径解析和环境变量覆盖 |
| [configs/config.yaml](/Volumes/Elements/code/codingagents/xxyCodingAgents/configs/config.yaml) | 已新增 `agent_runtime` 配置块 |
| [internal/runtime/browse_test.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/runtime/browse_test.go) | 覆盖 `BuildEnv()`、`EnsureDaemon()`、`ReadState()` |
| [internal/scheduler/scheduler_test.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/scheduler/scheduler_test.go) | 覆盖 QA browse env 注入和 workspace 占用判断 |

### 5.5 补充落地（2026-05-02）

1. `browse_cli_path` 已切到本仓库 wrapper：`./scripts/browse`（内部调用 gstack `browse/src/cli.ts`）。
2. 新增 [scripts/browse](/Volumes/Elements/code/codingagents/xxyCodingAgents/scripts/browse)，避免本机 `browse/dist/browse` 被 `SIGKILL`（137）导致的不可用。
3. 新增 [scripts/smoke-browse-qa.sh](/Volumes/Elements/code/codingagents/xxyCodingAgents/scripts/smoke-browse-qa.sh)，提供可重复 smoke 流程：
   - 有 `tmux`：跑 scheduler 全链路（API 建 run + SQLite 注入 QA task + 轮询完成 + 校验截图）
   - 无 `tmux`：自动退化为 direct CLI smoke（校验 `browse.json` + 截图）

### 5.6 目标配置

```yaml
agent_runtime:
  base_dir: "./data/agent-runtime"
  browse_cli_path: "./scripts/browse"
  prompt_template_dir: "./configs/prompts"
```

### 5.7 运行约束

1. 同一 workspace 同时只允许一个 `qa` / `browser-qa` 任务。
2. QA 任务的 launcher 需要带上 `BROWSE_STATE_FILE`。
3. 若 `browse_cli_path` 为空，则 QA 任务退化为无浏览器模式并记录 warning。

### 5.8 安全边界

接入 browse 后，网页内容属于不可信输入。

Step 2 至少要做：

1. QA prompt 中加入不可信网页内容规则。
2. 浏览器文本进入 prompt 前用 `BEGIN/END UNTRUSTED WEB CONTENT` 包裹。
3. 引入 canary，用于检测 prompt injection 内容回流。

### 5.9 验收状态

已完成：

1. QA 任务可按 workspace 级方式注入 browse 环境变量。
2. QA 调度已限制“同一 workspace 同时一个 browser QA”。
3. `go test ./...` 通过。
4. `go vet ./...` 通过。
5. `scripts/smoke-browse-qa.sh` 在当前环境通过（direct fallback 与 scheduler 两种模式均通过，均产出 `browse.json` 与截图）。

---

## 6. Step 3：PromptEngine + phase YAML

当前状态：**已完成（含测试）**

### 6.1 目标

把当前 `buildPrompt()` 的临时拼接替换成可维护的阶段模板系统。

### 6.2 设计原则

1. 不直接解析或执行 gstack 的 `SKILL.md`。
2. 从 gstack 提取“工作流正文”，手工沉淀为本项目自己的 YAML。
3. Prompt 作为 xxyCodingAgents 的资产，由本仓库维护。

### 6.3 已落地结构

#### 新增文件（已完成）

- `internal/prompt/engine.go`
- `internal/prompt/engine_test.go`
- `configs/prompts/review.yaml`
- `configs/prompts/qa.yaml`
- `configs/prompts/ship.yaml`
- `configs/prompts/retro.yaml`
- `configs/prompts/think.yaml`
- `configs/prompts/plan.yaml`
- `configs/prompts/build.yaml`

#### 已修改文件（已完成）

- [internal/scheduler/scheduler.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/scheduler/scheduler.go)
- [internal/scheduler/scheduler_test.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/scheduler/scheduler_test.go)
- [internal/config/config.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/config/config.go)
- [configs/config.yaml](/Volumes/Elements/code/codingagents/xxyCodingAgents/configs/config.yaml)

#### 运行时接口（实际）

```go
BuildPrompt(opts BuildOptions) (string, error)
```

### 6.4 四层注入

1. Layer 1：Agent role / system instruction
2. Layer 2：Phase body（来自 `configs/prompts/*.yaml`）
3. Layer 3：Learnings / patches
4. Layer 4：Runtime state（git status、workspace、最近失败等）

### 6.5 验收标准

已完成：

1. `scheduler.buildPrompt()` 优先走 PromptEngine，失败自动回退 legacy（不中断任务）。
2. phase 模板目录已独立维护，支持 alias（如 `browser-qa -> qa`）。
3. 已覆盖 `review / qa / ship / retro / think / plan / build` 七个 phase 模板。
4. PromptEngine 单测与 scheduler 侧集成单测已补齐并通过。

---

## 7. Step 4：trust boundary + learnings JSONL

### 7.1 目标

把 browse 带来的不可信输入风险和 learnings 复用能力一起补上。

当前状态：**进行中（已完成第一批代码落地）**

已完成：

1. 新增 [internal/learning/store.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/learning/store.go)，支持 gstack 兼容 JSONL append/read：
   - `FilePath(projectSlug) -> {learnings_root_dir}/{slug}/learnings.jsonl`
   - `Append()` append-only 写入
   - `ReadAll()` 容错读取（坏行跳过）
2. 新增 [internal/learning/search.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/learning/search.go)，提供 phase + query 的轻量检索与去重排序，输出可直接注入 Prompt Layer 3 的 insights。
3. 新增 [internal/prompt/security.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/prompt/security.go)：
   - `NewCanary()`
   - `WrapUntrustedContent()`
   - `QATrustBoundaryRule()`
4. 调度器已接入：
   - [internal/scheduler/scheduler.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/scheduler/scheduler.go) `buildPrompt()` 在 QA/browse 任务中自动包裹 InputData，并注入 trust-boundary 规则
   - 同时调用 learnings searcher，把检索结果注入 PromptEngine 的 `Learnings`（Layer 3）
5. 失败链路自动写入：
   - `checkTaskCompletion()` 检测到 `[TASK_FAILED]` 时，自动落一条 pitfall learning
   - `checkTaskTimeouts()` 超时失败时自动落 learning
   - `launchAgent()` 的启动失败路径（tmux/launcher/runtime/markRunning）自动落 learning
6. canary 泄漏检测闭环：
   - 调度器为 QA 任务记录 task 级 canary
   - `checkTaskCompletion()` 若输出中命中 canary，立即判定潜在 prompt injection
   - 自动执行：`FailTask` + `security_alert` event + learning 记录 + artifact 清理
7. 配置已补齐：
   - [internal/config/config.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/config/config.go) 新增 `agent_runtime.learnings_root_dir`
   - [configs/config.yaml](/Volumes/Elements/code/codingagents/xxyCodingAgents/configs/config.yaml) 默认值：`~/.gstack/projects`
8. 单测覆盖：
   - [internal/learning/store_test.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/learning/store_test.go)
   - [internal/learning/search_test.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/learning/search_test.go)
   - [internal/prompt/security_test.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/prompt/security_test.go)
   - [internal/scheduler/scheduler_test.go](/Volumes/Elements/code/codingagents/xxyCodingAgents/internal/scheduler/scheduler_test.go)（learnings 注入、QA trust-boundary）

### 7.2 Learnings 兼容策略

兼容 gstack 的项目级 JSONL：

```text
~/.gstack/projects/{slug}/learnings.jsonl
```

本项目新增方向：

1. `internal/learning/store.go`
2. `internal/learning/search.go`
3. 在 PromptEngine 的 Layer 3 做注入

### 7.3 下一步（Step 4 剩余）

1. 补充跨 run 的检索策略（例如按 `branch/files` 进一步加权）。
2. 增加低噪声清洗策略（避免重复失败 reason 过度写入）。
3. 增加 `security_alert` 的前端可视化与告警面板。

### 7.4 后做什么（Step 4+）

1. 自动提炼 pattern
2. 自动 patch promotion
3. Memory promotion / demotion

### 7.5 验收标准（更新）

已达成：
1. learnings JSONL 读写与检索能力可用。
2. buildPrompt 可注入相关 learnings（Layer 3）。
3. QA 输入可走 trust boundary 包裹。
4. canary 泄漏可触发自动失败与安全事件。

待达成：
1. 泄漏事件的告警展示与跨 run 趋势统计。

---

## 8. Step 5：七阶段工作流与 Gate

这一步不属于 runtime 基础设施，而属于更高层的工作流编排。

当前判断：

1. Step 2 和 Step 3 已完成，Step 4 已进入收尾阶段（安全边界 + learnings JSONL）。
2. 再把 `think -> plan -> build -> review -> qa -> ship -> retro` 变成默认工作流模板。
3. Gate 必须基于当前项目已有的 DAG 模型实现，不改成 `steps[]` 管道模型。

推荐做法：

1. Gate 作为显式 DAG 节点。
2. `node.kind = gate`
3. `node.config_json` 保存 `type / conditions / timeout`

---

## 9. 当前未决事项

1. `security_alert` 事件仍缺前端可视化与告警面板。
2. learnings 的项目 slug 规则需进一步和 project/repo 标识严格对齐（当前为多级回退策略）。
3. learnings 检索需做降噪与重复写入抑制（避免同类失败刷屏）。
4. 需在 CI 或其他开发机复跑 `scripts/smoke-browse-qa.sh`，确保环境迁移后仍稳定通过。

---

## 10. 建议执行顺序

1. 完成本文 Step 4 收尾：security_alert 可视化 + learnings 检索降噪。
2. 再推进 Step 5：七阶段工作流和 Gate。
3. 最后补 CI 稳定性验证与发布前回归脚本。

---

## 11. 文档关系

| 文档 | 角色 |
|------|------|
| [docs/gstack-integration-progress.md](/Volumes/Elements/code/codingagents/xxyCodingAgents/docs/gstack-integration-progress.md) | 当前活跃实施主文档 |
| [docs/architecture-design.md](/Volumes/Elements/code/codingagents/xxyCodingAgents/docs/architecture-design.md) | 目标架构文档 |
| [docs/development-plan.md](/Volumes/Elements/code/codingagents/xxyCodingAgents/docs/development-plan.md) | MVP 基线计划文档 |
| [docs/task-list.md](/Volumes/Elements/code/codingagents/xxyCodingAgents/docs/task-list.md) | MVP 历史任务清单 |

后续如果 Step 2/3/4 有代码落地，优先更新本文，再同步到其他文档。
