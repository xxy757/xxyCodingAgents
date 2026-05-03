# AI Dev Platform — MVP 开发任务清单（历史基线）

> 基于 2026-04-12 文档与代码一致性检查报告生成
> 本文反映的是 MVP 基线阶段的任务拆分，当前已不再作为活跃开发跟踪文档。
> 当前 gstack 集成的真实进度、Step 1 完成情况和 Step 2/3/4 路线，请参考 [gstack-integration-progress.md](/Volumes/Elements/code/codingagents/xxyCodingAgents/docs/gstack-integration-progress.md)。

## 当前状态说明（2026-05-02）

1. 本文中的 P0/P1/P2 任务大部分已经完成或关闭，详见文末状态追踪和 `docs/reports/`。
2. 其中和 gstack 集成直接相关的基础能力已经完成：
   - Agent 启动流程
   - Checkpoint / recovery 基础链路
   - Watchdog / timeout / 任务完成检测
   - DAG 执行、终端持久化、前端基础页
3. 当前活跃工作不再是 MVP 补洞，而是：
   - Step 4：trust boundary + learnings 兼容（进行中）
   - Step 5：Prompt Composer MVP（草稿生成/用户确认/发送，设计中）
   - Step 6：七阶段工作流与 Gate（设计中）

---

## 任务优先级说明

| 级别 | 含义 | SLA |
|------|------|-----|
| **P0** | 阻塞 MVP 核心功能，必须立即完成 | 优先处理 |
| **P1** | 重要功能完善，影响用户体验 | P0 之后处理 |
| **P2** | 质量保障与锦上添花 | P1 之后处理 |

---

## P0 — 阻塞 MVP 核心功能（7 项）

### P0-1：实现幂等语义（pause/resume/stop）

| 属性 | 内容 |
|------|------|
| 任务ID | P0-1 |
| 优先级 | P0 |
| 负责人 | 待分配 |
| 预计工时 | 2h |
| 对应问题 | N3：幂等语义未完整实现 |
| 影响标准 | S9：重复 pause/resume/stop 幂等安全 |

**完成标准：**
- [ ] `POST /api/agents/:id/pause`：Agent 已 paused 时返回 200（非 400）
- [ ] `POST /api/agents/:id/resume`：Agent 已 running 时返回 200（非 400）
- [ ] `POST /api/agents/:id/stop`：Agent 已 stopped 时返回 200（已实现）
- [ ] 对不可转换状态（如 starting → paused）仍返回 400
- [ ] 编写单元测试验证幂等行为

**涉及文件：**
- `internal/api/handlers.go` — handlePauseAgent、handleResumeAgent

**开发报告：** `docs/reports/P0-1-idempotent-semantics.md`

---

### P0-2：接入 Agent 启动流程（调度器 → 适配器）

| 属性 | 内容 |
|------|------|
| 任务ID | P0-2 |
| 优先级 | P0 |
| 负责人 | 待分配 |
| 预计工时 | 6h |
| 对应问题 | N8：Agent 启动流程未接入 |
| 影响标准 | S3：启动至少 2 个受管 Agent |

**完成标准：**
- [ ] scheduler.scheduleTasks 在 Task 从 queued → admitted 后，创建 AgentInstance 记录
- [ ] scheduler 调用 AgentRuntime.Start 启动 Agent 进程
- [ ] AgentInstance 状态从 starting → running 正确流转
- [ ] AgentInstance 记录 PID、TmuxSession 等运行时信息
- [ ] 支持至少 2 个 Agent 同时运行
- [ ] Agent 启动失败时正确标记为 failed 并记录事件

**涉及文件：**
- `internal/scheduler/scheduler.go` — scheduleTasks 扩展
- `internal/runtime/adapters.go` — 可能需要调整
- `internal/storage/repositories.go` — AgentInstance CRUD

**开发报告：** `docs/reports/P0-2-agent-launch-pipeline.md`

---

### P0-3：实现资源降载策略（WARN/HIGH/CRITICAL）

| 属性 | 内容 |
|------|------|
| 任务ID | P0-3 |
| 优先级 | P0 |
| 负责人 | 待分配 |
| 预计工时 | 8h |
| 对应问题 | N4：降载策略不完整 |
| 依赖 | P0-2 |
| 影响标准 | S6：内存/磁盘压力时暂停/驱逐低优先级 Agent |

**完成标准：**
- [ ] WARN 级别：停止重型任务（resource_class=heavy）准入
- [ ] HIGH 级别：暂停 preemptible=true + 低 priority 的 Agent
- [ ] CRITICAL 级别：先 checkpoint，再驱逐低优先级且可恢复 Agent
- [ ] 资源恢复后按 priority 排序恢复排队任务
- [ ] 降载操作写入 event 日志
- [ ] 压力等级变更写入 event 日志

