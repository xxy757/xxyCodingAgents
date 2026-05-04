package scheduler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xxy757/xxyCodingAgents/internal/config"
	"github.com/xxy757/xxyCodingAgents/internal/domain"
	learningengine "github.com/xxy757/xxyCodingAgents/internal/learning"
	promptengine "github.com/xxy757/xxyCodingAgents/internal/prompt"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
)

func newTestScheduler() *Scheduler {
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			TickSeconds:         3,
			MaxConcurrentAgents: 2,
			MaxHeavyAgents:      1,
			MaxTestJobs:         1,
		},
		Thresholds: config.ThresholdsConfig{
			WarnMemoryPercent:     70,
			HighMemoryPercent:     80,
			CriticalMemoryPercent: 88,
			DiskWarnPercent:       80,
			DiskHighPercent:       90,
		},
		Timeouts: config.TimeoutsConfig{
			HeartbeatTimeoutSeconds:   30,
			OutputTimeoutSeconds:      900,
			StallTimeoutSeconds:       900,
			CheckpointIntervalSeconds: 30,
		},
		AgentRuntime: config.AgentRuntimeConfig{
			BaseDir: "/tmp/agent-runtime",
		},
	}
	return &Scheduler{
		cfg: cfg,
		newBrowseManager: func(cliPath, workspacePath string) browseEnvManager {
			return nil
		},
	}
}

func TestCanAdmit_UnderLimit(t *testing.T) {
	s := newTestScheduler()
	if !s.CanAdmit(0, 0, domain.ResourceClassLight) {
		t.Error("expected to admit with 0 active agents")
	}
	if !s.CanAdmit(1, 0, domain.ResourceClassLight) {
		t.Error("expected to admit with 1 active agent")
	}
}

func TestCanAdmit_AtLimit(t *testing.T) {
	s := newTestScheduler()
	if s.CanAdmit(2, 0, domain.ResourceClassLight) {
		t.Error("expected to reject at MaxConcurrentAgents=2")
	}
}

func TestCanAdmit_OverLimit(t *testing.T) {
	s := newTestScheduler()
	if s.CanAdmit(5, 0, domain.ResourceClassLight) {
		t.Error("expected to reject over MaxConcurrentAgents")
	}
}

func TestCanAdmit_HeavyUnderHeavyLimit(t *testing.T) {
	s := newTestScheduler()
	if !s.CanAdmit(0, 0, domain.ResourceClassHeavy) {
		t.Error("expected to admit heavy task with 0 active agents")
	}
}

func TestCanAdmit_HeavyAtHeavyLimit(t *testing.T) {
	s := newTestScheduler()
	if s.CanAdmit(0, 1, domain.ResourceClassHeavy) {
		t.Error("expected to reject heavy task at MaxHeavyAgents=1")
	}
}

func TestCanAdmit_LightAtHeavyLimit(t *testing.T) {
	s := newTestScheduler()
	if !s.CanAdmit(1, 1, domain.ResourceClassLight) {
		t.Error("expected light task to be admitted even at MaxHeavyAgents")
	}
}

func TestCanAdmit_HeavyCountSeparate(t *testing.T) {
	s := newTestScheduler()
	// MaxHeavyAgents=1, heavyCount=1 should be at limit
	if s.CanAdmit(1, 1, domain.ResourceClassHeavy) {
		t.Error("expected to reject heavy when heavyCount=1 >= MaxHeavyAgents=1")
	}
	// Light agent shouldn't affect heavy admission
	if !s.CanAdmit(1, 0, domain.ResourceClassHeavy) {
		t.Error("expected to admit heavy when heavyCount=0 < MaxHeavyAgents=1")
	}
	// Key test: 1 active light agent, 0 heavy — should admit heavy
	// This would have failed with the old code (activeCount=1 >= MaxHeavyAgents=1)
	if !s.CanAdmit(1, 0, domain.ResourceClassHeavy) {
		t.Error("expected to admit heavy when heavyCount=0 even with activeCount=1")
	}
}

func TestDeterminePressure_Normal(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(50, 50); level != PressureNormal {
		t.Errorf("expected normal, got %s", level)
	}
}

func TestDeterminePressure_WarnMemory(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(75, 50); level != PressureWarn {
		t.Errorf("expected warn, got %s", level)
	}
}

func TestDeterminePressure_WarnDisk(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(50, 80); level != PressureWarn {
		t.Errorf("expected warn at disk=80, got %s", level)
	}
}

func TestDeterminePressure_HighMemory(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(85, 50); level != PressureHigh {
		t.Errorf("expected high, got %s", level)
	}
}

func TestDeterminePressure_HighDisk(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(50, 85); level != PressureHigh {
		t.Errorf("expected high at disk=85, got %s", level)
	}
}

func TestDeterminePressure_CriticalMemory(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(90, 50); level != PressureCritical {
		t.Errorf("expected critical, got %s", level)
	}
}

func TestDeterminePressure_CriticalDisk(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(50, 95); level != PressureCritical {
		t.Errorf("expected critical, got %s", level)
	}
}

