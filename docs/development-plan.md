# AI Dev Platform — MVP 开发计划（基线文档）

> 状态更新：2026-05-02
> 本文记录的是 MVP 基线开发计划，不再承担 gstack 集成的实施跟踪职责。
> 当前活跃实施路线、Step 1/2/3/4 进度、文件落点和验收标准，请参考 [gstack-integration-progress.md](/Volumes/Elements/code/codingagents/xxyCodingAgents/docs/gstack-integration-progress.md)。
>
> 当前实现快照：
> - 核心运行时闭环已建立：scheduler / watchdog / orchestrator / runtime adapter / tmux / workspace
> - gstack 集成 Step 1 已完成：prompt file + launcher script + recovery/timeout/cleanup 闭环
> - gstack 集成 Step 2 已完成：workspace 级 browse CLI + wrapper + smoke 脚本已落地（scheduler 全链路需 tmux 环境复核）
> - gstack 集成 Step 3 已完成：PromptEngine + phase YAML + scheduler 注入链已落地
> - gstack 集成 Step 4 进行中：已完成 learnings JSONL 检索注入 + QA trust-boundary + 失败链路自动写入 + canary 泄漏检测
> - 当前后续阶段：Step 4 收尾（检索降噪/可视化）、Step 5 Prompt Composer MVP（草稿生成/用户确认/发送）、Step 6 Gate 工作流

## 阶段一：项目基础搭建

1.1 Go 项目初始化（`go mod init`、目录结构创建）
1.2 配置文件加载（`config.yaml` 解析，涵盖 server、runtime、scheduler、thresholds、timeouts、sqlite 六大配置块）
1.3 配置优先级机制：`config.yaml < .env < environment variables`，支持环境变量覆盖
1.4 SQLite 数据库初始化（连接、WAL 模式、foreign key、busy timeout）
1.5 基础表结构创建（projects、runs、tasks、agent_instances、workspaces、terminal_sessions、checkpoints、resource_snapshots、events、command_logs、task_specs、agent_specs、workflow_templates 共 13 张表）
1.6 数据库索引与唯一约束设计：
- 索引：`runs(project_id, status, created_at)`、`tasks(run_id, status, priority, created_at)`、`agent_instances(run_id, status, last_heartbeat_at)`、`terminal_sessions(task_id, status)`、`events(run_id, created_at)`、`resource_snapshots(created_at)`、`command_logs(task_id, created_at)`、`checkpoints(task_id, created_at desc)`
- 唯一约束：workspace path 唯一、terminal session name 唯一、tmux session name 唯一、run external key 唯一
1.7 基础日志框架搭建（结构化日志，选用 `slog` 或 `zap`）
1.8 健康检查与诊断接口：
- `/healthz`：进程存活
- `/readyz`：数据库、调度器、tmux 可用
- `/debug/pprof`：Go 进程性能分析
- `/api/system/diagnostics`：核心状态概览

---

## 阶段二：领域模型与数据访问层

2.1 领域模型定义（Project、Run、Task、AgentInstance、Workspace、TerminalSession）
2.2 执行契约层定义（详见"执行契约层"章节）：
- TaskSpec：任务定义模板
- AgentSpec：Agent 类型定义
- WorkflowTemplate：工作流模板定义
2.3 状态机定义：
- Run 状态：pending → running → completed/failed/cancelled
- Task 状态：queued → admitted → running → completed/failed/cancelled/evicted
- Task 增加 `attempt_no` 字段，retry 时生成新 attempt，不覆盖旧记录
- AgentInstance 状态：starting → running → paused → stopped → failed → recoverable → orphaned
2.4 队列与优先级模型（详见"队列与优先级模型"章节）：
- Task 增加：priority（low / normal / high）、queue_status（queued / admitted / running / paused / evicted）、resource_class（light / heavy）、preemptible（是否允许被驱逐）、restart_policy
2.5 取消语义与幂等语义定义（详见"取消与幂等语义"章节）：
- Cancel Task：取消单任务
- Stop Agent：停止 Agent 进程
- Cancel Run：取消整个 Run，未开始任务直接 cancelled，运行中任务走 stop
- Graceful stop 超时后 force kill
- 重复调用 /pause、/resume、/stop 不报错（幂等）
2.6 Repository 层实现（每张表的 CRUD 操作）
2.7 数据库迁移机制（简易版本管理，确保表结构可演进）

