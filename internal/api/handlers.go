// Package api 的 handlers 文件实现所有 API 端点的请求处理逻辑。
// 涵盖项目、运行、任务、Agent、终端和系统的 CRUD 操作。
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
	"github.com/xxy757/xxyCodingAgents/internal/prompt"
)

// handleCreateProject 处理创建项目的请求，需要提供项目名称。
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

// handleListProjects 处理列出所有项目的请求。
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

// handleGetProject 处理根据 ID 获取单个项目的请求。
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

// handleCreateRun 处理创建运行的请求，需要关联已有项目并提供标题。
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

	// 验证关联的项目是否存在
	project, err := s.repos.Projects.GetByID(req.ProjectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify project")
		return
	}
	if project == nil {
		writeError(w, http.StatusBadRequest, "project not found")
		return
	}

	// 通过编排器创建运行，可能实例化工作流
	run, err := s.orch.CreateRun(r.Context(), req.ProjectID, req.TemplateID, req.Title, req.Description)
	if err != nil {
		slog.Error("create run", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create run")
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

// handleListAllRuns 处理列出所有运行的请求。
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

// handleGetRun 处理根据 ID 获取单个运行的请求。
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

// handleGetRunTimeline 处理获取运行事件时间线的请求。
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

// handleListRuns 处理列出指定项目下所有运行的请求。
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

// handleListTasks 处理列出指定运行下所有任务的请求。
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

// handleRunWorkflow 处理获取运行工作流图的请求，返回 ReactFlow 兼容的图数据。
func (s *Server) handleRunWorkflow(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	run, err := s.repos.Runs.GetByID(runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get run")
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	tasks, _ := s.repos.Tasks.ListByRun(runID)
	if tasks == nil {
		tasks = []*domain.Task{}
	}

	// 加载门禁数据
	gates, _ := s.repos.Gates.ListByRun(runID)

	// 定义 ReactFlow 兼容的图数据结构
	type NodeData struct {
		Label    string `json:"label"`
		Status   string `json:"status"`
		TaskType string `json:"task_type"`
		TaskID   string `json:"task_id"`
		GateID   string `json:"gate_id,omitempty"`
		GateType string `json:"gate_type,omitempty"`
	}
	type EdgeData struct {
		ID     string `json:"id"`
		Source string `json:"source"`
		Target string `json:"target"`
	}
	type WorkflowGraph struct {
		Nodes []struct {
			ID       string   `json:"id"`
			Type     string   `json:"type"`
			Data     NodeData `json:"data"`
			Position struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
			} `json:"position"`
		} `json:"nodes"`
		Edges []EdgeData `json:"edges"`
	}

	graph := WorkflowGraph{}

	// 如果运行关联了工作流模板，按模板构建图
	if run.WorkflowTemplateID != "" {
		template, _ := s.repos.WorkflowTemplates.GetByID(run.WorkflowTemplateID)
		if template != nil {
			var nodes []domain.WorkflowNode
			json.Unmarshal([]byte(template.NodesJSON), &nodes)
			var edges []domain.WorkflowEdge
			if template.EdgesJSON != "" {
				json.Unmarshal([]byte(template.EdgesJSON), &edges)
			}

			// 建立节点到任务状态和 ID 的映射
			nodeStatusMap := make(map[string]string)
			nodeTaskIDMap := make(map[string]string)
			for _, t := range tasks {
				for _, n := range nodes {
					if t.TaskSpecID == n.TaskSpecID {
						nodeStatusMap[n.ID] = string(t.Status)
						nodeTaskIDMap[n.ID] = t.ID
					}
				}
			}

			// 建立节点到门禁状态和 ID 的映射
			nodeGateStatusMap := make(map[string]string)
			nodeGateIDMap := make(map[string]string)
			nodeGateTypeMap := make(map[string]string)
			for _, g := range gates {
				nodeGateStatusMap[g.NodeID] = string(g.Status)
				nodeGateIDMap[g.NodeID] = g.ID
				nodeGateTypeMap[g.NodeID] = string(g.GateType)
			}

			// 构建节点数据
			for i, n := range nodes {
				nodeType := "task"
				if n.Kind == "gate" {
					nodeType = "gate"
				}

				// 根据节点类型选择状态来源
				status := nodeStatusMap[n.ID]
				if nodeType == "gate" {
					status = nodeGateStatusMap[n.ID]
				}

				graph.Nodes = append(graph.Nodes, struct {
					ID       string   `json:"id"`
					Type     string   `json:"type"`
					Data     NodeData `json:"data"`
					Position struct {
						X float64 `json:"x"`
						Y float64 `json:"y"`
					} `json:"position"`
				}{
					ID:   n.ID,
					Type: nodeType,
					Data: NodeData{
						Label:    n.Label,
						Status:   status,
						TaskType: n.Label,
						TaskID:   nodeTaskIDMap[n.ID],
						GateID:   nodeGateIDMap[n.ID],
						GateType: nodeGateTypeMap[n.ID],
					},
					Position: struct {
						X float64 `json:"x"`
						Y float64 `json:"y"`
					}{
						X: float64(i % 4) * 250,
						Y: float64(i/4) * 120,
					},
				})
			}

			// 构建边数据
			for i, e := range edges {
				graph.Edges = append(graph.Edges, EdgeData{
					ID:     fmt.Sprintf("edge-%d", i),
					Source: e.From,
					Target: e.To,
				})
			}
		}
	}

	// 如果没有模板或模板没有节点，则按任务列表构建简化图
	if len(graph.Nodes) == 0 {
		for i, t := range tasks {
			graph.Nodes = append(graph.Nodes, struct {
				ID       string   `json:"id"`
				Type     string   `json:"type"`
				Data     NodeData `json:"data"`
				Position struct {
					X float64 `json:"x"`
					Y float64 `json:"y"`
				} `json:"position"`
			}{
				ID:   t.ID,
				Type: "task",
				Data: NodeData{
					Label:    t.Title,
					Status:   string(t.Status),
					TaskType: t.TaskType,
					TaskID:   t.ID,
				},
				Position: struct {
					X float64 `json:"x"`
					Y float64 `json:"y"`
				}{
					X: float64(i % 4) * 250,
					Y: float64(i/4) * 120,
				},
			})
		}
	}

	writeJSON(w, http.StatusOK, graph)
}

// handleRetryTask 处理重试失败或已取消任务的请求，创建新的任务实例。
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
	// 只有失败或已取消的任务才能重试
	if task.Status != domain.TaskStatusFailed && task.Status != domain.TaskStatusCancelled {
		writeError(w, http.StatusBadRequest, "only failed or cancelled tasks can be retried")
		return
	}

	// 检查重试次数限制（最多 3 次）
	maxAttempt, err := s.repos.Tasks.MaxAttemptNo(task.RunID, task.TaskType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get max attempt")
		return
	}
	if maxAttempt >= 3 {
		writeError(w, http.StatusBadRequest, "max retry attempts exceeded")
		return
	}

	// 创建新的任务实例作为重试
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

// handleCancelTask 处理取消任务的请求。
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
	// 已经处于终态的任务直接返回
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

// isTerminalTaskStatus 判断任务状态是否为终态（不可再变更）。
func isTerminalTaskStatus(status domain.TaskStatus) bool {
	return status == domain.TaskStatusCompleted || status == domain.TaskStatusCancelled || status == domain.TaskStatusFailed || status == domain.TaskStatusEvicted
}

// handleListAgents 处理列出所有 Agent 实例的请求。
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

// handleGetAgent 处理根据 ID 获取单个 Agent 实例的请求。
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

// handlePauseAgent 处理暂停 Agent 的请求，仅运行中的 Agent 可暂停。
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
	if agent.Status == domain.AgentStatusPaused {
		writeJSON(w, http.StatusOK, agent)
		return
	}
	if agent.Status != domain.AgentStatusRunning {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("cannot pause agent in %s state", agent.Status))
		return
	}

	// 调用运行时暂停 Agent 进程
	rt := s.runtimeRegistry.GetOrDefault(agent.AgentKind)
	if err := rt.Pause(r.Context(), agent.TmuxSession); err != nil {
		slog.Warn("runtime pause agent", "agent_id", id, "error", err)
	}

	if err := s.repos.AgentInstances.UpdateStatus(id, domain.AgentStatusPaused); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to pause agent")
		return
	}
	agent.Status = domain.AgentStatusPaused
	writeJSON(w, http.StatusOK, agent)
}

