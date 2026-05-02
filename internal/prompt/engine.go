// Package prompt 提供阶段化提示词模板引擎。
// 目标：将 prompt 构建逻辑从 scheduler 的硬编码拼接中解耦，支持模板化维护与四层注入。
package prompt

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/xxy757/xxyCodingAgents/internal/domain"
	"gopkg.in/yaml.v3"
)

// PhaseTemplate 定义一个阶段模板（YAML 文件对应结构）。
type PhaseTemplate struct {
	Phase               string   `yaml:"phase"`
	Title               string   `yaml:"title"`
	System              []string `yaml:"system"`
	Steps               []string `yaml:"steps"`
	Checklist           []string `yaml:"checklist"`
	OutputFormat        string   `yaml:"output_format"`
	SafetyRules         []string `yaml:"safety_rules"`
	TaskContextTemplate string   `yaml:"task_context_template"`
}

// RuntimeState 是构建 prompt 时的运行时信息（Layer 4）。
type RuntimeState struct {
	WorkspacePath   string
	GitBranch       string
	GitStatus       string
	BrowseEnabled   bool
	BrowseStateFile string
}

// BuildOptions 是构建 prompt 所需参数。
type BuildOptions struct {
	Phase     string
	Task      *domain.Task
	AgentKind string
	TaskSpec  *domain.TaskSpec
	Learnings []string
	Runtime   RuntimeState
}

// Engine 负责加载模板并渲染完整 prompt。
type Engine struct {
	templateDir string
	templates   map[string]PhaseTemplate
}

// NewEngine 创建模板引擎。
func NewEngine(templateDir string) *Engine {
	return &Engine{
		templateDir: templateDir,
		templates:   make(map[string]PhaseTemplate),
	}
}

// BuildPrompt 根据阶段模板构建完整 prompt。
func (e *Engine) BuildPrompt(opts BuildOptions) (string, error) {
	if opts.Task == nil {
		return "", fmt.Errorf("task is required")
	}
	if err := e.ensureLoaded(); err != nil {
		return "", err
	}

	tpl, phaseName, err := e.selectTemplate(opts.Phase, opts.Task.TaskType)
	if err != nil {
		return "", err
	}

	taskContext, err := e.renderTaskContext(tpl, opts)
	if err != nil {
		return "", err
	}

	var parts []string
	title := strings.TrimSpace(tpl.Title)
	if title == "" {
		title = strings.ToUpper(phaseName)
	}
	parts = append(parts, fmt.Sprintf("# %s\n", title))
	parts = append(parts, fmt.Sprintf("阶段: %s", phaseName))

	parts = append(parts, renderListSection("## Layer 1: System Instructions", tpl.System))
	parts = append(parts, renderTextSection("## Layer 2: Task Context", taskContext))
	parts = append(parts, renderListSection("## Workflow Steps", tpl.Steps))
	parts = append(parts, renderListSection("## Quality Checklist", tpl.Checklist))

	if len(opts.Learnings) > 0 {
		parts = append(parts, renderListSection("## Layer 3: Past Learnings", opts.Learnings))
	} else {
		parts = append(parts, renderTextSection("## Layer 3: Past Learnings", "无匹配的历史经验。"))
	}

	parts = append(parts, renderTextSection("## Layer 4: Runtime State", renderRuntimeState(opts.Runtime)))
	parts = append(parts, renderTextSection("## Output Format", strings.TrimSpace(tpl.OutputFormat)))
	parts = append(parts, renderListSection("## Safety Rules", tpl.SafetyRules))

	return strings.TrimSpace(strings.Join(compact(parts), "\n\n")) + "\n", nil
}

func (e *Engine) ensureLoaded() error {
	if len(e.templates) > 0 {
		return nil
	}
	return e.Load()
}

// Load 从模板目录加载所有 YAML 模板。
func (e *Engine) Load() error {
	if strings.TrimSpace(e.templateDir) == "" {
		return fmt.Errorf("template directory is empty")
	}

	entries, err := os.ReadDir(e.templateDir)
	if err != nil {
		return fmt.Errorf("read template directory: %w", err)
	}

	loaded := make(map[string]PhaseTemplate)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(e.templateDir, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read template file %s: %w", path, err)
		}

		var tpl PhaseTemplate
		if err := yaml.Unmarshal(content, &tpl); err != nil {
			return fmt.Errorf("parse template file %s: %w", path, err)
		}

		phase := strings.ToLower(strings.TrimSpace(tpl.Phase))
		if phase == "" {
			phase = strings.TrimSuffix(strings.ToLower(entry.Name()), ext)
		}
		if phase == "" {
			return fmt.Errorf("template %s has empty phase", path)
		}

		if strings.TrimSpace(tpl.OutputFormat) == "" {
			tpl.OutputFormat = "## Summary\n- result: <required>"
		}
		tpl.Phase = phase
		loaded[phase] = tpl
	}

	if len(loaded) == 0 {
		return fmt.Errorf("no prompt templates found in %s", e.templateDir)
	}

	e.templates = loaded
	return nil
}

