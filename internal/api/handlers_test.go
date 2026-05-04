package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xxy757/xxyCodingAgents/internal/config"
	"github.com/xxy757/xxyCodingAgents/internal/domain"
	"github.com/xxy757/xxyCodingAgents/internal/orchestrator"
	agentruntime "github.com/xxy757/xxyCodingAgents/internal/runtime"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
	"github.com/xxy757/xxyCodingAgents/internal/terminal"
)

func newTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			HTTPAddr:       ":0",
			AllowedOrigins: []string{"*"},
		},
		Runtime: config.RuntimeConfig{
			WorkspaceRoot:  "./data/workspaces",
			LogRoot:        "./data/logs",
			CheckpointRoot: "./data/checkpoints",
		},
		Scheduler: config.SchedulerConfig{
			TickSeconds:         3,
			MaxConcurrentAgents: 2,
			MaxHeavyAgents:      1,
			MaxTestJobs:         1,
		},
		Thresholds: config.ThresholdsConfig{
			WarnMemoryPercent:        70,
			HighMemoryPercent:        80,
			CriticalMemoryPercent:    88,
			DiskWarnPercent:          80,
			DiskHighPercent:          90,
			WorkspaceMaxSizeMB:       2048,
			LogRetentionDays:         7,
			MaxTotalLogSizeMB:        1024,
			MaxChildProcessesPerAgent: 10,
		},
		Timeouts: config.TimeoutsConfig{
			HeartbeatTimeoutSeconds:   30,
			OutputTimeoutSeconds:      900,
			StallTimeoutSeconds:       900,
			CheckpointIntervalSeconds: 30,
		},
		SQLite: config.SQLiteConfig{
			Path:         ":memory:",
			WALMode:       false,
			BusyTimeoutMs: 1000,
		},
	}
}

