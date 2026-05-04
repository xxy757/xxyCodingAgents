// Package orchestrator 实现运行编排器，负责任务运行的创建、工作流实例化、
// 任务完成/失败处理、工作区克隆以及依赖任务解除阻塞等核心编排逻辑。
package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xxy757/xxyCodingAgents/internal/domain"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
	"github.com/xxy757/xxyCodingAgents/internal/workspace"
)

// Orchestrator 是运行编排器，协调运行（Run）和任务（Task）的生命周期。
type Orchestrator struct {
	repos      *storage.Repos
	gitManager *workspace.GitManager
}

// NewOrchestrator 创建编排器实例。
func NewOrchestrator(repos *storage.Repos, gitMgr *workspace.GitManager) *Orchestrator {
	return &Orchestrator{repos: repos, gitManager: gitMgr}
}

// CreateRun 创建一个新的运行实例。如果指定了工作流模板，会自动实例化其中的任务。
func (o *Orchestrator) CreateRun(ctx context.Context, projectID, templateID, title, description string) (*domain.Run, error) {
	run := &domain.Run{
		ID:                 uuid.New().String(),
		ProjectID:          projectID,
		WorkflowTemplateID: templateID,
		Title:              title,
		Description:        description,
		Status:             domain.RunStatusPending,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}
	if err := o.repos.Runs.Create(run); err != nil {
		return nil, err
	}

	// 如果关联了工作流模板，实例化模板中的所有任务
	if templateID != "" {
		if err := o.instantiateWorkflow(ctx, run); err != nil {
			slog.Error("instantiate workflow", "run_id", run.ID, "error", err)
		}
	}

	o.emitEvent(ctx, run.ID, nil, nil, domain.EventTypeTaskStarted, "run created")
	return run, nil
}

// SimpleRunResult 封装 CreateSimpleRun 的返回结果。
type SimpleRunResult struct {
	Run      *domain.Run
	Task     *domain.Task
	Warnings []string // 非致命警告（如工作区克隆失败）
}

