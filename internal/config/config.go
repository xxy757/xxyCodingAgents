// Package config 提供应用程序配置的加载和管理功能。
// 支持从 YAML 文件读取配置，通过环境变量覆盖，并为所有配置项提供合理的默认值。
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config 是应用程序的顶层配置结构，包含所有子模块的配置。
type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Runtime      RuntimeConfig      `yaml:"runtime"`
	Scheduler    SchedulerConfig    `yaml:"scheduler"`
	Thresholds   ThresholdsConfig   `yaml:"thresholds"`
	Timeouts     TimeoutsConfig     `yaml:"timeouts"`
	SQLite       SQLiteConfig       `yaml:"sqlite"`
	AgentRuntime AgentRuntimeConfig `yaml:"agent_runtime"`
}

// ServerConfig 定义 HTTP 服务器相关配置。
type ServerConfig struct {
	HTTPAddr       string   `yaml:"http_addr"`       // API 服务监听地址，如 ":8080"
	PprofAddr      string   `yaml:"pprof_addr"`      // pprof 性能分析服务监听地址
	AllowedOrigins []string `yaml:"allowed_origins"` // CORS 允许的来源域名列表
}

// RuntimeConfig 定义运行时目录路径配置。
type RuntimeConfig struct {
	WorkspaceRoot  string `yaml:"workspace_root"`  // Agent 工作区的根目录
	LogRoot        string `yaml:"log_root"`        // 日志文件存储根目录
	CheckpointRoot string `yaml:"checkpoint_root"` // 检查点文件存储根目录
}

// SchedulerConfig 定义调度器相关配置。
type SchedulerConfig struct {
	TickSeconds         int `yaml:"tick_seconds"`          // 调度器轮询间隔（秒）
	MaxConcurrentAgents int `yaml:"max_concurrent_agents"` // 最大并发 Agent 数量
	MaxHeavyAgents      int `yaml:"max_heavy_agents"`      // 最大重型（高资源）Agent 数量
	MaxTestJobs         int `yaml:"max_test_jobs"`         // 最大测试任务数量
}

// ThresholdsConfig 定义资源阈值和清理策略配置。
type ThresholdsConfig struct {
	WarnMemoryPercent         int `yaml:"warn_memory_percent"`           // 内存使用率告警阈值（%）
	HighMemoryPercent         int `yaml:"high_memory_percent"`           // 内存使用率高水位（%），触发负载保护
	CriticalMemoryPercent     int `yaml:"critical_memory_percent"`       // 内存使用率临界值（%），触发驱逐
	DiskWarnPercent           int `yaml:"disk_warn_percent"`             // 磁盘使用率告警阈值（%）
	DiskHighPercent           int `yaml:"disk_high_percent"`             // 磁盘使用率高水位（%）
	WorkspaceMaxSizeMB        int `yaml:"workspace_max_size_mb"`         // 单个工作区最大体积（MB）
	LogRetentionDays          int `yaml:"log_retention_days"`            // 日志保留天数
	MaxTotalLogSizeMB         int `yaml:"max_total_log_size_mb"`         // 日志总大小上限（MB）
	MaxChildProcessesPerAgent int `yaml:"max_child_processes_per_agent"` // 每个 Agent 最大子进程数
}

// TimeoutsConfig 定义各类超时时间配置。
type TimeoutsConfig struct {
	HeartbeatTimeoutSeconds   int `yaml:"heartbeat_timeout_seconds"`   // Agent 心跳超时时间（秒）
	OutputTimeoutSeconds      int `yaml:"output_timeout_seconds"`      // Agent 输出超时时间（秒）
	StallTimeoutSeconds       int `yaml:"stall_timeout_seconds"`       // Agent 停滞超时时间（秒）
	CheckpointIntervalSeconds int `yaml:"checkpoint_interval_seconds"` // 定期检查点间隔（秒）
}

// SQLiteConfig 定义 SQLite 数据库连接配置。
type SQLiteConfig struct {
	Path          string `yaml:"path"`            // 数据库文件路径
	WALMode       bool   `yaml:"wal_mode"`        // 是否启用 WAL 日志模式
	BusyTimeoutMs int    `yaml:"busy_timeout_ms"` // 数据库忙等待超时（毫秒）
}

