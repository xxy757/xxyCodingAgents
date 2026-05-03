# xxyCodingAgents 全功能架构设计文档

## Context

xxyCodingAgents 是一个 AI Dev Platform（Agent OS），当前 MVP 已完成核心后端运行时（scheduler、watchdog、orchestrator、terminal manager、workspace git toolkit）和基础前端页面（仪表盘、项目列表、运行详情、终端管理）。

状态更新（2026-05-01）：

1. 原先定义本阶段工作的核心 gap 已基本修复：
   - 任务完成检测：已接通
   - `recoverFromCheckpoint()`：已接通
   - `TaskSpec.TimeoutSeconds`：已强制
   - `AgentSpec` 路由：已接通
   - `GitManager` / `WorkspacePath` / `InputData`：已接入基础流程
   - pause/resume runtime 调用：已接通
2. 当前新的主问题已经转为如何把 gstack 能力按正确边界接入现有 runtime。
3. 本文仍然描述目标架构，但当前活跃实施路线请优先参考 [gstack-integration-progress.md](/Volumes/Elements/code/codingagents/xxyCodingAgents/docs/gstack-integration-progress.md)。

目标：设计一个**小白都能快速上手**的全功能架构，整合 gstack 开发流程、Hermes 自动学习、资源编排、任务分解、提示词草稿生成与运行时注入。

---

