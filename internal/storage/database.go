package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func NewDB(dbPath string, walMode bool, busyTimeoutMs int) (*DB, error) {
	dsn := fmt.Sprintf("%s?_busy_timeout=%d&_journal_mode=WAL&_foreign_keys=1", dbPath, busyTimeoutMs)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

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

func (db *DB) RunMigrations() error {
	migrations := []string{
		migrateProjects,
		migrateRuns,
		migrateTasks,
		migrateAgentInstances,
		migrateWorkspaces,
		migrateTerminalSessions,
		migrateCheckpoints,
		migrateResourceSnapshots,
		migrateEvents,
		migrateCommandLogs,
		migrateTaskSpecs,
		migrateAgentSpecs,
		migrateWorkflowTemplates,
		migrateIndexes,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("run migration: %w\nSQL: %s", err, m)
		}
	}
	return nil
}

const migrateProjects = `
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    repo_url TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);`

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

const migrateWorkflowTemplates = `
CREATE TABLE IF NOT EXISTS workflow_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    nodes_json TEXT NOT NULL DEFAULT '[]',
    edges_json TEXT NOT NULL DEFAULT '[]',
    on_failure TEXT NOT NULL DEFAULT 'abort'
);`

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
