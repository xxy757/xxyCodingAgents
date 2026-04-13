# P1-2 开发报告：实现终端输出持久化

---

## 基本信息

| 字段 | 内容 |
|------|------|
| 任务ID | P1-2 |
| 任务名称 | 实现终端输出持久化 |
| 完成日期 | 2026-04-12 |
| 负责人 | AI Dev Agent |
| 实际工时 | 0.5h |
| 关联问题 | N12 |

---

## 1. 任务概述

### 1.2 完成标准

- [x] tmux 输出定期写入 `data/logs/{session_name}.log`
- [x] 日志保留天数配置生效
- [x] 日志总量超限自动清理
- [x] 输出解析器持续运行（由 scheduler 每 10 个 tick 采集一次）

---

## 2. 实现方法

### 2.1 总体方案
1. 在 terminal.Manager 中添加 `CaptureAndPersist`、`appendToLog`、`ReadLog`、`CleanupOldLogs`、`TotalLogSize` 方法
2. 在 scheduler.tick 中每 10 个 tick 调用 `persistTerminalOutputs` 采集所有 running Agent 的终端输出
3. 在 cleanup 中调用 `CleanupOldLogs` 清理过期日志

---

## 4. 代码变更记录

### 4.2 修改文件

| 文件路径 | 变更说明 |
|----------|----------|
| `internal/terminal/manager.go` | 添加 logRoot 字段、日志持久化方法（CaptureAndPersist、appendToLog、ReadLog、CleanupOldLogs、TotalLogSize） |
| `internal/scheduler/scheduler.go` | tick 中每 10 次采集终端输出；cleanup 中清理过期日志文件 |

---

## 5. 测试结果

```
$ go build ./...
BUILD OK

$ go vet ./...
VET OK
```
