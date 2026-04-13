# P0-3 开发报告：实现资源降载策略

---

## 基本信息

| 字段 | 内容 |
|------|------|
| 任务ID | P0-3 |
| 任务名称 | 实现资源降载策略（WARN/HIGH/CRITICAL） |
| 完成日期 | 2026-04-12 |
| 负责人 | AI Dev Agent |
| 实际工时 | 1h |
| 关联问题 | N4：降载策略不完整 |
| 依赖 | P0-2 |

---

## 1. 任务概述

### 1.1 任务目标
在调度器的 tick 循环中根据资源压力等级执行不同的降载操作，防止系统资源耗尽。

### 1.2 完成标准

- [x] WARN 级别：停止重型任务（resource_class=heavy）准入
- [x] HIGH 级别：暂停 preemptible=true + 低 priority 的 Agent
- [x] CRITICAL 级别：先 checkpoint，再驱逐低优先级且可恢复 Agent
- [x] 资源恢复后按 priority 排序恢复排队任务（NORMAL 级别正常调度）
- [x] 降载操作写入 event 日志
- [x] 压力等级变更写入 event 日志

---

## 2. 实现方法

### 2.1 总体方案
在 tick 方法中，根据 `determinePressure` 返回的压力等级执行不同策略：

| 压力等级 | 调度策略 | 降载操作 |
|----------|----------|----------|
| NORMAL | 全量调度 | 无 |
| WARN | 仅准入 light/medium 任务 | 无 |
| HIGH | 停止准入 | 暂停 preemptible+low priority Agent |
| CRITICAL | 停止准入 | Checkpoint → 驱逐 preemptible Agent |

### 2.2 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 降载顺序 | 先降载再调度 | 避免刚准入就被降载 |
| HIGH 级别只暂停不驱逐 | 暂停 tmux 进程 | 资源恢复后可快速 resume |
| CRITICAL 先 checkpoint | 保存状态后驱逐 | 保证可恢复性 |
| eviction 使用 TaskStatusEvicted | 新状态 | 区别于 failed，支持后续恢复 |

---

## 3. 技术难点及解决方案

### 难点 1：降载与调度的协调
**问题描述：** 降载操作释放的资源应立即可用于调度，但同一 tick 内两者存在先后顺序

**解决方案：** 采用"先降载后调度"策略。降载在 tick 开头执行，释放资源后 activeCount 可能减少，但当前实现保守地不在同一 tick 内立即利用释放的资源，而是在下一个 tick 重新评估

### 难点 2：checkpoint 与 eviction 的原子性
**问题描述：** checkpoint 可能失败，此时是否应继续驱逐

**解决方案：** checkpoint 失败只记录 warning，不阻止驱逐。在 CRITICAL 级别下系统稳定性优先于单个任务的可恢复性

---

## 4. 代码变更记录

### 4.2 修改文件

| 文件路径 | 变更说明 |
|----------|----------|
| `internal/scheduler/scheduler.go` | 新增 handleLoadShedding、pauseLowPriorityAgents、evictAgents、scheduleTasksLightOnly 方法；修改 tick 方法集成降载逻辑 |

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

1. 实现被驱逐任务的自动恢复：当压力回到 NORMAL 时，将 evicted 任务重新排队
2. 添加降载冷却时间：避免在压力临界值附近频繁降载/恢复
3. 支持渐进式降载：HIGH 级别先暂停一部分，而非一次全部暂停
4. 添加降载指标到 `/api/system/metrics`

---

## 7. 影响范围评估

| 影响范围 | 说明 |
|----------|------|
| 数据库 | 无变更 |
| API | 无接口变更 |
| 前端 | 无需修改 |
| 配置 | 使用已有阈值配置 |
| 兼容性 | 完全兼容 |
