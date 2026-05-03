# TODOS

## DAG P0 Bugs 修复
- **What:** 修复 cycle detection（三色 DFS）、output propagation（CompleteTask 时写回 Task.OutputData + advanceFromNode 收集上游输出）、清理死代码（UnblockDependentTasks + getTaskDependencies）
- **Why:** DAG 工作流根本不能正确工作。菱形依赖会误报环。output propagation 在创建时快照是死代码。
- **Pros:** DAG 工作流可以正确运行
- **Cons:** ~2 小时工作量，已修复 cycle detection 和 output propagation
- **Depends on:** 无
- **Blocks:** Prompt Composer 的多任务 DAG 场景
- **Status:** 部分完成（cycle detection ✅, output propagation ✅, all-predecessor 检查已确认正确）

## 学习数据迁移到 SQLite
- **What:** 新 migration（learning_entries 表）+ 新 repository + 删除 internal/learning/ 的 JSONL 实现
- **Why:** JSONL 违反单 SQLite 架构。Go 的 os.Write 不是原子的，并发写入会交错字节。SQLite WAL 模式已经是存储层，统一到 SQLite 减少数据分叉。
- **Pros:** 统一存储层，备份/迁移只需处理 SQLite，并发安全
- **Cons:** ~2 小时工作量，需要重写 learning/store.go 和 learning/search.go
- **Depends on:** 无
- **Blocks:** Prompt Composer 的学习数据注入（Expansion 1）

## Scheduler 拆分
- **What:** 把 prompt 构建 + 学习注入从 scheduler.go 抽成独立 PromptService
- **Why:** scheduler.go 1455 行，10+ 职责，越来越难维护
- **Pros:** 降低认知负荷，提高可测试性
- **Cons:** 重构成本 ~2 小时，需更新接口和测试
- **Depends on:** 无

## 任务类型统一枚举
- **What:** 把 "qa"、"browser-qa"、"build" 等字符串字面量统一到 domain 包常量，全项目引用
- **Why:** 同一字符串散落 4 个包，inferTaskType 已出现行为不一致
- **Pros:** 编译期检查，消除拼写错误
- **Cons:** 改动面广但简单（查找替换）
- **Depends on:** 无