**涉及文件：**
- `internal/scheduler/scheduler.go` — tick 方法扩展
- `internal/scheduler/reconciler.go` — 可能需要扩展

**开发报告：** `docs/reports/P0-3-load-shedding-strategy.md`

---

### P0-4：实现 Checkpoint 机制与恢复流程

| 属性 | 内容 |
|------|------|
| 任务ID | P0-4 |
| 优先级 | P0 |
| 负责人 | 待分配 |
| 预计工时 | 6h |
| 对应问题 | N5：Checkpoint 为空壳 |
| 依赖 | P0-2, P0-3 |
| 影响标准 | S7：被驱逐任务可从 checkpoint 恢复 |

**完成标准：**
- [ ] 周期性 checkpoint（默认 30 秒间隔）协程运行
- [ ] checkpoint 保存 Agent 实际状态（命令历史、工作目录、环境变量）
- [ ] 驱逐前触发 checkpoint
- [ ] 从 checkpoint 恢复任务流程：读取 checkpoint → 重建 tmux session → 恢复 Agent 状态
- [ ] Checkpoint 数据写入 checkpoints 表
- [ ] 恢复操作写入 event 日志

**涉及文件：**
- `internal/runtime/adapters.go` — Checkpoint 方法重写
- `internal/scheduler/scheduler.go` — 新增 checkpoint 协程
- `internal/storage/repositories.go` — CheckpointRepo 扩展

**开发报告：** `docs/reports/P0-4-checkpoint-recovery.md`

---

### P0-5：实现 Watchdog 监控与心跳机制

| 属性 | 内容 |
|------|------|
| 任务ID | P0-5 |
| 优先级 | P0 |
| 负责人 | 待分配 |
| 预计工时 | 5h |
| 对应问题 | N6：Watchdog 未实现, N7：心跳未实现 |
| 依赖 | P0-2 |
| 影响标准 | S10：系统持续运行不失控 |

**完成标准：**
- [ ] Watchdog 协程定期检查所有 running Agent：
  - 心跳超时（默认 30 秒）→ 标记为 failed
  - 无输出超时（默认 900 秒）→ 标记为 failed
  - 进程崩溃（tmux session 不存在）→ 标记为 failed/recoverable
- [ ] 心跳采集：Agent 运行时定期更新 last_heartbeat_at
- [ ] 异常检测触发事件写入 event 日志
- [ ] 超时 Task 自动转 failed

**涉及文件：**
- `internal/scheduler/watchdog.go` — 新建
- `internal/scheduler/scheduler.go` — 集成 watchdog
- `cmd/server/main.go` — 启动 watchdog

**开发报告：** `docs/reports/P0-5-watchdog-heartbeat.md`

---

### P0-6：前端集成 xterm.js 终端展示

| 属性 | 内容 |
|------|------|
| 任务ID | P0-6 |
| 优先级 | P0 |
| 负责人 | 待分配 |
| 预计工时 | 4h |
| 对应问题 | N1：缺少 xterm.js |
| 影响标准 | S4：在 Web 中看到终端输出 |

**完成标准：**
- [ ] 安装 xterm.js 和 xterm-addon-fit 依赖
- [ ] 新增终端详情页 `/terminals/[id]`
- [ ] WebSocket 连接 `/api/terminals/:id/ws` 实时显示终端输出
- [ ] 支持键盘输入发送到 tmux session
- [ ] 终端自适应容器大小
- [ ] 终端列表页增加"打开终端"链接

**涉及文件：**
- `web/package.json` — 新增 xterm 依赖
- `web/src/app/terminals/[id]/page.tsx` — 新建
- `web/src/app/terminals/page.tsx` — 增加链接

**开发报告：** `docs/reports/P0-6-xtermjs-integration.md`

---

### P0-7：编写核心模块单元测试

| 属性 | 内容 |
|------|------|
| 任务ID | P0-7 |
| 优先级 | P0 |
| 负责人 | 待分配 |
| 预计工时 | 8h |
| 对应问题 | N15：测试完全缺失 |
| 依赖 | P0-1 ~ P0-5 |

**完成标准：**
- [ ] Repository 层测试（CRUD 操作）
- [ ] Scheduler 降载策略测试
- [ ] Orchestrator 工作流实例化测试
- [ ] Reconciler 对账逻辑测试
- [ ] 幂等语义测试
- [ ] 审计脱敏函数测试
- [ ] 测试覆盖率 > 60%

**涉及文件：**
- `internal/storage/*_test.go` — 新建
- `internal/scheduler/*_test.go` — 新建
- `internal/orchestrator/*_test.go` — 新建
- `internal/audit/*_test.go` — 新建

**开发报告：** `docs/reports/P0-7-unit-tests.md`

---

## P1 — 重要功能完善（4 项）

### P1-1：实现 Task 依赖顺序执行（DAG）

