# AGENTS.md

This file provides guidance to Codex (Codex.ai/code) when working with code in this repository.

## 开发命令

```bash
# Go 后端
go build -o bin/ai-dev-platform ./cmd/server     # 编译
./bin/ai-dev-platform                              # 运行（默认配置）
go vet ./...                                       # 静态检查
go test ./...                                      # 全部测试（用内存 SQLite，无外部依赖）
go test -race ./...                                # 竞态检测
go test ./internal/api/ -v                         # 单包测试

# Next.js 前端
cd web && npm run dev                              # 开发服务器（:3000，API 代理到 :8080）
çΩΩ                        # 代码检查
```

前后端需同时运行才能使用完整控制台功能。`web/next.config.js` 在 dev 模式下将 `/api/*` 代理到 `http://localhost:8080/api/*`。

## 技术栈

- **后端**: Go 1.26.1，标准库 HTTP 路由（Go 1.22+ 模式匹配），`github.com/mattn/go-sqlite3`，`github.com/gorilla/websocket`，`gopkg.in/yaml.v3`
- **前端**: Next.js 16（App Router），React 19，Tailwind CSS 4，xterm.js，@xyflow/react
- **基础设施**: SQLite（WAL 模式）、tmux（PTY 终端管理）、gopsutil（资源指标）

## 架构总览

```
cmd/server/main.go          # 入口：加载配置 → 数据库 → 迁移 → 仓库 → 终端管理器 → 编排器 → 调度器 → HTTP 服务
internal/
  config/config.go           # YAML + .env + 环境变量（前缀 AI_DEV_）三层配置覆盖
  domain/models.go           # 核心领域模型（Project/Run/Task/AgentInstance/TerminalSession/Checkpoint 等）
  domain/specs.go            # 规约定义（TaskSpec/AgentSpec/WorkflowTemplate DAG）
  storage/database.go        # SQLite 连接 + 14 版迁移
  storage/repositories.go    # 13 个实体仓库的 CRUD（原生 SQL，无 ORM）
  api/server.go              # HTTP 服务，CORS + 日志中间件
  api/handlers.go            # ~25 个 API 端点（项目/Run/任务/Agent/终端/系统/规约）
  api/websocket.go           # WebSocket Pub/Sub 中心（终端实时输出）
  orchestrator/orchestrator.go  # Run/Task 生命周期，WorkflowTemplate→DAG 实例化，依赖解析，环检测
  scheduler/scheduler.go     # 3s 滴答调度：资源指标采集 → 压力分级 → 降载 → 任务准入 → 定期检查点
  scheduler/watchdog.go      # 15s 心跳/输出超时检测
  scheduler/reconciler.go    # 启动时状态修复（孤儿进程回收、状态不一致修复）
  runtime/adapter.go         # AgentRuntime 接口（Start/Pause/Resume/Stop/Checkpoint/Inspect）
  runtime/adapters.go        # ClaudeCodeAdapter + GenericShellAdapter
  terminal/manager.go        # tmux 会话生命周期 + 日志持久化
  workspace/git.go           # Git 工作区管理（clone/checkout/branch/commit/diff）
web/
  src/lib/api.ts             # 前端统一 API 客户端，与后端所有领域模型对应的 TS 接口
  src/app/page.tsx           # 仪表盘：5 秒轮询系统指标（内存/CPU/磁盘/压力）+ 健康状态
  src/app/layout.tsx         # 根布局：Sidebar + 主内容区
```

## 关键状态机

- **Run**: pending → running → completed / failed / cancelled
- **Task**: queued → admitted → running → completed / failed / cancelled / evicted；`blocked` 等待依赖
- **AgentInstance**: starting → running → paused → stopped / failed；异常态 `recoverable`（有检查点）和 `orphaned`（无匹配进程）
- **TerminalSession**: active → detached → closed

## 调度器压力分级

基于内存使用率的四级压力：NORMAL (<70%) → WARN (70-80%) → HIGH (80-88%，暂停低优先级可抢占 Agent) → CRITICAL (>88%，检查点后驱逐全部可抢占 Agent)。

## 代码约定

- 所有注释使用中文
- 实体 ID 统一使用 `github.com/google/uuid`
- 数据库操作使用参数化原生 SQL，可空字段用 `sql.Null*` 类型
- 错误用 `fmt.Errorf("%w", err)` 包装传递
- 日志用 `log/slog` 结构化输出
- HTTP 测试用 `httptest.ResponseRecorder`，数据库测试用内存 SQLite `:memory:`，无需外部依赖
- 测试文件对应的源文件：`xxx.go` → `xxx_test.go`（同目录）

## Skill routing

When the user's request matches an available skill, ALWAYS invoke it using the Skill
tool as your FIRST action. Do NOT answer directly, do NOT use other tools first.
The skill has specialized workflows that produce better results than ad-hoc answers.

Key routing rules:
- Product ideas, "is this worth building", brainstorming → invoke office-hours
- Bugs, errors, "why is this broken", 500 errors → invoke investigate
- Ship, deploy, push, create PR → invoke ship
- QA, test the site, find bugs → invoke qa
- Code review, check my diff → invoke review
- Update docs after shipping → invoke document-release
- Weekly retro → invoke retro
- Design system, brand → invoke design-consultation
- Visual audit, design polish → invoke design-review
- Architecture review → invoke plan-eng-review
- Save progress, checkpoint, resume → invoke checkpoint
- Code quality, health check → invoke health
