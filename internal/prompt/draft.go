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

// TechStackOption 定义可选的技术方案预设。
type TechStackOption struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Context string `json:"context"`
}

// techStackPresets 定义所有可选的技术方案。
var techStackPresets = []TechStackOption{
	{
		ID:    "go-nextjs",
		Label: "Go + Next.js",
		Context: `技术栈：Go 1.22+ 后端（标准库 HTTP 路由、go-sqlite3、gorilla/websocket）+ Next.js 前端（App Router、React、Tailwind CSS）。
代码约定：中文注释、uuid 实体 ID、参数化原生 SQL、sql.Null* 可空字段、fmt.Errorf("%w", err) 错误包装、log/slog 结构化日志。
分层架构：domain/models.go（类型）→ storage/database.go（迁移）→ storage/repositories.go（CRUD）→ api/handlers.go（处理器）→ api/server.go（路由）。
测试：go test ./...（内存 SQLite :memory:）、HTTP 测试用 httptest.ResponseRecorder。`,
	},
	{
		ID:    "go-react",
		Label: "Go + React",
		Context: `技术栈：Go 1.22+ 后端（Gin/Echo 框架、GORM/sqlx 数据库）+ React 前端（Create React App / Vite、React Router、Tailwind CSS）。
代码约定：RESTful API 设计、JWT 认证、参数化 SQL 查询、结构化错误码。
后端分层：handler → service → repository → model。前端分层：pages → components → hooks → api。
测试：go test ./...、React Testing Library + Vitest。`,
	},
	{
		ID:    "python-react",
		Label: "Python + React",
		Context: `技术栈：Python 3.11+ 后端（FastAPI/Django、SQLAlchemy/Django ORM、Pydantic 数据验证）+ React 前端（Vite、React Router、Tailwind CSS）。
代码约定：type hints、docstring 文档、PEP 8 风格、Alembic 数据库迁移。
后端分层：routers → services → repositories → models。前端分层：pages → components → hooks → api。
测试：pytest（fixtures、mock）、React Testing Library + Vitest。`,
	},
	{
		ID:    "python-vue",
		Label: "Python + Vue",
		Context: `技术栈：Python 3.11+ 后端（FastAPI/Django、SQLAlchemy/Django ORM）+ Vue 3 前端（Vite、Vue Router、Pinia、Element Plus/Naive UI）。
代码约定：type hints、docstring 文档、PEP 8 风格、Alembic 数据库迁移。
后端分层：api → service → dao → model。前端分层：views → components → composables → api。
测试：pytest、Vitest + Vue Test Utils。`,
	},
	{
		ID:    "java-vue",
		Label: "Java + Vue",
		Context: `技术栈：Java 17+ 后端（Spring Boot 3、Spring Security、MyBatis-Plus/JPA、Maven/Gradle）+ Vue 3 前端（Vite、Vue Router、Pinia、Element Plus）。
代码约定：RESTful API、Swagger/OpenAPI 文档、统一响应封装、全局异常处理。
后端分层：controller → service → mapper/dao → entity。前端分层：views → components → composables → api。
测试：JUnit 5 + Mockito、Spring Boot Test、Vitest。`,
	},
	{
		ID:    "java-react",
		Label: "Java + React",
		Context: `技术栈：Java 17+ 后端（Spring Boot 3、Spring Security、MyBatis-Plus/JPA）+ React 前端（Vite、React Router、Ant Design、Tailwind CSS）。
代码约定：RESTful API、Swagger/OpenAPI 文档、统一响应封装、全局异常处理。
后端分层：controller → service → mapper/dao → entity。前端分层：pages → components → hooks → api。
测试：JUnit 5 + Mockito、React Testing Library + Vitest。`,
	},
	{
		ID:    "node-react",
		Label: "Node.js + React",
		Context: `技术栈：Node.js 20+ 后端（Express/Fastify/NestJS、Prisma/TypeORM、TypeScript）+ React 前端（Vite、React Router、Tailwind CSS、Zustand/Redux）。
代码约定：TypeScript 严格模式、ESLint + Prettier、RESTful API 设计、Zod 验证。
后端分层：routes → controllers → services → repositories → models。前端分层：pages → components → hooks → api。
测试：Jest/Vitest、Supertest（API）、React Testing Library。`,
	},
	{
		ID:    "node-vue",
		Label: "Node.js + Vue",
		Context: `技术栈：Node.js 20+ 后端（Express/NestJS、Prisma/TypeORM、TypeScript）+ Vue 3 前端（Vite、Vue Router、Pinia、Naive UI/Element Plus）。
代码约定：TypeScript 严格模式、ESLint + Prettier、RESTful API 设计。
后端分层：routes → controllers → services → repositories → models。前端分层：views → components → composables → api。
测试：Jest/Vitest、Supertest（API）、Vue Test Utils。`,
	},
	{
		ID:    "rust-react",
		Label: "Rust + React",
		Context: `技术栈：Rust 后端（Actix-web/Axum、SQLx/Diesel、Tokio 异步运行时）+ React 前端（Vite、React Router、Tailwind CSS）。
代码约定：cargo clippy 无警告、Result<T,E> 错误处理、trait 抽象、生命周期标注、模块化设计。
后端分层：handlers → services → repositories → models。前端分层：pages → components → hooks → api。
测试：cargo test、集成测试、React Testing Library。`,
	},
	{
		ID:    "go-vue",
		Label: "Go + Vue",
		Context: `技术栈：Go 1.22+ 后端（Gin/Echo 框架、GORM/sqlx）+ Vue 3 前端（Vite、Vue Router、Pinia、Element Plus）。
代码约定：RESTful API、参数化 SQL、结构化日志、中文注释。
后端分层：handler → service → repository → model。前端分层：views → components → composables → api。
测试：go test ./...、Vitest + Vue Test Utils。`,
	},
	{
		ID:    "custom",
		Label: "自定义 / 不指定",
		Context: `请根据项目实际情况和用户输入自行判断技术栈和架构。
通用编码规范：清晰的代码结构、完善的错误处理、充分的测试覆盖、合理的 API 设计。
请先阅读项目中的 CLAUDE.md、README.md 或其他文档了解项目上下文。`,
	},
}