| 属性 | 内容 |
|------|------|
| 任务ID | P1-1 |
| 优先级 | P1 |
| 负责人 | 待分配 |
| 预计工时 | 6h |
| 对应问题 | N9：Task 依赖未实现 |
| 依赖 | P0-2 |

**完成标准：**
- [ ] 解析 WorkflowTemplate.edges_json 构建 DAG
- [ ] Task 只在其所有前置 Task 完成后才进入 queued 状态
- [ ] 前置 Task 失败时根据 on_failure 策略处理（retry/skip/abort）
- [ ] DAG 环路检测
- [ ] 支持并行执行无依赖关系的 Task

**涉及文件：**
- `internal/orchestrator/orchestrator.go` — instantiateWorkflow 重写
- `internal/orchestrator/dag.go` — 新建 DAG 引擎

**开发报告：** `docs/reports/P1-1-dag-execution.md`

---

### P1-2：实现终端输出持久化

| 属性 | 内容 |
|------|------|
| 任务ID | P1-2 |
| 优先级 | P1 |
| 负责人 | 待分配 |
| 预计工时 | 3h |
| 对应问题 | N12：终端输出未持久化 |

**完成标准：**
- [ ] tmux 输出定期写入 `data/logs/{session_name}.log`
- [ ] 日志文件自动轮转（单文件大小限制）
- [ ] 日志保留天数配置生效
- [ ] 日志总量超限自动清理
- [ ] 输出解析器持续运行，结构化事件写入 events 表

**涉及文件：**
- `internal/terminal/manager.go` — 新增持久化方法
- `internal/scheduler/scheduler.go` — 集成日志采集

**开发报告：** `docs/reports/P1-2-terminal-persistence.md`

---

### P1-3：前端补全（Template 选择器 + Run 详情 + 时间线）

| 属性 | 内容 |
|------|------|
| 任务ID | P1-3 |
| 优先级 | P1 |
| 负责人 | 待分配 |
| 预计工时 | 6h |
| 对应问题 | N13, N14：前端页面缺失 |
| 影响标准 | S1, S5 |

**完成标准：**
- [ ] 创建 Run 时可选择 WorkflowTemplate（下拉框）
- [ ] 新增 Run 详情页 `/runs/[id]`
  - Task 列表 + 状态
  - 事件时间线
  - 资源使用趋势
- [ ] Run 列表页增加"查看详情"链接
- [ ] Task 操作按钮（retry/cancel）

**涉及文件：**
- `web/src/app/runs/page.tsx` — 增加 Template 选择器
- `web/src/app/runs/[id]/page.tsx` — 新建

**开发报告：** `docs/reports/P1-3-frontend-run-detail.md`

---

### P1-4：前端集成 React Flow 工作流可视化

| 属性 | 内容 |
|------|------|
| 任务ID | P1-4 |
| 优先级 | P1 |
| 负责人 | 待分配 |
| 预计工时 | 5h |
| 对应问题 | N2：缺少 React Flow |

**完成标准：**
- [ ] 安装 reactflow 依赖
- [ ] Run 详情页展示 Task DAG 图
- [ ] 节点颜色反映 Task 状态
- [ ] 点击节点查看 Task 详情
- [ ] 边（edge）反映依赖关系

**涉及文件：**
- `web/package.json` — 新增 reactflow 依赖
- `web/src/app/runs/[id]/workflow.tsx` — 新建

**开发报告：** `docs/reports/P1-4-react-flow.md`

---

## P2 — 质量保障与锦上添花（4 项）

### P2-1：挂载 pprof + .env 支持

| 属性 | 内容 |
|------|------|
| 任务ID | P2-1 |
| 优先级 | P2 |
| 负责人 | 待分配 |
| 预计工时 | 2h |
| 对应问题 | N10, N11 |

**完成标准：**
- [ ] `/debug/pprof/` 可访问
- [ ] 支持 `.env` 文件加载
- [ ] 配置优先级：config.yaml < .env < env vars

**涉及文件：**
- `internal/api/server.go` — 挂载 pprof
- `internal/config/config.go` — 添加 .env 加载

**开发报告：** `docs/reports/P2-1-pprof-env.md`

---

### P2-2：实现磁盘治理与进程树治理

| 属性 | 内容 |
|------|------|
| 任务ID | P2-2 |
| 优先级 | P2 |
| 负责人 | 待分配 |
| 预计工时 | 4h |
| 对应问题 | N16, N17 |

**完成标准：**
- [ ] workspace 超过 max_size_mb 时告警
- [ ] log_retention_days 日志文件自动清理
- [ ] max_total_log_size_mb 总量限制
- [ ] Agent 子进程数超过 max_child_processes_per_agent 时告警
- [ ] orphan 进程检测与回收

**涉及文件：**
- `internal/scheduler/scheduler.go` — 扩展 cleanup

