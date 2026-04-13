package orchestrator

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/xxy757/xxyCodingAgents/internal/domain"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
)

type Orchestrator struct {
	repos *storage.Repos
}

func NewOrchestrator(repos *storage.Repos) *Orchestrator {
	return &Orchestrator{repos: repos}
}

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

	if templateID != "" {
		if err := o.instantiateWorkflow(ctx, run); err != nil {
			slog.Error("instantiate workflow", "run_id", run.ID, "error", err)
		}
	}

	o.emitEvent(ctx, run.ID, nil, nil, domain.EventTypeTaskStarted, "run created")
	return run, nil
}

func (o *Orchestrator) instantiateWorkflow(ctx context.Context, run *domain.Run) error {
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
		if err := json.Unmarshal([]byte(template.EdgesJSON), &edges); err != nil {
			slog.Warn("parse edges json", "error", err)
		}
	}

	dependencyMap := make(map[string][]string)
	for _, edge := range edges {
		dependencyMap[edge.To] = append(dependencyMap[edge.To], edge.From)
	}

	visited := make(map[string]bool)
	var dfs func(nodeID string) bool
	dfs = func(nodeID string) bool {
		if visited[nodeID] {
			return false
		}
		visited[nodeID] = true
		for _, dep := range dependencyMap[nodeID] {
			if visited[dep] {
				return true
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
			edges = nil
			dependencyMap = make(map[string][]string)
			break
		}
	}

	nodeTaskMap := make(map[string]string)
	for _, node := range nodes {
		taskSpec, _ := o.repos.TaskSpecs.GetByID(node.TaskSpecID)

		resourceClass := domain.ResourceClassLight
		if taskSpec != nil {
			resourceClass = taskSpec.ResourceClass
		}

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
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := o.repos.Tasks.Create(task); err != nil {
			return err
		}

		nodeTaskMap[node.ID] = task.ID
		slog.Info("task created from template", "run_id", run.ID, "task_id", task.ID, "label", node.Label, "status", initialStatus)
	}

	if err := o.repos.Runs.UpdateStatus(run.ID, domain.RunStatusRunning); err != nil {
		return err
	}

	return nil
}

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

	nodeID = taskNodeMap[task.ID]

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

func (o *Orchestrator) CompleteTask(ctx context.Context, taskID string) error {
	task, err := o.repos.Tasks.GetByID(taskID)
	if err != nil || task == nil {
		return err
	}

	now := time.Now()
	task.Status = domain.TaskStatusCompleted
	task.CompletedAt = &now
	if err := o.repos.Tasks.UpdateStatus(taskID, domain.TaskStatusCompleted); err != nil {
		return err
	}

	o.emitEvent(ctx, task.RunID, &taskID, nil, domain.EventTypeTaskCompleted, "task completed")

	if err := o.UnblockDependentTasks(ctx, taskID); err != nil {
		slog.Error("unblock dependent tasks", "error", err)
	}

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

func (o *Orchestrator) FailTask(ctx context.Context, taskID, reason string) error {
	task, err := o.repos.Tasks.GetByID(taskID)
	if err != nil || task == nil {
		return err
	}

	if err := o.repos.Tasks.UpdateStatus(taskID, domain.TaskStatusFailed); err != nil {
		return err
	}

	o.emitEvent(ctx, task.RunID, &taskID, nil, domain.EventTypeTaskFailed, reason)

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