// taskTemplate 定义每种任务类型的模板段落。
type taskTemplate struct {
	goalPrefix  string // 任务目标前缀
	requireBase string // 基础执行要求（不含上下文）
	acceptBase  string // 基础验收标准
	outputReq   string // 输出要求
}

// templates 定义 6 种任务类型的规则模板。
// contextHint 不再硬编码，改为运行时根据 techStack 动态拼接。
var templates = map[string]taskTemplate{
	TaskTypeBugfix: {
		goalPrefix:  "定位并修复以下错误",
		requireBase: "1. 复现并确认错误\n2. 定位根因（精确到文件和行号）\n3. 检查相关状态转换是否正确\n4. 实施最小化修复\n5. 运行测试验证无回归",
		acceptBase:  "1. 错误不再复现\n2. 测试全部通过\n3. 静态检查无告警\n4. 修复范围最小化，不改动无关代码\n5. 如果涉及数据库变更，迁移脚本可重复执行",
		outputReq:   "修改的文件列表、修复说明、验证步骤和测试输出",
	},
	TaskTypeBuild: {
		goalPrefix:  "实现以下功能或重构",
		requireBase: "1. 分析需求，明确边界\n2. 设计实现方案（列出涉及的文件和层）\n3. 按项目分层顺序实现\n4. 补充单元测试\n5. 运行构建和测试验证",
		acceptBase:  "1. 功能按需求正常工作\n2. 静态检查无告警\n3. 测试全部通过\n4. 不破坏现有功能\n5. 代码风格与项目一致",
		outputReq:   "新增/修改的文件列表、实现说明、验证结果",
	},
	TaskTypeReview: {
		goalPrefix:  "审查以下代码变更",
		requireBase: "1. 理解变更目的\n2. 检查逻辑正确性（状态转换、并发安全）\n3. 检查安全漏洞（注入、认证、授权）\n4. 检查错误处理\n5. 检查测试覆盖\n6. 检查数据库操作安全性",
		acceptBase:  "1. 列出所有发现的问题\n2. 按严重程度分级（critical/major/minor/nit）\n3. 给出具体修改建议（精确到文件:行号）",
		outputReq:   "审查报告：结论（APPROVE/REQUEST_CHANGES/BLOCK）、问题列表、严重程度、修改建议",
	},
	TaskTypeQA: {
		goalPrefix:  "对以下页面或功能进行质量验证",
		requireBase: "1. 导航到目标页面或启动目标功能\n2. 验证核心功能正常（表单提交、数据展示、页面跳转）\n3. 测试边界情况和异常输入\n4. 检查控制台错误和警告\n5. 截图记录关键状态",
		acceptBase:  "1. 所有关键路径验证通过\n2. 发现的 bug 有截图证据\n3. 控制台无严重错误\n4. 响应式布局正常（如适用）",
		outputReq:   "测试报告：通过/失败项、截图路径、bug 列表和复现步骤",
	},
	TaskTypeDocs: {
		goalPrefix:  "更新或创建以下文档",
		requireBase: "1. 分析现有文档状态\n2. 确定需要新增/更新的内容\n3. 编写清晰、准确的文档\n4. 确保与代码实现一致\n5. 更新相关的 README 或配置说明",
		acceptBase:  "1. 文档内容准确\n2. 格式统一（Markdown）\n3. 覆盖所有关键点\n4. 与代码实现一致",
		outputReq:   "修改的文档文件列表、变更摘要",
	},
	TaskTypeArchitecture: {
		goalPrefix:  "分析架构或提出设计方案",
		requireBase: "1. 分析现有架构（读取各包/模块的职责和依赖关系）\n2. 识别问题或改进点\n3. 提出设计方案（含权衡分析）\n4. 给出实施路径（分阶段、可验证）",
		acceptBase:  "1. 分析透彻，有代码路径支撑\n2. 方案可行，考虑了现有约束\n3. 权衡清晰，决策有理\n4. 实施路径可分阶段执行",
		outputReq:   "架构分析报告：现状、问题、方案、权衡、实施建议",
	},
}