func TestDeterminePressure_BoundaryWarn(t *testing.T) {
	s := newTestScheduler()
	cfg := s.cfg.Thresholds
	if level := s.determinePressure(float64(cfg.WarnMemoryPercent-1), float64(cfg.DiskWarnPercent-1)); level != PressureNormal {
		t.Errorf("expected normal below boundary, got %s", level)
	}
	if level := s.determinePressure(float64(cfg.WarnMemoryPercent), float64(cfg.DiskWarnPercent-1)); level != PressureWarn {
		t.Errorf("expected warn at exact memory boundary, got %s", level)
	}
	if level := s.determinePressure(float64(cfg.WarnMemoryPercent-1), float64(cfg.DiskWarnPercent)); level != PressureWarn {
		t.Errorf("expected warn at exact disk boundary, got %s", level)
	}
}

func TestDeterminePressure_ZeroValues(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(0, 0); level != PressureNormal {
		t.Errorf("expected normal at 0%%, got %s", level)
	}
}

func TestHandleLoadShedding_NormalDoesNothing(t *testing.T) {
	s := newTestScheduler()
	s.handleLoadShedding(nil, PressureNormal, nil)
}

func TestHandleLoadShedding_WarnDoesNothing(t *testing.T) {
	s := newTestScheduler()
	s.handleLoadShedding(nil, PressureWarn, nil)
}

func TestPressureLevelConstants(t *testing.T) {
	if PressureNormal != "normal" {
		t.Errorf("expected 'normal', got %s", PressureNormal)
	}
	if PressureWarn != "warn" {
		t.Errorf("expected 'warn', got %s", PressureWarn)
	}
	if PressureHigh != "high" {
		t.Errorf("expected 'high', got %s", PressureHigh)
	}
	if PressureCritical != "critical" {
		t.Errorf("expected 'critical', got %s", PressureCritical)
	}
}

type stubBrowseManager struct {
	env map[string]string
	err error
}

func (m *stubBrowseManager) EnsureDaemon(_ context.Context) error { return m.err }
func (m *stubBrowseManager) BuildEnv() map[string]string          { return m.env }

func TestBuildEnv_IncludesBrowseForQATask(t *testing.T) {
	s := newTestScheduler()
	s.cfg.AgentRuntime.BrowseCLIPath = "/opt/gstack/browse"

	called := false
	s.newBrowseManager = func(cliPath, workspacePath string) browseEnvManager {
		called = true
		if cliPath != "/opt/gstack/browse" {
			t.Fatalf("unexpected cli path: %s", cliPath)
		}
		if workspacePath != "/tmp/ws" {
			t.Fatalf("unexpected workspace path: %s", workspacePath)
		}
		return &stubBrowseManager{
			env: map[string]string{
				"BROWSE_STATE_FILE": "/tmp/ws/.gstack/browse.json",
				"PATH":              "/opt/gstack:$PATH",
			},
		}
	}

	env, err := s.buildEnv(context.Background(), &domain.Task{
		ID:            "task-qa",
		TaskType:      "qa",
		WorkspacePath: "/tmp/ws",
	})
	if err != nil {
		t.Fatalf("buildEnv failed: %v", err)
	}
	if !called {
		t.Fatal("expected browse manager to be invoked")
	}
	if env["BROWSE_STATE_FILE"] != "/tmp/ws/.gstack/browse.json" {
		t.Fatalf("expected browse state env, got %#v", env)
	}
}

func TestActiveBrowseWorkspaces(t *testing.T) {
	s := newTestScheduler()

	workspaces := s.activeBrowseWorkspaces([]storage.ActiveAgentsResult{
		{
			Task: &domain.Task{TaskType: "qa", WorkspacePath: "/tmp/a"},
		},
		{
			Task: &domain.Task{TaskType: "build", WorkspacePath: "/tmp/b"},
		},
		{
			Task: &domain.Task{TaskType: "browser-qa", WorkspacePath: "/tmp/c"},
		},
	})

	if len(workspaces) != 2 {
		t.Fatalf("expected 2 occupied browse workspaces, got %d", len(workspaces))
	}
	if _, ok := workspaces["/tmp/a"]; !ok {
		t.Fatal("expected /tmp/a to be occupied")
	}
	if _, ok := workspaces["/tmp/c"]; !ok {
		t.Fatal("expected /tmp/c to be occupied")
	}
	if _, ok := workspaces["/tmp/b"]; ok {
		t.Fatal("did not expect non-QA workspace to be occupied")
	}
}

type stubPromptBuilder struct {
	prompt string
	err    error
	opts   promptengine.BuildOptions
}

func (s *stubPromptBuilder) BuildPrompt(opts promptengine.BuildOptions) (string, error) {
	s.opts = opts
	if s.err != nil {
		return "", s.err
	}
	return s.prompt, nil
}

type stubLearningSearcher struct {
	insights []string
	err      error
	opts     learningSearchCall
}

type learningSearchCall struct {
	projectSlug string
	phase       string
	queryText   string
	limit       int
}

