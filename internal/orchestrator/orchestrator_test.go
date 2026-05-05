package orchestrator

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/xxy757/xxyCodingAgents/internal/domain"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
)

func setupOrchestratorTest(t *testing.T) (*Orchestrator, *storage.Repos) {
	t.Helper()
	db, err := storage.NewDB(":memory:", false, 1000)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.RunMigrations(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repos := storage.NewRepos(db)
	return NewOrchestrator(repos, nil), repos
}

func seedProjectAndTemplate(t *testing.T, repos *storage.Repos) (string, string) {
	t.Helper()
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	repos.TaskSpecs.Create(&domain.TaskSpec{
		ID: "ts-build", Name: "build", TaskType: "build", CommandTemplate: "make build", ResourceClass: domain.ResourceClassLight,
	})
	repos.TaskSpecs.Create(&domain.TaskSpec{
		ID: "ts-test", Name: "test", TaskType: "test", CommandTemplate: "make test", ResourceClass: domain.ResourceClassMedium,
	})

	wt := &domain.WorkflowTemplate{
		ID:        "wt1",
		Name:      "ci",
		NodesJSON: `[{"id":"n1","task_spec_id":"ts-build","label":"build"},{"id":"n2","task_spec_id":"ts-test","label":"test"}]`,
		EdgesJSON: `[{"from":"n1","to":"n2"}]`,
		OnFailure: "abort",
	}
	repos.WorkflowTemplates.Create(wt)

	return "p1", "wt1"
}

func TestCreateRun_WithoutTemplate(t *testing.T) {
	o, repos := setupOrchestratorTest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	run, err := o.CreateRun(context.Background(), "p1", "", "my-run", "desc")
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	if run.Status != domain.RunStatusPending {
		t.Errorf("expected pending, got %s", run.Status)
	}
	if run.Title != "my-run" {
		t.Errorf("expected my-run, got %s", run.Title)
	}

	tasks, _ := repos.Tasks.ListByRun(run.ID)
	if len(tasks) != 0 {
		t.Errorf("expected 0 tasks without template, got %d", len(tasks))
	}
}

func TestCreateRun_WithTemplate(t *testing.T) {
	o, repos := setupOrchestratorTest(t)
	projectID, templateID := seedProjectAndTemplate(t, repos)

	run, err := o.CreateRun(context.Background(), projectID, templateID, "ci-run", "")
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	gotRun, _ := repos.Runs.GetByID(run.ID)
	if gotRun.Status != domain.RunStatusRunning {
		t.Errorf("expected running after workflow instantiation, got %s", gotRun.Status)
	}

	tasks, _ := repos.Tasks.ListByRun(run.ID)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	buildTask := tasks[0]
	testTask := tasks[1]

	if buildTask.Status != domain.TaskStatusQueued {
		t.Errorf("build task (no deps) should be queued, got %s", buildTask.Status)
	}
	if testTask.Status != domain.TaskStatusBlocked {
		t.Errorf("test task (depends on build) should be blocked, got %s", testTask.Status)
	}
}

func TestCompleteTask(t *testing.T) {
	o, repos := setupOrchestratorTest(t)
	projectID, templateID := seedProjectAndTemplate(t, repos)

	run, _ := o.CreateRun(context.Background(), projectID, templateID, "ci-run", "")
	tasks, _ := repos.Tasks.ListByRun(run.ID)

	buildTask := tasks[0]
	if err := o.CompleteTask(context.Background(), buildTask.ID, `{"result":"ok"}`); err != nil {
		t.Fatalf("CompleteTask: %v", err)
	}

	gotTask, _ := repos.Tasks.GetByID(buildTask.ID)
	if gotTask.Status != domain.TaskStatusCompleted {
		t.Errorf("expected completed, got %s", gotTask.Status)
	}
	if gotTask.CompletedAt == nil {
		t.Error("expected completed_at to be persisted")
	}

	tasksAfter, _ := repos.Tasks.ListByRun(run.ID)
	testTask := tasksAfter[1]
	if testTask.Status != domain.TaskStatusQueued {
		t.Errorf("test task should be unblocked after build completes, got %s", testTask.Status)
	}
}

func TestCompleteTask_AllDoneFinalizesRun(t *testing.T) {
	o, repos := setupOrchestratorTest(t)
	projectID, _ := seedProjectAndTemplate(t, repos)

	run, _ := o.CreateRun(context.Background(), projectID, "", "simple-run", "")
	repos.Tasks.Create(&domain.Task{
		ID: "t1", RunID: run.ID, TaskType: "code", AttemptNo: 1,
		Status: domain.TaskStatusRunning, Priority: domain.PriorityNormal,
		QueueStatus: "running", ResourceClass: domain.ResourceClassLight,
		Preemptible: true, RestartPolicy: "never", Title: "task-1",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	o.CompleteTask(context.Background(), "t1", "")

	gotRun, _ := repos.Runs.GetByID(run.ID)
	if gotRun.Status != domain.RunStatusCompleted {
		t.Errorf("expected run completed, got %s", gotRun.Status)
	}
}

func TestFailTask_AbortPolicy(t *testing.T) {
	o, repos := setupOrchestratorTest(t)
	projectID, templateID := seedProjectAndTemplate(t, repos)

	run, _ := o.CreateRun(context.Background(), projectID, templateID, "ci-run", "")
	tasks, _ := repos.Tasks.ListByRun(run.ID)

	if err := o.FailTask(context.Background(), tasks[0].ID, "build failed"); err != nil {
		t.Fatalf("FailTask: %v", err)
	}

	gotTask, _ := repos.Tasks.GetByID(tasks[0].ID)
	if gotTask.Status != domain.TaskStatusFailed {
		t.Errorf("expected failed, got %s", gotTask.Status)
	}

	gotRun, _ := repos.Runs.GetByID(run.ID)
	if gotRun.Status != domain.RunStatusFailed {
		t.Errorf("expected run failed (abort policy), got %s", gotRun.Status)
	}
}

// TestCreateRun_WithTemplate_WorkspaceFallback 验证 instantiateWorkflow 在无 gitManager 时
// 为每个任务创建临时工作区，避免 workspacePath 为空。
func TestCreateRun_WithTemplate_WorkspaceFallback(t *testing.T) {
	o, repos := setupOrchestratorTest(t)
	// setupOrchestratorTest 传入 gitManager=nil，project 无 RepoURL
	projectID, templateID := seedProjectAndTemplate(t, repos)

	run, err := o.CreateRun(context.Background(), projectID, templateID, "ci-run", "")
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}
	// 从数据库重新读取，instantiateWorkflow 内部更新了 status
	gotRun, _ := repos.Runs.GetByID(run.ID)
	if gotRun.Status != domain.RunStatusRunning {
		t.Fatalf("expected running, got %s", gotRun.Status)
	}

	tasks, _ := repos.Tasks.ListByRun(run.ID)
	if len(tasks) == 0 {
		t.Fatal("expected at least one task")
	}

	for _, task := range tasks {
		if task.WorkspacePath == "" {
			t.Errorf("task %s (%s): expected non-empty workspace path, got empty", task.ID, task.Title)
		}
		// 验证临时目录确实存在于文件系统
		if _, err := os.Stat(task.WorkspacePath); os.IsNotExist(err) {
			t.Errorf("task %s: workspace path %s does not exist", task.ID, task.WorkspacePath)
		}
	}
}

// TestCreateRun_WithTemplate_NoRepoURL_ProjectWithRepoURL 验证项目有 RepoURL 但 gitManager 为 nil
// 时（无法克隆），也能正确回退到临时工作区。
func TestCreateRun_WithTemplate_NoRepoURL_ProjectWithRepoURL(t *testing.T) {
	o, repos := setupOrchestratorTest(t)
	// 创建带有 RepoURL 的项目，但 gitManager 仍为 nil
	repos.Projects.Create(&domain.Project{
		ID: "p2", Name: "repo-project", RepoURL: "https://github.com/example/repo.git",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	repos.TaskSpecs.Create(&domain.TaskSpec{
		ID: "ts1", Name: "build", TaskType: "build", CommandTemplate: "make", ResourceClass: domain.ResourceClassLight,
	})
	repos.WorkflowTemplates.Create(&domain.WorkflowTemplate{
		ID: "wt2", Name: "single", NodesJSON: `[{"id":"n1","task_spec_id":"ts1","label":"build"}]`,
		EdgesJSON: `[]`, OnFailure: "abort",
	})

	run, err := o.CreateRun(context.Background(), "p2", "wt2", "repo-run", "")
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	tasks, _ := repos.Tasks.ListByRun(run.ID)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].WorkspacePath == "" {
		t.Error("expected fallback temp workspace, got empty workspace path")
	}
}

// TestCreateRun_WithTemplate_MultipleTasksShareWorkspace 验证同一 workflow 中的多个任务
// 共享同一个临时工作区路径。
func TestCreateRun_WithTemplate_MultipleTasksShareWorkspace(t *testing.T) {
	o, repos := setupOrchestratorTest(t)
	projectID, templateID := seedProjectAndTemplate(t, repos)

	run, err := o.CreateRun(context.Background(), projectID, templateID, "shared-ws", "")
	if err != nil {
		t.Fatalf("CreateRun: %v", err)
	}

	tasks, _ := repos.Tasks.ListByRun(run.ID)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(tasks))
	}

	// 所有任务的 workspace 应该相同（同一个临时目录）
	ws := tasks[0].WorkspacePath
	for _, task := range tasks {
		if task.WorkspacePath != ws {
			t.Errorf("task %s: expected workspace %s, got %s", task.ID, ws, task.WorkspacePath)
		}
	}
}

// TestCreateSimpleRun_NoRepo_FallbackWorkspace 验证 CreateSimpleRun 在无仓库时
// 也会创建临时工作区。
func TestCreateSimpleRun_NoRepo_FallbackWorkspace(t *testing.T) {
	o, repos := setupOrchestratorTest(t)
	repos.Projects.Create(&domain.Project{ID: "p3", Name: "no-repo", CreatedAt: time.Now(), UpdatedAt: time.Now()})

	result, err := o.CreateSimpleRun(context.Background(), "p3", "simple", "desc", "input data", "code")
	if err != nil {
		t.Fatalf("CreateSimpleRun: %v", err)
	}
	if result.Task.WorkspacePath == "" {
		t.Error("expected non-empty workspace path for simple run without repo")
	}
	if _, err := os.Stat(result.Task.WorkspacePath); os.IsNotExist(err) {
		t.Errorf("workspace path %s does not exist", result.Task.WorkspacePath)
	}
}

func TestFailTask_NoAbortPolicy(t *testing.T) {
	o, repos := setupOrchestratorTest(t)
	repos.Projects.Create(&domain.Project{ID: "p1", Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	repos.TaskSpecs.Create(&domain.TaskSpec{ID: "ts1", Name: "build", TaskType: "build", ResourceClass: domain.ResourceClassLight})
	repos.WorkflowTemplates.Create(&domain.WorkflowTemplate{
		ID: "wt1", Name: "tolerant", NodesJSON: `[{"id":"n1","task_spec_id":"ts1","label":"build"}]`,
		EdgesJSON: `[]`, OnFailure: "continue",
	})

	run, _ := o.CreateRun(context.Background(), "p1", "wt1", "run", "")
	tasks, _ := repos.Tasks.ListByRun(run.ID)

	o.FailTask(context.Background(), tasks[0].ID, "failed but continue")

	gotRun, _ := repos.Runs.GetByID(run.ID)
	if gotRun.Status == domain.RunStatusFailed {
		t.Error("expected run NOT to be failed with continue policy")
	}
}
