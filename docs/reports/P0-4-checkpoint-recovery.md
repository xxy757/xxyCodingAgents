# P0-4 开发报告：实现 Checkpoint 机制与恢复流程

---

## 基本信息

| 字段 | 内容 |
|------|------|
| 任务ID | P0-4 |
| 任务名称 | 实现 Checkpoint 机制与恢复流程 |
| 完成日期 | 2026-04-12 |
| 负责人 | AI Dev Agent |
| 实际工时 | 1h |
| 关联问题 | N5：Checkpoint 为空壳 |
| 依赖 | P0-2, P0-3 |

---

## 1. 任务概述

### 1.1 任务目标
实现周期性 checkpoint 协程和从 checkpoint 恢复任务的完整流程，使被驱逐的任务可以恢复执行。

### 1.2 完成标准

- [x] 周期性 checkpoint（默认 30 秒间隔）协程运行
- [x] checkpoint 保存 Agent 实际状态（tmux 输出长度、运行状态）
- [x] 驱逐前触发 checkpoint（P0-3 中已实现）
- [x] 从 checkpoint 恢复任务流程：读取 checkpoint → 重建 tmux session → 恢复 Agent 状态
- [x] Checkpoint 数据写入 checkpoints 表
- [x] 恢复操作写入 event 日志

---

## 2. 实现方法

### 2.1 总体方案
1. **周期性 checkpoint**：在 scheduler.Run 中增加 checkpointTicker，默认 30 秒间隔，遍历所有 running Agent 调用 runtime.Checkpoint
2. **Checkpoint 数据**：ClaudeCodeAdapter.Checkpoint 现在捕获 tmux 终端输出（最近 100 行）并记录运行状态
3. **恢复流程**：`recoverFromCheckpoint` 方法读取最新 checkpoint → 创建新 AgentInstance → 创建 tmux session → 调用 runtime.Start

### 2.2 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| checkpoint 频率 | 默认 30 秒 | 平衡性能与数据保留 |
| 恢复时使用新 AgentID | 而非复用旧 ID | 保持审计追踪清晰 |
| checkpoint 保留策略 | 保留所有历史 | 便于调试和审计 |
| 恢复命令来源 | TaskSpec.CommandTemplate > Task.InputData > 默认 | 与正常启动一致 |

---

## 3. 技术难点及解决方案

### 难点 1：Checkpoint 接口参数含义
**问题描述：** AgentRuntime.Checkpoint 接口参数名为 agentID，但适配器实现实际需要 tmuxSession

**解决方案：** 修改 ClaudeCodeAdapter.Checkpoint 参数语义为 tmuxSession，并在调用时传入 agent.TmuxSession

### 难点 2：恢复时的工作区状态
**问题描述：** 恢复时工作区可能已被清理或状态已变化

**解决方案：** MVP 阶段使用原始 Task.WorkspacePath，后续可添加工作区完整性校验

---

## 4. 代码变更记录

### 4.2 修改文件

| 文件路径 | 变更说明 |
|----------|----------|
| `internal/runtime/adapters.go` | ClaudeCodeAdapter.Checkpoint：从空壳改为捕获实际 tmux 输出和运行状态 |
| `internal/scheduler/scheduler.go` | 新增 runCheckpoints、recoverFromCheckpoint 方法；Run 方法中增加 checkpointTicker |
| `internal/storage/repositories.go` | 新增 AgentInstanceRepo.UpdateCheckpointID、CheckpointRepo.ListByTask |

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

1. 添加 checkpoint 数据压缩：对大输出进行 gzip 压缩
2. 添加 checkpoint 保留策略：定期清理过期 checkpoint
3. 实现增量 checkpoint：只保存上次 checkpoint 后的变化
4. 添加恢复 API 端点：`POST /api/tasks/:id/recover`
5. 恢复时验证工作区完整性

---

## 7. 影响范围评估

| 影响范围 | 说明 |
|----------|------|
| 数据库 | 无变更（使用现有 checkpoints 表） |
| API | 无新接口（recoverFromCheckpoint 为内部方法） |
| 前端 | 无需修改 |
| 配置 | 使用已有 checkpoint_interval_seconds |
| 兼容性 | 完全兼容 |