// CreateSimpleRun 创建一个简单运行，只包含单个任务，不需要工作流模板。
// 用于 Prompt Composer 场景：用户确认草稿后直接创建 Run + Task。
// Task.InputData 取 final_prompt 而非 original_input。
func (o *Orchestrator) CreateSimpleRun(ctx context.Context, projectID, title, description, inputData, taskType string) (*SimpleRunResult, error) {
	run := &domain.Run{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Title:     title,
		Status:    domain.RunStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := o.repos.Runs.Create(run); err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}

	// 为运行创建工作区（克隆项目仓库）
	var workspacePath string
	var warnings []string
	project, _ := o.repos.Projects.GetByID(projectID)
	if project != nil && project.RepoURL != "" && o.gitManager != nil {
		wsDir, err := o.gitManager.CreateWorkspace(ctx, run.ID)
		if err == nil {
			if err := o.gitManager.Clone(ctx, project.RepoURL, wsDir); err != nil {
				warnings = append(warnings, fmt.Sprintf("仓库克隆失败，任务将在无代码上下文中运行: %v", err))
				slog.Warn("clone repo for simple run, continuing with empty workspace", "run_id", run.ID, "error", err)
			} else {
				workspacePath = wsDir
			}
		} else {
			warnings = append(warnings, fmt.Sprintf("创建工作区失败: %v", err))
		}
	}

	task := &domain.Task{
		ID:            uuid.New().String(),
		RunID:         run.ID,
		TaskType:      taskType,
		AttemptNo:     1,
		Status:        domain.TaskStatusQueued,
		Priority:      domain.PriorityNormal,
		QueueStatus:   string(domain.TaskStatusQueued),
		ResourceClass: domain.ResourceClassMedium,
		Preemptible:   true,
		RestartPolicy: "never",
		Title:         title,
		Description:   description,
		InputData:     inputData,
		WorkspacePath: workspacePath,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := o.repos.Tasks.Create(task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	o.emitEvent(ctx, run.ID, &task.ID, nil, domain.EventTypeTaskStarted, "simple run created")
	return &SimpleRunResult{Run: run, Task: task, Warnings: warnings}, nil
}

// instantiateWorkflow 根据工作流模板实例化任务节点和依赖关系。
// 会检测工作流中的环，发现环时清除所有边以避免死锁。
func (o *Orchestrator) instantiateWorkflow(ctx context.Context, run *domain.Run) error {
	template, err := o.repos.WorkflowTemplates.GetByID(run.WorkflowTemplateID)
	if err != nil {
		return err
	}
	if template == nil {
		return fmt.Errorf("workflow template not found: %s", run.WorkflowTemplateID)
	}

	// 解析节点和边
	var nodes []domain.WorkflowNode
	if err := json.Unmarshal([]byte(template.NodesJSON), &nodes); err != nil {
		return err
	}

	var edges []domain.WorkflowEdge
	if template.EdgesJSON != "" {
		if err := json.Unmarshal([]byte(template.EdgesJSON), &edges); err != nil {
			slog.Warn("parse edges json", "error", err)
		}
	}

	// 构建依赖映射：目标节点 -> 其依赖的源节点列表
	dependencyMap := make(map[string][]string)
	for _, edge := range edges {
		dependencyMap[edge.To] = append(dependencyMap[edge.To], edge.From)
	}

	// 使用三色 DFS 检测工作流中是否存在环
	// 白色(0): 未访问, 灰色(1): 当前 DFS 路径中, 黑色(2): 已完成探索
	color := make(map[string]int) // 默认值 0 = 白色
	var dfs func(nodeID string) bool
	dfs = func(nodeID string) bool {
		color[nodeID] = 1 // 标记为灰色（进入当前路径）
		for _, dep := range dependencyMap[nodeID] {
			if color[dep] == 1 {
				return true // 遇到灰色节点，发现环
			}
			if color[dep] == 0 && dfs(dep) {
				return true
			}
		}
		color[nodeID] = 2 // 标记为黑色（探索完成）
		return false
	}
	for _, node := range nodes {
		if color[node.ID] == 0 && dfs(node.ID) {
			slog.Warn("cycle detected in workflow edges", "run_id", run.ID)
			// 发现环时清除所有边，降级为无依赖模式
			edges = nil
			dependencyMap = make(map[string][]string)
			break
		}
	}

	// 为运行创建工作区（克隆项目仓库）
	var workspacePath string
	project, _ := o.repos.Projects.GetByID(run.ProjectID)
	if project != nil && project.RepoURL != "" && o.gitManager != nil {
		wsDir, err := o.gitManager.CreateWorkspace(ctx, run.ID)
		if err == nil {
			if err := o.gitManager.Clone(ctx, project.RepoURL, wsDir); err != nil {
				slog.Warn("clone repo for run, continuing with empty workspace", "run_id", run.ID, "repo", project.RepoURL, "error", err)
			} else {
				workspacePath = wsDir
				slog.Info("workspace cloned for run", "run_id", run.ID, "path", wsDir)
			}
		}
	}

	// 为每个节点创建任务实例；gate 节点不在此处创建，等 advanceFromNode 到达时再创建
	nodeTaskMap := make(map[string]string)
	for _, node := range nodes {
		if node.Kind == "gate" {
			// gate 节点延迟创建：当其前驱完成时由 advanceFromNode 触发
			continue
		}

		taskSpec, _ := o.repos.TaskSpecs.GetByID(node.TaskSpecID)

		// 确定任务的资源等级
		resourceClass := domain.ResourceClassLight
		if taskSpec != nil {
			resourceClass = taskSpec.ResourceClass
		}

		// 有依赖的任务初始为 blocked 状态，无依赖的为 queued
		hasDeps := len(dependencyMap[node.ID]) > 0
		initialStatus := domain.TaskStatusQueued
		if hasDeps {
			initialStatus = domain.TaskStatusBlocked
		}

		task := &domain.Task{
			ID:            uuid.New().String(),
			RunID:         run.ID,
			TaskSpecID:    node.TaskSpecID,
			TaskType:      node.Label,
			AttemptNo:     1,
			Status:        initialStatus,
			Priority:      domain.PriorityNormal,
			QueueStatus:   string(initialStatus),
			ResourceClass: resourceClass,
			Preemptible:   true,
			RestartPolicy: "never",
			Title:         node.Label,
			WorkspacePath: workspacePath,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := o.repos.Tasks.Create(task); err != nil {
			return err
		}

		nodeTaskMap[node.ID] = task.ID
		slog.Info("task created from template", "run_id", run.ID, "task_id", task.ID, "label", node.Label, "status", initialStatus, "workspace", workspacePath)
	}

	// 所有任务创建后，将运行状态更新为 running
	if err := o.repos.Runs.UpdateStatus(run.ID, domain.RunStatusRunning); err != nil {
		return err
	}

	return nil
}

// CompleteTask 标记任务为已完成，持久化输出数据，并推进工作流。
func (o *Orchestrator) CompleteTask(ctx context.Context, taskID string, outputData string) error {
	task, err := o.repos.Tasks.GetByID(taskID)
	if err != nil || task == nil {
		return err
	}

	now := time.Now()
	task.Status = domain.TaskStatusCompleted
	task.CompletedAt = &now
	task.OutputData = outputData
	if err := o.repos.Tasks.MarkCompleted(taskID, now); err != nil {
		return err
	}
	if outputData != "" {
		if err := o.repos.Tasks.UpdateOutput(taskID, outputData); err != nil {
			slog.Error("persist task output", "task_id", taskID, "error", err)
		}
	}

	o.emitEvent(ctx, task.RunID, &taskID, nil, domain.EventTypeTaskCompleted, "task completed")

	// 推进工作流：找到当前任务对应的节点，触发下游
	if err := o.advanceFromTask(ctx, task); err != nil {
		slog.Error("advance workflow from task", "task_id", taskID, "error", err)
	}

	// 检查运行中的所有任务和门禁是否都已结束
	if err := o.checkRunCompletion(ctx, task.RunID); err != nil {
		slog.Error("check run completion", "run_id", task.RunID, "error", err)
	}

	return nil
}

// FailTask 标记任务为失败。如果工作流模板的失败策略为 abort，则同时中止整个运行。
func (o *Orchestrator) FailTask(ctx context.Context, taskID, reason string) error {
	task, err := o.repos.Tasks.GetByID(taskID)
	if err != nil || task == nil {
		return err
	}

	if err := o.repos.Tasks.UpdateStatus(taskID, domain.TaskStatusFailed); err != nil {
		return err
	}

	o.emitEvent(ctx, task.RunID, &taskID, nil, domain.EventTypeTaskFailed, reason)

	// 检查工作流模板的失败策略
	if task.RunID != "" {
		run, _ := o.repos.Runs.GetByID(task.RunID)
		if run != nil && run.WorkflowTemplateID != "" {
			template, _ := o.repos.WorkflowTemplates.GetByID(run.WorkflowTemplateID)
			if template != nil && template.OnFailure == "abort" {
				o.repos.Runs.UpdateStatus(run.ID, domain.RunStatusFailed)
				o.emitEvent(ctx, run.ID, nil, nil, domain.EventTypeTaskFailed, "run aborted due to task failure")
			}
		}
	}

	return nil
}

// advanceFromTask 任务完成后推进工作流，找到对应节点并触发下游节点。
func (o *Orchestrator) advanceFromTask(ctx context.Context, task *domain.Task) error {
	run, err := o.repos.Runs.GetByID(task.RunID)
	if err != nil || run == nil || run.WorkflowTemplateID == "" {
		return err
	}

	template, err := o.repos.WorkflowTemplates.GetByID(run.WorkflowTemplateID)
	if err != nil || template == nil {
		return err
	}

	var nodes []domain.WorkflowNode
	if err := json.Unmarshal([]byte(template.NodesJSON), &nodes); err != nil {
		return err
	}

	var edges []domain.WorkflowEdge
	if template.EdgesJSON != "" {
		json.Unmarshal([]byte(template.EdgesJSON), &edges)
	}

	// 找到当前任务对应的节点 ID
	var taskNodeID string
	for _, node := range nodes {
		if node.TaskSpecID == task.TaskSpecID || node.Label == task.TaskType {
			taskNodeID = node.ID
			break
		}
	}
	if taskNodeID == "" {
		return nil
	}

	return o.advanceFromNode(ctx, run, nodes, edges, taskNodeID)
}

// advanceFromGate 门禁通过后推进工作流，触发下游节点。
func (o *Orchestrator) advanceFromGate(ctx context.Context, gate *domain.Gate) error {
	run, err := o.repos.Runs.GetByID(gate.RunID)
	if err != nil || run == nil || run.WorkflowTemplateID == "" {
		return err
	}

	template, err := o.repos.WorkflowTemplates.GetByID(run.WorkflowTemplateID)
	if err != nil || template == nil {
		return err
	}

	var nodes []domain.WorkflowNode
	if err := json.Unmarshal([]byte(template.NodesJSON), &nodes); err != nil {
		return err
	}

	var edges []domain.WorkflowEdge
	if template.EdgesJSON != "" {
		json.Unmarshal([]byte(template.EdgesJSON), &edges)
	}

	return o.advanceFromNode(ctx, run, nodes, edges, gate.NodeID)
}

// advanceFromNode 根据已完成的节点 ID，找到下游节点并激活。
func (o *Orchestrator) advanceFromNode(ctx context.Context, run *domain.Run, nodes []domain.WorkflowNode, edges []domain.WorkflowEdge, completedNodeID string) error {
	// 找到所有从 completedNodeID 出发的边
	for _, edge := range edges {
		if edge.From != completedNodeID {
			continue
		}

		// 找到下游节点
		var nextNode *domain.WorkflowNode
		for i := range nodes {
			if nodes[i].ID == edge.To {
				nextNode = &nodes[i]
				break
			}
		}
		if nextNode == nil {
			continue
		}

		if nextNode.Kind == "gate" {
			// 下游是门禁节点：先检查是否已存在，避免重复创建
			existingGates, _ := o.repos.Gates.ListByRun(run.ID)
			alreadyExists := false
			for _, g := range existingGates {
				if g.NodeID == nextNode.ID {
					alreadyExists = true
					break
				}
			}
			if alreadyExists {
				slog.Info("gate already exists for node, skipping", "node_id", nextNode.ID)
				continue
			}

			gate := &domain.Gate{
				ID:         uuid.New().String(),
				RunID:      run.ID,
				NodeID:     nextNode.ID,
				GateType:   domain.GateTypeAuto,
				Status:     domain.GateStatusPending,
				ConfigJSON: nextNode.Config,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
			if nextNode.Config != "" {
				var cfg domain.GateConfig
				if err := json.Unmarshal([]byte(nextNode.Config), &cfg); err == nil && cfg.Type != "" {
					gate.GateType = cfg.Type
				}
			}
			if err := o.repos.Gates.Create(gate); err != nil {
				slog.Error("create downstream gate", "node_id", nextNode.ID, "error", err)
				continue
			}
			slog.Info("downstream gate created", "gate_id", gate.ID, "node_id", nextNode.ID, "type", gate.GateType)

			// auto 类型立即评估
			if gate.GateType == domain.GateTypeAuto {
				go o.EvaluateGate(context.Background(), gate.ID)
			}
		} else {
			// 下游是任务节点：需要所有前驱节点都已完成才能解除 blocked
			if !o.allPredecessorsSatisfied(run.ID, nextNode.ID, nodes, edges) {
				slog.Info("task still blocked, not all predecessors satisfied", "node_id", nextNode.ID)
				continue
			}

			// 收集所有前驱任务的输出，构建下游任务的输入数据
			inputData := o.collectUpstreamOutputs(run.ID, nextNode.ID, nodes, edges)

			tasks, _ := o.repos.Tasks.ListByRun(run.ID)
			for _, t := range tasks {
				if t.TaskSpecID == nextNode.TaskSpecID && t.Status == domain.TaskStatusBlocked {
					// 写入上游输出到下游任务的 InputData
					if inputData != "" {
						if err := o.repos.Tasks.UpdateInputData(t.ID, inputData); err != nil {
							slog.Error("update downstream task input", "task_id", t.ID, "error", err)
						}
					}
					if err := o.repos.Tasks.UpdateStatus(t.ID, domain.TaskStatusQueued); err != nil {
						slog.Error("unblock downstream task", "task_id", t.ID, "error", err)
						continue
					}
					o.emitEvent(ctx, run.ID, &t.ID, nil, "task_unblocked", "task unblocked, all predecessors satisfied")
					slog.Info("task unblocked", "task_id", t.ID, "run_id", run.ID)
				}
			}
		}
	}

	return nil
}

// allPredecessorsSatisfied 检查指定节点的所有前驱是否都已完成或通过。
// 用于 DAG 中多入边场景，确保所有前驱都满足后才解除 blocked。
func (o *Orchestrator) allPredecessorsSatisfied(runID, nodeID string, nodes []domain.WorkflowNode, edges []domain.WorkflowEdge) bool {
	// 收集所有指向 nodeID 的前驱节点 ID
	var predecessorNodeIDs []string
	for _, edge := range edges {
		if edge.To == nodeID {
			predecessorNodeIDs = append(predecessorNodeIDs, edge.From)
		}
	}
	if len(predecessorNodeIDs) == 0 {
		return true
	}

	// 构建节点 ID -> kind 映射
	nodeKindMap := make(map[string]string)
	for _, n := range nodes {
		nodeKindMap[n.ID] = n.Kind
	}

	// 检查每个前驱节点的状态
	for _, predNodeID := range predecessorNodeIDs {
		kind := nodeKindMap[predNodeID]
		if kind == "gate" {
			// 前驱是 gate：查找该 gate 的状态
			gates, _ := o.repos.Gates.ListByRun(runID)
			found := false
			for _, g := range gates {
				if g.NodeID == predNodeID {
					if g.Status != domain.GateStatusPassed && g.Status != domain.GateStatusSkipped {
						return false
					}
					found = true
					break
				}
			}
			if !found {
				return false // gate 尚未创建
			}
		} else {
			// 前驱是 task：查找对应 task 的状态
			var predTaskSpecID string
			for _, n := range nodes {
				if n.ID == predNodeID {
					predTaskSpecID = n.TaskSpecID
					break
				}
			}
			tasks, _ := o.repos.Tasks.ListByRun(runID)
			found := false
			for _, t := range tasks {
				if t.TaskSpecID == predTaskSpecID {
					if t.Status != domain.TaskStatusCompleted {
						return false
					}
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

// collectUpstreamOutputs 收集指定节点的所有前驱任务的输出数据。
// 返回 JSON 数组字符串，格式: [{"from":"nodeID","output":...}, ...]
// 如果没有前驱输出，返回空字符串。
func (o *Orchestrator) collectUpstreamOutputs(runID, nodeID string, nodes []domain.WorkflowNode, edges []domain.WorkflowEdge) string {
	// 收集所有指向 nodeID 的前驱节点 ID
	var predecessorNodeIDs []string
	for _, edge := range edges {
		if edge.To == nodeID {
			predecessorNodeIDs = append(predecessorNodeIDs, edge.From)
		}
	}
	if len(predecessorNodeIDs) == 0 {
		return ""
	}

	// 构建节点 ID -> TaskSpecID 映射
	nodeSpecMap := make(map[string]string)
	for _, n := range nodes {
		nodeSpecMap[n.ID] = n.TaskSpecID
	}

	// 查询运行中的所有任务
	tasks, _ := o.repos.Tasks.ListByRun(runID)
	if len(tasks) == 0 {
		return ""
	}

	// 构建 TaskSpecID -> 最新已完成任务映射
	specTaskMap := make(map[string]*domain.Task)
	for _, t := range tasks {
		if t.Status == domain.TaskStatusCompleted && t.OutputData != "" {
			specTaskMap[t.TaskSpecID] = t
		}
	}

	// 收集前驱任务的输出
	var outputs []string
	for _, predNodeID := range predecessorNodeIDs {
		specID := nodeSpecMap[predNodeID]
		if specID == "" {
			continue
		}
		if task, ok := specTaskMap[specID]; ok {
			outputs = append(outputs, fmt.Sprintf(`{"from":"%s","output":%s}`, predNodeID, task.OutputData))
		}
	}

	if len(outputs) == 0 {
		return ""
	}
	return "[" + strings.Join(outputs, ",") + "]"
}

// EvaluateGate 评估一个门禁。auto 类型执行验证命令，manual/approval 类型等待用户操作。
func (o *Orchestrator) EvaluateGate(ctx context.Context, gateID string) error {
	gate, err := o.repos.Gates.GetByID(gateID)
	if err != nil || gate == nil {
		return err
	}

	if gate.Status != domain.GateStatusPending {
		return nil
	}

	var cfg domain.GateConfig
	if gate.ConfigJSON != "" {
		if err := json.Unmarshal([]byte(gate.ConfigJSON), &cfg); err != nil {
			slog.Error("parse gate config", "gate_id", gateID, "error", err)
		}
	}

	switch gate.GateType {
	case domain.GateTypeAuto:
		// 执行验证命令
		if cfg.VerifyCommand == "" {
			// 无验证命令，默认通过
			o.repos.Gates.UpdateVerifyResult(gateID, "no verify command configured", domain.GateStatusPassed)
			o.advanceFromGate(ctx, gate)
			return nil
		}

		// 在项目工作区中执行验证命令
		var workDir string
		run, _ := o.repos.Runs.GetByID(gate.RunID)
		if run != nil {
			project, _ := o.repos.Projects.GetByID(run.ProjectID)
			if project != nil {
				// 尝试使用运行的工作区
				tasks, _ := o.repos.Tasks.ListByRun(run.ID)
				for _, t := range tasks {
					if t.WorkspacePath != "" {
						workDir = t.WorkspacePath
						break
					}
				}
			}
		}

		cmd := exec.CommandContext(ctx, "sh", "-c", cfg.VerifyCommand)
		if workDir != "" {
			cmd.Dir = workDir
		}
		output, err := cmd.CombinedOutput()
		result := string(output)
		if len(result) > 4096 {
			result = result[:4096] // 截断过长输出
		}

		if err != nil {
			o.repos.Gates.UpdateVerifyResult(gateID, result, domain.GateStatusFailed)
			slog.Warn("gate auto verify failed", "gate_id", gateID, "command", cfg.VerifyCommand, "error", err)
			o.emitEvent(ctx, gate.RunID, nil, nil, domain.EventTypeTaskFailed, fmt.Sprintf("gate %s auto verify failed: %s", gateID, err))

			// 检查工作流失败策略，决定是否中止整个 run
			run, _ := o.repos.Runs.GetByID(gate.RunID)
			if run != nil && run.WorkflowTemplateID != "" {
				template, _ := o.repos.WorkflowTemplates.GetByID(run.WorkflowTemplateID)
				if template != nil && template.OnFailure == "abort" {
					o.repos.Runs.UpdateStatus(run.ID, domain.RunStatusFailed)
					o.emitEvent(ctx, run.ID, nil, nil, domain.EventTypeTaskFailed, "run aborted due to gate failure")
					slog.Warn("run aborted due to gate failure", "run_id", run.ID, "gate_id", gateID)
				}
			}
			return nil
		}

		o.repos.Gates.UpdateVerifyResult(gateID, result, domain.GateStatusPassed)
		slog.Info("gate auto verify passed", "gate_id", gateID, "command", cfg.VerifyCommand)
		o.advanceFromGate(ctx, gate)

		// gate 通过后检查 run 是否完成
		o.checkRunCompletion(ctx, gate.RunID)

	case domain.GateTypeManual, domain.GateTypeApproval:
		// 等待用户通过 API 操作，不做任何处理
		slog.Info("gate waiting for approval", "gate_id", gateID, "type", gate.GateType, "prompt", cfg.Prompt)
	}

	return nil
}

// ApproveGate 用户通过门禁，触发下游工作流推进。
func (o *Orchestrator) ApproveGate(ctx context.Context, gateID, approvedBy string) error {
	gate, err := o.repos.Gates.GetByID(gateID)
	if err != nil || gate == nil {
		return err
	}

	if gate.Status != domain.GateStatusPending {
		return fmt.Errorf("gate %s is not pending (status: %s)", gateID, gate.Status)
	}

	if err := o.repos.Gates.Approve(gateID, approvedBy); err != nil {
		return err
	}

	o.emitEvent(ctx, gate.RunID, nil, nil, "gate_approved", fmt.Sprintf("gate %s approved by %s", gateID, approvedBy))

	// 推进工作流
	gate.Status = domain.GateStatusPassed // 更新本地副本
	if err := o.advanceFromGate(ctx, gate); err != nil {
		return err
	}

	// gate 通过后检查 run 是否完成
	return o.checkRunCompletion(ctx, gate.RunID)
}

// checkRunCompletion 检查运行中的所有任务和门禁是否都已完成。
func (o *Orchestrator) checkRunCompletion(ctx context.Context, runID string) error {
	tasks, err := o.repos.Tasks.ListByRun(runID)
	if err != nil {
		return err
	}

	// 检查所有任务是否都已结束
	allTasksDone := true
	hasFailed := false
	for _, t := range tasks {
		switch t.Status {
		case domain.TaskStatusCompleted, domain.TaskStatusCancelled, domain.TaskStatusEvicted:
			// 已结束
		case domain.TaskStatusFailed:
			hasFailed = true
		default:
			allTasksDone = false
		}
	}

	if !allTasksDone {
		return nil
	}

	// 检查所有门禁是否都已通过或跳过
	gates, _ := o.repos.Gates.ListByRun(runID)
	allGatesDone := true
	for _, g := range gates {
		switch g.Status {
		case domain.GateStatusPassed, domain.GateStatusSkipped:
			// 已完成
		case domain.GateStatusFailed:
			hasFailed = true
		default:
			allGatesDone = false
		}
	}

	if !allGatesDone {
		return nil
	}

	// 所有任务和门禁都已结束，更新运行最终状态
	finalStatus := domain.RunStatusCompleted
	if hasFailed {
		finalStatus = domain.RunStatusFailed
	}
	if err := o.repos.Runs.UpdateStatus(runID, finalStatus); err != nil {
		return err
	}
	o.emitEvent(ctx, runID, nil, nil, domain.EventTypeTaskCompleted, "run completed")
	return nil
}

// emitEvent 创建并持久化一个系统事件。
func (o *Orchestrator) emitEvent(ctx context.Context, runID string, taskID, agentID *string, eventType domain.EventType, message string) {
	event := &domain.Event{
		ID:        uuid.New().String(),
		RunID:     runID,
		TaskID:    taskID,
		AgentID:   agentID,
		EventType: eventType,
		Message:   message,
		CreatedAt: time.Now(),
	}
	if err := o.repos.Events.Create(event); err != nil {
		slog.Error("create event", "error", err)
	}
}
