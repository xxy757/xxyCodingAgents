// Package storage 提供数据库连接管理和版本化迁移功能。
// 使用 SQLite 作为底层存储，支持 WAL 模式以提升并发性能。
package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite 驱动
)

// DB 封装 sql.DB，提供数据库连接和迁移管理。
type DB struct {
	*sql.DB
}

// NewDB 创建并初始化 SQLite 数据库连接。
// walMode 为 true 时启用 WAL 日志模式以支持并发读写。
func NewDB(dbPath string, walMode bool, busyTimeoutMs int) (*DB, error) {
	dsn := fmt.Sprintf("%s?_busy_timeout=%d&_journal_mode=WAL&_foreign_keys=1", dbPath, busyTimeoutMs)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// 配置连接池参数
	if walMode {
		db.SetMaxOpenConns(5)
	} else {
		db.SetMaxOpenConns(1)
	}
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{db}, nil
}

// RunMigrations 执行所有未应用的数据库迁移，按版本号顺序执行。
func (db *DB) RunMigrations() error {
	// 创建迁移版本记录表
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				name TEXT NOT NULL,
				applied_at DATETIME NOT NULL DEFAULT (datetime('now'))
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	// 按顺序定义所有迁移
	migrations := []struct {
		Name string
		SQL  string
	}{
		{"projects", migrateProjects},
		{"runs", migrateRuns},
		{"tasks", migrateTasks},
		{"agent_instances", migrateAgentInstances},
		{"workspaces", migrateWorkspaces},
		{"terminal_sessions", migrateTerminalSessions},
		{"checkpoints", migrateCheckpoints},
		{"resource_snapshots", migrateResourceSnapshots},
		{"events", migrateEvents},
		{"command_logs", migrateCommandLogs},
		{"task_specs", migrateTaskSpecs},
		{"agent_specs", migrateAgentSpecs},
		{"workflow_templates", migrateWorkflowTemplates},
		{"indexes", migrateIndexes},
		{"prompt_drafts", migratePromptDrafts},
		{"gates", migrateGates},
	}

	for i, m := range migrations {
		// 检查该版本是否已应用
		var count int
		db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", i+1).Scan(&count)
		if count > 0 {
			continue
		}

		// 在事务中执行迁移
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration transaction for %s: %w", m.Name, err)
		}

		if _, err := tx.Exec(m.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("run migration %s: %w\nSQL: %s", m.Name, err, m.SQL)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version, name) VALUES (?, ?)", i+1, m.Name); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", m.Name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", m.Name, err)
		}
	}
	return nil
}

// CurrentVersion 返回当前已应用的最高迁移版本号。
func (db *DB) CurrentVersion() (int, error) {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	return version, err
}

// 以下是所有迁移的 SQL 定义，每个常量对应一个表或索引的创建语句。

// migrateProjects 创建项目表
const migrateProjects = `
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    repo_url TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);`

// migrateRuns 创建运行表，关联项目
const migrateRuns = `
CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    workflow_template_id TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'pending',
    external_key TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (project_id) REFERENCES projects(id)
);`

// migrateTasks 创建任务表，关联运行
const migrateTasks = `
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    task_spec_id TEXT NOT NULL DEFAULT '',
    task_type TEXT NOT NULL DEFAULT '',
    attempt_no INTEGER NOT NULL DEFAULT 1,
    status TEXT NOT NULL DEFAULT 'queued',
    priority TEXT NOT NULL DEFAULT 'normal',
    queue_status TEXT NOT NULL DEFAULT 'queued',
    resource_class TEXT NOT NULL DEFAULT 'light',
    preemptible INTEGER NOT NULL DEFAULT 1,
    restart_policy TEXT NOT NULL DEFAULT 'never',
    title TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    input_data TEXT NOT NULL DEFAULT '',
    output_data TEXT NOT NULL DEFAULT '',
    workspace_path TEXT NOT NULL DEFAULT '',
    parent_task_id TEXT,
    started_at DATETIME,
    completed_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (run_id) REFERENCES runs(id)
);`

// migrateAgentInstances 创建 Agent 实例表，关联运行和任务
const migrateAgentInstances = `
CREATE TABLE IF NOT EXISTS agent_instances (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    agent_spec_id TEXT NOT NULL DEFAULT '',
    agent_kind TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'starting',
    pid INTEGER,
    tmux_session TEXT NOT NULL DEFAULT '',
    workspace_path TEXT NOT NULL DEFAULT '',
    last_heartbeat_at DATETIME,
    last_output_at DATETIME,
    checkpoint_id TEXT,
    metadata TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (run_id) REFERENCES runs(id),
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);`

// migrateWorkspaces 创建工作区表
const migrateWorkspaces = `
CREATE TABLE IF NOT EXISTS workspaces (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    path TEXT NOT NULL UNIQUE,
    branch TEXT NOT NULL DEFAULT '',
    commit_sha TEXT NOT NULL DEFAULT '',
    size_bytes INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (task_id) REFERENCES tasks(id),
    FOREIGN KEY (project_id) REFERENCES projects(id)
);`

// migrateTerminalSessions 创建终端会话表
const migrateTerminalSessions = `
CREATE TABLE IF NOT EXISTS terminal_sessions (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    agent_id TEXT,
    tmux_session TEXT NOT NULL UNIQUE,
    tmux_pane TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    log_file_path TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);`