// handleResumeAgent 处理恢复 Agent 的请求，仅已暂停的 Agent 可恢复。
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
	if agent.Status == domain.AgentStatusRunning {
		writeJSON(w, http.StatusOK, agent)
		return
	}
	if agent.Status != domain.AgentStatusPaused {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("cannot resume agent in %s state", agent.Status))
		return
	}

	// 调用运行时恢复 Agent 进程
	rt := s.runtimeRegistry.GetOrDefault(agent.AgentKind)
	if err := rt.Resume(r.Context(), agent.TmuxSession); err != nil {
		slog.Warn("runtime resume agent", "agent_id", id, "error", err)
	}

	if err := s.repos.AgentInstances.UpdateStatus(id, domain.AgentStatusRunning); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to resume agent")
		return
	}
	agent.Status = domain.AgentStatusRunning
	writeJSON(w, http.StatusOK, agent)
}

// handleStopAgent 处理停止 Agent 的请求，终止其 tmux 会话并更新状态。
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
	if agent.Status == domain.AgentStatusStopped || agent.Status == domain.AgentStatusFailed {
		writeJSON(w, http.StatusOK, agent)
		return
	}

	// 终止 Agent 的 tmux 会话
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

// handleListTerminals 处理列出所有终端会话的请求。
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

// handleCreateTerminal 处理创建终端会话的请求，创建 tmux 会话并持久化记录。
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

	// 创建 tmux 会话
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		slog.Error("create tmux session", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create tmux session")
		return
	}

	// 持久化终端会话记录，失败时清理 tmux 会话
	if err := s.repos.TerminalSessions.Create(ts); err != nil {
		slog.Error("save terminal session", "error", err)
		_ = exec.Command("tmux", "kill-session", "-t", sessionName).Run()
		writeError(w, http.StatusInternalServerError, "failed to save terminal session")
		return
	}
	writeJSON(w, http.StatusCreated, ts)
}

