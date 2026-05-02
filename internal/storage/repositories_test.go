package storage

import (
	"testing"
	"time"

	"github.com/xxy757/xxyCodingAgents/internal/domain"
)

func setupRepoTestDB(t *testing.T) *Repos {
	t.Helper()
	db, err := NewDB(":memory:", false, 1000)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.RunMigrations(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewRepos(db)
}

func TestProjectRepo_CRUD(t *testing.T) {
	repos := setupRepoTestDB(t)

	p := &domain.Project{
		ID:          "p1",
		Name:        "test-project",
		RepoURL:     "https://github.com/test/repo",
		Description: "A test project",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repos.Projects.Create(p); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repos.Projects.GetByID("p1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("expected project, got nil")
	}
	if got.Name != "test-project" {
		t.Errorf("expected name test-project, got %s", got.Name)
	}

	list, err := repos.Projects.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 project, got %d", len(list))
	}

	none, err := repos.Projects.GetByID("nonexistent")
	if err != nil {
		t.Fatalf("get nonexistent: %v", err)
	}
	if none != nil {
		t.Error("expected nil for nonexistent project")
	}
}

func TestRunRepo_CRUD(t *testing.T) {
	repos := setupRepoTestDB(t)

	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	run := &domain.Run{
		ID:        "r1",
		ProjectID: "p1",
		Title:     "test-run",
		Status:    domain.RunStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := repos.Runs.Create(run); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repos.Runs.GetByID("r1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "test-run" {
		t.Errorf("expected title test-run, got %s", got.Title)
	}

	if err := repos.Runs.UpdateStatus("r1", domain.RunStatusRunning); err != nil {
		t.Fatalf("update status: %v", err)
	}
	got, _ = repos.Runs.GetByID("r1")
	if got.Status != domain.RunStatusRunning {
		t.Errorf("expected running, got %s", got.Status)
	}

	byProject, _ := repos.Runs.ListByProject("p1")
	if len(byProject) != 1 {
		t.Errorf("expected 1 run, got %d", len(byProject))
	}

	all, _ := repos.Runs.ListAll()
	if len(all) != 1 {
		t.Errorf("expected 1 run, got %d", len(all))
	}
}

func TestTaskRepo_CRUD(t *testing.T) {
	repos := setupRepoTestDB(t)

	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	task := &domain.Task{
		ID:            "t1",
		RunID:         "r1",
		TaskType:      "code",
		AttemptNo:     1,
		Status:        domain.TaskStatusQueued,
		Priority:      domain.PriorityNormal,
		QueueStatus:   "queued",
		ResourceClass: domain.ResourceClassLight,
		Preemptible:   true,
		RestartPolicy: "never",
		Title:         "test-task",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := repos.Tasks.Create(task); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repos.Tasks.GetByID("t1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "test-task" {
		t.Errorf("expected test-task, got %s", got.Title)
	}

	if err := repos.Tasks.UpdateStatus("t1", domain.TaskStatusRunning); err != nil {
		t.Fatalf("update status: %v", err)
	}

	byStatus, _ := repos.Tasks.ListByStatus(domain.TaskStatusRunning)
	if len(byStatus) != 1 {
		t.Errorf("expected 1 running task, got %d", len(byStatus))
	}

	byRun, _ := repos.Tasks.ListByRun("r1")
	if len(byRun) != 1 {
		t.Errorf("expected 1 task for run, got %d", len(byRun))
	}

	maxAttempt, _ := repos.Tasks.MaxAttemptNo("r1", "code")
	if maxAttempt != 1 {
		t.Errorf("expected max attempt 1, got %d", maxAttempt)
	}
}

func TestTaskRepo_MarkRunningAndCompleted(t *testing.T) {
	repos := setupRepoTestDB(t)

	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	task := &domain.Task{
		ID:            "t-running",
		RunID:         "r1",
		TaskType:      "code",
		AttemptNo:     1,
		Status:        domain.TaskStatusQueued,
		Priority:      domain.PriorityNormal,
		QueueStatus:   "queued",
		ResourceClass: domain.ResourceClassLight,
		Preemptible:   true,
		RestartPolicy: "never",
		Title:         "test-task",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := repos.Tasks.Create(task); err != nil {
		t.Fatalf("create: %v", err)
	}

	startedAt := time.Now().Add(-time.Minute).UTC().Truncate(time.Second)
	if err := repos.Tasks.MarkRunning(task.ID, startedAt); err != nil {
		t.Fatalf("mark running: %v", err)
	}

	runningTask, err := repos.Tasks.GetByID(task.ID)
	if err != nil {
		t.Fatalf("get after running: %v", err)
	}
	if runningTask.Status != domain.TaskStatusRunning {
		t.Fatalf("expected running status, got %s", runningTask.Status)
	}
	if runningTask.StartedAt == nil || !runningTask.StartedAt.Equal(startedAt) {
		t.Fatalf("expected started_at %v, got %v", startedAt, runningTask.StartedAt)
	}

	completedAt := time.Now().UTC().Truncate(time.Second)
	if err := repos.Tasks.MarkCompleted(task.ID, completedAt); err != nil {
		t.Fatalf("mark completed: %v", err)
	}

	completedTask, err := repos.Tasks.GetByID(task.ID)
	if err != nil {
		t.Fatalf("get after completed: %v", err)
	}
	if completedTask.Status != domain.TaskStatusCompleted {
		t.Fatalf("expected completed status, got %s", completedTask.Status)
	}
	if completedTask.CompletedAt == nil || !completedTask.CompletedAt.Equal(completedAt) {
		t.Fatalf("expected completed_at %v, got %v", completedAt, completedTask.CompletedAt)
	}
}

func TestAgentInstanceRepo_CRUD(t *testing.T) {
	repos := setupRepoTestDB(t)

	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Tasks.Create(&domain.Task{ID: "t1", RunID: "r1", TaskType: "code", AttemptNo: 1, Status: domain.TaskStatusQueued, Priority: domain.PriorityNormal, QueueStatus: "queued", ResourceClass: domain.ResourceClassLight, Preemptible: true, RestartPolicy: "never", Title: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	agent := &domain.AgentInstance{
		ID:          "a1",
		RunID:       "r1",
		TaskID:      "t1",
		AgentKind:   "generic-shell",
		Status:      domain.AgentStatusStarting,
		TmuxSession: "agent-test",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repos.AgentInstances.Create(agent); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repos.AgentInstances.GetByID("a1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AgentKind != "generic-shell" {
		t.Errorf("expected generic-shell, got %s", got.AgentKind)
	}

	if err := repos.AgentInstances.UpdateStatus("a1", domain.AgentStatusRunning); err != nil {
		t.Fatalf("update status: %v", err)
	}

	if err := repos.AgentInstances.UpdatePID("a1", 12345); err != nil {
		t.Fatalf("update pid: %v", err)
	}

	if err := repos.AgentInstances.UpdateHeartbeat("a1"); err != nil {
		t.Fatalf("update heartbeat: %v", err)
	}

	got, _ = repos.AgentInstances.GetByID("a1")
	if got.Status != domain.AgentStatusRunning {
		t.Errorf("expected running, got %s", got.Status)
	}
	if got.PID == nil || *got.PID != 12345 {
		t.Errorf("expected pid 12345, got %v", got.PID)
	}
	if got.LastHeartbeatAt == nil {
		t.Error("expected heartbeat to be set")
	}

	cpID := "cp-1"
	repos.AgentInstances.UpdateCheckpointID("a1", cpID)
	got, _ = repos.AgentInstances.GetByID("a1")
	if got.CheckpointID == nil || *got.CheckpointID != cpID {
		t.Errorf("expected checkpoint_id %s, got %v", cpID, got.CheckpointID)
	}

	byRun, _ := repos.AgentInstances.ListByRun("r1")
	if len(byRun) != 1 {
		t.Errorf("expected 1 agent, got %d", len(byRun))
	}

	byStatus, _ := repos.AgentInstances.ListByStatus(domain.AgentStatusRunning)
	if len(byStatus) != 1 {
		t.Errorf("expected 1 running agent, got %d", len(byStatus))
	}

	all, _ := repos.AgentInstances.ListAll()
	if len(all) != 1 {
		t.Errorf("expected 1 agent total, got %d", len(all))
	}
}

func TestEventRepo_CRUD(t *testing.T) {
	repos := setupRepoTestDB(t)

	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	event := &domain.Event{
		ID:        "e1",
		RunID:     "r1",
		EventType: domain.EventTypeTaskStarted,
		Message:   "task started",
		CreatedAt: time.Now(),
	}
	if err := repos.Events.Create(event); err != nil {
		t.Fatalf("create: %v", err)
	}

	byRun, _ := repos.Events.ListByRun("r1")
	if len(byRun) != 1 {
		t.Errorf("expected 1 event, got %d", len(byRun))
	}
	if byRun[0].Message != "task started" {
		t.Errorf("expected 'task started', got %s", byRun[0].Message)
	}

	cutoff := time.Now().Add(1 * time.Hour)
	repos.Events.DeleteOlderThan(cutoff)
	byRun, _ = repos.Events.ListByRun("r1")
	if len(byRun) != 0 {
		t.Errorf("expected 0 events after delete, got %d", len(byRun))
	}
}

func TestCheckpointRepo_CRUD(t *testing.T) {
	repos := setupRepoTestDB(t)

	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Tasks.Create(&domain.Task{ID: "t1", RunID: "r1", TaskType: "code", AttemptNo: 1, Status: domain.TaskStatusQueued, Priority: domain.PriorityNormal, QueueStatus: "queued", ResourceClass: domain.ResourceClassLight, Preemptible: true, RestartPolicy: "never", Title: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.AgentInstances.Create(&domain.AgentInstance{ID: "a1", RunID: "r1", TaskID: "t1", AgentKind: "shell", Status: domain.AgentStatusRunning, TmuxSession: "agent-1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	cp := &domain.Checkpoint{
		ID:        "cp1",
		AgentID:   "a1",
		TaskID:    "t1",
		RunID:     "r1",
		Phase:     "mid",
		StateData: "some-state",
		Reason:    "periodic",
		CreatedAt: time.Now(),
	}
	if err := repos.Checkpoints.Create(cp); err != nil {
		t.Fatalf("create: %v", err)
	}

	latest, err := repos.Checkpoints.LatestByTask("t1")
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if latest.Phase != "mid" {
		t.Errorf("expected phase mid, got %s", latest.Phase)
	}

	list, _ := repos.Checkpoints.ListByTask("t1")
	if len(list) != 1 {
		t.Errorf("expected 1 checkpoint, got %d", len(list))
	}

	noCp, _ := repos.Checkpoints.LatestByTask("nonexistent")
	if noCp != nil {
		t.Error("expected nil for nonexistent task")
	}
}

func TestResourceSnapshotRepo_CRUD(t *testing.T) {
	repos := setupRepoTestDB(t)

	snap := &domain.ResourceSnapshot{
		ID:            "rs1",
		MemoryPercent: 65.5,
		CPUPercent:    30.2,
		DiskPercent:   45.0,
		ActiveAgents:  2,
		PressureLevel: "normal",
		CreatedAt:     time.Now(),
	}
	if err := repos.ResourceSnapshots.Create(snap); err != nil {
		t.Fatalf("create: %v", err)
	}

	latest, err := repos.ResourceSnapshots.Latest()
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	if latest.MemoryPercent != 65.5 {
		t.Errorf("expected 65.5, got %f", latest.MemoryPercent)
	}
}

func TestWorkflowTemplateRepo_CRUD(t *testing.T) {
	repos := setupRepoTestDB(t)

	wt := &domain.WorkflowTemplate{
		ID:          "wt1",
		Name:        "ci-pipeline",
		Description: "CI pipeline template",
		NodesJSON:   `[{"id":"n1","task_spec_id":"ts1","label":"build"}]`,
		EdgesJSON:   `[]`,
		OnFailure:   "abort",
	}
	if err := repos.WorkflowTemplates.Create(wt); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repos.WorkflowTemplates.GetByID("wt1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "ci-pipeline" {
		t.Errorf("expected ci-pipeline, got %s", got.Name)
	}
	if got.OnFailure != "abort" {
		t.Errorf("expected abort, got %s", got.OnFailure)
	}

	list, _ := repos.WorkflowTemplates.List()
	if len(list) != 1 {
		t.Errorf("expected 1 template, got %d", len(list))
	}
}

func TestTaskSpecRepo_CRUD(t *testing.T) {
	repos := setupRepoTestDB(t)

	ts := &domain.TaskSpec{
		ID:              "ts1",
		Name:            "build",
		TaskType:        "build",
		RuntimeType:     "shell",
		CommandTemplate: "make build",
		TimeoutSeconds:  300,
		RetryPolicy:     "never",
		ResourceClass:   domain.ResourceClassLight,
	}
	if err := repos.TaskSpecs.Create(ts); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repos.TaskSpecs.GetByID("ts1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.CommandTemplate != "make build" {
		t.Errorf("expected 'make build', got %s", got.CommandTemplate)
	}
}

func TestTerminalSessionRepo_CRUD(t *testing.T) {
	repos := setupRepoTestDB(t)

	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Tasks.Create(&domain.Task{ID: "t1", RunID: "r1", TaskType: "code", AttemptNo: 1, Status: domain.TaskStatusQueued, Priority: domain.PriorityNormal, QueueStatus: "queued", ResourceClass: domain.ResourceClassLight, Preemptible: true, RestartPolicy: "never", Title: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	agentID := "a1"
	ts := &domain.TerminalSession{
		ID:          "term1",
		TaskID:      "t1",
		AgentID:     &agentID,
		TmuxSession: "agent-test",
		Status:      domain.TerminalStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := repos.TerminalSessions.Create(ts); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repos.TerminalSessions.GetByID("term1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.TmuxSession != "agent-test" {
		t.Errorf("expected agent-test, got %s", got.TmuxSession)
	}

	repos.TerminalSessions.UpdateStatus("term1", domain.TerminalStatusClosed)
	got, _ = repos.TerminalSessions.GetByID("term1")
	if got.Status != domain.TerminalStatusClosed {
		t.Errorf("expected closed, got %s", got.Status)
	}

	all, _ := repos.TerminalSessions.ListAll()
	if len(all) != 1 {
		t.Errorf("expected 1 terminal, got %d", len(all))
	}
}

func TestListActiveWithTasks(t *testing.T) {
	repos := setupRepoTestDB(t)

	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Tasks.Create(&domain.Task{ID: "t1", RunID: "r1", TaskType: "code", AttemptNo: 1, Status: domain.TaskStatusRunning, Priority: domain.PriorityNormal, QueueStatus: "running", ResourceClass: domain.ResourceClassLight, Preemptible: true, RestartPolicy: "never", Title: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	agent := &domain.AgentInstance{
		ID: "a1", RunID: "r1", TaskID: "t1", AgentKind: "shell",
		Status: domain.AgentStatusRunning, TmuxSession: "agent-1",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	repos.AgentInstances.Create(agent)

	results, err := repos.AgentInstances.ListActiveWithTasks()
	if err != nil {
		t.Fatalf("ListActiveWithTasks: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Agent.ID != "a1" {
		t.Errorf("expected agent a1, got %s", results[0].Agent.ID)
	}
	if results[0].Task.ID != "t1" {
		t.Errorf("expected task t1, got %s", results[0].Task.ID)
	}
}

func TestWorkspaceRepo_CRUD(t *testing.T) {
	repos := setupRepoTestDB(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Tasks.Create(&domain.Task{ID: "t1", RunID: "r1", TaskType: "code", AttemptNo: 1, Status: domain.TaskStatusQueued, Priority: domain.PriorityNormal, QueueStatus: "queued", ResourceClass: domain.ResourceClassLight, Preemptible: true, RestartPolicy: "never", Title: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	w := &domain.Workspace{ID: "ws1", TaskID: "t1", ProjectID: "p1", Path: "/data/workspaces/task-t1", Branch: "feature/test", CommitSHA: "abc123", SizeBytes: 1024000, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repos.Workspaces.Create(w); err != nil {
		t.Fatalf("create workspace: %v", err)
	}

	got, _ := repos.Workspaces.GetByTaskID("t1")
	if got.Path != "/data/workspaces/task-t1" {
		t.Errorf("expected path, got %s", got.Path)
	}

	none, _ := repos.Workspaces.GetByTaskID("nonexistent")
	if none != nil {
		t.Error("expected nil for nonexistent workspace")
	}

	list, _ := repos.Workspaces.ListActive()
	if len(list) != 1 {
		t.Errorf("expected 1 workspace, got %d", len(list))
	}
}

func TestCommandLogRepo_CRUD(t *testing.T) {
	repos := setupRepoTestDB(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Tasks.Create(&domain.Task{ID: "t1", RunID: "r1", TaskType: "code", AttemptNo: 1, Status: domain.TaskStatusQueued, Priority: domain.PriorityNormal, QueueStatus: "queued", ResourceClass: domain.ResourceClassLight, Preemptible: true, RestartPolicy: "never", Title: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	agentID := "a1"
	exitCode := 0
	cmd := &domain.CommandLog{ID: "cl1", TaskID: "t1", AgentID: &agentID, Command: "make build", ExitCode: &exitCode, Output: "Build completed", Duration: 15000, CreatedAt: time.Now()}
	if err := repos.CommandLogs.Create(cmd); err != nil {
		t.Fatalf("create command log: %v", err)
	}

	cmd2 := &domain.CommandLog{ID: "cl2", TaskID: "t1", Command: "echo hello", Duration: 500, CreatedAt: time.Now()}
	if err := repos.CommandLogs.Create(cmd2); err != nil {
		t.Fatalf("create command log with nil fields: %v", err)
	}
}

func TestAgentSpecRepo_CRUD(t *testing.T) {
	repos := setupRepoTestDB(t)

	as := &domain.AgentSpec{ID: "as1", Name: "Claude Code Agent", AgentKind: "claude-code", SupportedTaskTypes: "planner,coder,tester", DefaultCommand: "claude", MaxConcurrency: 2, ResourceWeight: 1.5, HeartbeatMode: "periodic", OutputParser: "claude-code"}
	if err := repos.AgentSpecs.Create(as); err != nil {
		t.Fatalf("create agent spec: %v", err)
	}

	got, _ := repos.AgentSpecs.GetByID("as1")
	if got.Name != "Claude Code Agent" {
		t.Errorf("expected 'Claude Code Agent', got %s", got.Name)
	}
	if got.MaxConcurrency != 2 {
		t.Errorf("expected max_concurrency 2, got %d", got.MaxConcurrency)
	}

	none, _ := repos.AgentSpecs.GetByID("nonexistent")
	if none != nil {
		t.Error("expected nil for nonexistent agent spec")
	}

	list, _ := repos.AgentSpecs.List()
	if len(list) != 1 {
		t.Errorf("expected 1 agent spec, got %d", len(list))
	}
}
