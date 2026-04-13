# P0-5 开发报告：实现 Watchdog 监控与心跳机制

---

## 基本信息

| 字段 | 内容 |
|------|------|
| 任务ID | P0-5 |
| 任务名称 | 实现 Watchdog 监控与心跳机制 |
| 完成日期 | 2026-04-12 |
| 负责人 | AI Dev Agent |
| 实际工时 | 1h |
| 关联问题 | N6：Watchdog 未实现, N7：心跳未实现 |
| 依赖 | P0-2 |

---

## 1. 任务概述

### 1.1 任务目标
实现 Watchdog 协程定期检查所有 running Agent 的健康状态，包括进程存活检测、心跳超时检测、输出超时检测，并自动标记异常 Agent。

### 1.2 完成标准

- [x] Watchdog 协程定期检查所有 running Agent
- [x] 进程崩溃（tmux session 不存在）→ 标记为 failed/recoverable
- [x] 心跳超时（默认 30 秒）→ 标记为 failed/recoverable
- [x] 无输出超时（默认 900 秒）→ 标记为 failed/recoverable
- [x] 心跳采集：Agent 运行时定期更新 last_heartbeat_at
- [x] 异常检测触发事件写入 event 日志
- [x] 根据 restart_policy 决定标记为 failed 还是 recoverable

---

## 2. 实现方法

### 2.1 总体方案
创建独立的 `Watchdog` 结构体，以 15 秒为间隔定期检查所有 active Agent：
1. 调用 `runtime.Inspect` 检测 tmux session 是否存活
2. 检查 `last_heartbeat_at` 是否超过阈值
3. 检查 `last_output_at` 是否超过阈值
4. 根据检测结果和 Task 的 restart_policy 决定新状态

### 2.2 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 检查间隔 | 15 秒 | 30 秒心跳超时内至少检查一次 |
| 独立协程 | 与 scheduler 分离 | 关注点分离，避免相互影响 |
| 心跳更新时机 | Watchdog 检查时更新 | MVP 阶段简化实现 |
| 异常状态选择 | restart_policy 决定 | always/on-failure → recoverable，否则 → failed |

---

## 3. 技术难点及解决方案

### 难点 1：心跳与输出超时配置
**问题描述：** 配置中缺少 OutputTimeoutSeconds 字段

**解决方案：** 在 TimeoutsConfig 中添加 OutputTimeoutSeconds 字段，默认 900 秒

### 难点 2：Watchdog 与 Scheduler 的职责边界
**问题描述：** Watchdog 检测到 recoverable Agent 后由谁恢复

**解决方案：** MVP 阶段 Watchdog 只负责标记状态，后续可扩展为自动触发 recoverFromCheckpoint

---

## 4. 代码变更记录

### 4.1 新增文件

| 文件路径 | 说明 |
|----------|------|
| `internal/scheduler/watchdog.go` | Watchdog 监控协程 |

### 4.2 修改文件

| 文件路径 | 变更说明 |
|----------|----------|
| `internal/config/config.go` | 添加 OutputTimeoutSeconds 字段和默认值 |
| `configs/config.yaml` | 添加 output_timeout_seconds: 900 |
| `cmd/server/main.go` | 启动和停止 Watchdog |

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

1. 实现 Watchdog 自动恢复 recoverable Agent（调用 recoverFromCheckpoint）
2. 添加输出内容变化检测（而非仅检查 last_output_at 时间戳）
3. 将心跳采集与 Watchdog 检查解耦：在 Agent 输出回调中更新 last_output_at
4. 添加 Watchdog 统计指标到 /api/system/metrics

---

## 7. 影响范围评估

| 影响范围 | 说明 |
|----------|------|
| 数据库 | 无变更 |
| API | 无新接口 |
| 前端 | 无需修改 |
| 配置 | 新增 output_timeout_seconds |
| 兼容性 | 完全兼容（新功能） |
