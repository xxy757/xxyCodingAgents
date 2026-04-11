package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xxy757/xxyCodingAgents/internal/domain"
)

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		RepoURL     string `json:"repo_url"`
		Description string `json:"description"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	p := &domain.Project{
		ID:          uuid.New().String(),
		Name:        req.Name,
		RepoURL:     req.RepoURL,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := s.repos.Projects.Create(p); err != nil {
		slog.Error("create project", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create project")
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.repos.Projects.List()
	if err != nil {
		slog.Error("list projects", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list projects")
		return
	}
	if projects == nil {
		projects = []*domain.Project{}
	}
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := s.repos.Projects.GetByID(id)
	if err != nil {
		slog.Error("get project", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get project")
		return
	}
	if p == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID   string `json:"project_id"`
		TemplateID  string `json:"workflow_template_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ProjectID == "" || req.Title == "" {
		writeError(w, http.StatusBadRequest, "project_id and title are required")
		return
	}

	project, err := s.repos.Projects.GetByID(req.ProjectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify project")
		return
	}
	if project == nil {
		writeError(w, http.StatusBadRequest, "project not found")
		return
	}

	run, err := s.orch.CreateRun(r.Context(), req.ProjectID, req.TemplateID, req.Title, req.Description)
	if err != nil {
		slog.Error("create run", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create run")
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) handleListAllRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.repos.Runs.ListAll()
	if err != nil {
		slog.Error("list all runs", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}
	if runs == nil {
		runs = []*domain.Run{}
	}
	writeJSON(w, http.StatusOK, runs)
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	run, err := s.repos.Runs.GetByID(id)
	if err != nil {
		slog.Error("get run", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get run")
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) handleGetRunTimeline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	events, err := s.repos.Events.ListByRun(id)
	if err != nil {
		slog.Error("get run timeline", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get timeline")
		return
	}
	if events == nil {
		events = []*domain.Event{}
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	runs, err := s.repos.Runs.ListByProject(projectID)
	if err != nil {
		slog.Error("list runs", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}
	if runs == nil {
		runs = []*domain.Run{}
	}
	writeJSON(w, http.StatusOK, runs)
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	tasks, err := s.repos.Tasks.ListByRun(runID)
	if err != nil {
		slog.Error("list tasks", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	if tasks == nil {
		tasks = []*domain.Task{}
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (s *Server) handleRetryTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := s.repos.Tasks.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if task.Status != domain.TaskStatusFailed && task.Status != domain.TaskStatusCancelled {
		writeError(w, http.StatusBadRequest, "only failed or cancelled tasks can be retried")
		return
	}

	maxAttempt, err := s.repos.Tasks.MaxAttemptNo(task.RunID, task.TaskType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get max attempt")
		return
	}
	if maxAttempt >= 3 {
		writeError(w, http.StatusBadRequest, "max retry attempts exceeded")
		return
	}

	newTask := &domain.Task{
		ID:            uuid.New().String(),
		RunID:         task.RunID,
		TaskSpecID:    task.TaskSpecID,
		TaskType:      task.TaskType,
		AttemptNo:     maxAttempt + 1,
		Status:        domain.TaskStatusQueued,
		Priority:      task.Priority,
		QueueStatus:   "queued",
		ResourceClass: task.ResourceClass,
		Preemptible:   task.Preemptible,
		RestartPolicy: task.RestartPolicy,
		Title:         task.Title,
		Description:   task.Description,
		InputData:     task.InputData,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := s.repos.Tasks.Create(newTask); err != nil {
		slog.Error("retry task", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create retry task")
		return
	}
	writeJSON(w, http.StatusCreated, newTask)
}

func (s *Server) handleCancelTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	task, err := s.repos.Tasks.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if isTerminalTaskStatus(task.Status) {
		writeJSON(w, http.StatusOK, task)
		return
	}
	if err := s.repos.Tasks.UpdateStatus(id, domain.TaskStatusCancelled); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to cancel task")
		return
	}
	task.Status = domain.TaskStatusCancelled
	writeJSON(w, http.StatusOK, task)
}

func isTerminalTaskStatus(status domain.TaskStatus) bool {
	return status == domain.TaskStatusCompleted || status == domain.TaskStatusCancelled || status == domain.TaskStatusFailed || status == domain.TaskStatusEvicted
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	agents, err := s.repos.AgentInstances.ListAll()
	if err != nil {
		slog.Error("list agents", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list agents")
		return
	}
	if agents == nil {
		agents = []*domain.AgentInstance{}
	}
	writeJSON(w, http.StatusOK, agents)
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	agent, err := s.repos.AgentInstances.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get agent")
		return
	}
	if agent == nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (s *Server) handlePauseAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	agent, err := s.repos.AgentInstances.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get agent")
		return
	}
	if agent == nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	if agent.Status != domain.AgentStatusRunning {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("cannot pause agent in %s state", agent.Status))
		return
	}
	if err := s.repos.AgentInstances.UpdateStatus(id, domain.AgentStatusPaused); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to pause agent")
		return
	}
	agent.Status = domain.AgentStatusPaused
	writeJSON(w, http.StatusOK, agent)
}

func (s *Server) handleResumeAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	agent, err := s.repos.AgentInstances.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get agent")
		return
	}
	if agent == nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	if agent.Status != domain.AgentStatusPaused {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("cannot resume agent in %s state", agent.Status))
		return
	}
	if err := s.repos.AgentInstances.UpdateStatus(id, domain.AgentStatusRunning); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resume agent")
		return
	}
	agent.Status = domain.AgentStatusRunning
	writeJSON(w, http.StatusOK, agent)
}

func (s *Server) handleStopAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	agent, err := s.repos.AgentInstances.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get agent")
		return
	}
	if agent == nil {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	if agent.Status == domain.AgentStatusStopped {
		writeJSON(w, http.StatusOK, agent)
		return
	}

	if agent.TmuxSession != "" {
		if err := s.termMgr.KillSession(r.Context(), agent.TmuxSession); err != nil {
			slog.Warn("kill tmux session on agent stop", "agent_id", id, "session", agent.TmuxSession, "error", err)
		}
	}

	if err := s.repos.AgentInstances.UpdateStatus(id, domain.AgentStatusStopped); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to stop agent")
		return
	}
	agent.Status = domain.AgentStatusStopped
	writeJSON(w, http.StatusOK, agent)
}