## 一、系统整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        Web UI (Next.js 16)                       │
│  仪表盘 │ 项目管理 │ 运行详情 │ 终端 │ 知识库 │ 学习面板 │ 模板市场  │
├─────────────────────────────────────────────────────────────────┤
│                        API Gateway (Go stdlib)                    │
│  /api/v1/*  版本化 API，JWT 认证，速率限制                         │
├──────────┬──────────┬───────────┬──────────┬────────────────────┤
│ Orchestrator│ Scheduler │ Learning  │ Prompt   │ Resource          │
│ (工作流编排) │ (任务调度) │ Engine    │ Draft/Engine│ Manager        │
│           │          │ (自动学习) │ (草稿/注入) │ (资源管理)       │
├──────────┴──────────┴───────────┴──────────┴────────────────────┤
│                    Agent Runtime Layer                            │
│  ClaudeCodeAdapter │ KimiAdapter │ GLMAdapter │ CodexAdapter     │
├──────────────────────────────────────────────────────────────────┤
│                    Infrastructure                                 │
│  tmux PTY │ Git Workspace │ SQLite │ Checkpoint Store │ Log Store│
└──────────────────────────────────────────────────────────────────┘
```

### 核心原则

- **一切皆 Task**：任何操作（代码生成、审查、部署、学习）都建模为 Task
- **Gate 检查点**：关键状态转换必须通过 Gate 验证（gstack 风格）
- **自动学习闭环**：失败→捕获→提炼→修补→记忆 全自动（Hermes 风格）
- **用户确认优先**：用户原始输入先生成 Prompt Draft，用户可编辑确认后才进入 Run / Task
- **分层提示词注入**：确认后的任务输入再进入 System → Task Context → Learning → Runtime State 四层
- **零配置启动**：`make dev` 一键启动全部服务，内置默认模板

---

## 二、gstack 工作流管道设计

### 2.1 七步管道（Think → Plan → Build → Review → QA → Ship → Retro）

每个步骤对应一个 Task，步骤间通过 Gate 检查点连接：

```
Think ──► Plan ──► Build ──► Review ──► QA ──► Ship ──► Retro
  │         │         │          │         │        │         │
  └─ Gate ──┴─ Gate ──┴─ Gate ───┴─ Gate ──┴─ Gate ─┴─ Gate ──┘
```

### 2.2 Gate 机制设计

Gate 是状态转换的守卫，但在当前项目中 **不挂在 `WorkflowTemplate.Steps[]` 上**。  
当前代码的工作流持久化模型是 `NodesJSON + EdgesJSON`，因此 Gate 应建模为 **显式 DAG 节点**：

```go
// 示例：node.kind = "gate"
type GateConfig struct {
    Type       GateType `json:"type"`        // auto / manual / approval
    Conditions []string `json:"conditions"`  // 通过条件表达式
    Timeout    int      `json:"timeout_sec"` // 超时秒数
}

type GateType string
const (
    GateAuto     GateType = "auto"     // 自动检查（脚本验证）
    GateManual   GateType = "manual"   // 人工确认（UI 按钮）
    GateApproval GateType = "approval" // 审批流（多人）
)
```

**每步 Gate 的检查内容**：

| 步骤 | Gate 类型 | 检查内容 |
|------|----------|---------|
| Think → Plan | auto | 需求分析完整性检查（是否有输入/输出定义） |
| Plan → Build | manual | 用户确认实施计划 |
| Build → Review | auto | 编译通过 + 基础 lint 通过 |
| Review → QA | auto | Code review 完成（AI review 评分 > 阈值） |
| QA → Ship | manual | 测试通过 + 用户确认部署 |
| Ship → Retro | auto | 部署状态确认 |

### 2.3 工作流模板系统

建议继续沿用 DAG 持久化，只是在 node 上增加类型和配置：

```json
{
  "nodes": [
    {"id":"think","kind":"task","task_spec_id":"ts-think","label":"Think"},
    {"id":"gate-plan","kind":"gate","label":"Gate: Plan Approval","config":{"type":"manual"}},
    {"id":"plan","kind":"task","task_spec_id":"ts-plan","label":"Plan"}
  ],
  "edges": [
    {"from":"think","to":"gate-plan"},
    {"from":"gate-plan","to":"plan"}
  ]
}
```

### 2.4 实现要点

1. **`internal/orchestrator/orchestrator.go`** 改造：
   - `instantiateWorkflow()` 已为 Task 注入 `InputData`（上一步输出）和 `WorkspacePath`
   - 后续新增 `evaluateGate()` 方法执行 Gate 条件检查
   - 后续新增 `advanceWorkflow()` 在 Task / Gate 完成后自动触发下一步

2. **`internal/scheduler/scheduler.go`** 改造：
   - `tick()` 中新增任务完成检测：解析 Agent 输出，匹配完成标记（已完成）
   - 后续在任务完成后触发 `advanceWorkflow()`

3. **前端新增页面**：
   - `web/src/app/runs/[id]/gates/` - Gate 审批页面（manual/approval 类型）
   - 在运行详情页显示当前 Gate 状态和 pipeline 进度条

---

## 三、Hermes 自动学习层设计

### 3.1 四层学习模型

```
Layer 1: Failure Capture     → 捕获每次失败的完整上下文
Layer 2: Pattern Distiller   → 从失败中提炼通用模式
Layer 3: Skill Patcher       → 将模式转化为可复用的技能补丁
Layer 4: Memory Promotion    → 高频使用的补丁升级为永久记忆
```

### 3.2 数据模型新增

```go
// internal/domain/models.go 新增

// FailureRecord 记录一次失败的完整上下文
type FailureRecord struct {
    ID           string    `json:"id"`
    TaskID       string    `json:"task_id"`
    AgentKind    string    `json:"agent_kind"`
    ErrorType    string    `json:"error_type"`    // compile_error / test_failure / lint_error / runtime_error / timeout
    ErrorMessage string    `json:"error_message"`
    ContextData  string    `json:"context_data"`  // 失败时的代码、日志、环境信息 JSON
    FixAttempts  int       `json:"fix_attempts"`
    Resolved     bool      `json:"resolved"`
    ResolvedBy   string    `json:"resolved_by"`   // 解决该失败的 Pattern ID
    CreatedAt    time.Time `json:"created_at"`
}

// LearningPattern 从失败中提炼的通用模式
type LearningPattern struct {
    ID              string    `json:"id"`
    Name            string    `json:"name"`
    Description     string    `json:"description"`
    ErrorSignature  string    `json:"error_signature"`  // 错误特征（正则）
    FixTemplate     string    `json:"fix_template"`     // 修复模板
    ApplicableKinds []string  `json:"applicable_kinds"` // 适用的 Agent 类型
    SuccessCount    int       `json:"success_count"`    // 成功应用次数
    ConfidenceScore float64   `json:"confidence_score"` // 置信度 0-1
    Status          string    `json:"status"`           // active / deprecated / promoted
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}

// SkillPatch 可复用的技能补丁
type SkillPatch struct {
    ID             string    `json:"id"`
    PatternID      string    `json:"pattern_id"`
    Name           string    `json:"name"`
    PatchType      string    `json:"patch_type"`      // prompt_injection / hook / script / config
    PatchContent   string    `json:"patch_content"`   // 补丁具体内容
    TargetPhase    string    `json:"target_phase"`    // think / plan / build / review / qa / ship
    UsageCount     int       `json:"usage_count"`
    CreatedAt      time.Time `json:"created_at"`
}

// LongTermMemory 长期记忆（高频 SkillPatch 升级而来）
type LongTermMemory struct {
    ID          string    `json:"id"`
    Category    string    `json:"category"`     // project_convention / bug_pattern / best_practice / user_preference
    Key         string    `json:"key"`
    Value       string    `json:"value"`
    Context     string    `json:"context"`      // 适用上下文描述
    AccessCount int       `json:"access_count"`
    LastAccess  time.Time `json:"last_access"`
    CreatedAt   time.Time `json:"created_at"`
}
```

### 3.3 学习引擎实现

**新文件：`internal/learning/engine.go`**

核心流程：
```
Task Failed
    │
    ▼
FailureRecorder.Capture(task, error)     ← Layer 1: 捕获上下文
    │
    ▼ (批量，每 N 次失败或定时触发)
PatternDistiller.Distill(failures)       ← Layer 2: 聚类分析，提炼模式
    │
    ▼ (有新模式发现时)
SkillPatcher.Generate(pattern)           ← Layer 3: 生成补丁
    │
    ▼ (补丁使用次数 > 阈值)
MemoryManager.Promote(patch)             ← Layer 4: 升级为长期记忆
    │
    ▼ (记忆超过 M 天未使用)
MemoryManager.Demote(memory)             ← Layer 4: 降级或归档
```

关键方法：
- `RecordFailure(taskID, error)` - 存储失败记录
- `DistillPatterns()` - 对未处理的失败进行聚类，生成 Pattern
- `GetRelevantPatches(taskContext)` - 根据当前任务上下文检索相关补丁
- `InjectPatches(prompt, patches)` - 将补丁注入到提示词中（Learning Layer）

### 3.4 触发时机

| 事件 | 触发动作 |
|------|---------|
| Task 状态变为 `failed` | `RecordFailure()` |
| 每 10 次新失败 | `DistillPatterns()` |
| Task 开始执行 | `GetRelevantPatches()` 注入提示词 |
| Pattern 使用 10 次+ | `Promote()` 升级为长期记忆 |
| Memory 30 天未访问 | `Demote()` 归档 |

---

## 四、任务分解与提示词草稿系统

### 4.1 任务分解系统

**新文件：`internal/decomposer/decomposer.go`**

用户输入自然语言需求 → 自动分解为 DAG 子任务：

```go
type Decomposer interface {
    // Decompose 将需求文本分解为子任务列表
    Decompose(ctx context.Context, requirement string, projectContext ProjectContext) (*TaskTree, error)
}

type TaskTree struct {
    Root    *DecomposedTask   // 根任务
    Tasks   []*DecomposedTask // 所有子任务（扁平化）
    Edges   []TaskEdge        // 依赖关系
}

type DecomposedTask struct {
    Title       string   // 任务标题
    Description string   // 任务描述
    TaskType    string   // 任务类型标签
    AgentKind   string   // 推荐 Agent
    Priority    string   // 推荐优先级
    EstimatedMinutes int // 预估时间
    AcceptCriteria []string // 验收标准
    SuggestedFiles  []string // 可能涉及的文件
}
```

分解策略：
1. **关键词匹配**：识别 "创建API"、"修改UI"、"数据库迁移" 等模式
2. **文件依赖分析**：基于项目文件依赖图确定执行顺序
3. **Agent 路由**：根据任务类型自动选择最适合的 Agent

### 4.2 Prompt Draft / Composer（MVP）

定位：Prompt Draft 是用户发送任务前的可编辑草稿层，不替代运行时 `PromptEngine`。它解决“用户不知道如何写好提示词”的入口问题；`PromptEngine` 继续解决 agent 执行时的角色、阶段模板、learnings 和运行时状态注入问题。

MVP 约束：

1. 不直接把用户原始输入发送给 agent。
2. 后端先用规则模板生成结构化提示词草稿。
3. 前端展示草稿，用户可以编辑、重新生成、确认发送。
4. 只有用户确认后的 `final_prompt` 可以进入 Run / Task。

#### 数据模型

```go
type PromptDraftStatus string

const (
    PromptDraftStatusDraft     PromptDraftStatus = "draft"
    PromptDraftStatusConfirmed PromptDraftStatus = "confirmed"
    PromptDraftStatusSent      PromptDraftStatus = "sent"
)

type PromptDraft struct {
    ID              uuid.UUID         `json:"id"`
    ProjectID       uuid.UUID         `json:"project_id"`
    OriginalInput   string            `json:"original_input"`
    GeneratedPrompt string            `json:"generated_prompt"`
    FinalPrompt     string            `json:"final_prompt"`
    TaskType        string            `json:"task_type"`
    Status          PromptDraftStatus `json:"status"`
    CreatedAt       time.Time         `json:"created_at"`
    UpdatedAt       time.Time         `json:"updated_at"`
}
```

#### API 设计

| API | 作用 |
|-----|------|
| `POST /api/prompt-drafts/generate` | 根据 `original_input`、`project_id`、可选 `task_type` 生成草稿 |
| `PUT /api/prompt-drafts/{id}` | 保存用户编辑后的 `final_prompt` |
| `POST /api/prompt-drafts/{id}/send` | 将确认后的 `final_prompt` 创建为 Run / Task |

#### 前端交互

首页任务入口升级为 Prompt Composer：

```text
用户原始输入
  -> 点击“优化提示词”
  -> 生成结构化草稿
  -> 用户编辑 / 重新生成
  -> 用户确认发送
  -> 创建 Run / Task
```

草稿默认包含固定区块：

```text
任务目标：
上下文：
执行要求：
验收标准：
输出要求：
```

第一版生成器使用规则模板即可覆盖高频场景：

1. 修 bug
2. 实现功能
3. 代码审查
4. QA 测试
5. 文档更新
6. 架构分析

#### 和运行时 PromptEngine 的边界

```text
Prompt Draft / Composer
  输入：用户原始输入
  输出：用户确认后的 final_prompt
  时机：创建 Run / Task 前

PromptEngine
  输入：Task + AgentKind + Learnings + RuntimeState
  输出：agent 实际执行的完整 prompt
  时机：Scheduler launchAgent 前
```

运行时仍采用四层注入：

```
┌─────────────────────────────────────┐
│ Layer 1: System Prompt (基础层)      │ ← 角色定义、通用规则
├─────────────────────────────────────┤
│ Layer 2: Task Context (任务层)       │ ← final_prompt + task metadata
├─────────────────────────────────────┤
│ Layer 3: Learning Layer (学习层)     │ ← 相关经验、失败模式、patch
├─────────────────────────────────────┤
│ Layer 4: Runtime State (运行时层)    │ ← workspace、git status、近期日志
└─────────────────────────────────────┘
```

现有 `configs/prompts/*.yaml` 是运行时阶段模板，不直接暴露为用户草稿编辑器。后续可以在 Prompt Composer 中展示“模板来源”和“重新生成策略”，但 MVP 不做复杂模板市场。

---

## 五、多模型 Worker 集成

### 5.1 AgentSpec 系统激活

当前 `AgentSpec` 模型已定义但未使用。需要：

1. **`internal/storage/` 新增 `agentspec_repo.go`**（已有测试，需要实现）
2. **`configs/agents/` 目录**存放 Agent 配置：

```yaml
# configs/agents/claude-code.yaml
kind: claude-code
display_name: "Claude Code (claude.ai)"
runtime_adapter: ClaudeCodeAdapter
capabilities: [think, plan, build, review, qa, retro]
default_model: claude-sonnet-4-6
rate_limit: { max_concurrent: 3, requests_per_hour: 100 }
prompt_template: claude_default

# configs/agents/kimi.yaml  
kind: kimi
display_name: "Kimi (Moonshot)"
runtime_adapter: KimiAdapter
capabilities: [think, plan, review]
default_model: kimi-k2
rate_limit: { max_concurrent: 2, requests_per_hour: 50 }

# configs/agents/glm.yaml
kind: glm
display_name: "GLM (Zhipu)"
runtime_adapter: GLMAdapter
capabilities: [build, qa]
default_model: glm-4.6
rate_limit: { max_concurrent: 2, requests_per_hour: 60 }
```

### 5.2 新增 Adapter

在 `internal/runtime/` 下新增：

| 文件 | 说明 |
|------|------|
| `adapters_kimi.go` | Kimi API 适配器，封装 HTTP 调用 |
| `adapters_glm.go` | GLM API 适配器 |
| `adapters_codex.go` | OpenAI Codex 适配器（可选） |
| `adapter_registry.go` | Agent 注册表，按 kind 路由 |

### 5.3 Agent 路由逻辑

```
Task 创建时自动选择 Agent:
  1. 若 Task.AgentKind 已指定 → 使用指定 Agent
  2. 若 Task.TaskType 为 "think"/"plan" → 默认 Claude Code
  3. 若 Task.TaskType 为 "build" → 选择可用率最高的 Agent
  4. 若多个 Agent 可用 → 优先选择 rate_limit 剩余最多的
```

---

## 六、资源编排完善

### 6.1 当前 Gap 修复

| Gap | 修复方案 |
|-----|---------|
| 任务完成检测 | `scheduler.tick()` 中解析 tmux capture-pane 输出，匹配完成标记 |
| recoverFromCheckpoint | 在 `handleDeadAgent()` 中检查 `AgentInstance.CheckpointID`，有则调用 `agentRuntime.Checkpoint()` 恢复 |
| LastOutputAt 更新 | 每次 watchdog 检查时更新 `AgentInstance.LastOutputAt` |
| TaskSpec 超时 | `scheduler.tick()` 中检查 `started_at + timeout_seconds < now` |
| AgentSpec 使用 | `startAgent()` 从 DB 加载 AgentSpec 配置选择 runtime |
| Git 集成 | `instantiateWorkflow()` 中调用 `GitManager.Clone()` |
| pause/resume | API handler 调用 `AgentRuntime.Pause()/Resume()` |

### 6.2 准入控制增强

```
Admission Control (新增):
  1. 检查资源：当前内存/CPU/磁盘是否足够
  2. 检查并发：当前运行的 Agent 数 < max_concurrent (agent_spec)
  3. 检查速率：过去 1 小时的请求数 < requests_per_hour (agent_spec)
  4. 队列优先级排序：Priority + 等待时间加权
```

---

## 七、小白友好 UX 设计

### 7.1 首页改造：自然语言输入框

```
┌──────────────────────────────────────────────────┐
│                                                  │
│   今天想让我帮你做什么？                            │
│                                                  │
│   ┌──────────────────────────────────────────┐   │
│   │ 帮我把登录页面的按钮改成蓝色，并加个加载   │   │
│   │ 动画...                          [发送]  │   │
│   └──────────────────────────────────────────┘   │
│                                                  │
│   快捷模板:                                       │
│   [创建新API] [修复Bug] [添加测试] [代码重构]     │
│   [部署上线] [数据库迁移] [写文档]  [代码审查]    │
│                                                  │
│   最近运行:                          [查看全部]    │
│   ┌──────────────────────────────────────────┐   │
│   │ OK 添加用户登录功能      2分钟前  完成    │   │
│   │ .. 重构数据库查询        10分钟前  进行中 │   │
│   │ XX 修复首页加载慢        1小时前   失败   │   │
│   └──────────────────────────────────────────┘   │
│                                                  │
└──────────────────────────────────────────────────┘
```

### 7.2 新页面清单

| 页面 | 路由 | 功能 |
|------|------|------|
| 首页（改造） | `/` | 自然语言输入 + Prompt Composer + 最近运行 |
| 运行详情（增强） | `/runs/[id]` | Pipeline 进度条 + Gate 状态 + ReactFlow |
| Gate 审批 | `/runs/[id]/gates` | 人工确认/审批页面 |
| 知识库 | `/knowledge` | 长期记忆浏览、搜索、管理 |
| 学习面板 | `/learning` | Failure → Pattern → Patch 可视化链路 |
| 模板市场 | `/templates` | 工作流模板浏览、导入、自定义 |
| 提示词草稿 | `/prompt-drafts` | 草稿历史、编辑、重新生成、确认发送 |
| 提示词模板（后置） | `/prompts` | 运行时模板查看、编辑、版本管理 |
| Agent 管理 | `/agents` | Agent 配置、状态、速率监控 |
| 设置 | `/settings` | 全局配置、API Key、偏好设置 |

### 7.3 新手引导

- **首次启动向导**：3 步完成配置（选择默认 Agent → 配置工作区路径 → 选择模板）
- **模板市场**：内置 5+ 常用工作流模板，一键导入
- **错误友好提示**：失败时自动展示修复建议（来自 Learning Engine）
- **实时进度**：Pipeline 进度条 + 动画，清晰展示当前在做什么

---

## 八、分阶段开发路线图

### Phase 1: 核心 Gap 修复（Week 1-2）
**目标**：让现有系统完整运作

- [x] 1.1 任务完成检测（scheduler tick 中解析输出）
- [x] 1.2 recoverFromCheckpoint 激活
- [x] 1.3 LastOutputAt 更新 + 超时检测生效
- [x] 1.4 TaskSpec 超时强制
- [x] 1.5 AgentSpec 系统激活 + adapter_registry
- [x] 1.6 Git workspace 集成到 workflow instantiation
- [x] 1.7 API pause/resume 调用实际 runtime 方法
- [x] 1.8 InputData/WorkspacePath 设置

补充说明：

1. Step 1 的 runtime 基础设施（launcher script + prompt file + cleanup chain）也已经完成。
2. 后续活跃开发重心不再是 Gap 修复，而是 gstack 集成的 Step 2/3/4。

### Phase 2: gstack 工作流管道（Week 3-4）
**目标**：实现完整 7 步管道 + Gate 检查点

- [ ] 2.1 WorkflowTemplate Gate 配置支持
- [ ] 2.2 Gate 评估引擎（`evaluateGate()`）
- [ ] 2.3 工作流自动推进（`advanceWorkflow()`）
- [ ] 2.4 内置 6 阶段提示词模板
- [ ] 2.5 Gate 审批前端页面
- [ ] 2.6 运行详情页 Pipeline 进度条

### Phase 3: 提示词草稿与确认发送（Week 5-6）
**目标**：Prompt Composer + 用户确认后发送

- [ ] 3.1 PromptDraft 模型、迁移、Repository
- [ ] 3.2 `generate / update / send` 三个 API
- [ ] 3.3 规则模板生成器（bug / build / review / qa / docs / architecture）
- [ ] 3.4 首页 Prompt Composer UI（编辑、重新生成、确认发送）
- [ ] 3.5 `send` 仅允许确认后的 `final_prompt` 创建 Run / Task
- [x] 3.6 运行时 PromptEngine 已落地（四层注入 + phase YAML + scheduler 接入）

### Phase 4: 任务分解系统（Week 7-8）
**目标**：自然语言 → DAG 子任务

- [ ] 4.1 Decomposer 接口和基础实现
- [ ] 4.2 关键词匹配分解策略
- [ ] 4.3 文件依赖分析
- [ ] 4.4 前端自然语言输入框
- [ ] 4.5 分解结果可视化和确认 UI

### Phase 5: Hermes 学习引擎（Week 9-11）
**目标**：完整四层自动学习闭环

- [ ] 5.1 FailureRecorder 实现
- [ ] 5.2 PatternDistiller 聚类分析
- [ ] 5.3 SkillPatcher 补丁生成
- [ ] 5.4 MemoryManager Promotion/Demotion
- [ ] 5.5 学习面板前端页面（Failure → Pattern → Patch 可视化）
- [ ] 5.6 知识库管理页面

### Phase 6: 多模型集成（Week 12-13）
**目标**：支持多种 AI 模型并行工作

- [ ] 6.1 KimiAdapter 实现
- [ ] 6.2 GLMAdapter 实现
- [ ] 6.3 Agent 路由逻辑（根据 task type 自动选择）
- [ ] 6.4 Agent 管理前端页面
- [ ] 6.5 速率限制和并发控制

### Phase 7: 小白体验优化（Week 14-15）
**目标**：零门槛上手

- [ ] 7.1 首页改造（自然语言输入 + Prompt Composer + 快捷模板）
- [ ] 7.2 首次启动向导
- [ ] 7.3 模板市场页面
- [ ] 7.4 错误友好提示（Learning Engine 建议）
- [ ] 7.5 设置页面
- [ ] 7.6 全局 UI/UX 打磨

---

## 九、验证方案

### 每个 Phase 的验证标准

1. **Phase 1**：运行 `go test ./...` 全部通过；`make dev` 启动后手动创建一个 Run，确认任务能完整执行并检测完成
2. **Phase 2**：创建含全部 7 步的工作流模板，手动触发后确认每个 Gate 正确拦截/放行
3. **Phase 3**：在首页输入自然语言需求，生成 Prompt Draft，编辑确认后创建 Run，并确认 Task 使用的是 `final_prompt`
4. **Phase 4**：在首页输入 "给用户表加个 email 字段"，确认自动分解为合理的子任务 DAG
5. **Phase 5**：故意制造失败场景（如语法错误），确认 Failure → Pattern → Patch 链路完整
6. **Phase 6**：确认不同 Agent kind 的任务被正确路由到对应的 adapter
7. **Phase 7**：邀请未接触过本项目的同事试用，收集体验反馈

### 持续集成

```bash
# 后端
go test ./... -v -cover
go vet ./...

# 前端
cd web && npm run build && npm run lint

# 端到端
make dev && curl http://localhost:8080/healthz
```

---

## 十、关键文件清单

### 需要修改的文件

| 文件 | 修改内容 |
|------|---------|
| `internal/domain/models.go` | 新增 FailureRecord, LearningPattern, SkillPatch, LongTermMemory, GateConfig, PromptDraft, DecomposedTask 等模型 |
| `internal/scheduler/scheduler.go` | 已完成任务完成检测、超时检查、launcher 集成；后续继续承载 Gate / PromptEngine / browse env |
| `internal/scheduler/watchdog.go` | 已完成恢复链与清理链；后续继续承载 browse/QA 相关保护逻辑 |
| `internal/orchestrator/orchestrator.go` | 已完成 InputData/WorkspacePath 设置与完成态持久化；后续继续扩展 Gate/自动推进 |
| `internal/api/handlers.go` | pause/resume 调用实际 runtime 方法、新 API 端点 |

### 需要新增的文件

| 文件 | 说明 |
|------|------|
| `internal/agentlauncher/launcher.go` | 已新增：launcher 生成器，是 gstack 集成 Step 1 的基础设施 |
| `internal/learning/engine.go` | Hermes 学习引擎 |
| `internal/learning/distiller.go` | Pattern 提炼器 |
| `internal/learning/patcher.go` | SkillPatcher |
| `internal/prompt/draft.go` | Prompt Draft 规则模板生成器 |
| `internal/prompt/engine.go` | 已新增：运行时提示词引擎 |
| `internal/prompt/templates.go` | 后续：模板加载和管理 |
| `internal/decomposer/decomposer.go` | 任务分解器 |
| `internal/runtime/adapters_kimi.go` | Kimi 适配器 |
| `internal/runtime/adapters_glm.go` | GLM 适配器 |
| `internal/runtime/adapter_registry.go` | 已存在：Agent 注册表 |
| `configs/agents/*.yaml` | Agent 配置文件 |
| `configs/prompts/*.yaml` | 提示词模板文件 |
| `configs/workflows/*.yaml` | 工作流模板文件 |
| `web/src/app/gates/[id]/page.tsx` | Gate 审批页面 |
| `web/src/app/prompt-drafts/page.tsx` | Prompt Draft 历史和编辑页面 |
| `web/src/app/knowledge/page.tsx` | 知识库页面 |
| `web/src/app/learning/page.tsx` | 学习面板页面 |
| `web/src/app/templates/page.tsx` | 模板市场页面 |
| `web/src/app/prompts/page.tsx` | 后续：运行时提示词模板管理页面 |
| `web/src/app/settings/page.tsx` | 设置页面 |
