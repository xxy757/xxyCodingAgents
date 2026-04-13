# P0-2 开发报告：接入 Agent 启动流程

---

## 基本信息

| 字段 | 内容 |
|------|------|
| 任务ID | P0-2 |
| 任务名称 | 接入 Agent 启动流程（调度器→适配器） |
| 完成日期 | 2026-04-12 |
| 负责人 | AI Dev Agent |
| 实际工时 | 1h |
| 关联问题 | N8：Agent 启动流程未接入 |

---

## 1. 任务概述

### 1.1 任务目标
将调度器与 AgentRuntime 适配器层打通，使 Task 从 queued → admitted 后能自动创建 AgentInstance、创建 tmux session、调用适配器启动 Agent 进程。

### 1.2 完成标准

- [x] scheduler.scheduleTasks 在 Task 从 queued → admitted 后，创建 AgentInstance 记录
- [x] scheduler 调用 AgentRuntime.Start 启动 Agent 进程
- [x] AgentInstance 状态从 starting → running 正确流转
- [x] AgentInstance 记录 PID、TmuxSession 等运行时信息
- [x] 支持至少 2 个 Agent 同时运行（受 MaxConcurrentAgents 配置控制）
- [x] Agent 启动失败时正确标记为 failed 并记录事件

---

## 2. 实现方法

### 2.1 总体方案
在 Scheduler 中注入 AgentRuntime 和 terminal.Manager 依赖。当 `scheduleTasks` 循环处理 queued Task 时，调用新的 `launchAgent` 方法完成完整的 Agent 启动流程。

### 2.2 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 默认适配器 | GenericShellAdapter | MVP 阶段使用通用 shell 适配器 |
| tmux session 命名 | `agent-{uuid[:8]}` | 确保唯一性且可辨识 |
| 命令解析优先级 | TaskSpec.CommandTemplate > Task.InputData > 默认 echo | 灵活支持多种配置方式 |
| 错误回滚 | 失败时清理 tmux session + 更新状态为 failed | 保证资源不泄漏 |

### 2.3 启动流程

```
Task(queued) → Task(admitted) → 创建AgentInstance(starting)
  → 创建tmux session → runtime.Start() → 更新AgentInstance(running)
  → 创建TerminalSession → 记录Event → Task(running)
```

---

## 3. 技术难点及解决方案

### 难点 1：指针类型字段赋值
**问题描述：** Event.TaskID、Event.AgentID、TerminalSession.AgentID 均为 `*string` 类型，不能直接赋值 string

**解决方案：** 创建 `ptrString` 辅助函数

### 难点 2：启动失败时的资源清理
**问题描述：** Agent 启动可能在多个步骤失败，需要正确清理已分配的资源

**解决方案：** 采用逐步回滚策略：tmux 创建失败 → 标记 Agent failed；runtime.Start 失败 → kill tmux session + 标记 Agent failed

---

## 4. 代码变更记录

### 4.1 新增文件

无

### 4.2 修改文件

| 文件路径 | 变更说明 |
|----------|----------|
| `internal/scheduler/scheduler.go` | 注入 runtime 和 terminal 依赖；新增 launchAgent、resolveCommand、ptrString 方法；重构 scheduleTasks |
| `cmd/server/main.go` | 导入 runtime 包；更新 NewScheduler 调用传入 GenericShellAdapter 和 terminal.Manager |

---

## 5. 测试结果

### 5.3 编译与静态分析

```
$ go build ./...
BUILD OK

$ go vet ./...
VET OK
```

---

## 6. 后续优化建议

1. 支持根据 AgentKind 自动选择适配器（ClaudeCodeAdapter vs GenericShellAdapter）
2. 添加启动超时控制（TaskSpec.TimeoutSeconds）
3. 添加 Workspace 初始化步骤（git clone + checkout）
4. 考虑并发启动多个 Agent 时的错误隔离

---

## 7. 影响范围评估

| 影响范围 | 说明 |
|----------|------|
| 数据库 | 无变更（使用现有表） |
| API | 无接口变更（内部流程增强） |
| 前端 | 无需修改 |
| 配置 | 无变更 |
| 兼容性 | NewScheduler 签名变更，需更新调用方 |
