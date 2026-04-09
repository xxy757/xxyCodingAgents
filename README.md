# AI Dev Platform

一个面向**内部研发团队**的单机版 AI 开发编排平台。  
该平台运行在 **Mac mini（16GB）** 等本地开发机器上，通过 **Web 控制台 + Go 后端 + SQLite + tmux/PTY + Claude/GLM 类代码 Agent** 的方式，实现多终端、多 Agent、长时间运行、任务编排、资源调度、容错恢复等能力。

当前版本目标是构建一个 **MVP（最小可用版本）**，优先解决以下问题：

- 用 Web 界面统一管理 AI 开发任务
- 启动、托管、控制多个代码 Agent
- 支持多个终端会话与日志回放
- 支持任务拆分、排队、运行、暂停、恢复、终止
- 在资源有限的机器上实现稳定运行
- 在内存/CPU 紧张时自动降载、暂停、驱逐、恢复
- 为后续基于 **GLM-5.1** 的 AI 辅助开发流程提供统一运行底座

---

## 1. 项目定位

本项目不是一个公开的 SaaS 平台，也不是一个“聊天机器人外壳”。  
它的定位更接近于：

> **内部使用的单机版 AI 软件开发调度台**

它负责把如下能力统一起来：

- 任务管理
- Agent 生命周期管理
- 多终端管理
- AI 编码工具托管
- 工作区隔离
- Git 操作
- 测试执行
- 资源调度
- 审计日志
- 容错恢复

平台优先服务于以下场景：

- 需求分析
- 代码生成/修改
- 自动化测试
- 代码审查
- 问题修复
- 长时间后台运行的研发任务

---

## 2. 第一版目标

第一版只做 **单机 MVP**，不追求“平台大而全”，重点追求：

- 架构简单
- 稳定性优先
- 资源可控
- 可恢复
- 易于在本地开发和迭代

### 第一版必须实现

- Web 前端控制台
- Go 单体后端
- SQLite 持久化
- 本地工作区管理
- tmux + PTY 终端托管
- 多 Agent 任务运行
- 资源调度器（内存/CPU 感知）
- checkpoint 恢复机制
- 基础日志与审计

### 第一版暂不实现

- Kubernetes
- 微服务化
- 分布式调度
- 多租户
- 复杂权限体系
- 云端部署抽象
- 向量数据库
- 大规模插件生态
- 公网开放能力

---

## 3. 技术选型

### 后端
- **Go**
- 原因：
  - 常驻服务内存占用更低
  - 并发调度能力更适合本项目
  - 更适合实现守护进程、资源监控、终端控制、进程管理
  - 单二进制部署简单，适合本地机器长期运行

### 前端
- **Web 前端**
- 推荐：
  - Next.js
  - Tailwind CSS
  - xterm.js（终端展示）
  - React Flow（工作流/Agent 关系图）

### 数据存储
- **SQLite**
- 原因：
  - 零运维
  - 适合单机内部工具
  - 足够支撑 MVP 的运行状态、事件日志、checkpoint 元数据存储

### 终端运行
- **tmux + PTY**
- 用于实现：
  - 多终端会话
  - attach/detach
  - 输出流采集
  - 长时间运行 session 保活

### AI 模型层
- **GLM-5.1**
- 本项目后续开发将基于 **GLM-5.1** 进行辅助生成、编码、规划、审查等工作。
- 在系统设计上，模型能力应被视为 **可替换的 Agent 推理引擎**，而不是平台本身。

---

## 4. 总体架构

```text
Web Frontend
    |
    | HTTP / WebSocket
    v
Go Monolith Backend
 ├─ API Layer
 ├─ Orchestrator
 ├─ Resource Scheduler / Supervisor
 ├─ Agent Runtime Manager
 ├─ Terminal Manager
 ├─ Workspace / Git Manager
 └─ Audit / Event Log
    |
    +-- SQLite
    +-- tmux / PTY
    +-- Local Filesystem
    +-- External Tools (Git / Test / AI CLI)
```

---

## 5. 核心模块

### 5.1 API Layer
负责：
- REST API
- WebSocket 推送
- 项目、任务、Agent、终端等对象的对外访问入口

### 5.2 Orchestrator
负责：
- Run / Task 状态机推进
- 任务拆分
- Agent 分配
- 执行顺序控制
- 简单工作流模板编排

### 5.3 Resource Scheduler / Supervisor
负责：
- 系统资源采样
- Agent 准入控制
- 并发数量限制
- 内存/CPU 压力判断
- 暂停/驱逐/恢复 Agent
- 看门狗机制

