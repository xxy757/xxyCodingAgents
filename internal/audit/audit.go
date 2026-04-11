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

type Auditor struct {
	repos *storage.Repos
}

func NewAuditor(repos *storage.Repos) *Auditor {
	return &Auditor{repos: repos}
}

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

var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(token|key|secret|password|cookie|authorization|bearer|api[_-]?key)\s*[:=]\s*\S+`),
	regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),
	regexp.MustCompile(`(?i)"(token|key|secret|password|api_key|apikey)"\s*:\s*"[^"]*"`),
}

func SanitizeOutput(output string) string {
	result := output
	for _, pattern := range sensitivePatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
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

type OutputParser struct {
	patterns []OutputPattern
}

type OutputPattern struct {
	Name    string
	Pattern *regexp.Regexp
}

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

func (p *OutputParser) Parse(line string) (eventType string, matched bool) {
	for _, pattern := range p.patterns {
		if pattern.Pattern.MatchString(line) {
			return pattern.Name, true
		}
	}
	return "", false
}
