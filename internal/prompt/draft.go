// Package prompt 的 draft 文件实现提示词草稿的规则模板生成器。
// 根据用户输入和任务类型，生成结构化的提示词草稿。
// MVP 阶段使用确定性规则模板，不依赖外部 LLM。
package prompt

import (
	"fmt"
	"strings"
)

// TaskType 定义支持的任务类型常量。
const (
	TaskTypeBugfix       = "bugfix"
	TaskTypeBuild        = "build"
	TaskTypeReview       = "review"
	TaskTypeQA           = "qa"
	TaskTypeDocs         = "docs"
	TaskTypeArchitecture = "architecture"
)

// taskTemplate 定义每种任务类型的模板段落。
type taskTemplate struct {
	goalPrefix   string // 任务目标前缀
	contextHint  string // 上下文提示
	requirements string // 执行要求
	acceptance   string // 验收标准
	outputReq    string // 输出要求
}

// projectContext 是嵌入到草稿中的项目上下文信息。
const projectContext = `技术栈：Go 1.22+ 后端（标准库 HTTP 路由、go-sqlite3、gorilla/websocket）+ Next.js 16 前端（App Router、React 19、Tailwind CSS 4）。
代码约定：中文注释、uuid 实体 ID、参数化原生 SQL、sql.Null* 可空字段、fmt.Errorf("%w", err) 错误包装、log/slog 结构化日志。
分层架构：domain/models.go（类型）→ storage/database.go（迁移）→ storage/repositories.go（CRUD）→ api/handlers.go（处理器）→ api/server.go（路由）。
测试：go test ./...（内存 SQLite :memory:）、HTTP 测试用 httptest.ResponseRecorder。`

// templates 定义 6 种任务类型的规则模板。
var templates = map[string]taskTemplate{
	TaskTypeBugfix: {
		goalPrefix: "定位并修复以下错误",
		contextHint: projectContext + "\n请分析错误现象、堆栈信息、相关代码路径，确定根本原因。" +
			"\n重点关注：状态机转换是否正确、数据库操作是否使用参数化 SQL、错误是否被正确包装传递。",
		requirements: "1. 复现并确认错误\n2. 定位根因（精确到文件和行号）\n3. 检查相关的状态机转换（Run/Task/AgentInstance）\n4. 实施最小化修复\n5. 运行 go test 验证无回归",
		acceptance: "1. 错误不再复现\n2. go test ./... 全部通过\n3. go vet 无告警\n4. 修复范围最小化，不改动无关代码\n5. 如果涉及数据库变更，迁移脚本可重复执行",
		outputReq:  "修改的文件列表、修复说明、验证步骤（go test 输出）",
	},
	TaskTypeBuild: {
		goalPrefix: "实现以下功能或重构",
		contextHint: projectContext + "\n请理解需求背景，对照现有代码结构确认实现位置。" +
			"\n新增实体需要完整分层：domain → migration → repository → handler → route → 前端 API 客户端 → 前端页面。",
		requirements: "1. 分析需求，明确边界\n2. 设计实现方案（列出涉及的文件和层）\n3. 按分层顺序实现：先 domain 类型，再 migration，再 repository，再 handler，再 route\n4. 前端同步更新 api.ts 接口和页面组件\n5. 补充单元测试\n6. 运行 go vet && go test && npm run build 验证",
		acceptance: "1. 功能按需求正常工作\n2. go vet ./... 无告警\n3. go test ./... 全部通过\n4. npm run build 无错误\n5. 不破坏现有功能\n6. 新增 SQL 使用参数化查询",
		outputReq:  "新增/修改的文件列表、实现说明、验证结果",
	},
	TaskTypeReview: {
		goalPrefix: "审查以下代码变更",
		contextHint: projectContext + "\n请关注代码的正确性、安全性、可维护性。" +
			"\n项目安全要点：SQL 注入（参数化查询）、tmux 命令注入、WebSocket 注入、QA 场景的网页内容信任边界。",
		requirements: "1. 理解变更目的\n2. 检查逻辑正确性（状态机转换、并发安全）\n3. 检查安全漏洞（注入、信任边界）\n4. 检查错误处理（是否包装传递、是否有清理路径）\n5. 检查测试覆盖\n6. 检查数据库操作（参数化 SQL、可空字段类型）",
		acceptance: "1. 列出所有发现的问题\n2. 按严重程度分级（critical/major/minor/nit）\n3. 给出具体修改建议（精确到文件:行号）",
		outputReq:  "审查报告：结论（APPROVE/REQUEST_CHANGES/BLOCK）、问题列表、严重程度、修改建议",
	},
	TaskTypeQA: {
		goalPrefix: "对以下页面或功能进行质量验证",
		contextHint: projectContext + "\n前端运行在 localhost:3000，API 代理到 localhost:8080。" +
			"\n使用 gstack browse 工具进行页面交互和截图。",
		requirements: "1. 导航到目标页面\n2. 使用 snapshot -i 查看交互元素\n3. 验证核心功能正常（表单提交、数据展示、页面跳转）\n4. 测试边界情况\n5. 检查控制台错误（console --errors）\n6. 截图记录关键状态",
		acceptance: "1. 所有关键路径验证通过\n2. 发现的 bug 有截图证据\n3. 控制台无严重错误\n4. 响应式布局在移动端和桌面端正常",
		outputReq:  "测试报告：通过/失败项、截图路径、bug 列表和复现步骤",
	},
	TaskTypeDocs: {
		goalPrefix: "更新或创建以下文档",
		contextHint: projectContext + "\n项目文档在 docs/ 目录，CLAUDE.md 是项目级 Claude Code 指引。" +
			"\n文档应与代码实现保持一致。",
		requirements: "1. 分析现有文档状态\n2. 确定需要新增/更新的内容\n3. 编写清晰、准确的文档\n4. 确保与代码实现一致\n5. 更新 CLAUDE.md 中的架构总览或命令（如有变更）",
		acceptance: "1. 文档内容准确\n2. 格式统一（Markdown）\n3. 覆盖所有关键点\n4. 与代码实现一致",
		outputReq:  "修改的文档文件列表、变更摘要",
	},
	TaskTypeArchitecture: {
		goalPrefix: "分析架构或提出设计方案",
		contextHint: projectContext + "\n系统采用分层架构，核心模块包括：编排器（Run/Task 生命周期）、调度器（3s 滴答、压力分级）、" +
			"运行时适配器（ClaudeCode/GenericShell）、终端管理器（tmux PTY）、PromptEngine（4 层注入）。",
		requirements: "1. 分析现有架构（读取 internal/ 下各包的职责和依赖关系）\n2. 识别问题或改进点\n3. 提出设计方案（含权衡分析）\n4. 给出实施路径（分阶段、可验证）",
		acceptance: "1. 分析透彻，有代码路径支撑\n2. 方案可行，考虑了现有约束（SQLite 单机、tmux 进程管理）\n3. 权衡清晰，决策有理\n4. 实施路径可分阶段执行",
		outputReq:  "架构分析报告：现状、问题、方案、权衡、实施建议",
	},
}

