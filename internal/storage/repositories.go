package storage

import (
	"database/sql"
	"time"

	"github.com/xxy757/xxyCodingAgents/internal/domain"
)

type scanner interface {
	Scan(dest ...any) error
}

type ProjectRepo struct {
	db *DB
}

func NewProjectRepo(db *DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

func (r *ProjectRepo) Create(p *domain.Project) error {
	_, err := r.db.Exec(
		"INSERT INTO projects (id, name, repo_url, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		p.ID, p.Name, p.RepoURL, p.Description, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

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

type RunRepo struct {
	db *DB
}

func NewRunRepo(db *DB) *RunRepo {
	return &RunRepo{db: db}
}

func (r *RunRepo) Create(run *domain.Run) error {
	_, err := r.db.Exec(
		"INSERT INTO runs (id, project_id, workflow_template_id, title, description, status, external_key, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		run.ID, run.ProjectID, run.WorkflowTemplateID, run.Title, run.Description, run.Status, run.ExternalKey, run.CreatedAt, run.UpdatedAt,
	)
	return err
}

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

func (r *RunRepo) UpdateStatus(id string, status domain.RunStatus) error {
	_, err := r.db.Exec(
		"UPDATE runs SET status = ?, updated_at = ? WHERE id = ?", status, time.Now(), id,
	)
	return err
}

type TaskRepo struct {
	db *DB
}

func NewTaskRepo(db *DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) Create(t *domain.Task) error {
	_, err := r.db.Exec(
		`INSERT INTO tasks (id, run_id, task_spec_id, task_type, attempt_no, status, priority, queue_status, resource_class, preemptible, restart_policy, title, description, input_data, output_data, workspace_path, parent_task_id, started_at, completed_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.RunID, t.TaskSpecID, t.TaskType, t.AttemptNo, t.Status, t.Priority, t.QueueStatus, t.ResourceClass, t.Preemptible, t.RestartPolicy, t.Title, t.Description, t.InputData, t.OutputData, t.WorkspacePath, t.ParentTaskID, t.StartedAt, t.CompletedAt, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

func scanTask(row scanner) (*domain.Task, error) {
	t := &domain.Task{}
	err := row.Scan(&t.ID, &t.RunID, &t.TaskSpecID, &t.TaskType, &t.AttemptNo, &t.Status, &t.Priority, &t.QueueStatus, &t.ResourceClass, &t.Preemptible, &t.RestartPolicy, &t.Title, &t.Description, &t.InputData, &t.OutputData, &t.WorkspacePath, &t.ParentTaskID, &t.StartedAt, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return t, err
}

func (r *TaskRepo) GetByID(id string) (*domain.Task, error) {
	row := r.db.QueryRow(
		`SELECT id, run_id, task_spec_id, task_type, attempt_no, status, priority, queue_status, resource_class, preemptible, restart_policy, title, description, input_data, output_data, workspace_path, parent_task_id, started_at, completed_at, created_at, updated_at
		 FROM tasks WHERE id = ?`, id,
	)
	return scanTask(row)
}

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

func (r *TaskRepo) UpdateStatus(id string, status domain.TaskStatus) error {
	_, err := r.db.Exec(
		"UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?", status, time.Now(), id,
	)
	return err
}

func (r *TaskRepo) UpdateQueueStatus(id string, queueStatus string) error {
	_, err := r.db.Exec(
		"UPDATE tasks SET queue_status = ?, updated_at = ? WHERE id = ?", queueStatus, time.Now(), id,
	)
	return err
}

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

type AgentInstanceRepo struct {
	db *DB
}

func NewAgentInstanceRepo(db *DB) *AgentInstanceRepo {
	return &AgentInstanceRepo{db: db}
}

func (r *AgentInstanceRepo) Create(a *domain.AgentInstance) error {
	_, err := r.db.Exec(
		`INSERT INTO agent_instances (id, run_id, task_id, agent_spec_id, agent_kind, status, pid, tmux_session, workspace_path, last_heartbeat_at, last_output_at, checkpoint_id, metadata, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.RunID, a.TaskID, a.AgentSpecID, a.AgentKind, a.Status, a.PID, a.TmuxSession, a.WorkspacePath, a.LastHeartbeatAt, a.LastOutputAt, a.CheckpointID, a.Metadata, a.CreatedAt, a.UpdatedAt,
	)
	return err
}

func scanAgentInstance(row scanner) (*domain.AgentInstance, error) {
	a := &domain.AgentInstance{}
	err := row.Scan(&a.ID, &a.RunID, &a.TaskID, &a.AgentSpecID, &a.AgentKind, &a.Status, &a.PID, &a.TmuxSession, &a.WorkspacePath, &a.LastHeartbeatAt, &a.LastOutputAt, &a.CheckpointID, &a.Metadata, &a.CreatedAt, &a.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
}

func (r *AgentInstanceRepo) GetByID(id string) (*domain.AgentInstance, error) {
	row := r.db.QueryRow(
		`SELECT id, run_id, task_id, agent_spec_id, agent_kind, status, pid, tmux_session, workspace_path, last_heartbeat_at, last_output_at, checkpoint_id, metadata, created_at, updated_at
		 FROM agent_instances WHERE id = ?`, id,
	)
	return scanAgentInstance(row)
}

func (r *AgentInstanceRepo) UpdateStatus(id string, status domain.AgentInstanceStatus) error {
	_, err := r.db.Exec(
		"UPDATE agent_instances SET status = ?, updated_at = ? WHERE id = ?", status, time.Now(), id,
	)
	return err
}

func (r *AgentInstanceRepo) UpdateHeartbeat(id string) error {
	_, err := r.db.Exec(
		"UPDATE agent_instances SET last_heartbeat_at = ?, updated_at = ? WHERE id = ?", time.Now(), time.Now(), id,
	)
	return err
}

func (r *AgentInstanceRepo) UpdatePID(id string, pid int) error {
	_, err := r.db.Exec(
		"UPDATE agent_instances SET pid = ?, updated_at = ? WHERE id = ?", pid, time.Now(), id,
	)
	return err
}

func (r *AgentInstanceRepo) UpdateCheckpointID(id string, checkpointID string) error {
	_, err := r.db.Exec(
		"UPDATE agent_instances SET checkpoint_id = ?, updated_at = ? WHERE id = ?", checkpointID, time.Now(), id,
	)
	return err
}

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

type EventRepo struct {
	db *DB
}

func NewEventRepo(db *DB) *EventRepo {
	return &EventRepo{db: db}
}

func (r *EventRepo) Create(e *domain.Event) error {
	_, err := r.db.Exec(
		"INSERT INTO events (id, run_id, task_id, agent_id, event_type, message, metadata, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		e.ID, e.RunID, e.TaskID, e.AgentID, e.EventType, e.Message, e.Metadata, e.CreatedAt,
	)
	return err
}

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

func (r *EventRepo) DeleteOlderThan(cutoff time.Time) error {
	_, err := r.db.Exec("DELETE FROM events WHERE created_at < ?", cutoff)
	return err
}

type CheckpointRepo struct {
	db *DB
}

func NewCheckpointRepo(db *DB) *CheckpointRepo {
	return &CheckpointRepo{db: db}
}

func (r *CheckpointRepo) Create(c *domain.Checkpoint) error {
	_, err := r.db.Exec(
		"INSERT INTO checkpoints (id, agent_id, task_id, run_id, phase, state_data, reason, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		c.ID, c.AgentID, c.TaskID, c.RunID, c.Phase, c.StateData, c.Reason, c.CreatedAt,
	)
	return err
}

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

type ResourceSnapshotRepo struct {
	db *DB
}

func NewResourceSnapshotRepo(db *DB) *ResourceSnapshotRepo {
	return &ResourceSnapshotRepo{db: db}
}

func (r *ResourceSnapshotRepo) Create(s *domain.ResourceSnapshot) error {
	_, err := r.db.Exec(
		"INSERT INTO resource_snapshots (id, memory_percent, cpu_percent, disk_percent, active_agents, pressure_level, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		s.ID, s.MemoryPercent, s.CPUPercent, s.DiskPercent, s.ActiveAgents, s.PressureLevel, s.CreatedAt,
	)
	return err
}

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

func (r *ResourceSnapshotRepo) DeleteOlderThan(cutoff time.Time) error {
	_, err := r.db.Exec("DELETE FROM resource_snapshots WHERE created_at < ?", cutoff)
	return err
}

type WorkspaceRepo struct {
	db *DB
}

func NewWorkspaceRepo(db *DB) *WorkspaceRepo {
	return &WorkspaceRepo{db: db}
}

func (r *WorkspaceRepo) Create(w *domain.Workspace) error {
	_, err := r.db.Exec(
		"INSERT INTO workspaces (id, task_id, project_id, path, branch, commit_sha, size_bytes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		w.ID, w.TaskID, w.ProjectID, w.Path, w.Branch, w.CommitSHA, w.SizeBytes, w.CreatedAt, w.UpdatedAt,
	)
	return err
}

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

type TerminalSessionRepo struct {
	db *DB
}

func NewTerminalSessionRepo(db *DB) *TerminalSessionRepo {
	return &TerminalSessionRepo{db: db}
}

func (r *TerminalSessionRepo) Create(t *domain.TerminalSession) error {
	_, err := r.db.Exec(
		"INSERT INTO terminal_sessions (id, task_id, agent_id, tmux_session, tmux_pane, status, log_file_path, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		t.ID, t.TaskID, t.AgentID, t.TmuxSession, t.TmuxPane, t.Status, t.LogFilePath, t.CreatedAt, t.UpdatedAt,
	)
	return err
}

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

func (r *TerminalSessionRepo) UpdateStatus(id string, status domain.TerminalSessionStatus) error {
	_, err := r.db.Exec(
		"UPDATE terminal_sessions SET status = ?, updated_at = ? WHERE id = ?", status, time.Now(), id,
	)
	return err
}

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

type CommandLogRepo struct {
	db *DB
}

func NewCommandLogRepo(db *DB) *CommandLogRepo {
	return &CommandLogRepo{db: db}
}

func (r *CommandLogRepo) Create(c *domain.CommandLog) error {
	_, err := r.db.Exec(
		"INSERT INTO command_logs (id, task_id, agent_id, command, exit_code, output, duration_ms, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		c.ID, c.TaskID, c.AgentID, c.Command, c.ExitCode, c.Output, c.Duration, c.CreatedAt,
	)
	return err
}

type TaskSpecRepo struct {
	db *DB
}

func NewTaskSpecRepo(db *DB) *TaskSpecRepo {
	return &TaskSpecRepo{db: db}
}

func (r *TaskSpecRepo) Create(ts *domain.TaskSpec) error {
	_, err := r.db.Exec(
		"INSERT INTO task_specs (id, name, task_type, runtime_type, command_template, timeout_seconds, retry_policy, resource_class, can_pause, can_checkpoint, required_inputs, expected_outputs) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		ts.ID, ts.Name, ts.TaskType, ts.RuntimeType, ts.CommandTemplate, ts.TimeoutSeconds, ts.RetryPolicy, ts.ResourceClass, ts.CanPause, ts.CanCheckpoint, ts.RequiredInputs, ts.ExpectedOutputs,
	)
	return err
}

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

type AgentSpecRepo struct {
	db *DB
}

func NewAgentSpecRepo(db *DB) *AgentSpecRepo {
	return &AgentSpecRepo{db: db}
}

func (r *AgentSpecRepo) Create(as *domain.AgentSpec) error {
	_, err := r.db.Exec(
		"INSERT INTO agent_specs (id, name, agent_kind, supported_task_types, default_command, max_concurrency, resource_weight, heartbeat_mode, output_parser) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		as.ID, as.Name, as.AgentKind, as.SupportedTaskTypes, as.DefaultCommand, as.MaxConcurrency, as.ResourceWeight, as.HeartbeatMode, as.OutputParser,
	)
	return err
}

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

type WorkflowTemplateRepo struct {
	db *DB
}

func NewWorkflowTemplateRepo(db *DB) *WorkflowTemplateRepo {
	return &WorkflowTemplateRepo{db: db}
}

func (r *WorkflowTemplateRepo) Create(wt *domain.WorkflowTemplate) error {
	_, err := r.db.Exec(
		"INSERT INTO workflow_templates (id, name, description, nodes_json, edges_json, on_failure) VALUES (?, ?, ?, ?, ?, ?)",
		wt.ID, wt.Name, wt.Description, wt.NodesJSON, wt.EdgesJSON, wt.OnFailure,
	)
	return err
}

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
}

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
	}
}

type ActiveAgentsResult struct {
	Agent *domain.AgentInstance
	Task  *domain.Task
}

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
