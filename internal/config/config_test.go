package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSetDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.setDefaults()

	if cfg.Server.HTTPAddr != ":8080" {
		t.Errorf("expected HTTPAddr :8080, got %s", cfg.Server.HTTPAddr)
	}
	if cfg.Scheduler.TickSeconds != 3 {
		t.Errorf("expected TickSeconds 3, got %d", cfg.Scheduler.TickSeconds)
	}
	if cfg.Scheduler.MaxConcurrentAgents != 2 {
		t.Errorf("expected MaxConcurrentAgents 2, got %d", cfg.Scheduler.MaxConcurrentAgents)
	}
	if cfg.Scheduler.MaxHeavyAgents != 1 {
		t.Errorf("expected MaxHeavyAgents 1, got %d", cfg.Scheduler.MaxHeavyAgents)
	}
	if cfg.Scheduler.MaxTestJobs != 1 {
		t.Errorf("expected MaxTestJobs 1, got %d", cfg.Scheduler.MaxTestJobs)
	}
	if cfg.Thresholds.WarnMemoryPercent != 70 {
		t.Errorf("expected WarnMemoryPercent 70, got %d", cfg.Thresholds.WarnMemoryPercent)
	}
	if cfg.Thresholds.HighMemoryPercent != 80 {
		t.Errorf("expected HighMemoryPercent 80, got %d", cfg.Thresholds.HighMemoryPercent)
	}
	if cfg.Thresholds.CriticalMemoryPercent != 88 {
		t.Errorf("expected CriticalMemoryPercent 88, got %d", cfg.Thresholds.CriticalMemoryPercent)
	}
	if cfg.Thresholds.DiskWarnPercent != 80 {
		t.Errorf("expected DiskWarnPercent 80, got %d", cfg.Thresholds.DiskWarnPercent)
	}
	if cfg.Thresholds.DiskHighPercent != 90 {
		t.Errorf("expected DiskHighPercent 90, got %d", cfg.Thresholds.DiskHighPercent)
	}
	if cfg.Thresholds.WorkspaceMaxSizeMB != 2048 {
		t.Errorf("expected WorkspaceMaxSizeMB 2048, got %d", cfg.Thresholds.WorkspaceMaxSizeMB)
	}
	if cfg.Thresholds.LogRetentionDays != 7 {
		t.Errorf("expected LogRetentionDays 7, got %d", cfg.Thresholds.LogRetentionDays)
	}
	if cfg.Thresholds.MaxTotalLogSizeMB != 1024 {
		t.Errorf("expected MaxTotalLogSizeMB 1024, got %d", cfg.Thresholds.MaxTotalLogSizeMB)
	}
	if cfg.Thresholds.MaxChildProcessesPerAgent != 10 {
		t.Errorf("expected MaxChildProcessesPerAgent 10, got %d", cfg.Thresholds.MaxChildProcessesPerAgent)
	}
	if cfg.Timeouts.HeartbeatTimeoutSeconds != 30 {
		t.Errorf("expected HeartbeatTimeoutSeconds 30, got %d", cfg.Timeouts.HeartbeatTimeoutSeconds)
	}
	if cfg.Timeouts.OutputTimeoutSeconds != 900 {
		t.Errorf("expected OutputTimeoutSeconds 900, got %d", cfg.Timeouts.OutputTimeoutSeconds)
	}
	if cfg.Timeouts.StallTimeoutSeconds != 900 {
		t.Errorf("expected StallTimeoutSeconds 900, got %d", cfg.Timeouts.StallTimeoutSeconds)
	}
	if cfg.Timeouts.CheckpointIntervalSeconds != 30 {
		t.Errorf("expected CheckpointIntervalSeconds 30, got %d", cfg.Timeouts.CheckpointIntervalSeconds)
	}
	if cfg.SQLite.Path != "./data/app.db" {
		t.Errorf("expected SQLite Path ./data/app.db, got %s", cfg.SQLite.Path)
	}
	if cfg.SQLite.BusyTimeoutMs != 5000 {
		t.Errorf("expected BusyTimeoutMs 5000, got %d", cfg.SQLite.BusyTimeoutMs)
	}
	if cfg.Runtime.WorkspaceRoot != "./data/workspaces" {
		t.Errorf("expected WorkspaceRoot ./data/workspaces, got %s", cfg.Runtime.WorkspaceRoot)
	}
	if cfg.Runtime.LogRoot != "./data/logs" {
		t.Errorf("expected LogRoot ./data/logs, got %s", cfg.Runtime.LogRoot)
	}
	if cfg.Runtime.CheckpointRoot != "./data/checkpoints" {
		t.Errorf("expected CheckpointRoot ./data/checkpoints, got %s", cfg.Runtime.CheckpointRoot)
	}
	if cfg.AgentRuntime.BaseDir == "" {
		t.Error("expected AgentRuntime.BaseDir to be set")
	}
	if !filepath.IsAbs(cfg.AgentRuntime.BaseDir) {
		t.Errorf("expected AgentRuntime.BaseDir to be absolute, got %s", cfg.AgentRuntime.BaseDir)
	}
	if cfg.AgentRuntime.PromptTemplateDir == "" {
		t.Error("expected AgentRuntime.PromptTemplateDir to be set")
	}
	if !filepath.IsAbs(cfg.AgentRuntime.PromptTemplateDir) {
		t.Errorf("expected AgentRuntime.PromptTemplateDir to be absolute, got %s", cfg.AgentRuntime.PromptTemplateDir)
	}
	if cfg.AgentRuntime.LearningsRootDir == "" {
		t.Error("expected AgentRuntime.LearningsRootDir to be set")
	}
	if !filepath.IsAbs(cfg.AgentRuntime.LearningsRootDir) {
		t.Errorf("expected AgentRuntime.LearningsRootDir to be absolute, got %s", cfg.AgentRuntime.LearningsRootDir)
	}
}

