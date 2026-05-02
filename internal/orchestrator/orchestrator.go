// Package orchestrator 实现运行编排器，负责任务运行的创建、工作流实例化、
// 任务完成/失败处理、工作区克隆以及依赖任务解除阻塞等核心编排逻辑。
package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

// instantiateWorkflow 根据工作流模板实例化任务节点和依赖关系。
// 会检测工作流中的环，发现环时清除所有边以避免死锁。
func (o *Orchestrator) instantiateWorkflow(ctx context.Context, run *domain.Run) error {
	template, err := o.repos.WorkflowTemplates.GetByID(run.WorkflowTemplateID)
	if err != nil || template == nil {
		return err
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

	// 使用 DFS 检测工作流中是否存在环
	visited := make(map[string]bool)
	var dfs func(nodeID string) bool
	dfs = func(nodeID string) bool {
		if visited[nodeID] {
			return false
		}
		visited[nodeID] = true
		for _, dep := range dependencyMap[nodeID] {
			if visited[dep] {
				return true // 发现环
			}
			if dfs(dep) {
				return true
			}
		}
		delete(visited, nodeID)
		return false
	}
	for _, node := range nodes {
		if dfs(node.ID) {
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

	// 为每个节点创建任务实例
	nodeTaskMap := make(map[string]string)
	for _, node := range nodes {
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

		// 收集上游任务的输出作为当前任务的输入
		var inputData string
		if hasDeps {
			var upstreamOutputs []string
			for _, depNodeID := range dependencyMap[node.ID] {
				if depTaskID, ok := nodeTaskMap[depNodeID]; ok {
					depTask, err := o.repos.Tasks.GetByID(depTaskID)
					if err == nil && depTask != nil && depTask.OutputData != "" {
						upstreamOutputs = append(upstreamOutputs,
							fmt.Sprintf(`{"from":"%s","output":%s}`, depNodeID, depTask.OutputData))
					}
				}
			}
			if len(upstreamOutputs) > 0 {
				inputData = "[" + strings.Join(upstreamOutputs, ",") + "]"
			}
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
			InputData:     inputData,
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

// UnblockDependentTasks 检查并解除因依赖被阻塞的任务。
// 当一个任务完成后，检查所有被它阻塞的任务是否可以解除阻塞。
func (o *Orchestrator) UnblockDependentTasks(ctx context.Context, completedTaskID string) error {
	tasks, err := o.repos.Tasks.ListByStatus(domain.TaskStatusBlocked)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		if task.TaskSpecID == "" {
			continue
		}

		deps := o.getTaskDependencies(task)
		if len(deps) == 0 {
			continue
		}

		// 检查所有依赖任务是否都已完成
		allMet := true
		for _, depTaskID := range deps {
			depTask, err := o.repos.Tasks.GetByID(depTaskID)
			if err != nil || depTask == nil {
				allMet = false
				break
			}
			if depTask.Status != domain.TaskStatusCompleted {
				allMet = false
				break
			}
		}

		if allMet {
			if err := o.repos.Tasks.UpdateStatus(task.ID, domain.TaskStatusQueued); err != nil {
				slog.Error("unblock task", "task_id", task.ID, "error", err)
				continue
			}
			o.emitEvent(ctx, task.RunID, &task.ID, nil, "task_unblocked", "task unblocked, all dependencies completed")
			slog.Info("task unblocked", "task_id", task.ID, "run_id", task.RunID)
		}
	}

	return nil
}

// getTaskDependencies 获取任务的所有前置依赖任务 ID。
// 通过查询运行关联的工作流模板，找到指向当前任务节点的所有边。
func (o *Orchestrator) getTaskDependencies(task *domain.Task) []string {
	if task.RunID == "" {
		return nil
	}

	run, err := o.repos.Runs.GetByID(task.RunID)
	if err != nil || run == nil || run.WorkflowTemplateID == "" {
		return nil
	}

	template, err := o.repos.WorkflowTemplates.GetByID(run.WorkflowTemplateID)
	if err != nil || template == nil || template.EdgesJSON == "" {
		return nil
	}

	var edges []domain.WorkflowEdge
	if err := json.Unmarshal([]byte(template.EdgesJSON), &edges); err != nil {
		return nil
	}

	var nodeID string
	var nodes []domain.WorkflowNode
	if err := json.Unmarshal([]byte(template.NodesJSON), &nodes); err != nil {
		return nil
	}

	// 建立 节点 <-> 任务 的双向映射
	nodeTaskMap := make(map[string]string)
	taskNodeMap := make(map[string]string)
	for _, n := range nodes {
		tasks, _ := o.repos.Tasks.ListByRun(run.ID)
		for _, t := range tasks {
			if t.TaskSpecID == n.TaskSpecID {
				nodeTaskMap[n.ID] = t.ID
				taskNodeMap[t.ID] = n.ID
			}
		}
	}

	// 找到当前任务对应的节点
	nodeID = taskNodeMap[task.ID]

	// 收集所有指向该节点的上游任务 ID
	var depTaskIDs []string
	for _, edge := range edges {
		if edge.To == nodeID {
			if taskID, ok := nodeTaskMap[edge.From]; ok {
				depTaskIDs = append(depTaskIDs, taskID)
			}
		}
	}

	return depTaskIDs
}

// CompleteTask 标记任务为已完成，并检查运行中的所有任务是否都已完成。
func (o *Orchestrator) CompleteTask(ctx context.Context, taskID string) error {
	task, err := o.repos.Tasks.GetByID(taskID)
	if err != nil || task == nil {
		return err
	}

	now := time.Now()
	task.Status = domain.TaskStatusCompleted
	task.CompletedAt = &now
	if err := o.repos.Tasks.MarkCompleted(taskID, now); err != nil {
		return err
	}

	o.emitEvent(ctx, task.RunID, &taskID, nil, domain.EventTypeTaskCompleted, "task completed")

	// 尝试解除依赖该任务的阻塞任务
	if err := o.UnblockDependentTasks(ctx, taskID); err != nil {
		slog.Error("unblock dependent tasks", "error", err)
	}

	// 检查运行中的所有任务是否都已结束
	tasks, err := o.repos.Tasks.ListByRun(task.RunID)
	if err != nil {
		return err
	}

	allDone := true
	hasFailed := false
	for _, t := range tasks {
		switch t.Status {
		case domain.TaskStatusCompleted:
		case domain.TaskStatusCancelled:
		case domain.TaskStatusFailed:
			hasFailed = true
		case domain.TaskStatusEvicted:
		default:
			allDone = false
		}
	}

	// 所有任务结束后，更新运行的最终状态
	if allDone {
		finalStatus := domain.RunStatusCompleted
		if hasFailed {
			finalStatus = domain.RunStatusFailed
		}
		if err := o.repos.Runs.UpdateStatus(task.RunID, finalStatus); err != nil {
			return err
		}
		o.emitEvent(ctx, task.RunID, nil, nil, domain.EventTypeTaskCompleted, "run completed")
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
