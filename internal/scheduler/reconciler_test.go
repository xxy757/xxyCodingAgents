package scheduler

import (
	"context"
	"testing"

	"github.com/xxy757/xxyCodingAgents/internal/domain"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
)

func setupTestDB(t *testing.T) *storage.Repos {
	t.Helper()
	db, err := storage.NewDB(":memory:", false, 1000)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.RunMigrations(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return storage.NewRepos(db)
}

func createTestProject(t *testing.T, repos *storage.Repos) string {
	t.Helper()
	p := &domain.Project{ID: "proj-1", Name: "test-project", RepoURL: "https://github.com/test/repo"}
	if err := repos.Projects.Create(p); err != nil {
		t.Fatalf("create project: %v", err)
	}
	return p.ID
}

func createTestRun(t *testing.T, repos *storage.Repos, projectID string) *domain.Run {
	t.Helper()
	r := &domain.Run{
		ID:        "run-1",
		ProjectID: projectID,
		Title:     "test-run",
		Status:    domain.RunStatusPending,
	}
	if err := repos.Runs.Create(r); err != nil {
		t.Fatalf("create run: %v", err)
	}
	return r
}

type mockTerminalChecker struct {
	exists map[string]bool
}

func (m *mockTerminalChecker) SessionExists(_ context.Context, name string) bool {
	return m.exists[name]
}

func TestReconciler_AgentRunningWithAliveTmux(t *testing.T) {
	repos := setupTestDB(t)
	projectID := createTestProject(t, repos)
	_ = createTestRun(t, repos, projectID)

	task := &domain.Task{
		ID:     "task-1", RunID: "run-1", TaskType: "code",
		Status: domain.TaskStatusRunning, Title: "test-task",
		ResourceClass: domain.ResourceClassLight, Preemptible: true,
	}
	repos.Tasks.Create(task)

	agent := &domain.AgentInstance{
		ID: "agent-1", RunID: "run-1", TaskID: "task-1",
		AgentKind: "generic-shell", Status: domain.AgentStatusRunning,
		TmuxSession: "agent-test",
	}
	repos.AgentInstances.Create(agent)

	mock := &mockTerminalChecker{exists: map[string]bool{"agent-test": true}}
	r := NewReconciler(repos, mock)
	r.Run(context.Background())

	updated, _ := repos.AgentInstances.GetByID("agent-1")
	if updated.Status != domain.AgentStatusRunning {
		t.Errorf("expected running, got %s", updated.Status)
	}
}

func TestReconciler_AgentStartingWithDeadTmux(t *testing.T) {
	repos := setupTestDB(t)
	projectID := createTestProject(t, repos)
	_ = createTestRun(t, repos, projectID)

	task := &domain.Task{
		ID: "task-1", RunID: "run-1", TaskType: "code",
		Status: domain.TaskStatusRunning, Title: "test-task",
		ResourceClass: domain.ResourceClassLight, Preemptible: true,
	}
	repos.Tasks.Create(task)

	agent := &domain.AgentInstance{
		ID: "agent-2", RunID: "run-1", TaskID: "task-1",
		AgentKind: "generic-shell", Status: domain.AgentStatusStarting,
		TmuxSession: "agent-dead",
	}
	repos.AgentInstances.Create(agent)

	mock := &mockTerminalChecker{exists: map[string]bool{}}
	r := NewReconciler(repos, mock)
	r.Run(context.Background())

	updated, _ := repos.AgentInstances.GetByID("agent-2")
	if updated.Status != domain.AgentStatusFailed {
		t.Errorf("expected failed, got %s", updated.Status)
	}
}

func TestReconciler_AgentWithCheckpoint(t *testing.T) {
	repos := setupTestDB(t)
	projectID := createTestProject(t, repos)
	_ = createTestRun(t, repos, projectID)

	task := &domain.Task{
		ID: "task-1", RunID: "run-1", TaskType: "code",
		Status: domain.TaskStatusRunning, Title: "test-task",
		ResourceClass: domain.ResourceClassLight, Preemptible: true,
	}
	repos.Tasks.Create(task)

	cpID := "cp-123"
	agent := &domain.AgentInstance{
		ID: "agent-3", RunID: "run-1", TaskID: "task-1",
		AgentKind: "generic-shell", Status: domain.AgentStatusRunning,
		TmuxSession: "agent-gone", CheckpointID: &cpID,
	}
	repos.AgentInstances.Create(agent)

	mock := &mockTerminalChecker{exists: map[string]bool{}}
	r := NewReconciler(repos, mock)
	r.Run(context.Background())

	updated, _ := repos.AgentInstances.GetByID("agent-3")
	if updated.Status != domain.AgentStatusRecoverable {
		t.Errorf("expected recoverable, got %s", updated.Status)
	}
}

func TestReconciler_PausedAgentWithAliveTmux(t *testing.T) {
	repos := setupTestDB(t)
	projectID := createTestProject(t, repos)
	_ = createTestRun(t, repos, projectID)

	task := &domain.Task{
		ID: "task-1", RunID: "run-1", TaskType: "code",
		Status: domain.TaskStatusRunning, Title: "test-task",
		ResourceClass: domain.ResourceClassLight, Preemptible: true,
	}
	repos.Tasks.Create(task)

	agent := &domain.AgentInstance{
		ID: "agent-4", RunID: "run-1", TaskID: "task-1",
		AgentKind: "generic-shell", Status: domain.AgentStatusPaused,
		TmuxSession: "agent-paused",
	}
	repos.AgentInstances.Create(agent)

	mock := &mockTerminalChecker{exists: map[string]bool{"agent-paused": true}}
	r := NewReconciler(repos, mock)
	r.Run(context.Background())

	updated, _ := repos.AgentInstances.GetByID("agent-4")
	if updated.Status != domain.AgentStatusPaused {
		t.Errorf("expected paused, got %s", updated.Status)
	}
}

func TestReconciler_NoActiveAgents(t *testing.T) {
	repos := setupTestDB(t)
	mock := &mockTerminalChecker{exists: map[string]bool{}}
	r := NewReconciler(repos, mock)
	if err := r.Run(context.Background()); err != nil {
		t.Errorf("expected no error with no agents, got %v", err)
	}
}