**开发报告：** `docs/reports/P2-2-disk-process-governance.md`

---

### P2-3：数据库迁移版本管理

| 属性 | 内容 |
|------|------|
| 任务ID | P2-3 |
| 优先级 | P2 |
| 负责人 | 待分配 |
| 预计工时 | 3h |
| 对应问题 | N18 |

**完成标准：**
- [ ] 创建 schema_migrations 表记录版本号
- [ ] 支持增量迁移（UP）
- [ ] 迁移失败时回滚
- [ ] 启动时自动执行未应用的迁移

**涉及文件：**
- `internal/storage/database.go` — 重构迁移机制
- `internal/storage/migrations/` — 新建迁移目录

**开发报告：** `docs/reports/P2-3-migration-versioning.md`

---

### P2-4：集成测试与端到端测试

| 属性 | 内容 |
|------|------|
| 任务ID | P2-4 |
| 优先级 | P2 |
| 负责人 | 待分配 |
| 预计工时 | 6h |
| 依赖 | P0 全部完成 |

**完成标准：**
- [ ] 端到端测试：创建 Project → 创建 Run → Task 拆分 → Agent 启动 → 完成
- [ ] 降载与恢复流程测试
- [ ] Reconciler 对账测试
- [ ] 幂等性测试
- [ ] 长时间运行稳定性测试方案

**涉及文件：**
- `tests/integration/` — 新建

**开发报告：** `docs/reports/P2-4-integration-tests.md`

---

## 任务依赖关系

```
P0-1（幂等语义）         ─────────────────────────────────────┐
P0-2（Agent 启动流程）   ──┬──────────────────────────────────┤
                          ├── P0-3（降载策略）── P0-4（Checkpoint）├── P0-7（单元测试）
                          ├── P0-5（Watchdog）─────────────────┤
P0-6（xterm.js）         ──┴──────────────────────────────────┘
                                        │
                        ┌───────────────┤
                        ▼               ▼
              P1-1（DAG 执行）    P1-3（前端补全）
              P1-2（终端持久化）  P1-4（React Flow）
                                        │
                        ┌───────────────┤
                        ▼               ▼
              P2-1（pprof/.env）  P2-2（磁盘/进程治理）
              P2-3（迁移版本）    P2-4（集成测试）
```

---

## 工时汇总

| 优先级 | 任务数 | 预计总工时 |
|--------|--------|-----------|
| P0 | 7 项 | 39h |
| P1 | 4 项 | 20h |
| P2 | 4 项 | 15h |
| **合计** | **15 项** | **74h** |

---

## 任务完成与报告关联机制

每完成一项任务，需在 `docs/reports/` 目录下生成对应的开发报告文档。

### 报告命名规范
```
docs/reports/{任务ID}-{任务简称}.md
```

### 报告模板结构
见 `docs/reports/_template.md`

### 流程
1. 开始任务 → 更新本清单状态为 `🔄 进行中`
2. 完成任务 → 更新本清单状态为 `✅ 已完成`
3. 立即生成开发报告 → 在本清单中填入报告路径
4. 代码审查 → 确认报告内容与代码变更一致

---

## 状态追踪

| 任务ID | 状态 | 报告路径 |
|--------|------|----------|
| P0-1 | ✅ 已完成 | `docs/reports/P0-1-idempotent-semantics.md` |
| P0-2 | ✅ 已完成 | `docs/reports/P0-2-agent-launch-pipeline.md` |
| P0-3 | ✅ 已完成 | `docs/reports/P0-3-load-shedding-strategy.md` |
| P0-4 | ✅ 已完成 | `docs/reports/P0-4-checkpoint-recovery.md` |
| P0-5 | ✅ 已完成 | `docs/reports/P0-5-watchdog-heartbeat.md` |
| P0-6 | ✅ 已完成 | `docs/reports/P0-6-xtermjs-integration.md` |
| P0-7 | ✅ 已完成（与P2-4合并） | `docs/reports/P2-4-unit-tests.md` |
| P1-1 | ✅ 已完成 | `docs/reports/P1-1-dag-execution.md` |
| P1-2 | ✅ 已完成 | `docs/reports/P1-2-terminal-persistence.md` |
| P1-3 | ✅ 已完成 | `docs/reports/P1-3-frontend-run-detail.md` |
| P1-4 | ✅ 已完成 | `docs/reports/P1-4-reactflow-workflow.md` |
| P2-1 | ✅ 已完成 | `docs/reports/P2-1-pprof-env.md` |
| P2-2 | ✅ 已完成 | `docs/reports/P2-2-disk-process-governance.md` |
| P2-3 | ✅ 已完成 | `docs/reports/P2-3-migration-versioning.md` |
| P2-4 | ✅ 已完成 | `docs/reports/P2-4-unit-tests.md` |