---

## 阶段三：REST API 层

3.1 HTTP 服务框架搭建（路由、中间件、CORS、请求日志）
3.2 项目管理 API（`POST /api/projects`、`GET /api/projects`、`GET /api/projects/:id`）
3.3 Run 管理 API（`POST /api/runs`、`GET /api/runs/:id`、`GET /api/runs/:id/timeline`）
3.4 Task 管理 API（`GET /api/runs/:id/tasks`、`POST /api/tasks/:id/retry`、`POST /api/tasks/:id/cancel`）
3.5 Agent 管理 API（`GET /api/agents/:id`、`POST /api/agents/:id/pause`、`POST /api/agents/:id/resume`、`POST /api/agents/:id/stop`）
3.6 Terminal API（`POST /api/terminals`、`GET /api/terminals/:id`）
3.7 系统 API（`GET /api/system/metrics`、`GET /api/runs/:id/resources`）
3.8 统一错误处理与响应格式
3.9 密钥与敏感配置管理（详见"密钥与敏感配置管理"章节）

---

## 阶段四：终端管理模块

4.1 tmux session 管理封装（创建、销毁、列表、重命名）
4.2 tmux pane 管理封装（分屏、发送命令、获取输出）
4.3 PTY 进程管理（启动、attach、detach、输出流采集）
4.4 终端输出流缓冲与持久化（写入 `data/logs/` 目录）
4.5 Terminal Session 数据库记录与状态管理
4.6 WebSocket 终端实时推送（`GET /api/terminals/:id/ws`）

---

## 阶段五：工作区与 Git 管理

5.1 工作区创建与隔离（基于 `data/workspaces/` 按任务隔离）
5.2 Git 仓库 clone / checkout 封装
5.3 分支管理（为每个 Task 创建独立分支）
5.4 文件变更检测（`git status`、`git diff` 读取）
5.5 变更提交（`git add`、`git commit` 封装）
5.6 工作区清理与回收
5.7 Git 凭证独立管理（不与其他密钥混用）

---

## 阶段六：Agent 运行时管理（含适配器层）

6.1 AgentRuntime 适配器接口定义（详见"适配器层"章节）：
```go
type AgentRuntime interface {
    Start(ctx context.Context, req StartRequest) (*StartResult, error)
    Pause(ctx context.Context, agentID string) error
    Resume(ctx context.Context, agentID string) error
    Stop(ctx context.Context, agentID string) error
    Checkpoint(ctx context.Context, agentID string) (*CheckpointData, error)
    Inspect(ctx context.Context, agentID string) (*AgentStatus, error)
}
```
6.2 ClaudeCodeAdapter 实现（对接 Claude Code CLI）
6.3 GenericShellAdapter 预留（通用 Shell 脚本执行）
6.4 Agent 进程启动器（通过 tmux/PTY 启动 AI CLI 工具，基于 AgentSpec 配置）
6.5 Agent 心跳机制（定期上报状态、输出时间、资源占用，基于 AgentSpec.heartbeat_mode）
6.6 Agent 状态管理（starting → running → paused → stopped → failed）
6.7 Agent 输出规范化层（详见"输出规范化层"章节）：
- Raw output：原始终端流原封不动落盘
- Structured events：从输出中提取结构化事件（阶段变更、命令执行、测试通过/失败、生成补丁、请求人工介入）
- 首版仅做简单规则提取：正则匹配、前缀标记、已知 CLI 输出模式匹配
6.8 Agent 进程监控（进程存活检测、异常捕获）
6.9 Agent 手动控制接口（暂停 / 恢复 / 停止）

---

## 阶段七：资源调度器

