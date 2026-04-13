# P0-1 开发报告：实现幂等语义

---

## 基本信息

| 字段 | 内容 |
|------|------|
| 任务ID | P0-1 |
| 任务名称 | 实现幂等语义（pause/resume/stop） |
| 完成日期 | 2026-04-12 |
| 负责人 | AI Dev Agent |
| 实际工时 | 0.5h |
| 关联问题 | N3：幂等语义未完整实现 |

---

## 1. 任务概述

### 1.1 任务目标
使 `POST /api/agents/:id/pause`、`POST /api/agents/:id/resume`、`POST /api/agents/:id/stop` 三个接口对重复请求具有幂等性——当 Agent 已经处于目标状态时，应返回 200 而非 400 错误。

### 1.2 完成标准

- [x] `POST /api/agents/:id/pause`：Agent 已 paused 时返回 200（非 400）
- [x] `POST /api/agents/:id/resume`：Agent 已 running 时返回 200（非 400）
- [x] `POST /api/agents/:id/stop`：Agent 已 stopped/failed 时返回 200
- [x] 对不可转换状态（如 starting → paused）仍返回 400

---

## 2. 实现方法

### 2.1 总体方案
在每个 handler 中，在检查"当前状态是否允许转换"之前，先检查"当前状态是否已经是目标状态"。如果是目标状态，直接返回 200 和当前 Agent 数据。

### 2.2 关键设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| 幂等时返回的数据 | 返回当前 Agent 数据 | 前端可直接使用，无需额外 GET |
| stop 的终态扩展 | stopped + failed 均视为终态 | failed 状态再 stop 也应幂等成功 |
| 状态检查顺序 | 先幂等检查 → 再合法性检查 | 避免幂等请求被误判为非法 |

---

## 3. 技术难点及解决方案

### 难点 1：幂等 vs 非法的边界
**问题描述：** 需要区分"已经是目标状态"（幂等）和"当前状态不允许转换到目标状态"（非法）。

**解决方案：** 采用两阶段判断：
1. 先检查 `agent.Status == 目标状态` → 返回 200（幂等）
2. 再检查 `agent.Status != 允许的源状态` → 返回 400（非法）

---

## 4. 代码变更记录

### 4.2 修改文件

| 文件路径 | 变更说明 |
|----------|----------|
| `internal/api/handlers.go` | handlePauseAgent：增加已 paused 状态的幂等返回（第 299~302 行） |
| `internal/api/handlers.go` | handleResumeAgent：增加已 running 状态的幂等返回（第 326~329 行） |
| `internal/api/handlers.go` | handleStopAgent：扩展终态判断，failed 状态也幂等返回（第 350 行） |

---

## 5. 测试结果

### 5.3 编译与静态分析

```
$ go build ./...
# 通过（无输出）

$ go vet ./...
# 通过（无输出）
```

### 5.2 手动验证

| 验证项 | 结果 | 备注 |
|--------|------|------|
| 已 paused 的 Agent 再次 pause | ✅ | 返回 200 + agent 数据 |
| 已 running 的 Agent 再次 resume | ✅ | 返回 200 + agent 数据 |
| 已 stopped 的 Agent 再次 stop | ✅ | 返回 200 + agent 数据 |
| 已 failed 的 Agent 再次 stop | ✅ | 返回 200 + agent 数据 |
| starting 状态的 Agent pause | ✅ | 返回 400（非法转换） |

---

## 6. 后续优化建议

1. 补充单元测试：使用 httptest 模拟请求验证各种状态组合
2. 考虑添加 `X-Idempotent: true` 响应头，让调用方区分"本次操作生效"和"因幂等跳过"
3. 考虑为幂等请求添加 event 日志记录

---

## 7. 影响范围评估

| 影响范围 | 说明 |
|----------|------|
| 数据库 | 无变更 |
| API | 行为变更：pause/resume/stop 在目标状态下返回 200 而非 400 |
| 前端 | 无需修改，前端重试逻辑将更可靠 |
| 配置 | 无变更 |
| 兼容性 | 向后兼容（只是放宽了成功条件） |
