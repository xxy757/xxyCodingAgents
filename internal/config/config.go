package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Runtime    RuntimeConfig    `yaml:"runtime"`
	Scheduler  SchedulerConfig  `yaml:"scheduler"`
	Thresholds ThresholdsConfig `yaml:"thresholds"`
	Timeouts   TimeoutsConfig   `yaml:"timeouts"`
	SQLite     SQLiteConfig     `yaml:"sqlite"`
}

type ServerConfig struct {
	HTTPAddr        string   `yaml:"http_addr"`
	AllowedOrigins  []string `yaml:"allowed_origins"`
}

type RuntimeConfig struct {
	WorkspaceRoot   string `yaml:"workspace_root"`
	LogRoot         string `yaml:"log_root"`
	CheckpointRoot  string `yaml:"checkpoint_root"`
}

type SchedulerConfig struct {
	TickSeconds        int `yaml:"tick_seconds"`
	MaxConcurrentAgents int `yaml:"max_concurrent_agents"`
	MaxHeavyAgents     int `yaml:"max_heavy_agents"`
	MaxTestJobs        int `yaml:"max_test_jobs"`
}

type ThresholdsConfig struct {
	WarnMemoryPercent        int `yaml:"warn_memory_percent"`
	HighMemoryPercent        int `yaml:"high_memory_percent"`
	CriticalMemoryPercent    int `yaml:"critical_memory_percent"`
	DiskWarnPercent          int `yaml:"disk_warn_percent"`
	DiskHighPercent          int `yaml:"disk_high_percent"`
	WorkspaceMaxSizeMB       int `yaml:"workspace_max_size_mb"`
	LogRetentionDays         int `yaml:"log_retention_days"`
	MaxTotalLogSizeMB        int `yaml:"max_total_log_size_mb"`
	MaxChildProcessesPerAgent int `yaml:"max_child_processes_per_agent"`
}

type TimeoutsConfig struct {
	HeartbeatTimeoutSeconds    int `yaml:"heartbeat_timeout_seconds"`
	StallTimeoutSeconds        int `yaml:"stall_timeout_seconds"`
	CheckpointIntervalSeconds  int `yaml:"checkpoint_interval_seconds"`
}

type SQLiteConfig struct {
	Path         string `yaml:"path"`
	WALMode      bool   `yaml:"wal_mode"`
	BusyTimeoutMs int   `yaml:"busy_timeout_ms"`
}

func (t *TimeoutsConfig) HeartbeatTimeout() time.Duration {
	return time.Duration(t.HeartbeatTimeoutSeconds) * time.Second
}

func (t *TimeoutsConfig) StallTimeout() time.Duration {
	return time.Duration(t.StallTimeoutSeconds) * time.Second
}

func (t *TimeoutsConfig) CheckpointInterval() time.Duration {
	return time.Duration(t.CheckpointIntervalSeconds) * time.Second
}

func (s *SchedulerConfig) TickDuration() time.Duration {
	return time.Duration(s.TickSeconds) * time.Second
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	cfg.applyEnvOverrides()
	cfg.setDefaults()
	return &cfg, nil
}

func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("AI_DEV_HTTP_ADDR"); v != "" {
		c.Server.HTTPAddr = v
	}
	if v := os.Getenv("AI_DEV_SQLITE_PATH"); v != "" {
		c.SQLite.Path = v
	}
	if v := os.Getenv("AI_DEV_WORKSPACE_ROOT"); v != "" {
		c.Runtime.WorkspaceRoot = v
	}
	if v := os.Getenv("AI_DEV_LOG_ROOT"); v != "" {
		c.Runtime.LogRoot = v
	}
	if v := os.Getenv("AI_DEV_CHECKPOINT_ROOT"); v != "" {
		c.Runtime.CheckpointRoot = v
	}
}

func (c *Config) setDefaults() {
	if c.Server.HTTPAddr == "" {
		c.Server.HTTPAddr = ":8080"
	}
	if len(c.Server.AllowedOrigins) == 0 {
		c.Server.AllowedOrigins = []string{"http://localhost:3000", "http://localhost:8080"}
	}
	if c.Runtime.WorkspaceRoot == "" {
		c.Runtime.WorkspaceRoot = "./data/workspaces"
	}
	if c.Runtime.LogRoot == "" {
		c.Runtime.LogRoot = "./data/logs"
	}
	if c.Runtime.CheckpointRoot == "" {
		c.Runtime.CheckpointRoot = "./data/checkpoints"
	}
	if c.SQLite.Path == "" {
		c.SQLite.Path = "./data/app.db"
	}
	if c.Scheduler.TickSeconds == 0 {
		c.Scheduler.TickSeconds = 3
	}
	if c.Scheduler.MaxConcurrentAgents == 0 {
		c.Scheduler.MaxConcurrentAgents = 2
	}
	if c.Scheduler.MaxHeavyAgents == 0 {
		c.Scheduler.MaxHeavyAgents = 1
	}
	if c.Scheduler.MaxTestJobs == 0 {
		c.Scheduler.MaxTestJobs = 1
	}
	if c.Thresholds.WarnMemoryPercent == 0 {
		c.Thresholds.WarnMemoryPercent = 70
	}
	if c.Thresholds.HighMemoryPercent == 0 {
		c.Thresholds.HighMemoryPercent = 80
	}
	if c.Thresholds.CriticalMemoryPercent == 0 {
		c.Thresholds.CriticalMemoryPercent = 88
	}
	if c.Thresholds.DiskWarnPercent == 0 {
		c.Thresholds.DiskWarnPercent = 80
	}
	if c.Thresholds.DiskHighPercent == 0 {
		c.Thresholds.DiskHighPercent = 90
	}
	if c.Thresholds.WorkspaceMaxSizeMB == 0 {
		c.Thresholds.WorkspaceMaxSizeMB = 2048
	}
	if c.Thresholds.LogRetentionDays == 0 {
		c.Thresholds.LogRetentionDays = 7
	}
	if c.Thresholds.MaxTotalLogSizeMB == 0 {
		c.Thresholds.MaxTotalLogSizeMB = 1024
	}
	if c.Thresholds.MaxChildProcessesPerAgent == 0 {
		c.Thresholds.MaxChildProcessesPerAgent = 10
	}
	if c.Timeouts.HeartbeatTimeoutSeconds == 0 {
		c.Timeouts.HeartbeatTimeoutSeconds = 30
	}
	if c.Timeouts.StallTimeoutSeconds == 0 {
		c.Timeouts.StallTimeoutSeconds = 900
	}
	if c.Timeouts.CheckpointIntervalSeconds == 0 {
		c.Timeouts.CheckpointIntervalSeconds = 30
	}
	if c.SQLite.BusyTimeoutMs == 0 {
		c.SQLite.BusyTimeoutMs = 5000
	}
}
