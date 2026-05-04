// Package storage 的 repositories 文件实现所有领域实体的数据访问层。
// 为项目、运行、任务、Agent 实例、事件、检查点、资源快照等提供 CRUD 操作。
package storage

import (
	"database/sql"
	"time"

	"github.com/xxy757/xxyCodingAgents/internal/domain"
)

// scanner 是 sql.Row 和 sql.Rows 的公共接口，用于统一扫描逻辑。
type scanner interface {
	Scan(dest ...any) error
}

// ==================== ProjectRepo ====================

// ProjectRepo 提供项目实体的数据访问操作。
type ProjectRepo struct {
	db *DB
}

// NewProjectRepo 创建项目仓库实例。
func NewProjectRepo(db *DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

// Create 插入一个新项目记录。
func (r *ProjectRepo) Create(p *domain.Project) error {
	_, err := r.db.Exec(
		"INSERT INTO projects (id, name, repo_url, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		p.ID, p.Name, p.RepoURL, p.Description, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

// GetByID 根据 ID 查询项目，未找到时返回 nil。
func (r *ProjectRepo) GetByID(id string) (*domain.Project, error) {
	p := &domain.Project{}
	err := r.db.QueryRow(
		"SELECT id, name, repo_url, description, created_at, updated_at FROM projects WHERE id = ?", id,
	).Scan(&p.ID, &p.Name, &p.RepoURL, &p.Description, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

// List 查询所有项目，按创建时间降序排列。
func (r *ProjectRepo) List() ([]*domain.Project, error) {
	rows, err := r.db.Query("SELECT id, name, repo_url, description, created_at, updated_at FROM projects ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*domain.Project
	for rows.Next() {
		p := &domain.Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.RepoURL, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// ==================== RunRepo ====================

// RunRepo 提供运行实体的数据访问操作。
type RunRepo struct {
	db *DB
}

// NewRunRepo 创建运行仓库实例。
func NewRunRepo(db *DB) *RunRepo {
	return &RunRepo{db: db}
}

// Create 插入一个新运行记录。
func (r *RunRepo) Create(run *domain.Run) error {
	_, err := r.db.Exec(
		"INSERT INTO runs (id, project_id, workflow_template_id, title, description, status, external_key, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		run.ID, run.ProjectID, run.WorkflowTemplateID, run.Title, run.Description, run.Status, run.ExternalKey, run.CreatedAt, run.UpdatedAt,
	)
	return err
}

// GetByID 根据 ID 查询运行。
func (r *RunRepo) GetByID(id string) (*domain.Run, error) {
	run := &domain.Run{}
	err := r.db.QueryRow(
		"SELECT id, project_id, workflow_template_id, title, description, status, external_key, created_at, updated_at FROM runs WHERE id = ?", id,
	).Scan(&run.ID, &run.ProjectID, &run.WorkflowTemplateID, &run.Title, &run.Description, &run.Status, &run.ExternalKey, &run.CreatedAt, &run.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return run, err
}

// ListByProject 查询指定项目下的所有运行。
func (r *RunRepo) ListByProject(projectID string) ([]*domain.Run, error) {
	rows, err := r.db.Query(
		"SELECT id, project_id, workflow_template_id, title, description, status, external_key, created_at, updated_at FROM runs WHERE project_id = ? ORDER BY created_at DESC", projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*domain.Run
	for rows.Next() {
		run := &domain.Run{}
		if err := rows.Scan(&run.ID, &run.ProjectID, &run.WorkflowTemplateID, &run.Title, &run.Description, &run.Status, &run.ExternalKey, &run.CreatedAt, &run.UpdatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

// ListAll 查询所有运行。
func (r *RunRepo) ListAll() ([]*domain.Run, error) {
	rows, err := r.db.Query(
		"SELECT id, project_id, workflow_template_id, title, description, status, external_key, created_at, updated_at FROM runs ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*domain.Run
	for rows.Next() {
		run := &domain.Run{}
		if err := rows.Scan(&run.ID, &run.ProjectID, &run.WorkflowTemplateID, &run.Title, &run.Description, &run.Status, &run.ExternalKey, &run.CreatedAt, &run.UpdatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

// UpdateStatus 更新运行状态。
func (r *RunRepo) UpdateStatus(id string, status domain.RunStatus) error {
	_, err := r.db.Exec(
		"UPDATE runs SET status = ?, updated_at = ? WHERE id = ?", status, time.Now(), id,
	)
	return err
}

// ==================== TaskRepo ====================

// TaskRepo 提供任务实体的数据访问操作。
type TaskRepo struct {
	db *DB
}

// NewTaskRepo 创建任务仓库实例。
func NewTaskRepo(db *DB) *TaskRepo {
	return &TaskRepo{db: db}
}

// Create 插入一个新任务记录。
func (r *TaskRepo) Create(t *domain.Task) error {
	_, err := r.db.Exec(
		`INSERT INTO tasks (id, run_id, task_spec_id, task_type, attempt_no, status, priority, queue_status, resource_class, preemptible, restart_policy, title, description, input_data, output_data, workspace_path, parent_task_id, started_at, completed_at, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.RunID, t.TaskSpecID, t.TaskType, t.AttemptNo, t.Status, t.Priority, t.QueueStatus, t.ResourceClass, t.Preemptible, t.RestartPolicy, t.Title, t.Description, t.InputData, t.OutputData, t.WorkspacePath, t.ParentTaskID, t.StartedAt, t.CompletedAt, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

// scanTask 是任务扫描的通用辅助函数。
func scanTask(row scanner) (*domain.Task, error) {
	t := &domain.Task{}
	err := row.Scan(&t.ID, &t.RunID, &t.TaskSpecID, &t.TaskType, &t.AttemptNo, &t.Status, &t.Priority, &t.QueueStatus, &t.ResourceClass, &t.Preemptible, &t.RestartPolicy, &t.Title, &t.Description, &t.InputData, &t.OutputData, &t.WorkspacePath, &t.ParentTaskID, &t.StartedAt, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

// GetByID 根据 ID 查询任务。
func (r *TaskRepo) GetByID(id string) (*domain.Task, error) {
	row := r.db.QueryRow(
		`SELECT id, run_id, task_spec_id, task_type, attempt_no, status, priority, queue_status, resource_class, preemptible, restart_policy, title, description, input_data, output_data, workspace_path, parent_task_id, started_at, completed_at, created_at, updated_at
			 FROM tasks WHERE id = ?`, id,
	)
	return scanTask(row)
}

// ListByRun 查询指定运行下的所有任务，按创建时间升序排列。
func (r *TaskRepo) ListByRun(runID string) ([]*domain.Task, error) {
	rows, err := r.db.Query(
		`SELECT id, run_id, task_spec_id, task_type, attempt_no, status, priority, queue_status, resource_class, preemptible, restart_policy, title, description, input_data, output_data, workspace_path, parent_task_id, started_at, completed_at, created_at, updated_at
			 FROM tasks WHERE run_id = ? ORDER BY created_at ASC`, runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// ListByStatus 按状态查询任务，按优先级和创建时间排序。
func (r *TaskRepo) ListByStatus(status domain.TaskStatus) ([]*domain.Task, error) {
	rows, err := r.db.Query(
		`SELECT id, run_id, task_spec_id, task_type, attempt_no, status, priority, queue_status, resource_class, preemptible, restart_policy, title, description, input_data, output_data, workspace_path, parent_task_id, started_at, completed_at, created_at, updated_at
			 FROM tasks WHERE status = ? ORDER BY priority ASC, created_at ASC`, status,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// UpdateStatus 更新任务状态。
func (r *TaskRepo) UpdateStatus(id string, status domain.TaskStatus) error {
	_, err := r.db.Exec(
		"UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?", status, time.Now(), id,
	)
	return err
}

// MarkRunning 更新任务状态为 running，并持久化 started_at。
func (r *TaskRepo) MarkRunning(id string, startedAt time.Time) error {
	_, err := r.db.Exec(
		"UPDATE tasks SET status = ?, started_at = ?, updated_at = ? WHERE id = ?",
		domain.TaskStatusRunning, startedAt, time.Now(), id,
	)
	return err
}

// MarkCompleted 更新任务状态为 completed，并持久化 completed_at。
func (r *TaskRepo) MarkCompleted(id string, completedAt time.Time) error {
	_, err := r.db.Exec(
		"UPDATE tasks SET status = ?, completed_at = ?, updated_at = ? WHERE id = ?",
		domain.TaskStatusCompleted, completedAt, time.Now(), id,
	)
	return err
}

// UpdateQueueStatus 更新任务的队列状态。
func (r *TaskRepo) UpdateQueueStatus(id string, queueStatus string) error {
	_, err := r.db.Exec(
		"UPDATE tasks SET queue_status = ?, updated_at = ? WHERE id = ?", queueStatus, time.Now(), id,
	)
	return err
}

// UpdateOutput 更新任务的输出数据。
func (r *TaskRepo) UpdateOutput(id string, outputData string) error {
	_, err := r.db.Exec(
		"UPDATE tasks SET output_data = ?, updated_at = ? WHERE id = ?", outputData, time.Now(), id,
	)
	return err
}

// UpdateInputData 更新任务的输入数据。
func (r *TaskRepo) UpdateInputData(id string, inputData string) error {
	_, err := r.db.Exec(
		"UPDATE tasks SET input_data = ?, updated_at = ? WHERE id = ?", inputData, time.Now(), id,
	)
	return err
}

// MaxAttemptNo 查询指定运行和任务类型的最大尝试次数。
func (r *TaskRepo) MaxAttemptNo(runID string, taskType string) (int, error) {
	var maxAttempt sql.NullInt64
	err := r.db.QueryRow(
		"SELECT MAX(attempt_no) FROM tasks WHERE run_id = ? AND task_type = ?", runID, taskType,
	).Scan(&maxAttempt)
	if err != nil {
		return 0, err
	}
	if !maxAttempt.Valid {
		return 0, nil
	}
	return int(maxAttempt.Int64), nil
}

// ==================== AgentInstanceRepo ====================

// AgentInstanceRepo 提供 Agent 实例的数据访问操作。
type AgentInstanceRepo struct {
	db *DB
}

// NewAgentInstanceRepo 创建 Agent 实例仓库。
func NewAgentInstanceRepo(db *DB) *AgentInstanceRepo {
	return &AgentInstanceRepo{db: db}
}

// Create 插入一个新 Agent 实例记录。
func (r *AgentInstanceRepo) Create(a *domain.AgentInstance) error {
	_, err := r.db.Exec(
		`INSERT INTO agent_instances (id, run_id, task_id, agent_spec_id, agent_kind, status, pid, tmux_session, workspace_path, last_heartbeat_at, last_output_at, checkpoint_id, metadata, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.RunID, a.TaskID, a.AgentSpecID, a.AgentKind, a.Status, a.PID, a.TmuxSession, a.WorkspacePath, a.LastHeartbeatAt, a.LastOutputAt, a.CheckpointID, a.Metadata, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

// scanAgentInstance 是 Agent 实例扫描的通用辅助函数。
func scanAgentInstance(row scanner) (*domain.AgentInstance, error) {
	a := &domain.AgentInstance{}
	err := row.Scan(&a.ID, &a.RunID, &a.TaskID, &a.AgentSpecID, &a.AgentKind, &a.Status, &a.PID, &a.TmuxSession, &a.WorkspacePath, &a.LastHeartbeatAt, &a.LastOutputAt, &a.CheckpointID, &a.Metadata, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
}

// GetByID 根据 ID 查询 Agent 实例。
func (r *AgentInstanceRepo) GetByID(id string) (*domain.AgentInstance, error) {
	row := r.db.QueryRow(
		`SELECT id, run_id, task_id, agent_spec_id, agent_kind, status, pid, tmux_session, workspace_path, last_heartbeat_at, last_output_at, checkpoint_id, metadata, created_at, updated_at
			 FROM agent_instances WHERE id = ?`, id,
	)
	return scanAgentInstance(row)
}

// UpdateStatus 更新 Agent 实例状态。
func (r *AgentInstanceRepo) UpdateStatus(id string, status domain.AgentInstanceStatus) error {
	_, err := r.db.Exec(
		"UPDATE agent_instances SET status = ?, updated_at = ? WHERE id = ?", status, time.Now(), id,
	)
	return err
}

// UpdateHeartbeat 更新 Agent 的心跳时间。
func (r *AgentInstanceRepo) UpdateHeartbeat(id string) error {
	_, err := r.db.Exec(
		"UPDATE agent_instances SET last_heartbeat_at = ?, updated_at = ? WHERE id = ?", time.Now(), time.Now(), id,
	)
	return err
}

// UpdatePID 更新 Agent 的进程 ID。
func (r *AgentInstanceRepo) UpdatePID(id string, pid int) error {
	_, err := r.db.Exec(
		"UPDATE agent_instances SET pid = ?, updated_at = ? WHERE id = ?", pid, time.Now(), id,
	)
	return err
}
	// UpdateLastOutputAt 更新 Agent 的最后输出时间。
	func (r *AgentInstanceRepo) UpdateLastOutputAt(id string) error {
		now := time.Now()
		_, err := r.db.Exec(
			"UPDATE agent_instances SET last_output_at = ?, updated_at = ? WHERE id = ?", now, now, id,
		)
		return err
	}


// UpdateCheckpointID 更新 Agent 的最新检查点引用。
func (r *AgentInstanceRepo) UpdateCheckpointID(id string, checkpointID string) error {
	_, err := r.db.Exec(
		"UPDATE agent_instances SET checkpoint_id = ?, updated_at = ? WHERE id = ?", checkpointID, time.Now(), id,
	)
	return err
}

// ListByRun 查询指定运行下的所有 Agent 实例。
func (r *AgentInstanceRepo) ListByRun(runID string) ([]*domain.AgentInstance, error) {
	rows, err := r.db.Query(
		`SELECT id, run_id, task_id, agent_spec_id, agent_kind, status, pid, tmux_session, workspace_path, last_heartbeat_at, last_output_at, checkpoint_id, metadata, created_at, updated_at
			 FROM agent_instances WHERE run_id = ? ORDER BY created_at ASC`, runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*domain.AgentInstance
	for rows.Next() {
		a, err := scanAgentInstance(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// ListByStatus 按状态查询 Agent 实例。
func (r *AgentInstanceRepo) ListByStatus(status domain.AgentInstanceStatus) ([]*domain.AgentInstance, error) {
	rows, err := r.db.Query(
		`SELECT id, run_id, task_id, agent_spec_id, agent_kind, status, pid, tmux_session, workspace_path, last_heartbeat_at, last_output_at, checkpoint_id, metadata, created_at, updated_at
			 FROM agent_instances WHERE status = ?`, status,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*domain.AgentInstance
	for rows.Next() {
		a, err := scanAgentInstance(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// ListAll 查询所有 Agent 实例。
func (r *AgentInstanceRepo) ListAll() ([]*domain.AgentInstance, error) {
	rows, err := r.db.Query(
		`SELECT id, run_id, task_id, agent_spec_id, agent_kind, status, pid, tmux_session, workspace_path, last_heartbeat_at, last_output_at, checkpoint_id, metadata, created_at, updated_at
			 FROM agent_instances ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []*domain.AgentInstance
	for rows.Next() {
		a, err := scanAgentInstance(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, a)
	}
	return agents, nil
}

// ==================== EventRepo ====================

// EventRepo 提供事件的数据访问操作。
type EventRepo struct {
	db *DB
}

// NewEventRepo 创建事件仓库实例。
func NewEventRepo(db *DB) *EventRepo {
	return &EventRepo{db: db}
}

// Create 插入一个新事件记录。
func (r *EventRepo) Create(e *domain.Event) error {
	_, err := r.db.Exec(
		"INSERT INTO events (id, run_id, task_id, agent_id, event_type, message, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		e.ID, e.RunID, e.TaskID, e.AgentID, e.EventType, e.Message, e.Metadata, e.CreatedAt,
	)
	return err
}

// ListByRun 查询指定运行的所有事件，按创建时间升序排列。
func (r *EventRepo) ListByRun(runID string) ([]*domain.Event, error) {
	rows, err := r.db.Query(
		"SELECT id, run_id, task_id, agent_id, event_type, message, metadata, created_at FROM events WHERE run_id = ? ORDER BY created_at ASC", runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		e := &domain.Event{}
		if err := rows.Scan(&e.ID, &e.RunID, &e.TaskID, &e.AgentID, &e.EventType, &e.Message, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

// DeleteOlderThan 删除指定时间之前的所有事件。
func (r *EventRepo) DeleteOlderThan(cutoff time.Time) error {
	_, err := r.db.Exec("DELETE FROM events WHERE created_at < ?", cutoff)
	return err
}

// ==================== CheckpointRepo ====================

// CheckpointRepo 提供检查点的数据访问操作。
type CheckpointRepo struct {
	db *DB
}

// NewCheckpointRepo 创建检查点仓库实例。
func NewCheckpointRepo(db *DB) *CheckpointRepo {
	return &CheckpointRepo{db: db}
}

// Create 插入一个新检查点记录。
func (r *CheckpointRepo) Create(c *domain.Checkpoint) error {
	_, err := r.db.Exec(
		"INSERT INTO checkpoints (id, agent_id, task_id, run_id, phase, state_data, reason, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		c.ID, c.AgentID, c.TaskID, c.RunID, c.Phase, c.StateData, c.Reason, c.CreatedAt,
	)
	return err
}

// LatestByTask 获取指定任务的最新检查点。
func (r *CheckpointRepo) LatestByTask(taskID string) (*domain.Checkpoint, error) {
	c := &domain.Checkpoint{}
	err := r.db.QueryRow(
		"SELECT id, agent_id, task_id, run_id, phase, state_data, reason, created_at FROM checkpoints WHERE task_id = ? ORDER BY created_at DESC LIMIT 1", taskID,
	).Scan(&c.ID, &c.AgentID, &c.TaskID, &c.RunID, &c.Phase, &c.StateData, &c.Reason, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// ListByTask 查询指定任务的所有检查点，按创建时间降序排列。
func (r *CheckpointRepo) ListByTask(taskID string) ([]*domain.Checkpoint, error) {
	rows, err := r.db.Query(
		"SELECT id, agent_id, task_id, run_id, phase, state_data, reason, created_at FROM checkpoints WHERE task_id = ? ORDER BY created_at DESC", taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checkpoints []*domain.Checkpoint
	for rows.Next() {
		c := &domain.Checkpoint{}
		if err := rows.Scan(&c.ID, &c.AgentID, &c.TaskID, &c.RunID, &c.Phase, &c.StateData, &c.Reason, &c.CreatedAt); err != nil {
			return nil, err
		}
		checkpoints = append(checkpoints, c)
	}
	return checkpoints, nil
}

// ==================== ResourceSnapshotRepo ====================

// ResourceSnapshotRepo 提供资源快照的数据访问操作。
type ResourceSnapshotRepo struct {
	db *DB
}

// NewResourceSnapshotRepo 创建资源快照仓库实例。
func NewResourceSnapshotRepo(db *DB) *ResourceSnapshotRepo {
	return &ResourceSnapshotRepo{db: db}
}

// Create 插入一个新资源快照记录。
func (r *ResourceSnapshotRepo) Create(s *domain.ResourceSnapshot) error {
	_, err := r.db.Exec(
		"INSERT INTO resource_snapshots (id, memory_percent, cpu_percent, disk_percent, active_agents, pressure_level, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		s.ID, s.MemoryPercent, s.CPUPercent, s.DiskPercent, s.ActiveAgents, s.PressureLevel, s.CreatedAt,
	)
	return err
}

// Latest 获取最新的资源快照。
func (r *ResourceSnapshotRepo) Latest() (*domain.ResourceSnapshot, error) {
	s := &domain.ResourceSnapshot{}
	err := r.db.QueryRow(
		"SELECT id, memory_percent, cpu_percent, disk_percent, active_agents, pressure_level, created_at FROM resource_snapshots ORDER BY created_at DESC LIMIT 1",
	).Scan(&s.ID, &s.MemoryPercent, &s.CPUPercent, &s.DiskPercent, &s.ActiveAgents, &s.PressureLevel, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return s, err
}

// DeleteOlderThan 删除指定时间之前的所有资源快照。
func (r *ResourceSnapshotRepo) DeleteOlderThan(cutoff time.Time) error {
	_, err := r.db.Exec("DELETE FROM resource_snapshots WHERE created_at < ?", cutoff)
	return err
}

// ==================== WorkspaceRepo ====================

// WorkspaceRepo 提供工作区的数据访问操作。
type WorkspaceRepo struct {
	db *DB
}

// NewWorkspaceRepo 创建工作区仓库实例。
func NewWorkspaceRepo(db *DB) *WorkspaceRepo {
	return &WorkspaceRepo{db: db}
}

// Create 插入一个新工作区记录。
func (r *WorkspaceRepo) Create(w *domain.Workspace) error {
	_, err := r.db.Exec(
		"INSERT INTO workspaces (id, task_id, project_id, path, branch, commit_sha, size_bytes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		w.ID, w.TaskID, w.ProjectID, w.Path, w.Branch, w.CommitSHA, w.SizeBytes, w.CreatedAt, w.UpdatedAt,
	)
	return err
}

// GetByTaskID 根据任务 ID 查询工作区。
func (r *WorkspaceRepo) GetByTaskID(taskID string) (*domain.Workspace, error) {
	w := &domain.Workspace{}
	err := r.db.QueryRow(
		"SELECT id, task_id, project_id, path, branch, commit_sha, size_bytes, created_at, updated_at FROM workspaces WHERE task_id = ?", taskID,
	).Scan(&w.ID, &w.TaskID, &w.ProjectID, &w.Path, &w.Branch, &w.CommitSHA, &w.SizeBytes, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return w, err
}

// ListActive 查询所有工作区。
func (r *WorkspaceRepo) ListActive() ([]*domain.Workspace, error) {
	rows, err := r.db.Query(
		"SELECT id, task_id, project_id, path, branch, commit_sha, size_bytes, created_at, updated_at FROM workspaces",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workspaces []*domain.Workspace
	for rows.Next() {
		w := &domain.Workspace{}
		if err := rows.Scan(&w.ID, &w.TaskID, &w.ProjectID, &w.Path, &w.Branch, &w.CommitSHA, &w.SizeBytes, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, w)
	}
	return workspaces, nil
}

// ==================== TerminalSessionRepo ====================

// TerminalSessionRepo 提供终端会话的数据访问操作。
type TerminalSessionRepo struct {
	db *DB
}

// NewTerminalSessionRepo 创建终端会话仓库实例。
func NewTerminalSessionRepo(db *DB) *TerminalSessionRepo {
	return &TerminalSessionRepo{db: db}
}

// Create 插入一个新终端会话记录。
func (r *TerminalSessionRepo) Create(t *domain.TerminalSession) error {
	_, err := r.db.Exec(
		"INSERT INTO terminal_sessions (id, task_id, agent_id, tmux_session, tmux_pane, status, log_file_path, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		t.ID, t.TaskID, t.AgentID, t.TmuxSession, t.TmuxPane, t.Status, t.LogFilePath, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

// GetByID 根据 ID 查询终端会话。
func (r *TerminalSessionRepo) GetByID(id string) (*domain.TerminalSession, error) {
	t := &domain.TerminalSession{}
	err := r.db.QueryRow(
		"SELECT id, task_id, agent_id, tmux_session, tmux_pane, status, log_file_path, created_at, updated_at FROM terminal_sessions WHERE id = ?", id,
	).Scan(&t.ID, &t.TaskID, &t.AgentID, &t.TmuxSession, &t.TmuxPane, &t.Status, &t.LogFilePath, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

// UpdateStatus 更新终端会话状态。
func (r *TerminalSessionRepo) UpdateStatus(id string, status domain.TerminalSessionStatus) error {
	_, err := r.db.Exec(
		"UPDATE terminal_sessions SET status = ?, updated_at = ? WHERE id = ?", status, time.Now(), id,
	)
	return err
}

// ListAll 查询所有终端会话。
func (r *TerminalSessionRepo) ListAll() ([]*domain.TerminalSession, error) {
	rows, err := r.db.Query(
		"SELECT id, task_id, agent_id, tmux_session, tmux_pane, status, log_file_path, created_at, updated_at FROM terminal_sessions ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var terminals []*domain.TerminalSession
	for rows.Next() {
		t := &domain.TerminalSession{}
		if err := rows.Scan(&t.ID, &t.TaskID, &t.AgentID, &t.TmuxSession, &t.TmuxPane, &t.Status, &t.LogFilePath, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		terminals = append(terminals, t)
	}
	return terminals, nil
}

// ==================== CommandLogRepo ====================

// CommandLogRepo 提供命令日志的数据访问操作。
type CommandLogRepo struct {
	db *DB
}

// NewCommandLogRepo 创建命令日志仓库实例。
func NewCommandLogRepo(db *DB) *CommandLogRepo {
	return &CommandLogRepo{db: db}
}

// Create 插入一条命令日志记录。
func (r *CommandLogRepo) Create(c *domain.CommandLog) error {
	_, err := r.db.Exec(
		"INSERT INTO command_logs (id, task_id, agent_id, command, exit_code, output, duration_ms, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		c.ID, c.TaskID, c.AgentID, c.Command, c.ExitCode, c.Output, c.Duration, c.CreatedAt,
	)
	return err
}

// ==================== TaskSpecRepo ====================

// TaskSpecRepo 提供任务规格的数据访问操作。
type TaskSpecRepo struct {
	db *DB
}

// NewTaskSpecRepo 创建任务规格仓库实例。
func NewTaskSpecRepo(db *DB) *TaskSpecRepo {
	return &TaskSpecRepo{db: db}
}

// Create 插入一个新任务规格记录。
func (r *TaskSpecRepo) Create(ts *domain.TaskSpec) error {
	_, err := r.db.Exec(
		"INSERT INTO task_specs (id, name, task_type, runtime_type, command_template, timeout_seconds, retry_policy, resource_class, can_pause, can_checkpoint, required_inputs, expected_outputs) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		ts.ID, ts.Name, ts.TaskType, ts.RuntimeType, ts.CommandTemplate, ts.TimeoutSeconds, ts.RetryPolicy, ts.ResourceClass, ts.CanPause, ts.CanCheckpoint, ts.RequiredInputs, ts.ExpectedOutputs,
	)
	return err
}

// GetByID 根据 ID 查询任务规格。
func (r *TaskSpecRepo) GetByID(id string) (*domain.TaskSpec, error) {
	ts := &domain.TaskSpec{}
	err := r.db.QueryRow(
		"SELECT id, name, task_type, runtime_type, command_template, timeout_seconds, retry_policy, resource_class, can_pause, can_checkpoint, required_inputs, expected_outputs FROM task_specs WHERE id = ?", id,
	).Scan(&ts.ID, &ts.Name, &ts.TaskType, &ts.RuntimeType, &ts.CommandTemplate, &ts.TimeoutSeconds, &ts.RetryPolicy, &ts.ResourceClass, &ts.CanPause, &ts.CanCheckpoint, &ts.RequiredInputs, &ts.ExpectedOutputs)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return ts, err
}

// List 查询所有任务规格。
func (r *TaskSpecRepo) List() ([]*domain.TaskSpec, error) {
	rows, err := r.db.Query("SELECT id, name, task_type, runtime_type, command_template, timeout_seconds, retry_policy, resource_class, can_pause, can_checkpoint, required_inputs, expected_outputs FROM task_specs")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var specs []*domain.TaskSpec
	for rows.Next() {
		ts := &domain.TaskSpec{}
		if err := rows.Scan(&ts.ID, &ts.Name, &ts.TaskType, &ts.RuntimeType, &ts.CommandTemplate, &ts.TimeoutSeconds, &ts.RetryPolicy, &ts.ResourceClass, &ts.CanPause, &ts.CanCheckpoint, &ts.RequiredInputs, &ts.ExpectedOutputs); err != nil {
			return nil, err
		}
		specs = append(specs, ts)
	}
	return specs, nil
}

// ==================== AgentSpecRepo ====================

// AgentSpecRepo 提供 Agent 规格的数据访问操作。
type AgentSpecRepo struct {
	db *DB
}

// NewAgentSpecRepo 创建 Agent 规格仓库实例。
func NewAgentSpecRepo(db *DB) *AgentSpecRepo {
	return &AgentSpecRepo{db: db}
}

// Create 插入一个新 Agent 规格记录。
func (r *AgentSpecRepo) Create(as *domain.AgentSpec) error {
	_, err := r.db.Exec(
		"INSERT INTO agent_specs (id, name, agent_kind, supported_task_types, default_command, max_concurrency, resource_weight, heartbeat_mode, output_parser) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		as.ID, as.Name, as.AgentKind, as.SupportedTaskTypes, as.DefaultCommand, as.MaxConcurrency, as.ResourceWeight, as.HeartbeatMode, as.OutputParser,
	)
	return err
}

// GetByID 根据 ID 查询 Agent 规格。
func (r *AgentSpecRepo) GetByID(id string) (*domain.AgentSpec, error) {
	as := &domain.AgentSpec{}
	err := r.db.QueryRow(
		"SELECT id, name, agent_kind, supported_task_types, default_command, max_concurrency, resource_weight, heartbeat_mode, output_parser FROM agent_specs WHERE id = ?", id,
	).Scan(&as.ID, &as.Name, &as.AgentKind, &as.SupportedTaskTypes, &as.DefaultCommand, &as.MaxConcurrency, &as.ResourceWeight, &as.HeartbeatMode, &as.OutputParser)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return as, err
}


	// GetByKind 根据 agent_kind 查询 Agent 规格，未找到时返回 nil。
	func (r *AgentSpecRepo) GetByKind(kind string) (*domain.AgentSpec, error) {
		as := &domain.AgentSpec{}
		err := r.db.QueryRow(
			"SELECT id, name, agent_kind, supported_task_types, default_command, max_concurrency, resource_weight, heartbeat_mode, output_parser FROM agent_specs WHERE agent_kind = ?", kind,
		).Scan(&as.ID, &as.Name, &as.AgentKind, &as.SupportedTaskTypes, &as.DefaultCommand, &as.MaxConcurrency, &as.ResourceWeight, &as.HeartbeatMode, &as.OutputParser)
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return as, err
	}
// List 查询所有 Agent 规格。
func (r *AgentSpecRepo) List() ([]*domain.AgentSpec, error) {
	rows, err := r.db.Query("SELECT id, name, agent_kind, supported_task_types, default_command, max_concurrency, resource_weight, heartbeat_mode, output_parser FROM agent_specs")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var specs []*domain.AgentSpec
	for rows.Next() {
		as := &domain.AgentSpec{}
		if err := rows.Scan(&as.ID, &as.Name, &as.AgentKind, &as.SupportedTaskTypes, &as.DefaultCommand, &as.MaxConcurrency, &as.ResourceWeight, &as.HeartbeatMode, &as.OutputParser); err != nil {
			return nil, err
		}
		specs = append(specs, as)
	}
	return specs, nil
}

// ==================== WorkflowTemplateRepo ====================

// WorkflowTemplateRepo 提供工作流模板的数据访问操作。
type WorkflowTemplateRepo struct {
	db *DB
}

// NewWorkflowTemplateRepo 创建工作流模板仓库实例。
func NewWorkflowTemplateRepo(db *DB) *WorkflowTemplateRepo {
	return &WorkflowTemplateRepo{db: db}
}

// Create 插入一个新工作流模板记录。
func (r *WorkflowTemplateRepo) Create(wt *domain.WorkflowTemplate) error {
	_, err := r.db.Exec(
		"INSERT INTO workflow_templates (id, name, description, nodes_json, edges_json, on_failure) VALUES (?, ?, ?, ?, ?, ?)",
		wt.ID, wt.Name, wt.Description, wt.NodesJSON, wt.EdgesJSON, wt.OnFailure,
	)
	return err
}

// GetByID 根据 ID 查询工作流模板。
func (r *WorkflowTemplateRepo) GetByID(id string) (*domain.WorkflowTemplate, error) {
	wt := &domain.WorkflowTemplate{}
	err := r.db.QueryRow(
		"SELECT id, name, description, nodes_json, edges_json, on_failure FROM workflow_templates WHERE id = ?", id,
	).Scan(&wt.ID, &wt.Name, &wt.Description, &wt.NodesJSON, &wt.EdgesJSON, &wt.OnFailure)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return wt, err
}

// List 查询所有工作流模板。
func (r *WorkflowTemplateRepo) List() ([]*domain.WorkflowTemplate, error) {
	rows, err := r.db.Query("SELECT id, name, description, nodes_json, edges_json, on_failure FROM workflow_templates")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*domain.WorkflowTemplate
	for rows.Next() {
		wt := &domain.WorkflowTemplate{}
		if err := rows.Scan(&wt.ID, &wt.Name, &wt.Description, &wt.NodesJSON, &wt.EdgesJSON, &wt.OnFailure); err != nil {
			return nil, err
		}
		templates = append(templates, wt)
	}
	return templates, nil
}

// ==================== PromptDraftRepo ====================

// PromptDraftRepo 提供提示词草稿实体的数据访问操作。
type PromptDraftRepo struct {
	db *DB
}

// NewPromptDraftRepo 创建提示词草稿仓库实例。
func NewPromptDraftRepo(db *DB) *PromptDraftRepo {
	return &PromptDraftRepo{db: db}
}

// Create 插入一个新的提示词草稿记录。
func (r *PromptDraftRepo) Create(d *domain.PromptDraft) error {
	_, err := r.db.Exec(
		"INSERT INTO prompt_drafts (id, project_id, original_input, generated_prompt, final_prompt, task_type, status, run_id, sent_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		d.ID, d.ProjectID, d.OriginalInput, d.GeneratedPrompt, d.FinalPrompt, d.TaskType, d.Status, d.RunID, d.SentAt, d.CreatedAt, d.UpdatedAt,
	)
	return err
}

// GetByID 根据 ID 查询提示词草稿，未找到时返回 nil。
func (r *PromptDraftRepo) GetByID(id string) (*domain.PromptDraft, error) {
	d := &domain.PromptDraft{}
	err := r.db.QueryRow(
		"SELECT id, project_id, original_input, generated_prompt, final_prompt, task_type, status, run_id, sent_at, created_at, updated_at FROM prompt_drafts WHERE id = ?", id,
	).Scan(&d.ID, &d.ProjectID, &d.OriginalInput, &d.GeneratedPrompt, &d.FinalPrompt, &d.TaskType, &d.Status, &d.RunID, &d.SentAt, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return d, err
}

// ListByProject 查询指定项目的所有草稿，按创建时间降序排列。
func (r *PromptDraftRepo) ListByProject(projectID string) ([]*domain.PromptDraft, error) {
	rows, err := r.db.Query(
		"SELECT id, project_id, original_input, generated_prompt, final_prompt, task_type, status, run_id, sent_at, created_at, updated_at FROM prompt_drafts WHERE project_id = ? ORDER BY created_at DESC", projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var drafts []*domain.PromptDraft
	for rows.Next() {
		d := &domain.PromptDraft{}
		if err := rows.Scan(&d.ID, &d.ProjectID, &d.OriginalInput, &d.GeneratedPrompt, &d.FinalPrompt, &d.TaskType, &d.Status, &d.RunID, &d.SentAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		drafts = append(drafts, d)
	}
	return drafts, nil
}

// Update 更新提示词草稿的 final_prompt、status 和 updated_at。
func (r *PromptDraftRepo) Update(d *domain.PromptDraft) error {
	_, err := r.db.Exec(
		"UPDATE prompt_drafts SET final_prompt = ?, status = ?, task_type = ?, updated_at = ? WHERE id = ?",
		d.FinalPrompt, d.Status, d.TaskType, d.UpdatedAt, d.ID,
	)
	return err
}

// UpdateStatus 更新提示词草稿的状态和更新时间。
func (r *PromptDraftRepo) UpdateStatus(id string, status domain.PromptDraftStatus) error {
	_, err := r.db.Exec(
		"UPDATE prompt_drafts SET status = ?, updated_at = ? WHERE id = ?",
		status, time.Now(), id,
	)
	return err
}

// MarkSent CAS 更新草稿为 sent 状态，同时写入 run_id 和 sent_at。
// 只有当前状态为 draft 的草稿才会被更新，返回受影响行数（0 表示已被其他请求发送）。
func (r *PromptDraftRepo) MarkSent(id string, runID string) (int64, error) {
	now := time.Now()
	result, err := r.db.Exec(
		"UPDATE prompt_drafts SET status = ?, run_id = ?, sent_at = ?, updated_at = ? WHERE id = ? AND status = ?",
		domain.PromptDraftStatusSent, runID, now, now, id, domain.PromptDraftStatusDraft,
	)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// ResetToDraft 将已发送的草稿回滚为 draft 状态（用于 CAS 成功但 Run 创建失败时的回滚）。
func (r *PromptDraftRepo) ResetToDraft(id string) error {
	_, err := r.db.Exec(
		"UPDATE prompt_drafts SET status = ?, run_id = '', sent_at = NULL, updated_at = ? WHERE id = ? AND status = ?",
		domain.PromptDraftStatusDraft, time.Now(), id, domain.PromptDraftStatusSent,
	)
	return err
}

// UpdateRunID 更新已发送草稿的 run_id（用于 CAS 先行策略中 Run 创建后补填）。
func (r *PromptDraftRepo) UpdateRunID(id string, runID string) error {
	_, err := r.db.Exec(
		"UPDATE prompt_drafts SET run_id = ?, updated_at = ? WHERE id = ?",
		runID, time.Now(), id,
	)
	return err
}

// ==================== GateRepo ====================

// GateRepo 提供质量门禁实体的数据访问操作。
type GateRepo struct {
	db *DB
}

// NewGateRepo 创建门禁仓库实例。
func NewGateRepo(db *DB) *GateRepo {
	return &GateRepo{db: db}
}

// Create 插入一个新的门禁记录。
func (r *GateRepo) Create(g *domain.Gate) error {
	_, err := r.db.Exec(
		"INSERT INTO gates (id, run_id, node_id, gate_type, status, config_json, verify_result, approved_by, approved_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		g.ID, g.RunID, g.NodeID, g.GateType, g.Status, g.ConfigJSON, g.VerifyResult, g.ApprovedBy, g.ApprovedAt, g.CreatedAt, g.UpdatedAt,
	)
	return err
}

// GetByID 根据 ID 查询门禁，未找到时返回 nil。
func (r *GateRepo) GetByID(id string) (*domain.Gate, error) {
	g := &domain.Gate{}
	var approvedAt sql.NullTime
	err := r.db.QueryRow(
		"SELECT id, run_id, node_id, gate_type, status, config_json, verify_result, approved_by, approved_at, created_at, updated_at FROM gates WHERE id = ?", id,
	).Scan(&g.ID, &g.RunID, &g.NodeID, &g.GateType, &g.Status, &g.ConfigJSON, &g.VerifyResult, &g.ApprovedBy, &approvedAt, &g.CreatedAt, &g.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if approvedAt.Valid {
		g.ApprovedAt = &approvedAt.Time
	}
	return g, err
}

// GetByRunAndNode 根据运行 ID 和节点 ID 查询门禁。
func (r *GateRepo) GetByRunAndNode(runID, nodeID string) (*domain.Gate, error) {
	g := &domain.Gate{}
	var approvedAt sql.NullTime
	err := r.db.QueryRow(
		"SELECT id, run_id, node_id, gate_type, status, config_json, verify_result, approved_by, approved_at, created_at, updated_at FROM gates WHERE run_id = ? AND node_id = ?", runID, nodeID,
	).Scan(&g.ID, &g.RunID, &g.NodeID, &g.GateType, &g.Status, &g.ConfigJSON, &g.VerifyResult, &g.ApprovedBy, &approvedAt, &g.CreatedAt, &g.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if approvedAt.Valid {
		g.ApprovedAt = &approvedAt.Time
	}
	return g, err
}

// ListByRun 查询指定运行的所有门禁，按创建时间升序。
func (r *GateRepo) ListByRun(runID string) ([]*domain.Gate, error) {
	rows, err := r.db.Query(
		"SELECT id, run_id, node_id, gate_type, status, config_json, verify_result, approved_by, approved_at, created_at, updated_at FROM gates WHERE run_id = ? ORDER BY created_at ASC", runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gates []*domain.Gate
	for rows.Next() {
		g := &domain.Gate{}
		var approvedAt sql.NullTime
		if err := rows.Scan(&g.ID, &g.RunID, &g.NodeID, &g.GateType, &g.Status, &g.ConfigJSON, &g.VerifyResult, &g.ApprovedBy, &approvedAt, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		if approvedAt.Valid {
			g.ApprovedAt = &approvedAt.Time
		}
		gates = append(gates, g)
	}
	return gates, nil
}

// ListPendingByRun 查询指定运行中所有待处理的门禁。
func (r *GateRepo) ListPendingByRun(runID string) ([]*domain.Gate, error) {
	rows, err := r.db.Query(
		"SELECT id, run_id, node_id, gate_type, status, config_json, verify_result, approved_by, approved_at, created_at, updated_at FROM gates WHERE run_id = ? AND status = ? ORDER BY created_at ASC", runID, domain.GateStatusPending,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gates []*domain.Gate
	for rows.Next() {
		g := &domain.Gate{}
		var approvedAt sql.NullTime
		if err := rows.Scan(&g.ID, &g.RunID, &g.NodeID, &g.GateType, &g.Status, &g.ConfigJSON, &g.VerifyResult, &g.ApprovedBy, &approvedAt, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		if approvedAt.Valid {
			g.ApprovedAt = &approvedAt.Time
		}
		gates = append(gates, g)
	}
	return gates, nil
}

// UpdateStatus 更新门禁的状态和更新时间。
func (r *GateRepo) UpdateStatus(id string, status domain.GateStatus) error {
	_, err := r.db.Exec(
		"UPDATE gates SET status = ?, updated_at = ? WHERE id = ?",
		status, time.Now(), id,
	)
	return err
}

// Approve 通过门禁，设置状态为 passed、审批人和审批时间。
func (r *GateRepo) Approve(id string, approvedBy string) error {
	now := time.Now()
	_, err := r.db.Exec(
		"UPDATE gates SET status = ?, approved_by = ?, approved_at = ?, updated_at = ? WHERE id = ?",
		domain.GateStatusPassed, approvedBy, now, now, id,
	)
	return err
}

// UpdateVerifyResult 更新门禁的验证结果和状态。
func (r *GateRepo) UpdateVerifyResult(id string, result string, status domain.GateStatus) error {
	_, err := r.db.Exec(
		"UPDATE gates SET verify_result = ?, status = ?, updated_at = ? WHERE id = ?",
		result, status, time.Now(), id,
	)
	return err
}

// ==================== Repos 聚合 ====================

// Repos 是所有数据仓库的聚合容器，提供统一的访问入口。
type Repos struct {
	Projects          *ProjectRepo
	Runs              *RunRepo
	Tasks             *TaskRepo
	AgentInstances    *AgentInstanceRepo
	Events            *EventRepo
	Checkpoints       *CheckpointRepo
	ResourceSnapshots *ResourceSnapshotRepo
	Workspaces        *WorkspaceRepo
	TerminalSessions  *TerminalSessionRepo
	CommandLogs       *CommandLogRepo
	TaskSpecs         *TaskSpecRepo
	AgentSpecs        *AgentSpecRepo
	WorkflowTemplates *WorkflowTemplateRepo
	PromptDrafts      *PromptDraftRepo
	Gates             *GateRepo
}

// NewRepos 创建并初始化所有数据仓库。
func NewRepos(db *DB) *Repos {
	return &Repos{
		Projects:          NewProjectRepo(db),
		Runs:              NewRunRepo(db),
		Tasks:             NewTaskRepo(db),
		AgentInstances:    NewAgentInstanceRepo(db),
		Events:            NewEventRepo(db),
		Checkpoints:       NewCheckpointRepo(db),
		ResourceSnapshots: NewResourceSnapshotRepo(db),
		Workspaces:        NewWorkspaceRepo(db),
		TerminalSessions:  NewTerminalSessionRepo(db),
		CommandLogs:       NewCommandLogRepo(db),
		TaskSpecs:         NewTaskSpecRepo(db),
		AgentSpecs:        NewAgentSpecRepo(db),
		WorkflowTemplates: NewWorkflowTemplateRepo(db),
		PromptDrafts:      NewPromptDraftRepo(db),
		Gates:             NewGateRepo(db),
	}
}

// ActiveAgentsResult 包含 Agent 实例及其关联任务的联合查询结果。
type ActiveAgentsResult struct {
	Agent *domain.AgentInstance
	Task  *domain.Task
}

// ListActiveWithTasks 联合查询所有活跃 Agent 及其关联任务。
// 活跃状态包括 starting、running 和 paused。
func (r *AgentInstanceRepo) ListActiveWithTasks() ([]ActiveAgentsResult, error) {
	query := `SELECT a.id, a.run_id, a.task_id, a.agent_spec_id, a.agent_kind, a.status, a.pid, a.tmux_session, a.workspace_path, a.last_heartbeat_at, a.last_output_at, a.checkpoint_id, a.metadata, a.created_at, a.updated_at,
			        t.id, t.run_id, t.task_spec_id, t.task_type, t.attempt_no, t.status, t.priority, t.queue_status, t.resource_class, t.preemptible, t.restart_policy, t.title, t.description, t.input_data, t.output_data, t.workspace_path, t.parent_task_id, t.started_at, t.completed_at, t.created_at, t.updated_at
			 FROM agent_instances a
			 JOIN tasks t ON a.task_id = t.id
			 WHERE a.status IN (?, ?, ?)
			 ORDER BY t.priority ASC, a.created_at ASC`
	rows, err := r.db.Query(query, domain.AgentStatusStarting, domain.AgentStatusRunning, domain.AgentStatusPaused)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ActiveAgentsResult
	for rows.Next() {
		a := &domain.AgentInstance{}
		t := &domain.Task{}
		if err := rows.Scan(
			&a.ID, &a.RunID, &a.TaskID, &a.AgentSpecID, &a.AgentKind, &a.Status, &a.PID, &a.TmuxSession, &a.WorkspacePath, &a.LastHeartbeatAt, &a.LastOutputAt, &a.CheckpointID, &a.Metadata, &a.CreatedAt, &a.UpdatedAt,
			&t.ID, &t.RunID, &t.TaskSpecID, &t.TaskType, &t.AttemptNo, &t.Status, &t.Priority, &t.QueueStatus, &t.ResourceClass, &t.Preemptible, &t.RestartPolicy, &t.Title, &t.Description, &t.InputData, &t.OutputData, &t.WorkspacePath, &t.ParentTaskID, &t.StartedAt, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, ActiveAgentsResult{Agent: a, Task: t})
	}
	return results, nil
}