func TestSetDefaultsDoesNotOverrideExisting(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{HTTPAddr: ":9090"},
		Scheduler: SchedulerConfig{
			TickSeconds:         10,
			MaxConcurrentAgents: 5,
			MaxHeavyAgents:      3,
		},
		Thresholds: ThresholdsConfig{
			WarnMemoryPercent: 60,
		},
	}
	cfg.setDefaults()

	if cfg.Server.HTTPAddr != ":9090" {
		t.Errorf("expected HTTPAddr :9090, got %s", cfg.Server.HTTPAddr)
	}
	if cfg.Scheduler.TickSeconds != 10 {
		t.Errorf("expected TickSeconds 10, got %d", cfg.Scheduler.TickSeconds)
	}
	if cfg.Scheduler.MaxConcurrentAgents != 5 {
		t.Errorf("expected MaxConcurrentAgents 5, got %d", cfg.Scheduler.MaxConcurrentAgents)
	}
	if cfg.Scheduler.MaxHeavyAgents != 3 {
		t.Errorf("expected MaxHeavyAgents 3, got %d", cfg.Scheduler.MaxHeavyAgents)
	}
	if cfg.Thresholds.WarnMemoryPercent != 60 {
		t.Errorf("expected WarnMemoryPercent 60, got %d", cfg.Thresholds.WarnMemoryPercent)
	}
	if cfg.Thresholds.HighMemoryPercent != 80 {
		t.Errorf("expected HighMemoryPercent to default to 80, got %d", cfg.Thresholds.HighMemoryPercent)
	}
}

func TestTickDuration(t *testing.T) {
	s := &SchedulerConfig{TickSeconds: 5}
	if d := s.TickDuration(); d != 5*time.Second {
		t.Errorf("expected 5s, got %v", d)
	}
}

func TestHeartbeatTimeout(t *testing.T) {
	tm := &TimeoutsConfig{HeartbeatTimeoutSeconds: 60}
	if d := tm.HeartbeatTimeout(); d != 60*time.Second {
		t.Errorf("expected 60s, got %v", d)
	}
}

func TestStallTimeout(t *testing.T) {
	tm := &TimeoutsConfig{StallTimeoutSeconds: 120}
	if d := tm.StallTimeout(); d != 120*time.Second {
		t.Errorf("expected 120s, got %v", d)
	}
}

func TestCheckpointInterval(t *testing.T) {
	tm := &TimeoutsConfig{CheckpointIntervalSeconds: 45}
	if d := tm.CheckpointInterval(); d != 45*time.Second {
		t.Errorf("expected 45s, got %v", d)
	}
}

func TestLoadWithValidFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := `
server:
  http_addr: ":9090"
scheduler:
  tick_seconds: 10
  max_concurrent_agents: 5
thresholds:
  warn_memory_percent: 60
timeouts:
  heartbeat_timeout_seconds: 60
sqlite:
  path: "` + filepath.Join(dir, "test.db") + `"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Server.HTTPAddr != ":9090" {
		t.Errorf("expected :9090, got %s", cfg.Server.HTTPAddr)
	}
	if cfg.Scheduler.TickSeconds != 10 {
		t.Errorf("expected 10, got %d", cfg.Scheduler.TickSeconds)
	}
	if cfg.Scheduler.MaxConcurrentAgents != 5 {
		t.Errorf("expected 5, got %d", cfg.Scheduler.MaxConcurrentAgents)
	}
	if cfg.Thresholds.WarnMemoryPercent != 60 {
		t.Errorf("expected 60, got %d", cfg.Thresholds.WarnMemoryPercent)
	}
	if cfg.Timeouts.HeartbeatTimeoutSeconds != 60 {
		t.Errorf("expected 60, got %d", cfg.Timeouts.HeartbeatTimeoutSeconds)
	}
	if cfg.Thresholds.HighMemoryPercent != 80 {
		t.Errorf("expected default 80, got %d", cfg.Thresholds.HighMemoryPercent)
	}
}

func TestLoadWithNonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	cfg := &Config{}
	cfg.setDefaults()

	os.Setenv("AI_DEV_HTTP_ADDR", ":7070")
	os.Setenv("AI_DEV_SQLITE_PATH", "/tmp/test.db")
	os.Setenv("AI_DEV_WORKSPACE_ROOT", "/tmp/ws")
	os.Setenv("AI_DEV_AGENT_RUNTIME_BASE_DIR", "/tmp/agent-runtime")
	os.Setenv("AI_DEV_BROWSE_CLI_PATH", "/tmp/bin/browse")
	os.Setenv("AI_DEV_PROMPT_TEMPLATE_DIR", "/tmp/prompts")
	os.Setenv("AI_DEV_LEARNINGS_ROOT_DIR", "/tmp/learnings")
	defer func() {
		os.Unsetenv("AI_DEV_HTTP_ADDR")
		os.Unsetenv("AI_DEV_SQLITE_PATH")
		os.Unsetenv("AI_DEV_WORKSPACE_ROOT")
		os.Unsetenv("AI_DEV_AGENT_RUNTIME_BASE_DIR")
		os.Unsetenv("AI_DEV_BROWSE_CLI_PATH")
		os.Unsetenv("AI_DEV_PROMPT_TEMPLATE_DIR")
		os.Unsetenv("AI_DEV_LEARNINGS_ROOT_DIR")
	}()

	cfg.applyEnvOverrides()

	if cfg.Server.HTTPAddr != ":7070" {
		t.Errorf("expected :7070, got %s", cfg.Server.HTTPAddr)
	}
	if cfg.SQLite.Path != "/tmp/test.db" {
		t.Errorf("expected /tmp/test.db, got %s", cfg.SQLite.Path)
	}
	if cfg.Runtime.WorkspaceRoot != "/tmp/ws" {
		t.Errorf("expected /tmp/ws, got %s", cfg.Runtime.WorkspaceRoot)
	}
	if cfg.AgentRuntime.BaseDir != "/tmp/agent-runtime" {
		t.Errorf("expected /tmp/agent-runtime, got %s", cfg.AgentRuntime.BaseDir)
	}
	if cfg.AgentRuntime.BrowseCLIPath != "/tmp/bin/browse" {
		t.Errorf("expected /tmp/bin/browse, got %s", cfg.AgentRuntime.BrowseCLIPath)
	}
	if cfg.AgentRuntime.PromptTemplateDir != "/tmp/prompts" {
		t.Errorf("expected /tmp/prompts, got %s", cfg.AgentRuntime.PromptTemplateDir)
	}
	if cfg.AgentRuntime.LearningsRootDir != "/tmp/learnings" {
		t.Errorf("expected /tmp/learnings, got %s", cfg.AgentRuntime.LearningsRootDir)
	}
}

func TestAllowedOriginsDefault(t *testing.T) {
	cfg := &Config{}
	cfg.setDefaults()
	if len(cfg.Server.AllowedOrigins) != 2 {
		t.Errorf("expected 2 allowed origins, got %d", len(cfg.Server.AllowedOrigins))
	}
	if cfg.Server.AllowedOrigins[0] != "http://localhost:3000" {
		t.Errorf("expected first origin http://localhost:3000, got %s", cfg.Server.AllowedOrigins[0])
	}
}