// GetTechStackOptions 返回所有可选的技术方案预设。
func GetTechStackOptions() []TechStackOption {
	return techStackPresets
}

// GetTechStackContext 根据 ID 返回对应技术方案的上下文。
// 未找到时返回自定义方案的上下文。
func GetTechStackContext(id string) string {
	for _, ts := range techStackPresets {
		if ts.ID == id {
			return ts.Context
		}
	}
	return techStackPresets[len(techStackPresets)-1].Context
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

// GenerateDraft 根据用户原始输入、任务类型和技术方案生成结构化提示词草稿。
// taskType 为空时自动推断。techStackID 指定技术方案预设 ID。
func GenerateDraft(originalInput, taskType, techStackID string) string {
	if taskType == "" {
		taskType = InferTaskType(originalInput)
	}

	tmpl, ok := templates[taskType]
	if !ok {
		tmpl = templates[TaskTypeBuild]
	}

	contextStr := GetTechStackContext(techStackID)

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## 任务目标\n%s：%s\n\n", tmpl.goalPrefix, strings.TrimSpace(originalInput)))
	sb.WriteString(fmt.Sprintf("## 上下文\n%s\n\n", contextStr))
	sb.WriteString(fmt.Sprintf("## 执行要求\n%s\n\n", tmpl.requireBase))
	sb.WriteString(fmt.Sprintf("## 验收标准\n%s\n\n", tmpl.acceptBase))
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