### 5.4 Agent Runtime Manager
负责：
- 启动/停止 Agent
- 托管 AI 编码进程
- 记录心跳
- checkpoint 保存与恢复

### 5.5 Terminal Manager
负责：
- 管理 tmux session/pane
- 管理 PTY
- 终端 attach/detach
- 终端输出流实时推送

### 5.6 Workspace / Git Manager
负责：
- 创建工作区
- checkout 分支
- 管理任务独立 workspace
- 读取 diff / status
- 提交变更

### 5.7 Audit / Event Log
负责：
- 写入结构化事件日志
- 保存原始终端输出日志
- 为恢复、排障、审计提供依据

---

## 6. MVP 的运行对象

### Project
表示一个代码仓库项目。

### Run
表示一次完整执行，例如：
- “实现退款接口”
- “修复某个 bug 并补测试”

### Task
表示 Run 内的一个子任务，例如：
- planner
- coder
- tester
- reviewer

### AgentInstance
表示一个运行中的 Agent 实例。

### Workspace
表示某个任务对应的独立工作目录。

### TerminalSession
表示某个 tmux/PTY 终端会话。

---

## 7. 推荐的最小工作流

第一版建议只保留 4 类 Agent：

- **Planner**：负责理解需求与输出任务目标
- **Coder**：负责写代码/改代码
- **Tester**：负责运行测试与反馈失败信息
- **Reviewer**：负责检查代码质量和问题

### 最小工作流

```text
需求输入
  ↓
Planner
  ↓
Coder
  ↓
Tester
  ↓
Reviewer
  ↓
结果收敛
```

第一版先不要做过多复杂角色拆分。

---

## 8. 长时间运行设计原则

本系统的设计重点之一是：

> **在本地资源有限的机器上，让多个 AI Agent 可以长时间稳定运行。**

为实现这一点，必须具备以下机制：

### 8.1 心跳机制
每个运行中的 Agent 定期上报：
- 当前状态
- 最近输出时间
- 最近心跳时间
- 资源占用信息

### 8.2 Checkpoint
系统在以下时机写 checkpoint：
- 周期性保存
- 阶段完成时保存
- 驱逐前保存
- 异常恢复前保存

### 8.3 Watchdog
监控以下异常：
- 长时间无输出
- 心跳超时
- 进程崩溃
- 终端断开

### 8.4 自动恢复
资源恢复后，系统可以：
- 从 checkpoint 恢复任务
- 重建终端会话
- 重新启动受管 Agent

---

## 9. 资源调度策略

由于平台运行在 **Mac mini 16GB** 上，必须采用资源感知调度。

### 9.1 设计原则
- 不是追求最多并发
- 而是追求“有限资源下的稳定吞吐”

### 9.2 建议默认限制
- 同时活跃 Agent：**2 个**
- 同时重型任务：**1 个**
- 同时测试任务：**1 个**

### 9.3 内存压力分级
- **NORMAL**：< 70%
- **WARN**：70% - 80%
- **HIGH**：80% - 88%
- **CRITICAL**：> 88%

### 9.4 对应策略
#### NORMAL
- 正常准入
- 正常恢复排队任务

#### WARN
- 停止重型任务准入
- 新任务进入队列

#### HIGH
- 暂停低优先级 Agent
- 暂停非关键任务
- 禁止新任务进入运行态

#### CRITICAL
- 先 checkpoint
- 再驱逐/终止低优先级且可恢复 Agent
- 保证主控服务存活

### 9.5 关键设计要求
系统不是“内存超了就直接乱杀进程”，而是：

1. 先判断优先级
2. 先暂停再驱逐
3. 驱逐前尽量保存 checkpoint
4. 资源恢复后再继续执行

---

## 10. 目录结构建议

```text
ai-dev-platform/
├─ cmd/
│  └─ server/
│     └─ main.go
├─ internal/
│  ├─ api/
│  ├─ app/
│  ├─ domain/
│  ├─ service/
│  ├─ orchestrator/
│  ├─ scheduler/
│  ├─ runtime/
│  ├─ git/
│  ├─ storage/
│  ├─ monitor/
│  ├─ audit/
│  └─ util/
├─ web/
├─ data/
│  ├─ app.db
│  ├─ logs/
│  ├─ checkpoints/
│  └─ workspaces/
└─ configs/
   └─ config.yaml
```

---

## 11. 数据存储建议

SQLite 中至少应包含以下对象：

- projects
- runs
- tasks
- agent_instances
- workspaces
- terminal_sessions
- checkpoints
- resource_snapshots
- events
- command_logs