func setupAPITest(t *testing.T) (*Server, *storage.Repos, *orchestrator.Orchestrator) {
	t.Helper()
	cfg := newTestConfig()

	db, err := storage.NewDB(":memory:", false, 1000)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.RunMigrations(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repos := storage.NewRepos(db)
	orch := orchestrator.NewOrchestrator(repos, nil)
	tm := terminal.NewManager()
	registry := agentruntime.NewAdapterRegistry()
	srv := NewServer(cfg, db, repos, orch, tm, registry)
	return srv, repos, orch
}

func serveHTTP(srv *Server, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	return w
}

func postReq(t *testing.T, path string, body any) *http.Request {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func getReq(t *testing.T, path string) *http.Request {
	t.Helper()
	return httptest.NewRequest("GET", path, nil)
}

func seedProjectAndTemplate(t *testing.T, repos *storage.Repos) {
	t.Helper()
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test-project", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.TaskSpecs.Create(&domain.TaskSpec{ID: "ts-build", Name: "build", TaskType: "build", CommandTemplate: "make build", ResourceClass: domain.ResourceClassLight})
	repos.TaskSpecs.Create(&domain.TaskSpec{ID: "ts-test", Name: "test", TaskType: "test", CommandTemplate: "make test", ResourceClass: domain.ResourceClassMedium})
	repos.WorkflowTemplates.Create(&domain.WorkflowTemplate{
		ID: "wt1", Name: "ci-pipeline",
		NodesJSON: `[{"id":"n1","task_spec_id":"ts-build","label":"Build"},{"id":"n2","task_spec_id":"ts-test","label":"Test"}]`,
		EdgesJSON: `[{"from":"n1","to":"n2"}]`,
		OnFailure: "abort",
	})
}

func seedRunWithAgent(t *testing.T, repos *storage.Repos, agentStatus domain.AgentInstanceStatus) {
	t.Helper()
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusRunning, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Tasks.Create(&domain.Task{
		ID: "t1", RunID: "r1", TaskType: "code", AttemptNo: 1,
		Status: domain.TaskStatusRunning, Priority: domain.PriorityNormal,
		QueueStatus: "running", ResourceClass: domain.ResourceClassLight,
		Preemptible: true, RestartPolicy: "never", Title: "test",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	tmux := "agent-1"
	if agentStatus == domain.AgentStatusStopped || agentStatus == domain.AgentStatusFailed {
		tmux = ""
	}
	repos.AgentInstances.Create(&domain.AgentInstance{
		ID: "a1", RunID: "r1", TaskID: "t1", AgentKind: "shell",
		Status: agentStatus, TmuxSession: tmux,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
}

func TestAPI_CreateProject(t *testing.T) {
	srv, _, _ := setupAPITest(t)
	w := serveHTTP(srv, postReq(t, "/api/projects", map[string]string{
		"name":    "my-project",
		"repo_url": "https://github.com/test/repo",
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var p domain.Project
	json.Unmarshal(w.Body.Bytes(), &p)
	if p.Name != "my-project" {
		t.Errorf("expected name my-project, got %s", p.Name)
	}
	if p.ID == "" {
		t.Error("expected non-empty ID")
	}
}

func TestAPI_CreateProject_MissingName(t *testing.T) {
	srv, _, _ := setupAPITest(t)
	w := serveHTTP(srv, postReq(t, "/api/projects", map[string]string{}))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_ListProjects(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "proj1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	w := serveHTTP(srv, getReq(t, "/api/projects"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var list []*domain.Project
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Errorf("expected 1 project, got %d", len(list))
	}
}

func TestAPI_GetProject(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "proj1", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	w := serveHTTP(srv, getReq(t, "/api/projects/p1"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAPI_GetProject_NotFound(t *testing.T) {
	srv, _, _ := setupAPITest(t)
	w := serveHTTP(srv, getReq(t, "/api/projects/nonexistent"))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_CreateRun_WithTemplate(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	seedProjectAndTemplate(t, repos)

	w := serveHTTP(srv, postReq(t, "/api/runs", map[string]string{
		"project_id":           "p1",
		"workflow_template_id": "wt1",
		"title":                "CI Run",
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var run domain.Run
	json.Unmarshal(w.Body.Bytes(), &run)

	runAfter, _ := repos.Runs.GetByID(run.ID)
	if runAfter.Status != domain.RunStatusRunning {
		t.Errorf("expected running after template instantiation, got %s", runAfter.Status)
	}

	tasksW := serveHTTP(srv, getReq(t, "/api/runs/"+run.ID+"/tasks"))
	var tasks []*domain.Task
	json.Unmarshal(tasksW.Body.Bytes(), &tasks)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks from template, got %d", len(tasks))
	}
}

func TestAPI_CreateRun_WithoutTemplate(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	w := serveHTTP(srv, postReq(t, "/api/runs", map[string]string{
		"project_id": "p1",
		"title":      "Manual Run",
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var run domain.Run
	json.Unmarshal(w.Body.Bytes(), &run)
	if run.Status != domain.RunStatusPending {
		t.Errorf("expected pending without template, got %s", run.Status)
	}
}

func TestAPI_CreateRun_InvalidProject(t *testing.T) {
	srv, _, _ := setupAPITest(t)
	w := serveHTTP(srv, postReq(t, "/api/runs", map[string]string{
		"project_id": "nonexistent",
		"title":      "Test",
	}))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_GetRun(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	w := serveHTTP(srv, getReq(t, "/api/runs/r1"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAPI_ListAllRuns(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "run1", Status: domain.RunStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})

	w := serveHTTP(srv, getReq(t, "/api/runs"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var list []*domain.Run
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Errorf("expected 1 run, got %d", len(list))
	}
}

func TestAPI_GetRunTimeline(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusPending, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Events.Create(&domain.Event{ID: "e1", RunID: "r1", EventType: domain.EventTypeTaskStarted, Message: "task started", CreatedAt: time.Now()})

	w := serveHTTP(srv, getReq(t, "/api/runs/r1/timeline"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var events []*domain.Event
	json.Unmarshal(w.Body.Bytes(), &events)
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestAPI_RetryTask(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusRunning, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Tasks.Create(&domain.Task{
		ID: "t1", RunID: "r1", TaskType: "code", AttemptNo: 1,
		Status: domain.TaskStatusFailed, Priority: domain.PriorityNormal,
		QueueStatus: "failed", ResourceClass: domain.ResourceClassLight,
		Preemptible: true, RestartPolicy: "never", Title: "build",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	w := serveHTTP(srv, postReq(t, "/api/tasks/t1/retry", nil))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var newTask domain.Task
	json.Unmarshal(w.Body.Bytes(), &newTask)
	if newTask.AttemptNo != 2 {
		t.Errorf("expected attempt 2, got %d", newTask.AttemptNo)
	}
	if newTask.Status != domain.TaskStatusQueued {
		t.Errorf("expected queued, got %s", newTask.Status)
	}
}

func TestAPI_RetryTask_MaxAttempts(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusRunning, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Tasks.Create(&domain.Task{
		ID: "t1", RunID: "r1", TaskType: "code", AttemptNo: 3,
		Status: domain.TaskStatusFailed, Priority: domain.PriorityNormal,
		QueueStatus: "failed", ResourceClass: domain.ResourceClassLight,
		Preemptible: true, RestartPolicy: "never", Title: "build",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	w := serveHTTP(srv, postReq(t, "/api/tasks/t1/retry", nil))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for max attempts, got %d", w.Code)
	}
}

func TestAPI_CancelTask(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusRunning, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Tasks.Create(&domain.Task{
		ID: "t1", RunID: "r1", TaskType: "code", AttemptNo: 1,
		Status: domain.TaskStatusQueued, Priority: domain.PriorityNormal,
		QueueStatus: "queued", ResourceClass: domain.ResourceClassLight,
		Preemptible: true, RestartPolicy: "never", Title: "build",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	w := serveHTTP(srv, postReq(t, "/api/tasks/t1/cancel", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAPI_CancelTask_Idempotent(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Runs.Create(&domain.Run{ID: "r1", ProjectID: "p1", Title: "test", Status: domain.RunStatusRunning, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.Tasks.Create(&domain.Task{
		ID: "t1", RunID: "r1", TaskType: "code", AttemptNo: 1,
		Status: domain.TaskStatusCompleted, Priority: domain.PriorityNormal,
		QueueStatus: "completed", ResourceClass: domain.ResourceClassLight,
		Preemptible: true, RestartPolicy: "never", Title: "build",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	w := serveHTTP(srv, postReq(t, "/api/tasks/t1/cancel", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for idempotent cancel on completed task, got %d", w.Code)
	}
}

func TestAPI_PauseAgent_Idempotent(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	seedRunWithAgent(t, repos, domain.AgentStatusPaused)

	w := serveHTTP(srv, postReq(t, "/api/agents/a1/pause", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for idempotent pause, got %d", w.Code)
	}
}

func TestAPI_ResumeAgent_Idempotent(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	seedRunWithAgent(t, repos, domain.AgentStatusRunning)

	w := serveHTTP(srv, postReq(t, "/api/agents/a1/resume", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for idempotent resume, got %d", w.Code)
	}
}

func TestAPI_StopAgent_Idempotent(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	seedRunWithAgent(t, repos, domain.AgentStatusStopped)

	w := serveHTTP(srv, postReq(t, "/api/agents/a1/stop", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for idempotent stop, got %d", w.Code)
	}
}

func TestAPI_StopAgent_FailedIsIdempotent(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	seedRunWithAgent(t, repos, domain.AgentStatusFailed)

	w := serveHTTP(srv, postReq(t, "/api/agents/a1/stop", nil))
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for idempotent stop on failed agent, got %d", w.Code)
	}
}

func TestAPI_PauseAgent_WrongState(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	seedRunWithAgent(t, repos, domain.AgentStatusStarting)

	w := serveHTTP(srv, postReq(t, "/api/agents/a1/pause", nil))
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for pause on starting agent, got %d", w.Code)
	}
}

func TestAPI_ListAgents(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	seedRunWithAgent(t, repos, domain.AgentStatusRunning)

	w := serveHTTP(srv, getReq(t, "/api/agents"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var list []*domain.AgentInstance
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Errorf("expected 1 agent, got %d", len(list))
	}
}

func TestAPI_GetAgent(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	seedRunWithAgent(t, repos, domain.AgentStatusRunning)

	w := serveHTTP(srv, getReq(t, "/api/agents/a1"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAPI_GetAgent_NotFound(t *testing.T) {
	srv, _, _ := setupAPITest(t)
	w := serveHTTP(srv, getReq(t, "/api/agents/nonexistent"))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_SystemMetrics(t *testing.T) {
	srv, _, _ := setupAPITest(t)
	w := serveHTTP(srv, getReq(t, "/api/system/metrics"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var m map[string]any
	json.Unmarshal(w.Body.Bytes(), &m)
	if _, ok := m["pressure_level"]; !ok {
		t.Error("expected pressure_level in metrics")
	}
}

func TestAPI_WorkflowTemplates(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.WorkflowTemplates.Create(&domain.WorkflowTemplate{ID: "wt1", Name: "ci", NodesJSON: "[]", EdgesJSON: "[]", OnFailure: "abort"})

	w := serveHTTP(srv, getReq(t, "/api/workflow-templates"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var list []*domain.WorkflowTemplate
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Errorf("expected 1 template, got %d", len(list))
	}
}

func TestAPI_CreateWorkflowTemplate(t *testing.T) {
	srv, _, _ := setupAPITest(t)
	w := serveHTTP(srv, postReq(t, "/api/workflow-templates", map[string]string{
		"name":       "deploy",
		"nodes_json": `[{"id":"n1","task_spec_id":"ts1","label":"deploy"}]`,
		"edges_json": "[]",
	}))
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAPI_RunWorkflowEndpoint(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	seedProjectAndTemplate(t, repos)

	runW := serveHTTP(srv, postReq(t, "/api/runs", map[string]string{
		"project_id":           "p1",
		"workflow_template_id": "wt1",
		"title":                "CI Run",
	}))
	var run domain.Run
	json.Unmarshal(runW.Body.Bytes(), &run)

	w := serveHTTP(srv, getReq(t, "/api/runs/"+run.ID+"/workflow"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var graph map[string]any
	json.Unmarshal(w.Body.Bytes(), &graph)
	nodes := graph["nodes"].([]any)
	edges := graph["edges"].([]any)
	if len(nodes) != 2 {
		t.Errorf("expected 2 workflow nodes, got %d", len(nodes))
	}
	if len(edges) != 1 {
		t.Errorf("expected 1 workflow edge, got %d", len(edges))
	}
}

func TestAPI_Diagnostics(t *testing.T) {
	srv, _, _ := setupAPITest(t)
	w := serveHTTP(srv, getReq(t, "/api/system/diagnostics"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var diag map[string]any
	json.Unmarshal(w.Body.Bytes(), &diag)
	if _, ok := diag["tmux_sessions"]; !ok {
		t.Error("expected tmux_sessions in diagnostics")
	}
	if _, ok := diag["config"]; !ok {
		t.Error("expected config in diagnostics")
	}
}

func TestAPI_TaskSpecs(t *testing.T) {
	srv, _, _ := setupAPITest(t)
	w := serveHTTP(srv, getReq(t, "/api/task-specs"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAPI_AgentSpecs(t *testing.T) {
	srv, _, _ := setupAPITest(t)
	w := serveHTTP(srv, getReq(t, "/api/agent-specs"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAPI_HealthCheck(t *testing.T) {
	srv, _, _ := setupAPITest(t)
	w := serveHTTP(srv, getReq(t, "/healthz"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "ok" {
		t.Errorf("expected status ok, got %s", body["status"])
	}
}

func TestAPI_ReadyCheck(t *testing.T) {
	srv, _, _ := setupAPITest(t)
	w := serveHTTP(srv, getReq(t, "/readyz"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["status"] != "ready" {
		t.Errorf("expected status ready, got %s", body["status"])
	}
}

func TestAPI_ListTerminals(t *testing.T) {
	srv, _, _ := setupAPITest(t)
	w := serveHTTP(srv, getReq(t, "/api/terminals"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestE2E_FullWorkflowPipeline(t *testing.T) {
	srv, repos, orch := setupAPITest(t)
	seedProjectAndTemplate(t, repos)

	createW := serveHTTP(srv, postReq(t, "/api/runs", map[string]string{
		"project_id":           "p1",
		"workflow_template_id": "wt1",
		"title":                "Full Pipeline",
	}))
	if createW.Code != http.StatusCreated {
		t.Fatalf("create run: expected 201, got %d", createW.Code)
	}

	var run domain.Run
	json.Unmarshal(createW.Body.Bytes(), &run)

	tasksW := serveHTTP(srv, getReq(t, "/api/runs/"+run.ID+"/tasks"))
	var tasks []*domain.Task
	json.Unmarshal(tasksW.Body.Bytes(), &tasks)

	var buildTask, testTask *domain.Task
	for _, t2 := range tasks {
		if t2.Title == "Build" {
			buildTask = t2
		} else if t2.Title == "Test" {
			testTask = t2
		}
	}

	if buildTask == nil || testTask == nil {
		t.Fatal("expected build and test tasks")
	}
	if buildTask.Status != domain.TaskStatusQueued {
		t.Errorf("build should be queued (no deps), got %s", buildTask.Status)
	}
	if testTask.Status != domain.TaskStatusBlocked {
		t.Errorf("test should be blocked (depends on build), got %s", testTask.Status)
	}

	repos.Tasks.UpdateStatus(buildTask.ID, domain.TaskStatusRunning)
	orch.CompleteTask(context.Background(), buildTask.ID, `{"result":"ok"}`)

	testTaskAfter, _ := repos.Tasks.GetByID(testTask.ID)
	if testTaskAfter.Status != domain.TaskStatusQueued {
		t.Errorf("test should be queued after build completes, got %s", testTaskAfter.Status)
	}

	timelineW := serveHTTP(srv, getReq(t, "/api/runs/"+run.ID+"/timeline"))
	if timelineW.Code != http.StatusOK {
		t.Fatalf("timeline: expected 200, got %d", timelineW.Code)
	}

	workflowW := serveHTTP(srv, getReq(t, "/api/runs/"+run.ID+"/workflow"))
	if workflowW.Code != http.StatusOK {
		t.Fatalf("workflow: expected 200, got %d", workflowW.Code)
	}
}

func TestE2E_RunFailureAbort(t *testing.T) {
	srv, repos, orch := setupAPITest(t)
	seedProjectAndTemplate(t, repos)

	createW := serveHTTP(srv, postReq(t, "/api/runs", map[string]string{
		"project_id":           "p1",
		"workflow_template_id": "wt1",
		"title":                "Fail Test",
	}))
	var run domain.Run
	json.Unmarshal(createW.Body.Bytes(), &run)

	tasksW := serveHTTP(srv, getReq(t, "/api/runs/"+run.ID+"/tasks"))
	var tasks []*domain.Task
	json.Unmarshal(tasksW.Body.Bytes(), &tasks)

	if len(tasks) == 0 {
		t.Fatal("expected tasks")
	}
	orch.FailTask(context.Background(), tasks[0].ID, "build failed")

	runAfter, _ := repos.Runs.GetByID(run.ID)
	if runAfter.Status != domain.RunStatusFailed {
		t.Errorf("expected run failed (abort on failure), got %s", runAfter.Status)
	}
}

// ==================== Prompt Draft 测试 ====================

func putReq(t *testing.T, path string, body any) *http.Request {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest("PUT", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestAPI_GeneratePromptDraft(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	req := postReq(t, "/api/prompt-drafts/generate", map[string]string{
		"project_id":     "p1",
		"original_input": "修复登录页面的样式问题",
	})
	w := serveHTTP(srv, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var draft domain.PromptDraft
	if err := json.Unmarshal(w.Body.Bytes(), &draft); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if draft.Status != domain.PromptDraftStatusDraft {
		t.Errorf("expected status draft, got %s", draft.Status)
	}
	if draft.TaskType != "bugfix" {
		t.Errorf("expected task_type bugfix, got %s", draft.TaskType)
	}
	if draft.GeneratedPrompt == "" {
		t.Error("expected non-empty generated_prompt")
	}
}

func TestAPI_GeneratePromptDraft_MissingInput(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	req := postReq(t, "/api/prompt-drafts/generate", map[string]string{
		"project_id": "p1",
	})
	w := serveHTTP(srv, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAPI_GeneratePromptDraft_ProjectNotFound(t *testing.T) {
	srv, _, _ := setupAPITest(t)

	req := postReq(t, "/api/prompt-drafts/generate", map[string]string{
		"project_id":     "nonexistent",
		"original_input": "test",
	})
	w := serveHTTP(srv, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestAPI_UpdatePromptDraft(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	genReq := postReq(t, "/api/prompt-drafts/generate", map[string]string{
		"project_id":     "p1",
		"original_input": "添加暗色模式",
	})
	genW := serveHTTP(srv, genReq)
	var draft domain.PromptDraft
	json.Unmarshal(genW.Body.Bytes(), &draft)

	updateReq := putReq(t, "/api/prompt-drafts/"+draft.ID, map[string]string{
		"final_prompt": "自定义的 prompt 内容",
		"task_type":    "build",
	})
	updateW := serveHTTP(srv, updateReq)
	if updateW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", updateW.Code, updateW.Body.String())
	}

	var updated domain.PromptDraft
	json.Unmarshal(updateW.Body.Bytes(), &updated)
	if updated.FinalPrompt != "自定义的 prompt 内容" {
		t.Errorf("expected final_prompt updated, got %s", updated.FinalPrompt)
	}
}

func TestAPI_UpdatePromptDraft_EmptyPrompt(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	genReq := postReq(t, "/api/prompt-drafts/generate", map[string]string{
		"project_id":     "p1",
		"original_input": "test",
	})
	genW := serveHTTP(srv, genReq)
	var draft domain.PromptDraft
	json.Unmarshal(genW.Body.Bytes(), &draft)

	updateReq := putReq(t, "/api/prompt-drafts/"+draft.ID, map[string]string{
		"final_prompt": "   ",
	})
	updateW := serveHTTP(srv, updateReq)
	if updateW.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty prompt, got %d", updateW.Code)
	}
}

func TestAPI_SendPromptDraft(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", RepoURL: "", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	genReq := postReq(t, "/api/prompt-drafts/generate", map[string]string{
		"project_id":     "p1",
		"original_input": "写单元测试",
	})
	genW := serveHTTP(srv, genReq)
	var draft domain.PromptDraft
	json.Unmarshal(genW.Body.Bytes(), &draft)

	sendReq := postReq(t, "/api/prompt-drafts/"+draft.ID+"/send", nil)
	sendW := serveHTTP(srv, sendReq)
	if sendW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", sendW.Code, sendW.Body.String())
	}

	var result map[string]string
	json.Unmarshal(sendW.Body.Bytes(), &result)
	if result["status"] != "sent" {
		t.Errorf("expected status sent, got %s", result["status"])
	}
	if result["run_id"] == "" {
		t.Error("expected non-empty run_id")
	}

	draftAfter, _ := repos.PromptDrafts.GetByID(draft.ID)
	if draftAfter.Status != domain.PromptDraftStatusSent {
		t.Errorf("expected draft status sent, got %s", draftAfter.Status)
	}
}

func TestAPI_SendPromptDraft_NonDraft(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	draft := &domain.PromptDraft{
		ID:              "draft-sent",
		ProjectID:       "p1",
		OriginalInput:   "test",
		GeneratedPrompt: "prompt",
		TaskType:        "build",
		Status:          domain.PromptDraftStatusSent,
		RunID:           "existing-run",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	repos.PromptDrafts.Create(draft)

	// CAS 幂等：已 sent 的草稿重复发送返回 200 + 已有 run_id
	sendReq := postReq(t, "/api/prompt-drafts/draft-sent/send", nil)
	sendW := serveHTTP(srv, sendReq)
	if sendW.Code != http.StatusOK {
		t.Errorf("expected 200 for idempotent re-send, got %d", sendW.Code)
	}
	var resp map[string]any
	json.NewDecoder(sendW.Body).Decode(&resp)
	if resp["run_id"] != "existing-run" {
		t.Errorf("expected existing run_id, got %v", resp["run_id"])
	}
}

func TestAPI_ListPromptDrafts(t *testing.T) {
	srv, repos, _ := setupAPITest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	for _, input := range []string{"任务一", "任务二"} {
		req := postReq(t, "/api/prompt-drafts/generate", map[string]string{
			"project_id":     "p1",
			"original_input": input,
		})
		serveHTTP(srv, req)
	}

	listReq := getReq(t, "/api/prompt-drafts?project_id=p1")
	listW := serveHTTP(srv, listReq)
	if listW.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listW.Code)
	}

	var drafts []domain.PromptDraft
	json.Unmarshal(listW.Body.Bytes(), &drafts)
	if len(drafts) != 2 {
		t.Errorf("expected 2 drafts, got %d", len(drafts))
	}
}