func (s *Server) handleListTerminals(w http.ResponseWriter, r *http.Request) {
	terminals, err := s.repos.TerminalSessions.ListAll()
	if err != nil {
		slog.Error("list terminals", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list terminals")
		return
	}
	if terminals == nil {
		terminals = []*domain.TerminalSession{}
	}
	writeJSON(w, http.StatusOK, terminals)
}

func (s *Server) handleCreateTerminal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID string `json:"task_id"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TaskID == "" {
		writeError(w, http.StatusBadRequest, "task_id is required")
		return
	}

	sessionName := fmt.Sprintf("ai-dev-%s", uuid.New().String()[:8])
	ts := &domain.TerminalSession{
		ID:          uuid.New().String(),
		TaskID:      req.TaskID,
		TmuxSession: sessionName,
		Status:      domain.TerminalStatusActive,
		LogFilePath: fmt.Sprintf("%s/%s.log", s.cfg.Runtime.LogRoot, sessionName),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		slog.Error("create tmux session", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create tmux session")
		return
	}

	if err := s.repos.TerminalSessions.Create(ts); err != nil {
		slog.Error("save terminal session", "error", err)
		_ = exec.Command("tmux", "kill-session", "-t", sessionName).Run()
		writeError(w, http.StatusInternalServerError, "failed to save terminal session")
		return
	}
	writeJSON(w, http.StatusCreated, ts)
}

func (s *Server) handleGetTerminal(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ts, err := s.repos.TerminalSessions.GetByID(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get terminal")
		return
	}
	if ts == nil {
		writeError(w, http.StatusNotFound, "terminal not found")
		return
	}
	writeJSON(w, http.StatusOK, ts)
}

func (s *Server) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ts, err := s.repos.TerminalSessions.GetByID(id)
	if err != nil || ts == nil {
		writeError(w, http.StatusNotFound, "terminal not found")
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade", "error", err)
		return
	}
	defer conn.Close()

	output, _ := s.termMgr.CapturePane(r.Context(), ts.TmuxSession)
	if output != "" {
		conn.WriteJSON(map[string]string{"type": "output", "data": output})
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		var lastOutput string
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				out, err := s.termMgr.CapturePane(ctx, ts.TmuxSession)
				if err != nil {
					continue
				}
				if out != lastOutput {
					lastOutput = out
					conn.WriteJSON(map[string]string{"type": "output", "data": out})
				}
			}
		}
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var req struct {
			Type string `json:"type"`
			Data string `json:"data"`
		}
		if json.Unmarshal(msg, &req) == nil && req.Type == "input" {
			_ = s.termMgr.SendKeys(r.Context(), ts.TmuxSession, req.Data)
		}
	}
}

func (s *Server) handleSystemMetrics(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.repos.ResourceSnapshots.Latest()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get metrics")
		return
	}
	if snapshot == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"memory_percent": 0,
			"cpu_percent":    0,
			"disk_percent":   0,
			"active_agents":  0,
			"pressure_level": "normal",
		})
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	snapshot, _ := s.repos.ResourceSnapshots.Latest()

	tmuxSessions, err := exec.Command("tmux", "list-sessions").Output()
	tmuxList := "no active tmux sessions"
	if err == nil {
		tmuxList = strings.TrimSpace(string(tmuxSessions))
	}

	activeAgents := []string{}
	active, err := s.repos.AgentInstances.ListActiveWithTasks()
	if err == nil {
		for _, a := range active {
			id := a.Agent.ID
			if len(id) >= 8 {
				id = id[:8]
			}
			activeAgents = append(activeAgents, fmt.Sprintf("%s (%s)", id, a.Agent.Status))
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"snapshot":      snapshot,
		"tmux_sessions": tmuxList,
		"active_agents": activeAgents,
		"config": map[string]any{
			"max_concurrent_agents": s.cfg.Scheduler.MaxConcurrentAgents,
			"max_heavy_agents":      s.cfg.Scheduler.MaxHeavyAgents,
		},
	})
}

func (s *Server) handleListTaskSpecs(w http.ResponseWriter, r *http.Request) {
	specs, err := s.repos.TaskSpecs.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list task specs")
		return
	}
	if specs == nil {
		specs = []*domain.TaskSpec{}
	}
	writeJSON(w, http.StatusOK, specs)
}

func (s *Server) handleListAgentSpecs(w http.ResponseWriter, r *http.Request) {
	specs, err := s.repos.AgentSpecs.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list agent specs")
		return
	}
	if specs == nil {
		specs = []*domain.AgentSpec{}
	}
	writeJSON(w, http.StatusOK, specs)
}

func (s *Server) handleListWorkflowTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.repos.WorkflowTemplates.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list workflow templates")
		return
	}
	if templates == nil {
		templates = []*domain.WorkflowTemplate{}
	}
	writeJSON(w, http.StatusOK, templates)
}

func (s *Server) handleCreateWorkflowTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		NodesJSON   string `json:"nodes_json"`
		EdgesJSON   string `json:"edges_json"`
		OnFailure   string `json:"on_failure"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	wt := &domain.WorkflowTemplate{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		NodesJSON:   req.NodesJSON,
		EdgesJSON:   req.EdgesJSON,
		OnFailure:   req.OnFailure,
	}
	if wt.OnFailure == "" {
		wt.OnFailure = "abort"
	}
	if wt.NodesJSON == "" {
		wt.NodesJSON = "[]"
	}
	if wt.EdgesJSON == "" {
		wt.EdgesJSON = "[]"
	}
	if err := s.repos.WorkflowTemplates.Create(wt); err != nil {
		slog.Error("create workflow template", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create workflow template")
		return
	}
	writeJSON(w, http.StatusCreated, wt)
}