建议开启：
- WAL 模式
- foreign key
- busy timeout

---

## 12. 配置建议（MVP）

```yaml
server:
  http_addr: ":8080"

runtime:
  workspace_root: "./data/workspaces"
  log_root: "./data/logs"
  checkpoint_root: "./data/checkpoints"

scheduler:
  tick_seconds: 3
  max_concurrent_agents: 2
  max_heavy_agents: 1
  max_test_jobs: 1

thresholds:
  warn_memory_percent: 70
  high_memory_percent: 80
  critical_memory_percent: 88

timeouts:
  heartbeat_timeout_seconds: 30
  stall_timeout_seconds: 900
  checkpoint_interval_seconds: 30

sqlite:
  path: "./data/app.db"
  wal_mode: true
  busy_timeout_ms: 5000
```

---

## 13. API 方向（MVP）

### 项目
- `POST /api/projects`
- `GET /api/projects`
- `GET /api/projects/:id`

### Run
- `POST /api/runs`
- `GET /api/runs/:id`
- `GET /api/runs/:id/timeline`

### Task
- `GET /api/runs/:id/tasks`
- `POST /api/tasks/:id/retry`
- `POST /api/tasks/:id/cancel`

### Agent
- `GET /api/agents/:id`
- `POST /api/agents/:id/pause`
- `POST /api/agents/:id/resume`
- `POST /api/agents/:id/stop`

### Terminal
- `POST /api/terminals`
- `GET /api/terminals/:id`
- `GET /api/terminals/:id/ws`

### Resource
- `GET /api/system/metrics`
- `GET /api/runs/:id/resources`

---

## 14. 开发原则

### 14.1 优先顺序
开发顺序建议如下：

1. SQLite 与基础表结构
2. REST API 与基础对象管理
3. tmux + PTY 终端托管
4. Claude/GLM 类 Agent 受管启动
5. 资源调度器
6. Checkpoint 与恢复
7. 最小工作流编排

### 14.2 架构原则
- 单机优先
- 简单优先
- 稳定优先
- 可恢复优先
- 人工可接管优先

### 14.3 明确不做
在 MVP 阶段，不做以下事情：
- 复杂平台抽象
- 大规模插件系统
- 过早微服务拆分
- 过度通用化设计

---

## 15. 与 GLM-5.1 的关系

本项目后续开发会使用 AI 辅助完成代码生成、模块实现、文档编写、测试编写等工作，并以 **GLM-5.1** 作为重要模型能力来源之一。

在系统设计上，建议遵循以下原则：

1. **模型层可替换**
   - 平台不应绑定单一模型供应商
   - GLM-5.1 是当前主要使用模型，但运行架构不应写死

2. **模型只负责推理，不负责平台控制**
   - 任务调度、资源控制、终端管理必须由平台后端控制
   - 不把“是否 kill、是否恢复、是否排队”等核心控制逻辑交给模型

3. **AI 输出必须可审计**
   - 所有关键决策、命令、输出摘要都应记录

4. **AI 开发流程必须可中断/恢复**
   - 避免一次长链调用失控

---

## 16. 当前 MVP 成功标准

当以下目标达成时，可以认为第一版 MVP 完成：

- 可以通过 Web 创建一个 Run
- 可以为 Run 创建 Task
- 可以启动至少 2 个受管 Agent
- 可以在 Web 中看到终端输出
- 可以查看事件时间线
- 可以在内存压力升高时暂停/驱逐低优先级 Agent
- 被驱逐任务可以从 checkpoint 恢复
- 系统可以持续运行较长时间而不失控

---

## 17. 后续演进方向

MVP 完成后，未来可以逐步增加：

- 更丰富的 Agent 角色
- 更强的 Git 流程集成
- 更细的命令审批机制
- 更智能的任务拆分与重试策略
- 更完善的日志检索和运行分析
- 更丰富的模型适配层
- 更强的本地/远程执行隔离

但这些都应建立在：

> **MVP 先稳定可用**

的前提上。

---

## 18. 许可证与使用范围

本项目当前定位为：

- 内部工具
- 单机部署
- 非公开服务

是否开源、是否开放公共使用、是否支持多租户，由后续阶段再决定。

---

## 19. 一句话总结

这是一个：

> **基于 Go + Web + SQLite + tmux/PTY 的单机版 AI 开发调度平台 MVP**

它面向内部研发使用，运行在本地 Mac mini 上，强调：

- 多 Agent 执行
- 长时间稳定运行
- 资源感知调度
- 容错与恢复
- 为后续基于 **GLM-5.1** 的 AI 开发流程提供可靠底座