// migrateCheckpoints 创建检查点表
const migrateCheckpoints = `
CREATE TABLE IF NOT EXISTS checkpoints (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    phase TEXT NOT NULL DEFAULT '',
    state_data TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (agent_id) REFERENCES agent_instances(id),
    FOREIGN KEY (task_id) REFERENCES tasks(id),
    FOREIGN KEY (run_id) REFERENCES runs(id)
);`

// migrateResourceSnapshots 创建资源快照表
const migrateResourceSnapshots = `
CREATE TABLE IF NOT EXISTS resource_snapshots (
    id TEXT PRIMARY KEY,
    memory_percent REAL NOT NULL DEFAULT 0,
    cpu_percent REAL NOT NULL DEFAULT 0,
    disk_percent REAL NOT NULL DEFAULT 0,
    active_agents INTEGER NOT NULL DEFAULT 0,
    pressure_level TEXT NOT NULL DEFAULT 'normal',
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);`

// migrateEvents 创建事件表
const migrateEvents = `
CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    task_id TEXT,
    agent_id TEXT,
    event_type TEXT NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    metadata TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (run_id) REFERENCES runs(id)
);`

// migrateCommandLogs 创建命令日志表
const migrateCommandLogs = `
CREATE TABLE IF NOT EXISTS command_logs (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    agent_id TEXT,
    command TEXT NOT NULL,
    exit_code INTEGER,
    output TEXT NOT NULL DEFAULT '',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (task_id) REFERENCES tasks(id)
);`

// migrateTaskSpecs 创建任务规格表
const migrateTaskSpecs = `
CREATE TABLE IF NOT EXISTS task_specs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    task_type TEXT NOT NULL,
    runtime_type TEXT NOT NULL DEFAULT '',
    command_template TEXT NOT NULL DEFAULT '',
    timeout_seconds INTEGER NOT NULL DEFAULT 300,
    retry_policy TEXT NOT NULL DEFAULT 'never',
    resource_class TEXT NOT NULL DEFAULT 'light',
    can_pause INTEGER NOT NULL DEFAULT 1,
    can_checkpoint INTEGER NOT NULL DEFAULT 1,
    required_inputs TEXT NOT NULL DEFAULT '',
    expected_outputs TEXT NOT NULL DEFAULT ''
);`

// migrateAgentSpecs 创建 Agent 规格表
const migrateAgentSpecs = `
CREATE TABLE IF NOT EXISTS agent_specs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    agent_kind TEXT NOT NULL,
    supported_task_types TEXT NOT NULL DEFAULT '',
    default_command TEXT NOT NULL DEFAULT '',
    max_concurrency INTEGER NOT NULL DEFAULT 1,
    resource_weight REAL NOT NULL DEFAULT 1.0,
    heartbeat_mode TEXT NOT NULL DEFAULT 'process',
    output_parser TEXT NOT NULL DEFAULT 'default'
);`

// migrateWorkflowTemplates 创建工作流模板表
const migrateWorkflowTemplates = `
CREATE TABLE IF NOT EXISTS workflow_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    nodes_json TEXT NOT NULL DEFAULT '[]',
    edges_json TEXT NOT NULL DEFAULT '[]',
    on_failure TEXT NOT NULL DEFAULT 'abort'
);`

// migrateIndexes 创建常用查询的数据库索引
const migrateIndexes = `
CREATE INDEX IF NOT EXISTS idx_runs_project_status ON runs(project_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_tasks_run_status ON tasks(run_id, status, priority, created_at);
CREATE INDEX IF NOT EXISTS idx_agent_instances_run_status ON agent_instances(run_id, status, last_heartbeat_at);
CREATE INDEX IF NOT EXISTS idx_terminal_sessions_task ON terminal_sessions(task_id, status);
CREATE INDEX IF NOT EXISTS idx_events_run_created ON events(run_id, created_at);
CREATE INDEX IF NOT EXISTS idx_resource_snapshots_created ON resource_snapshots(created_at);
CREATE INDEX IF NOT EXISTS idx_command_logs_task_created ON command_logs(task_id, created_at);
CREATE INDEX IF NOT EXISTS idx_checkpoints_task_created ON checkpoints(task_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_runs_external_key ON runs(external_key) WHERE external_key != '';`

// migratePromptDrafts 创建提示词草稿表
const migratePromptDrafts = `
CREATE TABLE IF NOT EXISTS prompt_drafts (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    original_input TEXT NOT NULL,
    generated_prompt TEXT NOT NULL DEFAULT '',
    final_prompt TEXT NOT NULL DEFAULT '',
    task_type TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'draft',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (project_id) REFERENCES projects(id)
);
CREATE INDEX IF NOT EXISTS idx_prompt_drafts_project_status ON prompt_drafts(project_id, status, created_at);`

// migrateGates 创建质量门禁表
const migrateGates = `
CREATE TABLE IF NOT EXISTS gates (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    node_id TEXT NOT NULL,
    gate_type TEXT NOT NULL DEFAULT 'auto',
    status TEXT NOT NULL DEFAULT 'pending',
    config_json TEXT NOT NULL DEFAULT '{}',
    verify_result TEXT NOT NULL DEFAULT '',
    approved_by TEXT NOT NULL DEFAULT '',
    approved_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (run_id) REFERENCES runs(id)
);
CREATE INDEX IF NOT EXISTS idx_gates_run_status ON gates(run_id, status);`
