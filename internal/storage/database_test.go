package storage

import (
	"testing"
)

func TestNewDB_InMemory(t *testing.T) {
	db, err := NewDB(":memory:", false, 1000)
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestRunMigrations(t *testing.T) {
	db, err := NewDB(":memory:", false, 1000)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.RunMigrations(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	tables := []string{
		"projects", "runs", "tasks", "agent_instances",
		"workspaces", "terminal_sessions", "checkpoints",
		"resource_snapshots", "events", "command_logs",
		"task_specs", "agent_specs", "workflow_templates",
		"prompt_drafts", "gates", "schema_migrations",
	}

	for _, table := range tables {
		var count int
		err := db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil {
			t.Errorf("check table %s: %v", table, err)
		}
		if count != 1 {
			t.Errorf("expected table %s to exist", table)
		}
	}
}

func TestRunMigrations_Idempotent(t *testing.T) {
	db, err := NewDB(":memory:", false, 1000)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.RunMigrations(); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := db.RunMigrations(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestCurrentVersion(t *testing.T) {
	db, err := NewDB(":memory:", false, 1000)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	db.RunMigrations()

	v, err := db.CurrentVersion()
	if err != nil {
		t.Fatalf("version after migration: %v", err)
	}
	if v != 16 {
		t.Errorf("expected version 16 after migration, got %d", v)
	}
}

func TestNewRepos(t *testing.T) {
	db, err := NewDB(":memory:", false, 1000)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	db.RunMigrations()

	repos := NewRepos(db)
	if repos.Projects == nil {
		t.Error("Projects repo should not be nil")
	}
	if repos.Runs == nil {
		t.Error("Runs repo should not be nil")
	}
	if repos.Tasks == nil {
		t.Error("Tasks repo should not be nil")
	}
	if repos.AgentInstances == nil {
		t.Error("AgentInstances repo should not be nil")
	}
	if repos.Events == nil {
		t.Error("Events repo should not be nil")
	}
	if repos.Checkpoints == nil {
		t.Error("Checkpoints repo should not be nil")
	}
	if repos.ResourceSnapshots == nil {
		t.Error("ResourceSnapshots repo should not be nil")
	}
	if repos.Workspaces == nil {
		t.Error("Workspaces repo should not be nil")
	}
	if repos.TerminalSessions == nil {
		t.Error("TerminalSessions repo should not be nil")
	}
	if repos.CommandLogs == nil {
		t.Error("CommandLogs repo should not be nil")
	}
	if repos.TaskSpecs == nil {
		t.Error("TaskSpecs repo should not be nil")
	}
	if repos.AgentSpecs == nil {
		t.Error("AgentSpecs repo should not be nil")
	}
	if repos.WorkflowTemplates == nil {
		t.Error("WorkflowTemplates repo should not be nil")
	}
	if repos.PromptDrafts == nil {
		t.Error("PromptDrafts repo should not be nil")
	}
	if repos.Gates == nil {
		t.Error("Gates repo should not be nil")
	}
}