func (s *stubLearningSearcher) SearchInsights(opts learningengine.SearchOptions) ([]string, error) {
	s.opts = learningSearchCall{
		projectSlug: opts.ProjectSlug,
		phase:       opts.Phase,
		queryText:   opts.QueryText,
		limit:       opts.Limit,
	}
	if s.err != nil {
		return nil, s.err
	}
	return s.insights, nil
}

func TestBuildPrompt_UsesPromptBuilder(t *testing.T) {
	s := newTestScheduler()
	builder := &stubPromptBuilder{prompt: "prompt-from-engine"}
	searcher := &stubLearningSearcher{
		insights: []string{"[qa/pitfall/c=9] qa-screenshot: use absolute path"},
	}
	s.promptBuilder = builder
	s.learningSearcher = searcher

	task := &domain.Task{
		ID:          "task-12345678",
		TaskType:    "review",
		Title:       "review task",
		Description: "check regressions",
		InputData:   "diff context",
	}

	got := s.buildPrompt(context.Background(), task, "claude-code")
	if got != "prompt-from-engine" {
		t.Fatalf("expected engine prompt, got: %s", got)
	}
	if builder.opts.Phase != "review" {
		t.Fatalf("expected phase review, got: %s", builder.opts.Phase)
	}
	if builder.opts.AgentKind != "claude-code" {
		t.Fatalf("expected agent kind claude-code, got: %s", builder.opts.AgentKind)
	}
	if builder.opts.Task == nil || builder.opts.Task.ID != task.ID {
		t.Fatalf("expected task propagated to builder")
	}
	if len(builder.opts.Learnings) != 1 {
		t.Fatalf("expected learnings injected, got %#v", builder.opts.Learnings)
	}
	if searcher.opts.phase != "review" {
		t.Fatalf("expected learning search phase=review, got %s", searcher.opts.phase)
	}
}

func TestBuildPrompt_FallbackToLegacyWhenEngineFails(t *testing.T) {
	s := newTestScheduler()
	s.promptBuilder = &stubPromptBuilder{err: errors.New("template missing")}

	task := &domain.Task{
		ID:          "task-87654321",
		TaskType:    "review",
		Title:       "legacy fallback",
		Description: "legacy description",
		InputData:   "legacy input",
	}

	got := s.buildPrompt(context.Background(), task, "claude-code")
	if got == "" {
		t.Fatal("expected non-empty fallback prompt")
	}
	if got != s.buildPromptLegacy(task, nil) {
		t.Fatalf("expected legacy prompt fallback, got: %s", got)
	}
}

func TestPrepareTaskForPrompt_WrapsQATrustBoundary(t *testing.T) {
	s := newTestScheduler()
	task := &domain.Task{
		ID:          "task-qa-1",
		TaskType:    "qa",
		Description: "执行浏览器冒烟",
		InputData:   "browse snapshot -ic",
	}

	prepared := s.prepareTaskForPrompt(task)
	if prepared == task {
		t.Fatalf("expected qa task to be cloned before prompt mutation")
	}
	if !strings.Contains(prepared.Description, "Trust Boundary") {
		t.Fatalf("expected trust-boundary rule in description, got: %s", prepared.Description)
	}
	if !strings.Contains(prepared.InputData, "BEGIN UNTRUSTED WEB CONTENT") {
		t.Fatalf("expected wrapped untrusted content, got: %s", prepared.InputData)
	}
	if task.InputData != "browse snapshot -ic" {
		t.Fatalf("expected original task untouched, got: %s", task.InputData)
	}
}

func TestAppendFailureLearning_WritesJSONL(t *testing.T) {
	s := newTestScheduler()
	root := t.TempDir()
	s.learningStore = learningengine.NewStore(root)

	workspace := t.TempDir()
	task := &domain.Task{
		ID:            "task-fail-1",
		TaskType:      "qa",
		WorkspacePath: workspace,
	}

	s.appendFailureLearning(task, "task timeout: exceeded 30 seconds")

	slug := learningengine.SanitizeSlug(filepath.Base(workspace))
	path := filepath.Join(root, slug, "learnings.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read learnings file failed: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "task timeout: exceeded 30 seconds") {
		t.Fatalf("expected failure reason written to learnings jsonl, got: %s", text)
	}
	if !strings.Contains(text, "\"skill\":\"qa\"") {
		t.Fatalf("expected qa skill written, got: %s", text)
	}
}

func TestDetectCanaryLeak(t *testing.T) {
	s := newTestScheduler()
	task := &domain.Task{
		ID:       "task-qa-canary",
		TaskType: "qa",
	}
	s.setTaskCanary(task.ID, "abc123def456")

	leaked, reason := s.detectCanaryLeak(task, "model output ... abc123def456 ...")
	if !leaked {
		t.Fatalf("expected canary leak to be detected")
	}
	if !strings.Contains(reason, "potential prompt injection") {
		t.Fatalf("expected security reason, got: %s", reason)
	}

	s.clearTaskCanary(task.ID)
	leaked, _ = s.detectCanaryLeak(task, "abc123def456")
	if leaked {
		t.Fatalf("expected no leak after canary cleared")
	}
}