// handleGetTerminal 处理根据 ID 获取终端会话的请求。
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

// handleTerminalWS 处理终端 WebSocket 连接，实现双向终端交互。
func (s *Server) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ts, err := s.repos.TerminalSessions.GetByID(id)
	if err != nil || ts == nil {
		writeError(w, http.StatusNotFound, "terminal not found")
		return
	}

	// 升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade", "error", err)
		return
	}
	defer conn.Close()

	// 发送终端当前内容
	output, _ := s.termMgr.CapturePane(r.Context(), ts.TmuxSession)
	if output != "" {
		conn.WriteJSON(map[string]string{"type": "output", "data": output})
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// 后台 goroutine：定时轮询终端输出并推送增量更新
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
				// 仅在内容变化时发送更新
				if out != lastOutput {
					lastOutput = out
					conn.WriteJSON(map[string]string{"type": "output", "data": out})
				}
			}
		}
	}()

	// 主循环：读取客户端输入并转发到 tmux
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

// handleSystemMetrics 处理获取系统资源指标的请求。
func (s *Server) handleSystemMetrics(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.repos.ResourceSnapshots.Latest()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get metrics")
		return
	}
	if snapshot == nil {
		// 没有快照数据时返回零值
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

// handleDiagnostics 处理获取系统诊断信息的请求，包含资源快照、tmux 会话和活跃 Agent。
func (s *Server) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	snapshot, _ := s.repos.ResourceSnapshots.Latest()

	// 获取当前 tmux 会话列表
	tmuxSessions, err := exec.Command("tmux", "list-sessions").Output()
	tmuxList := "no active tmux sessions"
	if err == nil {
		tmuxList = strings.TrimSpace(string(tmuxSessions))
	}

	// 获取活跃 Agent 列表（带截断 ID）
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

// handleListTaskSpecs 处理列出所有任务规格的请求。
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

// handleListAgentSpecs 处理列出所有 Agent 规格的请求。
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

// handleListWorkflowTemplates 处理列出所有工作流模板的请求。
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

// handleCreateWorkflowTemplate 处理创建工作流模板的请求。
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
	// 设置默认值
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

// ==================== 提示词草稿 API ====================

// handleGeneratePromptDraft 处理生成提示词草稿的请求。
// 根据用户原始输入和任务类型，使用规则模板生成结构化草稿。
func (s *Server) handleGeneratePromptDraft(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID     string `json:"project_id"`
		OriginalInput string `json:"original_input"`
		TaskType      string `json:"task_type"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ProjectID == "" {
		writeError(w, http.StatusBadRequest, "project_id is required")
		return
	}
	if req.OriginalInput == "" {
		writeError(w, http.StatusBadRequest, "original_input is required")
		return
	}

	// 验证项目存在
	project, err := s.repos.Projects.GetByID(req.ProjectID)
	if err != nil {
		slog.Error("get project for prompt draft", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get project")
		return
	}
	if project == nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	// 使用规则模板生成结构化草稿
	generatedPrompt := prompt.GenerateDraft(req.OriginalInput, req.TaskType)

	// 推断实际使用的任务类型
	taskType := req.TaskType
	if taskType == "" {
		taskType = prompt.InferTaskType(req.OriginalInput)
	}

	draft := &domain.PromptDraft{
		ID:              uuid.New().String(),
		ProjectID:       req.ProjectID,
		OriginalInput:   req.OriginalInput,
		GeneratedPrompt: generatedPrompt,
		TaskType:        taskType,
		Status:          domain.PromptDraftStatusDraft,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if err := s.repos.PromptDrafts.Create(draft); err != nil {
		slog.Error("create prompt draft", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create prompt draft")
		return
	}
	writeJSON(w, http.StatusCreated, draft)
}

// handleUpdatePromptDraft 处理更新提示词草稿的请求。
// 用户编辑 final_prompt 后保存，仅 draft 状态的草稿可编辑。
func (s *Server) handleUpdatePromptDraft(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	draft, err := s.repos.PromptDrafts.GetByID(id)
	if err != nil {
		slog.Error("get prompt draft", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get prompt draft")
		return
	}
	if draft == nil {
		writeError(w, http.StatusNotFound, "prompt draft not found")
		return
	}
	if draft.Status != domain.PromptDraftStatusDraft {
		writeError(w, http.StatusBadRequest, "only draft status can be edited")
		return
	}

	var req struct {
		FinalPrompt string `json:"final_prompt"`
		TaskType    string `json:"task_type"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	draft.FinalPrompt = req.FinalPrompt
	if req.TaskType != "" {
		draft.TaskType = req.TaskType
	}
	draft.UpdatedAt = time.Now()

	if err := s.repos.PromptDrafts.Update(draft); err != nil {
		slog.Error("update prompt draft", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update prompt draft")
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

// handleSendPromptDraft 处理发送提示词草稿的请求。
// 将草稿状态更新为 confirmed，创建 Run 和 Task，然后更新为 sent。
// Task.InputData 取 final_prompt 而非 original_input。
func (s *Server) handleSendPromptDraft(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	draft, err := s.repos.PromptDrafts.GetByID(id)
	if err != nil {
		slog.Error("get prompt draft", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get prompt draft")
		return
	}
	if draft == nil {
		writeError(w, http.StatusNotFound, "prompt draft not found")
		return
	}
	if draft.Status != domain.PromptDraftStatusDraft {
		writeError(w, http.StatusBadRequest, "only draft status can be sent")
		return
	}

	// 确定最终使用的提示词：优先用 final_prompt，为空则用 generated_prompt
	finalPrompt := draft.FinalPrompt
	if strings.TrimSpace(finalPrompt) == "" {
		finalPrompt = draft.GeneratedPrompt
	}

	// 生成任务标题：取 final_prompt 首行或原始输入前 50 字符
	title := generateTaskTitle(draft.OriginalInput, finalPrompt)

	// 更新状态为 confirmed
	if err := s.repos.PromptDrafts.UpdateStatus(draft.ID, domain.PromptDraftStatusConfirmed); err != nil {
		slog.Error("confirm prompt draft", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to confirm prompt draft")
		return
	}

	// 创建 Run 和 Task
	run, task, err := s.orch.CreateSimpleRun(
		r.Context(),
		draft.ProjectID,
		title,
		draft.OriginalInput,
		finalPrompt,
		draft.TaskType,
	)
	if err != nil {
		slog.Error("create simple run from draft", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create run")
		return
	}

	// 更新状态为 sent
	if err := s.repos.PromptDrafts.UpdateStatus(draft.ID, domain.PromptDraftStatusSent); err != nil {
		slog.Error("mark draft as sent", "error", err)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"draft_id": draft.ID,
		"run_id":   run.ID,
		"task_id":  task.ID,
		"status":   "sent",
	})
}

// generateTaskTitle 从原始输入和最终提示词生成任务标题。
func generateTaskTitle(originalInput, finalPrompt string) string {
	// 优先从 finalPrompt 提取第一行非空内容
	for _, line := range strings.Split(finalPrompt, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "##") {
			if len(line) > 80 {
				return line[:80] + "..."
			}
			return line
		}
	}
	// 回退到原始输入前 50 字符
	input := strings.TrimSpace(originalInput)
	if len(input) > 50 {
		return input[:50] + "..."
	}
	return input
}

// handleListPromptDrafts 处理列出提示词草稿的请求。
// 支持按 project_id 过滤。
func (s *Server) handleListPromptDrafts(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		writeError(w, http.StatusBadRequest, "project_id query parameter is required")
		return
	}

	drafts, err := s.repos.PromptDrafts.ListByProject(projectID)
	if err != nil {
		slog.Error("list prompt drafts", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list prompt drafts")
		return
	}
	if drafts == nil {
		drafts = []*domain.PromptDraft{}
	}
	writeJSON(w, http.StatusOK, drafts)
}

// handleApproveGate 处理通过门禁的请求。
// POST /api/gates/{id}/approve
func (s *Server) handleApproveGate(w http.ResponseWriter, r *http.Request) {
	gateID := r.PathValue("id")
	if gateID == "" {
		writeError(w, http.StatusBadRequest, "gate id is required")
		return
	}

	var req struct {
		ApprovedBy string `json:"approved_by"`
	}
	readJSON(r, &req)
	if req.ApprovedBy == "" {
		req.ApprovedBy = "user"
	}

	if err := s.orch.ApproveGate(r.Context(), gateID, req.ApprovedBy); err != nil {
		slog.Error("approve gate", "gate_id", gateID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to approve gate: "+err.Error())
		return
	}

	gate, _ := s.repos.Gates.GetByID(gateID)
	writeJSON(w, http.StatusOK, gate)
}

// handleListGates 处理列出门禁的请求。
// 支持按 run_id 过滤。
func (s *Server) handleListGates(w http.ResponseWriter, r *http.Request) {
	runID := r.URL.Query().Get("run_id")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "run_id query parameter is required")
		return
	}

	gates, err := s.repos.Gates.ListByRun(runID)
	if err != nil {
		slog.Error("list gates", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list gates")
		return
	}
	if gates == nil {
		gates = []*domain.Gate{}
	}
	writeJSON(w, http.StatusOK, gates)
}

// handleGetGate 处理获取单个门禁的请求。
func (s *Server) handleGetGate(w http.ResponseWriter, r *http.Request) {
	gateID := r.PathValue("id")
	if gateID == "" {
		writeError(w, http.StatusBadRequest, "gate id is required")
		return
	}

	gate, err := s.repos.Gates.GetByID(gateID)
	if err != nil {
		slog.Error("get gate", "gate_id", gateID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get gate")
		return
	}
	if gate == nil {
		writeError(w, http.StatusNotFound, "gate not found")
		return
	}
	writeJSON(w, http.StatusOK, gate)
}
