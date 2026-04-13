# P2-4: 编写核心模块单元测试

## 任务概述

| 项目 | 内容 |
|------|------|
| 任务编号 | P2-4 |
| 优先级 | P2 (中) |
| 状态 | ✅ 已完成 |
| 预计工时 | 6h |
| 关联任务 | 所有 P0/P1 任务 |

## 实现方法

为 4 个核心包编写单元测试，采用不同测试策略：

### 1. config 包 — 纯单元测试

使用直接构造 `Config` 结构体测试 `setDefaults`、`applyEnvOverrides`、Duration 方法。使用 `t.TempDir()` 创建临时 YAML 文件测试 `Load` 函数。

### 2. scheduler 包 — 纯逻辑测试 + Mock 集成

- **CanAdmit / determinePressure**: 直接构造 `Config` 手动填充阈值，测试纯逻辑分支
- **Reconciler**: 使用 mock `TerminalChecker` 接口，配合内存 SQLite 数据库测试状态转换逻辑

### 3. storage 包 — 集成测试

使用 SQLite 内存数据库 (`:memory:`) 运行完整迁移后测试所有 13 个 Repository 的 CRUD 操作。

### 4. orchestrator 包 — 集成测试

使用内存 SQLite 测试完整的业务流程：创建 Run → 实例化工作流 → 完成任务 → 解除阻塞 → 完成/失败 Run。

## 技术难点及解决方案

### 难点1: setDefaults 是未导出方法

`Config.setDefaults()` 是小写开头的私有方法，外部测试包无法直接调用。

**解决方案**: config 包的测试文件在同一个包内，可以直接访问。scheduler 包测试改为手动构造完整的 `Config` 结构体，填充与 `setDefaults` 相同的默认值。

### 难点2: determinePressure 阈值计算复杂

磁盘 High 阈值使用中间值公式 `DiskWarnPercent + (DiskHighPercent - DiskWarnPercent) / 2 = 85`，不是简单的 `DiskHighPercent`。

**解决方案**: 仔细分析 `determinePressure` 源码中的三层条件判断，分别用精确的边界值测试每种压力级别。修正了初始错误的测试预期值。

### 难点3: 外键约束导致测试数据依赖

SQLite 外键约束要求插入顺序：Project → Run → Task → Agent → Checkpoint。

**解决方案**: 创建辅助函数 `setupRepoTestDB`、`seedProjectAndTemplate` 封装基础数据创建，确保外键依赖链完整。

### 难点4: CurrentVersion 在迁移前表不存在

`CurrentVersion` 查询 `schema_migrations` 表，但该表在 `RunMigrations` 之前不存在。

**解决方案**: 移除"迁移前版本为0"的测试用例，仅在迁移完成后验证版本号。

## 测试结果

| 包 | 测试数 | 通过 | 失败 | 耗时 |
|---|---|---|---|---|
| `config` | 10 | 10 | 0 | 0.14s |
| `orchestrator` | 7 | 7 | 0 | 0.13s |
| `scheduler` | 17 | 17 | 0 | 0.20s |
| `storage` | 16 | 16 | 0 | 0.32s |
| **合计** | **50** | **50** | **0** | **0.79s** |

## 代码变更记录

### 新增文件

| 文件 | 测试数 | 测试内容 |
|------|--------|----------|
| `internal/config/config_test.go` | 10 | setDefaults (20+默认值)、不覆盖已有值、Duration 方法、Load YAML、环境变量覆盖 |
| `internal/scheduler/scheduler_test.go` | 12 | CanAdmit (6场景)、determinePressure (7阈值场景)、PressureLevel 常量、handleLoadShedding |
| `internal/scheduler/reconciler_test.go` | 5 | tmux存活→running、tmux死亡→failed、有checkpoint→recoverable、paused保持、无agent |
| `internal/storage/database_test.go` | 4 | 内存DB连接、迁移创建14表、幂等迁移、版本号 |
| `internal/storage/repositories_test.go` | 12 | Project/Run/Task/Agent/Event/Checkpoint/ResourceSnapshot/WorkflowTemplate/TaskSpec/TerminalSession CRUD、ListActiveWithTasks |
| `internal/orchestrator/orchestrator_test.go` | 7 | 无模板创建Run、有模板创建Run、完成任务解除阻塞、全部完成终结Run、失败abort策略、失败continue策略、无依赖阻塞任务 |

### 关键测试覆盖

**CanAdmit 函数** (6个测试):
- 低于限制 → 通过
- 等于限制 → 拒绝
- 超过限制 → 拒绝
- Heavy任务低于Heavy限制 → 通过
- Heavy任务达到Heavy限制 → 拒绝
- Light任务不受Heavy限制影响 → 通过

**determinePressure 函数** (7个测试):
- mem=50, disk=50 → Normal
- mem=75 → Warn
- disk=80 → Warn
- mem=85 → High
- disk=85 → High (中间阈值)
- mem=90 → Critical
- disk=95 → Critical

**Reconciler** (5个测试):
- tmux alive + running → 保持 running
- tmux dead + starting → 转为 failed
- tmux dead + 有 checkpoint → 转为 recoverable
- tmux alive + paused → 保持 paused
- 无活跃 agent → 无错误

## 后续优化建议

1. **增加 API Handler 测试**: 使用 `httptest` 包测试 HTTP 端点的请求/响应
2. **增加 Watchdog 测试**: mock AgentRuntime 接口测试心跳超时和输出超时检测
3. **增加竞态测试**: 使用 `-race` 标志运行测试检测并发问题
4. **增加基准测试**: 为 Repository CRUD 添加 Benchmark 测试
5. **CI 集成**: 在 GitHub Actions 中自动运行 `go test ./...`
6. **测试覆盖率**: 使用 `go test -cover` 生成覆盖率报告，目标 > 70%