7.1 系统资源采样器：
- 内存使用率定时采集
- CPU 使用率定时采集
- 磁盘使用率定时采集
- 工作区目录大小采样
- 日志目录大小采样
- 进程树采样（每个 Agent 的子进程数量与资源占用）
7.2 资源压力分级判定：
- 内存压力：NORMAL / WARN / HIGH / CRITICAL
- 磁盘压力：基于 disk_warn_percent / disk_high_percent
7.3 Agent 准入控制器（根据并发限制、资源状态、Task priority、resource_class 决定是否允许启动）
7.4 调度循环（tick 驱动，默认 3 秒一次）
7.5 降载策略实现：
- WARN：停止重型任务准入
- HIGH：暂停低优先级、preemptible=true 的 Agent
- CRITICAL：checkpoint + 驱逐低优先级且可恢复 Agent
7.6 资源恢复后按优先级恢复排队任务
7.7 资源快照记录（写入 `resource_snapshots` 表）
7.8 磁盘治理：
- workspace_max_size_mb：单个工作区大小上限
- log_retention_days：日志保留天数
- max_total_log_size_mb：日志总量上限
- 超限自动清理
7.9 进程树治理：
- max_child_processes_per_agent：每个 Agent 最大子进程数
- orphan 进程检测与回收

---

## 阶段八：启动恢复 / 自愈对账 + Checkpoint 与容错

### 8.0 Startup Reconciler（启动自愈对账）

8.0.1 启动时扫描 DB 中 running / starting / paused 状态的对象
8.0.2 扫描本机 tmux session、相关 PID、工作目录
8.0.3 状态对账与修正：
- 进程存活 → running
- 进程不在但 checkpoint 存在 → recoverable
- 进程不在且无 checkpoint → failed
- tmux session 存在但 DB 无记录 → orphaned
8.0.4 已超时的 task 自动转 failed
8.0.5 orphan 进程回收
8.0.6 orphan tmux session 清理
8.0.7 对账结果写入 event 日志

### 8.1 Checkpoint 与容错恢复

8.1.1 Checkpoint 数据模型与存储（Agent 状态、上下文、进度快照）
8.1.2 Checkpoint 触发机制（周期性 / 阶段完成 / 驱逐前）
8.1.3 Watchdog 监控器（无输出超时、心跳超时、进程崩溃检测）
8.1.4 从 Checkpoint 恢复任务流程（基于 reconciler 标记的 recoverable 状态）
8.1.5 终端会话重建（恢复后重新创建 tmux session）
8.1.6 Agent 自动重启机制

---

## 阶段九：编排器（Orchestrator）

9.1 Run 生命周期管理（创建 → 按 WorkflowTemplate 拆分 Task → 分配 AgentSpec → 推进执行）
9.2 Task 依赖与顺序控制（基于 WorkflowTemplate 的节点依赖关系）
9.3 WorkflowTemplate 引擎（从模板实例化为具体 Task 列表，绑定 TaskSpec）
9.4 失败后策略（基于 WorkflowTemplate 定义：重试 / 跳过 / 中止整个 Run）
9.5 Task 重试策略（生成新 attempt_no，不覆盖旧记录）
9.6 Run 事件时间线记录
9.7 编排状态持久化（确保中断后可恢复编排进度）

---

## 阶段十：审计与事件日志

10.1 结构化事件写入（Task 启动/完成/失败、Agent 状态变更、资源事件、对账事件等）
10.2 终端原始输出日志持久化（`data/logs/`）
10.3 命令执行日志记录（`command_logs` 表，对 token / cookie / secret 做脱敏）
10.4 事件查询 API
10.5 审计日志清理策略（基于 log_retention_days 和 max_total_log_size_mb）

---

## 阶段十一：Web 前端控制台