// InferTaskType 从用户输入中推断任务类型。
// 基于关键词匹配，返回最匹配的任务类型，默认返回 build。
func InferTaskType(input string) string {
	lower := strings.ToLower(input)

	// bugfix 关键词
	bugfixKeywords := []string{"bug", "错误", "修复", "报错", "异常", "崩溃", "crash", "fix", "error", "fail", "失败", "问题"}
	for _, kw := range bugfixKeywords {
		if strings.Contains(lower, kw) {
			return TaskTypeBugfix
		}
	}

	// qa 关键词
	qaKeywords := []string{"测试", "检查", "验证", "test", "qa", "质量", "浏览", "页面"}
	for _, kw := range qaKeywords {
		if strings.Contains(lower, kw) {
			return TaskTypeQA
		}
	}

	// review 关键词
	reviewKeywords := []string{"审查", "review", "代码检查", "diff", "pr"}
	for _, kw := range reviewKeywords {
		if strings.Contains(lower, kw) {
			return TaskTypeReview
		}
	}

	// docs 关键词
	docsKeywords := []string{"文档", "doc", "readme", "说明", "注释"}
	for _, kw := range docsKeywords {
		if strings.Contains(lower, kw) {
			return TaskTypeDocs
		}
	}

	// architecture 关键词
	archKeywords := []string{"架构", "设计", "重构", "architecture", "design", "方案"}
	for _, kw := range archKeywords {
		if strings.Contains(lower, kw) {
			return TaskTypeArchitecture
		}
	}

	// 默认 build
	return TaskTypeBuild
}

// GenerateDraft 根据用户原始输入和任务类型生成结构化提示词草稿。
// taskType 为空时自动推断。返回格式化的草稿文本。
func GenerateDraft(originalInput, taskType string) string {
	if taskType == "" {
		taskType = InferTaskType(originalInput)
	}

	tmpl, ok := templates[taskType]
	if !ok {
		tmpl = templates[TaskTypeBuild]
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 任务目标\n%s：%s\n\n", tmpl.goalPrefix, strings.TrimSpace(originalInput)))
	sb.WriteString(fmt.Sprintf("## 上下文\n%s\n\n", tmpl.contextHint))
	sb.WriteString(fmt.Sprintf("## 执行要求\n%s\n\n", tmpl.requirements))
	sb.WriteString(fmt.Sprintf("## 验收标准\n%s\n\n", tmpl.acceptance))
	sb.WriteString(fmt.Sprintf("## 输出要求\n%s\n", tmpl.outputReq))

	return sb.String()
}

// GetSupportedTaskTypes 返回所有支持的任务类型。
func GetSupportedTaskTypes() []string {
	return []string{
		TaskTypeBugfix,
		TaskTypeBuild,
		TaskTypeReview,
		TaskTypeQA,
		TaskTypeDocs,
		TaskTypeArchitecture,
	}
}