// AgentRuntimeConfig 定义 agent 启动器相关配置。
type AgentRuntimeConfig struct {
	BaseDir           string `yaml:"base_dir"`            // prompt 和 launcher 文件的根目录
	BrowseCLIPath     string `yaml:"browse_cli_path"`     // gstack browse 二进制绝对路径（空则不启用）
	PromptTemplateDir string `yaml:"prompt_template_dir"` // PromptEngine 模板目录
	LearningsRootDir  string `yaml:"learnings_root_dir"`  // gstack learnings 根目录（通常是 ~/.gstack/projects）
}

// HeartbeatTimeout 将心跳超时秒数转换为 time.Duration。
func (t *TimeoutsConfig) HeartbeatTimeout() time.Duration {
	return time.Duration(t.HeartbeatTimeoutSeconds) * time.Second
}

// StallTimeout 将停滞超时秒数转换为 time.Duration。
func (t *TimeoutsConfig) StallTimeout() time.Duration {
	return time.Duration(t.StallTimeoutSeconds) * time.Second
}

// CheckpointInterval 将检查点间隔秒数转换为 time.Duration。
func (t *TimeoutsConfig) CheckpointInterval() time.Duration {
	return time.Duration(t.CheckpointIntervalSeconds) * time.Second
}

// TickDuration 将调度器轮询间隔秒数转换为 time.Duration。
func (s *SchedulerConfig) TickDuration() time.Duration {
	return time.Duration(s.TickSeconds) * time.Second
}

// Load 从指定路径加载 YAML 配置文件，并应用环境变量覆盖和默认值。
func Load(path string) (*Config, error) {
	// 加载 .env 文件（如存在）
	godotenv.Load()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	// 应用环境变量覆盖
	cfg.applyEnvOverrides()
	// 填充零值字段的默认值
	cfg.setDefaults()
	return &cfg, nil
}

// applyEnvOverrides 使用环境变量覆盖配置项，支持运行时动态配置。
func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("AI_DEV_HTTP_ADDR"); v != "" {
		c.Server.HTTPAddr = v
	}
	if v := os.Getenv("AI_DEV_PPROF_ADDR"); v != "" {
		c.Server.PprofAddr = v
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
	if v := os.Getenv("AI_DEV_AGENT_RUNTIME_BASE_DIR"); v != "" {
		c.AgentRuntime.BaseDir = v
	}
	if v := os.Getenv("AI_DEV_BROWSE_CLI_PATH"); v != "" {
		c.AgentRuntime.BrowseCLIPath = v
	}
	if v := os.Getenv("AI_DEV_PROMPT_TEMPLATE_DIR"); v != "" {
		c.AgentRuntime.PromptTemplateDir = v
	}
	if v := os.Getenv("AI_DEV_LEARNINGS_ROOT_DIR"); v != "" {
		c.AgentRuntime.LearningsRootDir = v
	}
}

// setDefaults 为零值配置字段设置合理的默认值。
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
	if c.Timeouts.OutputTimeoutSeconds == 0 {
		c.Timeouts.OutputTimeoutSeconds = 900
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
	if c.AgentRuntime.BaseDir == "" {
		c.AgentRuntime.BaseDir = "./data/agent-runtime"
	}
	if c.AgentRuntime.PromptTemplateDir == "" {
		c.AgentRuntime.PromptTemplateDir = "./configs/prompts"
	}
	if c.AgentRuntime.LearningsRootDir == "" {
		home, err := os.UserHomeDir()
		if err == nil && strings.TrimSpace(home) != "" {
			c.AgentRuntime.LearningsRootDir = filepath.Join(home, ".gstack", "projects")
		} else {
			c.AgentRuntime.LearningsRootDir = "./data/learnings/projects"
		}
	}
	// 将相对路径解析为绝对路径（tmux shell 的 cwd 可能不同于 server）
	c.AgentRuntime.BaseDir = resolveAbsPath(c.AgentRuntime.BaseDir)
	if c.AgentRuntime.BrowseCLIPath != "" {
		c.AgentRuntime.BrowseCLIPath = resolveAbsPath(c.AgentRuntime.BrowseCLIPath)
	}
	c.AgentRuntime.PromptTemplateDir = resolveAbsPath(c.AgentRuntime.PromptTemplateDir)
	c.AgentRuntime.LearningsRootDir = resolveAbsPath(c.AgentRuntime.LearningsRootDir)
}

func resolveAbsPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil && strings.TrimSpace(home) != "" {
			if path == "~" {
				path = home
			} else {
				path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
			}
		}
	}
	if filepath.IsAbs(path) {
		return path
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
