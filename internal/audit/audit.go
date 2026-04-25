// Package audit 提供审计日志功能，记录系统事件和 Agent 执行的命令。
// 包含敏感数据自动脱敏和输出解析功能。
package audit

import (
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xxy757/xxyCodingAgents/internal/domain"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
)

// Auditor 是审计日志记录器，负责持久化事件和命令日志。
type Auditor struct {
	repos *storage.Repos
}

// NewAuditor 创建审计记录器实例。
func NewAuditor(repos *storage.Repos) *Auditor {
	return &Auditor{repos: repos}
}

// LogEvent 记录一个系统事件到审计日志。
func (a *Auditor) LogEvent(runID string, taskID, agentID *string, eventType domain.EventType, message string) {
	event := &domain.Event{
		ID:        uuid.New().String(),
		RunID:     runID,
		TaskID:    taskID,
		AgentID:   agentID,
		EventType: eventType,
		Message:   message,
		CreatedAt: time.Now(),
	}
	if err := a.repos.Events.Create(event); err != nil {
		slog.Error("audit log event", "error", err)
	}
}

// LogCommand 记录一条命令执行日志，自动对敏感信息进行脱敏处理。
func (a *Auditor) LogCommand(taskID string, agentID *string, command string, exitCode *int, output string, durationMs int64) {
	sanitizedOutput := SanitizeOutput(output)
	sanitizedCommand := SanitizeOutput(command)

	log := &domain.CommandLog{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		AgentID:   agentID,
		Command:   sanitizedCommand,
		ExitCode:  exitCode,
		Output:    sanitizedOutput,
		Duration:  durationMs,
		CreatedAt: time.Now(),
	}
	if err := a.repos.CommandLogs.Create(log); err != nil {
		slog.Error("audit log command", "error", err)
	}
}

// sensitivePatterns 定义需要脱敏的敏感数据匹配模式。
// 匹配 token、key、secret、password 等键值对以及 OpenAI API Key 格式。
var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(token|key|secret|password|cookie|authorization|bearer|api[_-]?key)\s*[:=]\s*\S+`),
	regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),
	regexp.MustCompile(`(?i)"(token|key|secret|password|api_key|apikey)"\s*:\s*"[^"]*"`),
}

// SanitizeOutput 对输出文本进行敏感信息脱敏，将匹配到的敏感值替换为 ***REDACTED***。
func SanitizeOutput(output string) string {
	result := output
	for _, pattern := range sensitivePatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// 尝试保留键名，只替换值部分
			parts := strings.SplitN(match, "=", 2)
			if len(parts) == 2 {
				return parts[0] + "=***REDACTED***"
			}
			parts = strings.SplitN(match, ":", 2)
			if len(parts) == 2 {
				return parts[0] + ":***REDACTED***"
			}
			return "***REDACTED***"
		})
	}
	return result
}

// OutputParser 解析 Agent 终端输出，识别测试结果、命令执行和阶段变化等模式。
type OutputParser struct {
	patterns []OutputPattern
}

// OutputPattern 定义一个输出匹配模式。
type OutputPattern struct {
	Name    string         // 模式名称
	Pattern *regexp.Regexp // 匹配正则
}

// NewOutputParser 创建输出解析器，预置常见的匹配模式。
func NewOutputParser() *OutputParser {
	return &OutputParser{
		patterns: []OutputPattern{
			{Name: "test_passed", Pattern: regexp.MustCompile(`(?i)(PASS|OK|passed|success)\s*`)},
			{Name: "test_failed", Pattern: regexp.MustCompile(`(?i)(FAIL|ERROR|failed|failure)\s*`)},
			{Name: "command_exec", Pattern: regexp.MustCompile(`^\$\s+(.+)`)},
			{Name: "phase_change", Pattern: regexp.MustCompile(`(?i)(entering|starting|phase|step)\s*:?\s*(.+)`)},
		},
	}
}

// Parse 解析一行输出文本，返回匹配到的事件类型。
func (p *OutputParser) Parse(line string) (eventType string, matched bool) {
	for _, pattern := range p.patterns {
		if pattern.Pattern.MatchString(line) {
			return pattern.Name, true
		}
	}
	return "", false
}