11.1 Next.js 项目初始化 + Tailwind CSS 配置
11.2 页面路由与布局框架
11.3 项目管理页面（项目列表、创建项目）
11.4 Run 管理页面（创建 Run、选择 WorkflowTemplate、查看 Run 详情、时间线）
11.5 Task 管理页面（Task 列表、状态、attempt 历史、重试/取消操作）
11.6 Agent 监控页面（Agent 列表、状态、暂停/恢复/停止操作、子进程信息）
11.7 终端页面（xterm.js 集成、WebSocket 实时终端输出）
11.8 系统监控仪表盘（内存/CPU/磁盘使用率、内存压力等级、活跃 Agent 数、队列状态）
11.9 工作流可视化（React Flow 集成，展示 Run 内 Task 关系图）
11.10 前端安全：不暴露底层命令全文，敏感信息脱敏展示

---

## 阶段十二：集成测试与稳定性验证

12.1 单元测试覆盖核心模块（Repository、调度器、编排器、Reconciler）
12.2 集成测试（创建 Run → 启动 Agent → 执行任务 → 完成）
12.3 长时间运行稳定性测试
12.4 内存压力模拟测试（验证降载与恢复流程）
12.5 磁盘压力模拟测试（验证日志清理与工作区回收）
12.6 进程崩溃恢复测试（验证 Reconciler 对账流程）
12.7 幂等性测试（重复 pause/resume/stop 调用）
12.8 端到端演示流程跑通

---

## 关键设计章节

### 执行契约层

#### TaskSpec（任务定义模板）

定义一个任务"应该怎么跑"：

| 字段 | 说明 |
|------|------|
| task_type | planner / coder / tester / reviewer |
| runtime_type | claude-code / codex-cli / custom-script |
| command_template | 启动命令模板 |
| timeout_seconds | 超时时间 |
| retry_policy | 重试策略 |
| resource_class | light / medium / heavy |
| can_pause | 是否可暂停 |
| can_checkpoint | 是否可 checkpoint |
| required_inputs | 需要的输入 |
| expected_outputs | 期望的输出 |

#### AgentSpec（Agent 类型定义）

定义某类 Agent 的运行能力：

| 字段 | 说明 |
|------|------|
| agent_kind | Agent 类别标识 |
| supported_task_types | 支持的任务类型列表 |
| default_command | 默认启动命令 |
| max_concurrency | 最大并发数 |
| resource_weight | 资源权重 |
| heartbeat_mode | 心跳模式 |
| output_parser | 输出解析器类型 |

#### WorkflowTemplate（工作流模板）

定义一个 Run 的任务图：

| 字段 | 说明 |
|------|------|
| name | 模板名称 |
| nodes | 节点列表（每个节点绑定一个 TaskSpec） |
| dependencies | 节点依赖关系（DAG） |
| on_failure | 失败后策略（retry / skip / abort） |

---

### 适配器层

```go
type AgentRuntime interface {
    Start(ctx context.Context, req StartRequest) (*StartResult, error)
    Pause(ctx context.Context, agentID string) error
    Resume(ctx context.Context, agentID string) error
    Stop(ctx context.Context, agentID string) error
    Checkpoint(ctx context.Context, agentID string) (*CheckpointData, error)
    Inspect(ctx context.Context, agentID string) (*AgentStatus, error)
}
```

MVP 阶段实现：
- **ClaudeCodeAdapter**：对接 Claude Code CLI
- 预留 **CodexAdapter**、**GenericShellAdapter** 接口

---

### 队列与优先级模型

Task 增加字段：

| 字段 | 说明 |
|------|------|
| priority | low / normal / high |
| queue_status | queued / admitted / running / paused / evicted |
| resource_class | light / heavy |
| preemptible | 是否允许被驱逐 |
| restart_policy | 重启策略 |
| attempt_no | 执行次数编号（retry 时递增） |

调度判定逻辑：
- 准入：priority + resource_class + 当前资源状态
- 暂停：preemptible=true + 低 priority
- 驱逐：preemptible=true + 低 priority + 资源 CRITICAL
- 恢复：按 priority 排序，高 priority 先恢复

---

### 取消与幂等语义

#### 取消语义

