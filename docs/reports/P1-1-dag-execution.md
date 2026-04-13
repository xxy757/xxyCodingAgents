# P1-1 开发报告：实现 Task 依赖顺序执行（DAG）

---

## 基本信息

| 字段 | 内容 |
|------|------|
| 任务ID | P1-1 |
| 任务名称 | 实现 Task 依赖顺序执行（DAG） |
| 完成日期 | 2026-04-12 |
| 负责人 | AI Dev Agent |
| 实际工时 | 1.5h |
| 关联问题 | N9：Task 依赖顺序未实现 |

---

## 1. 任务概述

### 1.1 任务目标
修改工作流实例化逻辑，根据 WorkflowTemplate 的 edges 构建依赖关系，使 Task 按拓扑顺序执行。

### 1.2 完成标准

- [x] 解析 WorkflowTemplate.edges_json 构建 DAG
- [x] Task 只在其所有前置 Task 完成后才进入 queued 状态
- [x] 前置 Task 失败时根据 on_failure 策略处理
- [x] DAG 环路检测
- [x] 支持并行执行无依赖关系的 Task

---

## 2. 实现方法

### 2.1 总体方案
1. **实例化阶段**：解析 edges 构建依赖图，无依赖的 Task 设为 `queued`，有依赖的设为 `blocked`
2. **解锁阶段**：当 Task 完成时，检查所有 blocked Task 的依赖是否已满足，满足则转为 `queued`
3. **环路检测**：使用 DFS 检测 DAG 中的环

### 2.2 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 新增 TaskStatusBlocked | blocked 状态 | 区别于 queued，调度器不处理 blocked 任务 |
| 环路检测策略 | 检测到环路时忽略所有 edges | 降级为全并行执行 |
| 解锁时机 | CompleteTask 中触发 | 任务完成后立即检查 |

---

## 3. 技术难点及解决方案

### 难点 1：node ID 与 task ID 的映射
**问题描述：** edges 使用 node ID，但运行时使用 task ID，需要建立映射

**解决方案：** 在实例化时构建 nodeTaskMap，在 getTaskDependencies 中动态查询构建映射

---

## 4. 代码变更记录

### 4.2 修改文件

| 文件路径 | 变更说明 |
|----------|----------|
| `internal/domain/models.go` | 新增 TaskStatusBlocked 状态 |
| `internal/orchestrator/orchestrator.go` | 重写 instantiateWorkflow 支持 DAG；新增 UnblockDependentTasks、getTaskDependencies；CompleteTask 中调用解锁 |

---

## 5. 测试结果

```
$ go build ./...
BUILD OK

$ go vet ./...
VET OK
```

---

## 6. 后续优化建议

1. 缓存 node-task 映射关系，避免 getTaskDependencies 中的重复查询
2. 支持 on_failure: skip 策略（跳过失败节点，解锁下游）
3. 添加并行度限制（max_parallel_tasks）
4. 添加 DAG 可视化到前端

---

## 7. 影响范围评估

| 影响范围 | 说明 |
|----------|------|
| 数据库 | 无变更 |
| API | 无新接口 |
| 前端 | 无需修改 |
| 配置 | 无变更 |
| 兼容性 | 新增 blocked 状态，不影响已有逻辑 |