func (e *Engine) selectTemplate(phase, taskType string) (PhaseTemplate, string, error) {
	phase = normalizePhase(phase)
	taskType = normalizePhase(taskType)

	var candidates []string
	if phase != "" {
		candidates = append(candidates, phase)
	}
	if taskType != "" && taskType != phase {
		candidates = append(candidates, taskType)
	}
	if phase == "browser-qa" || taskType == "browser-qa" {
		candidates = append(candidates, "qa")
	}
	if phase != "" {
		if alias, ok := phaseAliases[phase]; ok {
			candidates = append(candidates, alias)
		}
	}
	if taskType != "" {
		if alias, ok := phaseAliases[taskType]; ok {
			candidates = append(candidates, alias)
		}
	}
	candidates = append(candidates, "review")

	seen := make(map[string]struct{})
	for _, candidate := range candidates {
		candidate = normalizePhase(candidate)
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		if tpl, ok := e.templates[candidate]; ok {
			return tpl, candidate, nil
		}
	}

	var available []string
	for k := range e.templates {
		available = append(available, k)
	}
	sort.Strings(available)
	return PhaseTemplate{}, "", fmt.Errorf("no prompt template matched phase=%q task_type=%q (available=%v)", phase, taskType, available)
}

func (e *Engine) renderTaskContext(tpl PhaseTemplate, opts BuildOptions) (string, error) {
	data := map[string]any{
		"Now":             time.Now().Format(time.RFC3339),
		"TaskID":          opts.Task.ID,
		"TaskType":        opts.Task.TaskType,
		"TaskTitle":       opts.Task.Title,
		"TaskDescription": opts.Task.Description,
		"InputData":       opts.Task.InputData,
		"AgentKind":       opts.AgentKind,
		"WorkspacePath":   opts.Task.WorkspacePath,
		"RequiredInputs":  "",
		"ExpectedOutputs": "",
	}
	if opts.TaskSpec != nil {
		data["RequiredInputs"] = opts.TaskSpec.RequiredInputs
		data["ExpectedOutputs"] = opts.TaskSpec.ExpectedOutputs
	}

	if strings.TrimSpace(tpl.TaskContextTemplate) != "" {
		t, err := template.New("task-context").Option("missingkey=zero").Parse(tpl.TaskContextTemplate)
		if err != nil {
			return "", fmt.Errorf("parse task_context_template for phase=%s: %w", tpl.Phase, err)
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return "", fmt.Errorf("render task_context_template for phase=%s: %w", tpl.Phase, err)
		}
		return strings.TrimSpace(buf.String()), nil
	}

	var sections []string
	sections = append(sections, fmt.Sprintf("- Task ID: `%s`", opts.Task.ID))
	sections = append(sections, fmt.Sprintf("- Task Type: `%s`", opts.Task.TaskType))
	if opts.AgentKind != "" {
		sections = append(sections, fmt.Sprintf("- Agent Kind: `%s`", opts.AgentKind))
	}
	if opts.Task.Title != "" {
		sections = append(sections, "### Task Title\n"+strings.TrimSpace(opts.Task.Title))
	}
	if opts.Task.Description != "" {
		sections = append(sections, "### Task Description\n"+strings.TrimSpace(opts.Task.Description))
	}
	if opts.Task.InputData != "" {
		sections = append(sections, "### Input Data\n"+strings.TrimSpace(opts.Task.InputData))
	}
	if opts.TaskSpec != nil && strings.TrimSpace(opts.TaskSpec.RequiredInputs) != "" {
		sections = append(sections, "### Required Inputs\n"+strings.TrimSpace(opts.TaskSpec.RequiredInputs))
	}
	if opts.TaskSpec != nil && strings.TrimSpace(opts.TaskSpec.ExpectedOutputs) != "" {
		sections = append(sections, "### Expected Outputs\n"+strings.TrimSpace(opts.TaskSpec.ExpectedOutputs))
	}

	return strings.TrimSpace(strings.Join(sections, "\n\n")), nil
}

func renderRuntimeState(state RuntimeState) string {
	var lines []string
	if state.WorkspacePath != "" {
		lines = append(lines, fmt.Sprintf("- Workspace: `%s`", state.WorkspacePath))
	}
	if state.GitBranch != "" {
		lines = append(lines, fmt.Sprintf("- Git Branch: `%s`", state.GitBranch))
	}
	if state.GitStatus != "" {
		lines = append(lines, "### Git Status\n```text\n"+state.GitStatus+"\n```")
	}
	if state.BrowseEnabled {
		lines = append(lines, "- Browser QA: enabled")
		if state.BrowseStateFile != "" {
			lines = append(lines, fmt.Sprintf("- BROWSE_STATE_FILE: `%s`", state.BrowseStateFile))
		}
	} else {
		lines = append(lines, "- Browser QA: disabled")
	}
	if len(lines) == 0 {
		return "无运行时状态。"
	}
	return strings.Join(lines, "\n\n")
}

func renderListSection(title string, items []string) string {
	var cleaned []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		cleaned = append(cleaned, "- "+item)
	}
	if len(cleaned) == 0 {
		return ""
	}
	return title + "\n" + strings.Join(cleaned, "\n")
}

func renderTextSection(title, body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	return title + "\n" + body
}

func compact(values []string) []string {
	var out []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func normalizePhase(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

var phaseAliases = map[string]string{
	"browser-qa":  "qa",
	"code-review": "review",
	"postmortem":  "retro",
}