| 操作 | 行为 |
|------|------|
| Cancel Task | 取消单个任务 |
| Stop Agent | 停止 Agent 进程 |
| Cancel Run | 取消整个 Run，未开始任务直接 cancelled，运行中任务走 stop |
| Graceful stop | 先发停止信号，超时后 force kill |

#### 幂等语义

| 操作 | 行为 |
|------|------|
| 重复 /pause | 不报错，保持 paused |
| 重复 /resume | 不报错，保持 running |
| 重复 /stop | 不报错，保持 stopped |
| /retry | 生成新 attempt_no，不覆盖旧 task 执行记录 |

---

### 输出规范化层

分两层处理：

**Raw output（原始层）**
- 原始终端流，原封不动落盘到 `data/logs/`
- 供完整回放和排障使用

**Structured events（结构化层）**
- 从输出中提取结构化事件，写入 `events` 表
- 事件类型：阶段变更、命令执行、测试通过、测试失败、生成补丁、请求人工介入
- 首版仅做简单规则提取：正则匹配、前缀标记、已知 CLI 输出模式匹配

---

### 密钥与敏感配置管理

- API key 不进数据库明文存储
- 命令日志对 token / cookie / secret 做脱敏处理
- Web 前端不暴露底层命令全文
- Git 凭证独立管理
- 配置优先级：`config.yaml < .env < environment variables`

---

### 启动自愈对账流程

系统每次启动时执行 Reconciler：

1. 扫描 DB 中 running / starting / paused 状态的对象
2. 扫描本机 tmux session、相关 PID、工作目录
3. 做状态对账，修正为：running / recoverable / failed / orphaned
4. 已超时的 task 转 failed
5. orphan 进程回收
6. orphan tmux session 清理
7. 对账结果写入 event 日志

---

## 阶段依赖关系

```text
阶段一（基础）
  → 阶段二（领域模型 + 执行契约 + 队列模型）
    → 阶段三（API）
    → 阶段四（终端）      ← 可与阶段五并行
    → 阶段五（工作区）    ← 可与阶段四并行
      → 阶段六（Agent 运行时 + 适配器层）
        → 阶段七（资源调度 + 磁盘/进程树治理）
        → 阶段八（Reconciler + Checkpoint）  ← 可与阶段七并行
          → 阶段九（编排器 + WorkflowTemplate 引擎）
            → 阶段十（审计）
              → 阶段十一（前端）
                → 阶段十二（测试）
```

---

## 默认调度参数（参考）

| 参数 | 默认值 |
|------|--------|
| 同时活跃 Agent | 2 |
| 同时重型任务 | 1 |
| 同时测试任务 | 1 |
| 调度 tick 周期 | 3 秒 |
| WARN 内存阈值 | 70% |
| HIGH 内存阈值 | 80% |
| CRITICAL 内存阈值 | 88% |
| disk_warn_percent | 80% |
| disk_high_percent | 90% |
| workspace_max_size_mb | 2048 |
| log_retention_days | 7 |
| max_total_log_size_mb | 1024 |
| max_child_processes_per_agent | 10 |
| 心跳超时 | 30 秒 |
| 无输出超时 | 900 秒 |
| Checkpoint 间隔 | 30 秒 |

---

## MVP 成功标准

当以下目标全部达成时，MVP 完成：

- [ ] 可以通过 Web 创建一个 Run，选择 WorkflowTemplate
- [ ] 可以为 Run 自动拆分 Task（基于 WorkflowTemplate）
- [ ] 可以启动至少 2 个受管 Agent（通过适配器层）
- [ ] 可以在 Web 中看到终端输出
- [ ] 可以查看事件时间线（含结构化事件）
- [ ] 可以在内存/磁盘压力升高时暂停/驱逐低优先级 Agent
- [ ] 被驱逐任务可以从 checkpoint 恢复
- [ ] 系统重启后可通过 Reconciler 自愈对账
- [ ] 重复操作（pause/resume/stop）幂等安全
- [ ] 系统可以持续运行较长时间而不失控
