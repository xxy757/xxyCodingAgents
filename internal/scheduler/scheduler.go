// Package scheduler 实现核心调度器，负责定时执行任务调度、资源监控、
// 负载保护（驱逐/暂停 Agent）、任务完成检测、超时强制、定期检查点、
// 日志清理和工作区大小检查。
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/xxy757/xxyCodingAgents/internal/agentlauncher"
	learningengine "github.com/xxy757/xxyCodingAgents/internal/learning"
	promptengine "github.com/xxy757/xxyCodingAgents/internal/prompt"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/xxy757/xxyCodingAgents/internal/config"
	"github.com/xxy757/xxyCodingAgents/internal/domain"
	agentruntime "github.com/xxy757/xxyCodingAgents/internal/runtime"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
	"github.com/xxy757/xxyCodingAgents/internal/terminal"
)

// Orchestrator 是调度器回调编排器所需的接口。
type Orchestrator interface {
	CompleteTask(ctx context.Context, taskID string, outputData string) error
	FailTask(ctx context.Context, taskID, reason string) error
}

// PressureLevel 表示系统资源压力等级。
type PressureLevel string

const (
	PressureNormal   PressureLevel = "normal"   // 正常，无限制
	PressureWarn     PressureLevel = "warn"     // 警告，仅调度轻量任务
	PressureHigh     PressureLevel = "high"     // 高压，暂停低优先级 Agent
	PressureCritical PressureLevel = "critical" // 临界，驱逐所有可抢占 Agent
)

// Scheduler 是核心调度器，定时执行任务调度和资源管理。
type Scheduler struct {
	cfg              *config.Config
	repos            *storage.Repos
	runtimeRegistry  *agentruntime.AdapterRegistry
	terminal         *terminal.Manager
	orch             Orchestrator
	newBrowseManager func(cliPath, workspacePath string) browseEnvManager
	promptBuilder    promptBuilder
	learningSearcher learningSearcher
	learningStore    *learningengine.Store
	qaCanaryMu       sync.RWMutex
	qaCanaries       map[string]string
	stop             chan struct{}
	tickCount        int64 // 调度周期计数器
}

type browseEnvManager interface {
	EnsureDaemon(ctx context.Context) error
	BuildEnv() map[string]string
}

type promptBuilder interface {
	BuildPrompt(opts promptengine.BuildOptions) (string, error)
}

type learningSearcher interface {
	SearchInsights(opts learningengine.SearchOptions) ([]string, error)
}

// NewScheduler 创建调度器实例。
func NewScheduler(cfg *config.Config, repos *storage.Repos, registry *agentruntime.AdapterRegistry, tm *terminal.Manager, orch Orchestrator) *Scheduler {
	return &Scheduler{
		cfg:             cfg,
		repos:           repos,
		runtimeRegistry: registry,
		terminal:        tm,
		orch:            orch,
		newBrowseManager: func(cliPath, workspacePath string) browseEnvManager {
			return agentruntime.NewBrowseManager(cliPath, workspacePath)
		},
		promptBuilder:    promptengine.NewEngine(promptTemplateDirFromConfig(cfg)),
		learningSearcher: learningengine.NewSearcher(learningsRootDirFromConfig(cfg)),
		learningStore:    learningengine.NewStore(learningsRootDirFromConfig(cfg)),
		qaCanaries:       make(map[string]string),
		stop:             make(chan struct{}),
	}
}

// Run 启动调度器的主循环，包含调度定时器和检查点定时器。
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.Scheduler.TickDuration())
	defer ticker.Stop()

	checkpointTicker := time.NewTicker(s.cfg.Timeouts.CheckpointInterval())
	defer checkpointTicker.Stop()

	slog.Info("scheduler started", "tick_seconds", s.cfg.Scheduler.TickSeconds)

	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler stopped")
			return
		case <-s.stop:
			slog.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.tick(ctx)
		case <-checkpointTicker.C:
			s.runCheckpoints(ctx)
		}
	}
}

// Stop 停止调度器。
func (s *Scheduler) Stop() {
	close(s.stop)
}

// tick 执行一次调度周期，包含资源监控、负载保护、任务调度、完成检测、超时检查和定期清理。
func (s *Scheduler) tick(ctx context.Context) {
	// 收集系统资源指标
	memPercent, cpuPercent, diskPercent := s.collectMetrics()
	activeAgents, _ := s.repos.AgentInstances.ListActiveWithTasks()
	activeCount := len(activeAgents)

	// 计算资源压力等级
	level := s.determinePressure(memPercent, diskPercent)

	// 保存资源快照
	snapshot := &domain.ResourceSnapshot{
		ID:            uuid.New().String(),
		MemoryPercent: memPercent,
		CPUPercent:    cpuPercent,
		DiskPercent:   diskPercent,
		ActiveAgents:  activeCount,
		PressureLevel: string(level),
		CreatedAt:     time.Now(),
	}
	if err := s.repos.ResourceSnapshots.Create(snapshot); err != nil {
		slog.Error("save resource snapshot", "error", err)
	}

	if level != PressureNormal {
		slog.Warn("resource pressure detected", "level", level, "memory", memPercent, "disk", diskPercent, "active_agents", activeCount)
	}

	// 检测任务完成和超时
	s.checkTaskCompletion(ctx, activeAgents)
	s.checkTaskTimeouts(ctx, activeAgents)

	// 执行负载保护（暂停或驱逐 Agent）
	s.handleLoadShedding(ctx, level, activeAgents)

	// 根据压力等级调度任务
	if level == PressureNormal {
		s.resumePausedAgents(ctx)
		s.scheduleTasks(ctx, activeCount, activeAgents)
	} else if level == PressureWarn {
		s.scheduleTasksLightOnly(ctx, activeCount, activeAgents)
	}

	s.tickCount++
	// 每 10 个周期持久化一次终端输出
	if s.tickCount%10 == 0 {
		s.persistTerminalOutputs(ctx, activeAgents)
	}
	// 每 100 个周期执行一次清理
	if s.tickCount%100 == 0 {
		s.cleanup(ctx)
	}

	slog.Debug("scheduler tick", "memory", memPercent, "cpu", cpuPercent, "disk", diskPercent, "agents", activeCount, "pressure", level)
}

// handleLoadShedding 根据压力等级执行负载保护策略。
func (s *Scheduler) handleLoadShedding(ctx context.Context, level PressureLevel, activeAgents []storage.ActiveAgentsResult) {
	switch level {
	case PressureHigh:
		s.pauseLowPriorityAgents(ctx, activeAgents)
	case PressureCritical:
		s.evictAgents(ctx, activeAgents)
	}
}

// pauseLowPriorityAgents 在高压时暂停低优先级且可抢占的 Agent。
func (s *Scheduler) pauseLowPriorityAgents(ctx context.Context, activeAgents []storage.ActiveAgentsResult) {
	for _, entry := range activeAgents {
		if entry.Agent.Status != domain.AgentStatusRunning {
			continue
		}
		if !entry.Task.Preemptible {
			continue
		}
		if entry.Task.Priority != domain.PriorityLow {
			continue
		}
		rt := s.runtimeRegistry.GetOrDefault(entry.Agent.AgentKind)
		if err := rt.Pause(ctx, entry.Agent.TmuxSession); err != nil {
			slog.Error("pause agent during load shedding", "agent_id", entry.Agent.ID, "error", err)
			continue
		}
		s.repos.AgentInstances.UpdateStatus(entry.Agent.ID, domain.AgentStatusPaused)
		s.repos.Tasks.UpdateStatus(entry.Task.ID, domain.TaskStatusEvicted)
		s.repos.Events.Create(&domain.Event{
			ID:        uuid.New().String(),
			RunID:     entry.Agent.RunID,
			TaskID:    ptrString(entry.Task.ID),
			AgentID:   ptrString(entry.Agent.ID),
			EventType: domain.EventTypeTaskEvicted,
			Message:   fmt.Sprintf("Agent %s paused due to HIGH resource pressure", entry.Agent.ID[:8]),
			Metadata:  fmt.Sprintf(`{"reason":"high_pressure","tmux_session":"%s"}`, entry.Agent.TmuxSession),
			CreatedAt: time.Now(),
		})
		slog.Info("agent paused due to high pressure", "agent_id", entry.Agent.ID, "task_id", entry.Task.ID)
	}
}

// resumePausedAgents 当压力恢复正常时，自动恢复所有暂停的 Agent。
func (s *Scheduler) resumePausedAgents(ctx context.Context) {
	agents, err := s.repos.AgentInstances.ListByStatus(domain.AgentStatusPaused)
	if err != nil || len(agents) == 0 {
		return
	}
	for _, agent := range agents {
		rt := s.runtimeRegistry.GetOrDefault(agent.AgentKind)
		// 检查 tmux 会话是否仍然存活
		inspect, err := rt.Inspect(ctx, agent.TmuxSession)
		if err != nil || !inspect.Running {
			s.repos.AgentInstances.UpdateStatus(agent.ID, domain.AgentStatusFailed)
			if agent.TaskID != "" {
				s.repos.Tasks.UpdateStatus(agent.TaskID, domain.TaskStatusFailed)
			}
			slog.Warn("paused agent tmux session gone, marking failed", "agent_id", agent.ID)
			continue
		}
		if err := rt.Resume(ctx, agent.TmuxSession); err != nil {
			slog.Error("resume paused agent", "agent_id", agent.ID, "error", err)
			continue
		}
		s.repos.AgentInstances.UpdateStatus(agent.ID, domain.AgentStatusRunning)
		if agent.TaskID != "" {
			s.repos.Tasks.UpdateStatus(agent.TaskID, domain.TaskStatusRunning)
		}
		s.repos.Events.Create(&domain.Event{
			ID:        uuid.New().String(),
			RunID:     agent.RunID,
			TaskID:    ptrString(agent.TaskID),
			AgentID:   ptrString(agent.ID),
			EventType: domain.EventTypeAgentResumed,
			Message:   fmt.Sprintf("Agent %s auto-resumed, pressure returned to normal", agent.ID[:8]),
			CreatedAt: time.Now(),
		})
		slog.Info("agent auto-resumed", "agent_id", agent.ID, "task_id", agent.TaskID)
	}
}

// evictAgents 在临界压力时先创建检查点再驱逐所有可抢占的 Agent。
func (s *Scheduler) evictAgents(ctx context.Context, activeAgents []storage.ActiveAgentsResult) {
	for _, entry := range activeAgents {
		if entry.Agent.Status != domain.AgentStatusRunning && entry.Agent.Status != domain.AgentStatusPaused {
			continue
		}
		if !entry.Task.Preemptible {
			continue
		}

		rt := s.runtimeRegistry.GetOrDefault(entry.Agent.AgentKind)

		// 驱逐前尝试创建检查点
		cp, err := rt.Checkpoint(ctx, entry.Agent.TmuxSession)
		if err != nil {
			slog.Warn("checkpoint before eviction failed", "agent_id", entry.Agent.ID, "error", err)
		}
		if cp != nil {
			s.repos.Checkpoints.Create(&domain.Checkpoint{
				ID:        uuid.New().String(),
				AgentID:   entry.Agent.ID,
				TaskID:    entry.Task.ID,
				RunID:     entry.Agent.RunID,
				Phase:     cp.Phase,
				StateData: cp.StateData,
				Reason:    "critical_eviction",
				CreatedAt: time.Now(),
			})
		}

		// 停止 Agent 并更新状态
		if err := rt.Stop(ctx, entry.Agent.TmuxSession); err != nil {
			slog.Error("stop agent during eviction", "agent_id", entry.Agent.ID, "error", err)
		}
		s.repos.AgentInstances.UpdateStatus(entry.Agent.ID, domain.AgentStatusStopped)
		s.repos.Tasks.UpdateStatus(entry.Task.ID, domain.TaskStatusEvicted)
		s.repos.Events.Create(&domain.Event{
			ID:        uuid.New().String(),
			RunID:     entry.Agent.RunID,
			TaskID:    ptrString(entry.Task.ID),
			AgentID:   ptrString(entry.Agent.ID),
			EventType: domain.EventTypeTaskEvicted,
			Message:   fmt.Sprintf("Agent %s evicted due to CRITICAL resource pressure", entry.Agent.ID[:8]),
			Metadata:  fmt.Sprintf(`{"reason":"critical_eviction","checkpoint_saved":%v}`, cp != nil),
			CreatedAt: time.Now(),
		})
		slog.Info("agent evicted due to critical pressure", "agent_id", entry.Agent.ID, "task_id", entry.Task.ID)
	}
}

// scheduleTasksLightOnly 在警告压力下仅调度轻量级任务。
func (s *Scheduler) scheduleTasksLightOnly(ctx context.Context, activeAgentCount int, activeAgents []storage.ActiveAgentsResult) {
	queuedTasks, err := s.repos.Tasks.ListByStatus(domain.TaskStatusQueued)
	if err != nil || len(queuedTasks) == 0 {
		return
	}

	occupiedQAWorkspaces := s.activeBrowseWorkspaces(activeAgents)
	for _, task := range queuedTasks {
		// 跳过重型任务
		if task.ResourceClass == domain.ResourceClassHeavy {
			slog.Info("skipping heavy task under WARN pressure", "task_id", task.ID)
			continue
		}
		if s.hasBrowseWorkspaceConflict(task, occupiedQAWorkspaces) {
			slog.Info("skipping browser qa task due to active workspace session", "task_id", task.ID, "workspace", task.WorkspacePath)
			continue
		}
		if !s.CanAdmit(activeAgentCount, task.ResourceClass) {
			break
		}
		if err := s.launchAgent(ctx, task); err != nil {
			slog.Error("launch agent for task", "task_id", task.ID, "error", err)
			continue
		}
		activeAgentCount++
		if isBrowseTask(task) && task.WorkspacePath != "" {
			occupiedQAWorkspaces[task.WorkspacePath] = struct{}{}
		}
	}
}

// runCheckpoints 为所有运行中的 Agent 创建定期检查点。
func (s *Scheduler) runCheckpoints(ctx context.Context) {
	agents, err := s.repos.AgentInstances.ListByStatus(domain.AgentStatusRunning)
	if err != nil || len(agents) == 0 {
		return
	}

	for _, agent := range agents {
		rt := s.runtimeRegistry.GetOrDefault(agent.AgentKind)
		cp, err := rt.Checkpoint(ctx, agent.TmuxSession)
		if err != nil {
			slog.Warn("periodic checkpoint failed", "agent_id", agent.ID, "error", err)
			continue
		}

		task, _ := s.repos.Tasks.GetByID(agent.TaskID)
		taskID := ""
		if task != nil {
			taskID = task.ID
		}

		checkpoint := &domain.Checkpoint{
			ID:        uuid.New().String(),
			AgentID:   agent.ID,
			TaskID:    taskID,
			RunID:     agent.RunID,
			Phase:     cp.Phase,
			StateData: cp.StateData,
			Reason:    "periodic",
			CreatedAt: time.Now(),
		}
		if err := s.repos.Checkpoints.Create(checkpoint); err != nil {
			slog.Error("save checkpoint", "agent_id", agent.ID, "error", err)
			continue
		}

		// 更新 Agent 的最新检查点引用
		s.repos.AgentInstances.UpdateCheckpointID(agent.ID, checkpoint.ID)
		slog.Debug("periodic checkpoint saved", "agent_id", agent.ID, "checkpoint_id", checkpoint.ID)
	}
}

// recoverFromCheckpoint 从检查点恢复任务，创建新的 Agent 实例并重新启动执行。
func (s *Scheduler) recoverFromCheckpoint(ctx context.Context, taskID string) error {
	task, err := s.repos.Tasks.GetByID(taskID)
	if err != nil || task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// 获取任务的检查点列表
	checkpoints, err := s.repos.Checkpoints.ListByTask(taskID)
	if err != nil || len(checkpoints) == 0 {
		return fmt.Errorf("no checkpoints found for task %s", taskID)
	}

	// 使用最新的检查点
	latest := checkpoints[0]
	agentKind := s.resolveAgentKind(task)
	rt := s.runtimeRegistry.GetOrDefault(agentKind)

	agentID := uuid.New().String()
	tmuxSession := fmt.Sprintf("agent-%s", agentID[:8])
	agentSpecID := s.resolveAgentSpecID(agentKind)

	// 创建恢复用的 Agent 实例
	agent := &domain.AgentInstance{
		ID:            agentID,
		RunID:         task.RunID,
		TaskID:        task.ID,
		AgentSpecID:   agentSpecID,
		AgentKind:     agentKind,
		Status:        domain.AgentStatusStarting,
		TmuxSession:   tmuxSession,
		WorkspacePath: task.WorkspacePath,
		CheckpointID:  ptrString(latest.ID),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := s.repos.AgentInstances.Create(agent); err != nil {
		return fmt.Errorf("create recovery agent: %w", err)
	}

	// 创建 tmux 会话
	if err := s.terminal.CreateSession(ctx, tmuxSession); err != nil {
		s.repos.AgentInstances.UpdateStatus(agentID, domain.AgentStatusFailed)
		return fmt.Errorf("create tmux session for recovery: %w", err)
	}

	launcherPath, err := s.buildLauncher(ctx, task, agentKind)
	if err != nil {
		s.terminal.KillSession(ctx, tmuxSession)
		s.repos.AgentInstances.UpdateStatus(agentID, domain.AgentStatusFailed)
		s.cleanupTaskArtifacts(task.ID)
		return fmt.Errorf("build launcher for recovery: %w", err)
	}

	// 启动 Agent 执行
	startReq := agentruntime.StartRequest{
		AgentID:       agentID,
		TaskID:        task.ID,
		RunID:         task.RunID,
		AgentKind:     agent.AgentKind,
		Command:       launcherPath,
		TmuxSession:   tmuxSession,
		WorkspacePath: task.WorkspacePath,
	}

	result, err := rt.Start(ctx, startReq)
	if err != nil {
		s.terminal.KillSession(ctx, tmuxSession)
		s.repos.AgentInstances.UpdateStatus(agentID, domain.AgentStatusFailed)
		s.cleanupTaskArtifacts(task.ID)
		return fmt.Errorf("start recovery agent: %w", err)
	}

	if result.PID > 0 {
		s.repos.AgentInstances.UpdatePID(agentID, result.PID)
	}

	now := time.Now()
	if err := s.repos.Tasks.MarkRunning(task.ID, now); err != nil {
		s.terminal.KillSession(ctx, tmuxSession)
		s.repos.AgentInstances.UpdateStatus(agentID, domain.AgentStatusFailed)
		s.cleanupTaskArtifacts(task.ID)
		return fmt.Errorf("mark recovered task running: %w", err)
	}

	s.repos.AgentInstances.UpdateStatus(agentID, domain.AgentStatusRunning)
	s.recordTerminalSession(task.ID, agentID, tmuxSession, result.TmuxPane, now)

	// 记录恢复事件
	s.repos.Events.Create(&domain.Event{
		ID:        uuid.New().String(),
		RunID:     task.RunID,
		TaskID:    ptrString(task.ID),
		AgentID:   ptrString(agentID),
		EventType: "agent_recovered",
		Message:   fmt.Sprintf("Agent recovered from checkpoint %s", latest.ID[:8]),
		Metadata:  fmt.Sprintf(`{"checkpoint_id":"%s","phase":"%s"}`, latest.ID, latest.Phase),
		CreatedAt: now,
	})

	slog.Info("agent recovered from checkpoint", "task_id", task.ID, "checkpoint_id", latest.ID)
	return nil
}

// checkTaskCompletion 检测运行中的任务是否已完成，通过解析终端输出中的完成标记。
func (s *Scheduler) checkTaskCompletion(ctx context.Context, activeAgents []storage.ActiveAgentsResult) {
	for _, entry := range activeAgents {
		if entry.Agent.Status != domain.AgentStatusRunning {
			continue
		}
		if entry.Agent.TmuxSession == "" {
			continue
		}

		// 捕获终端输出来判断任务是否完成
		output, err := s.terminal.CapturePane(ctx, entry.Agent.TmuxSession)
		if err != nil {
			continue
		}

		if leaked, reason := s.detectCanaryLeak(entry.Task, output); leaked {
			s.repos.AgentInstances.UpdateStatus(entry.Agent.ID, domain.AgentStatusFailed)
			s.repos.AgentInstances.UpdateLastOutputAt(entry.Agent.ID)
			if entry.Agent.TmuxSession != "" {
				rt := s.runtimeRegistry.GetOrDefault(entry.Agent.AgentKind)
				_ = rt.Stop(ctx, entry.Agent.TmuxSession)
			}
			if s.orch != nil {
				if err := s.orch.FailTask(ctx, entry.Task.ID, reason); err != nil {
					slog.Error("fail task on canary leak", "task_id", entry.Task.ID, "error", err)
				}
			}
			s.appendFailureLearning(entry.Task, reason)
			s.emitSecurityEvent(entry.Task, entry.Agent.ID, reason)
			s.clearTaskCanary(entry.Task.ID)
			s.cleanupTaskArtifacts(entry.Task.ID)
			slog.Warn("qa canary leak detected", "task_id", entry.Task.ID, "agent_id", entry.Agent.ID)
			continue
		}

		// 检测完成标记：[TASK_COMPLETED] 或 [TASK_FAILED]
		if strings.Contains(output, "[TASK_COMPLETED]") {
			s.repos.AgentInstances.UpdateStatus(entry.Agent.ID, domain.AgentStatusStopped)
			s.repos.AgentInstances.UpdateLastOutputAt(entry.Agent.ID)
			if s.orch != nil {
				if err := s.orch.CompleteTask(ctx, entry.Task.ID, ""); err != nil {
					slog.Error("complete task", "task_id", entry.Task.ID, "error", err)
				}
			}
			s.clearTaskCanary(entry.Task.ID)
			s.cleanupTaskArtifacts(entry.Task.ID)
			s.appendSuccessLearning(entry.Task)
			slog.Info("task completed (detected from output)", "task_id", entry.Task.ID, "agent_id", entry.Agent.ID)
		} else if strings.Contains(output, "[TASK_FAILED]") {
			s.repos.AgentInstances.UpdateStatus(entry.Agent.ID, domain.AgentStatusFailed)
			s.repos.AgentInstances.UpdateLastOutputAt(entry.Agent.ID)
			// 从输出中提取失败原因
			reason := "task failed (detected from output)"
			if idx := strings.Index(output, "[TASK_FAILED]"); idx >= 0 {
				end := strings.Index(output[idx:], "\n")
				if end > 0 && end < 200 {
					reason = strings.TrimSpace(output[idx : idx+end])
				}
			}
			if s.orch != nil {
				if err := s.orch.FailTask(ctx, entry.Task.ID, reason); err != nil {
					slog.Error("fail task", "task_id", entry.Task.ID, "error", err)
				}
			}
			s.appendFailureLearning(entry.Task, reason)
			s.clearTaskCanary(entry.Task.ID)
			s.cleanupTaskArtifacts(entry.Task.ID)
			slog.Info("task failed (detected from output)", "task_id", entry.Task.ID, "agent_id", entry.Agent.ID)
		}
	}
}

// checkTaskTimeouts 检查运行中的任务是否超时，超过 TaskSpec.TimeoutSeconds 则标记失败。
func (s *Scheduler) checkTaskTimeouts(ctx context.Context, activeAgents []storage.ActiveAgentsResult) {
	now := time.Now()
	for _, entry := range activeAgents {
		if entry.Task.Status != domain.TaskStatusRunning {
			continue
		}
		if entry.Task.StartedAt == nil {
			continue
		}

		// 获取 TaskSpec 中的超时配置
		timeoutSeconds := 0
		if entry.Task.TaskSpecID != "" {
			spec, err := s.repos.TaskSpecs.GetByID(entry.Task.TaskSpecID)
			if err == nil && spec != nil {
				timeoutSeconds = spec.TimeoutSeconds
			}
		}

		// 未配置超时则跳过
		if timeoutSeconds <= 0 {
			continue
		}

		elapsed := now.Sub(*entry.Task.StartedAt)
		if elapsed.Seconds() > float64(timeoutSeconds) {
			slog.Warn("task timeout exceeded", "task_id", entry.Task.ID, "elapsed", elapsed, "timeout_sec", timeoutSeconds)

			// 停止 Agent
			rt := s.runtimeRegistry.GetOrDefault(entry.Agent.AgentKind)
			if entry.Agent.TmuxSession != "" {
				if err := rt.Stop(ctx, entry.Agent.TmuxSession); err != nil {
					slog.Error("stop agent on timeout", "agent_id", entry.Agent.ID, "error", err)
				}
			}
			s.repos.AgentInstances.UpdateStatus(entry.Agent.ID, domain.AgentStatusFailed)

			// 标记任务失败
			reason := fmt.Sprintf("task timeout: exceeded %d seconds", timeoutSeconds)
			if s.orch != nil {
				if err := s.orch.FailTask(ctx, entry.Task.ID, reason); err != nil {
					slog.Error("fail task on timeout", "task_id", entry.Task.ID, "error", err)
				}
			}
			s.appendFailureLearning(entry.Task, reason)
			s.clearTaskCanary(entry.Task.ID)
			s.cleanupTaskArtifacts(entry.Task.ID)
		}
	}
}

// scheduleTasks 在正常压力下调度所有类型的排队任务。
func (s *Scheduler) scheduleTasks(ctx context.Context, activeAgentCount int, activeAgents []storage.ActiveAgentsResult) {
	queuedTasks, err := s.repos.Tasks.ListByStatus(domain.TaskStatusQueued)
	if err != nil || len(queuedTasks) == 0 {
		return
	}

	occupiedQAWorkspaces := s.activeBrowseWorkspaces(activeAgents)
	for _, task := range queuedTasks {
		if s.hasBrowseWorkspaceConflict(task, occupiedQAWorkspaces) {
			slog.Info("skipping browser qa task due to active workspace session", "task_id", task.ID, "workspace", task.WorkspacePath)
			continue
		}
		if !s.CanAdmit(activeAgentCount, task.ResourceClass) {
			break
		}
		if err := s.launchAgent(ctx, task); err != nil {
			slog.Error("launch agent for task", "task_id", task.ID, "error", err)
			continue
		}
		activeAgentCount++
		if isBrowseTask(task) && task.WorkspacePath != "" {
			occupiedQAWorkspaces[task.WorkspacePath] = struct{}{}
		}
		slog.Info("task admitted and agent launched", "task_id", task.ID, "resource_class", task.ResourceClass)
	}
}

// launchAgent 为任务创建并启动一个新的 Agent 实例。
func (s *Scheduler) launchAgent(ctx context.Context, task *domain.Task) error {
	// 更新任务状态为已接纳
	if err := s.repos.Tasks.UpdateStatus(task.ID, domain.TaskStatusAdmitted); err != nil {
		return fmt.Errorf("update task status to admitted: %w", err)
	}
	if err := s.repos.Tasks.UpdateQueueStatus(task.ID, "admitted"); err != nil {
		slog.Error("update queue status", "task_id", task.ID, "error", err)
	}

	// 根据 Task 的 TaskSpecID 查询 AgentSpec，确定使用哪种 Agent
	agentKind := s.resolveAgentKind(task)
	rt := s.runtimeRegistry.GetOrDefault(agentKind)

	agentID := uuid.New().String()
	tmuxSession := fmt.Sprintf("agent-%s", agentID[:8])

	agentSpecID := s.resolveAgentSpecID(agentKind)

	// 创建 Agent 实例记录
	agent := &domain.AgentInstance{
		ID:            agentID,
		RunID:         task.RunID,
		TaskID:        task.ID,
		AgentSpecID:   agentSpecID,
		AgentKind:     agentKind,
		Status:        domain.AgentStatusStarting,
		TmuxSession:   tmuxSession,
		WorkspacePath: task.WorkspacePath,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := s.repos.AgentInstances.Create(agent); err != nil {
		return fmt.Errorf("create agent instance: %w", err)
	}

	// 创建 tmux 会话
	if err := s.terminal.CreateSession(ctx, tmuxSession); err != nil {
		s.repos.AgentInstances.UpdateStatus(agentID, domain.AgentStatusFailed)
		s.repos.Tasks.UpdateStatus(task.ID, domain.TaskStatusFailed)
		s.appendFailureLearning(task, "create tmux session failed: "+err.Error())
		s.clearTaskCanary(task.ID)
		return fmt.Errorf("create tmux session: %w", err)
	}

	launcherPath, launcherErr := s.buildLauncher(ctx, task, agentKind)
	if launcherErr != nil {
		s.terminal.KillSession(ctx, tmuxSession)
		s.repos.AgentInstances.UpdateStatus(agentID, domain.AgentStatusFailed)
		s.repos.Tasks.UpdateStatus(task.ID, domain.TaskStatusFailed)
		s.appendFailureLearning(task, "build launcher script failed: "+launcherErr.Error())
		s.clearTaskCanary(task.ID)
		s.cleanupTaskArtifacts(task.ID)
		return fmt.Errorf("build launcher script: %w", launcherErr)
	}

	startReq := agentruntime.StartRequest{
		AgentID:       agentID,
		TaskID:        task.ID,
		RunID:         task.RunID,
		AgentKind:     agentKind,
		Command:       launcherPath,
		TmuxSession:   tmuxSession,
		WorkspacePath: task.WorkspacePath,
	}

	result, err := rt.Start(ctx, startReq)
	if err != nil {
		s.terminal.KillSession(ctx, tmuxSession)
		s.repos.AgentInstances.UpdateStatus(agentID, domain.AgentStatusFailed)
		s.repos.Tasks.UpdateStatus(task.ID, domain.TaskStatusFailed)
		s.appendFailureLearning(task, "start agent runtime failed: "+err.Error())
		s.clearTaskCanary(task.ID)
		s.cleanupTaskArtifacts(task.ID)
		return fmt.Errorf("start agent runtime: %w", err)
	}

	if result.PID > 0 {
		s.repos.AgentInstances.UpdatePID(agentID, result.PID)
	}

	now := time.Now()
	if err := s.repos.Tasks.MarkRunning(task.ID, now); err != nil {
		s.terminal.KillSession(ctx, tmuxSession)
		s.repos.AgentInstances.UpdateStatus(agentID, domain.AgentStatusFailed)
		s.repos.Tasks.UpdateStatus(task.ID, domain.TaskStatusFailed)
		s.appendFailureLearning(task, "mark task running failed: "+err.Error())
		s.clearTaskCanary(task.ID)
		s.cleanupTaskArtifacts(task.ID)
		return fmt.Errorf("mark task running: %w", err)
	}

	s.repos.AgentInstances.UpdateStatus(agentID, domain.AgentStatusRunning)
	s.recordTerminalSession(task.ID, agentID, tmuxSession, result.TmuxPane, now)

	// 记录 Agent 启动事件
	s.repos.Events.Create(&domain.Event{
		ID:        uuid.New().String(),
		RunID:     task.RunID,
		TaskID:    ptrString(task.ID),
		AgentID:   ptrString(agentID),
		EventType: "agent_started",
		Message:   fmt.Sprintf("Agent %s (%s) started for task %s", agentID[:8], agentKind, task.Title),
		Metadata:  fmt.Sprintf(`{"tmux_session":"%s","pid":%d,"agent_kind":"%s"}`, tmuxSession, result.PID, agentKind),
		CreatedAt: now,
	})

	return nil
}

// resolveAgentKind 根据任务类型和可用的 AgentSpec 确定使用哪种 Agent。
func (s *Scheduler) resolveAgentKind(task *domain.Task) string {
	// 首先检查 TaskSpec 指定的 runtime_type
	if task.TaskSpecID != "" {
		spec, err := s.repos.TaskSpecs.GetByID(task.TaskSpecID)
		if err == nil && spec != nil && spec.RuntimeType != "" {
			// 检查对应的 AgentSpec 是否存在
			if agentSpec, err := s.repos.AgentSpecs.GetByKind(spec.RuntimeType); err == nil && agentSpec != nil {
				return spec.RuntimeType
			}
		}
	}

	// 根据任务类型回退：think/plan → claude-code，其他 → generic-shell
	switch task.TaskType {
	case "think", "plan", "review", "retro":
		if _, err := s.repos.AgentSpecs.GetByKind("claude-code"); err == nil {
			return "claude-code"
		}
	}

	return "generic-shell"
}

func (s *Scheduler) resolveAgentSpecID(agentKind string) string {
	spec, err := s.repos.AgentSpecs.GetByKind(agentKind)
	if err != nil || spec == nil {
		return ""
	}
	return spec.ID
}

// resolveCommand 解析任务要执行的命令，优先使用 TaskSpec 中的命令模板。
func (s *Scheduler) resolveCommand(task *domain.Task) string {
	if task == nil {
		return "echo 'task started'"
	}
	// 尝试从 TaskSpec 获取命令模板
	if task.TaskSpecID != "" {
		spec, err := s.repos.TaskSpecs.GetByID(task.TaskSpecID)
		if err == nil && spec != nil && spec.CommandTemplate != "" {
			return spec.CommandTemplate
		}
	}
	// 回退到任务输入数据
	if task.InputData != "" {
		return task.InputData
	}
	shortID := strings.TrimSpace(task.ID)
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	if shortID == "" {
		shortID = "unknown"
	}
	return "echo 'task " + shortID + " started'"
}

func (s *Scheduler) buildLauncher(ctx context.Context, task *domain.Task, agentKind string) (string, error) {
	env, err := s.buildEnv(ctx, task)
	if err != nil {
		return "", err
	}

	cfg := agentlauncher.Config{
		TaskID:        task.ID,
		WorkspacePath: task.WorkspacePath,
		Env:           env,
		AgentKind:     agentKind,
		BaseDir:       s.cfg.AgentRuntime.BaseDir,
	}

	if agentKind == "claude-code" {
		cfg.PromptContent = s.buildPrompt(ctx, task, agentKind)
	} else {
		cfg.ShellCommand = s.resolveCommand(task)
	}

	return agentlauncher.Build(cfg)
}

func (s *Scheduler) cleanupTaskArtifacts(taskID string) {
	if taskID == "" {
		return
	}
	agentlauncher.Cleanup(s.cfg.AgentRuntime.BaseDir, taskID)
}

func (s *Scheduler) recordTerminalSession(taskID, agentID, tmuxSession, tmuxPane string, createdAt time.Time) {
	terminalSession := &domain.TerminalSession{
		ID:          uuid.New().String(),
		TaskID:      taskID,
		AgentID:     ptrString(agentID),
		TmuxSession: tmuxSession,
		TmuxPane:    tmuxPane,
		Status:      domain.TerminalStatusActive,
		LogFilePath: fmt.Sprintf("data/logs/%s.log", tmuxSession),
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
	if err := s.repos.TerminalSessions.Create(terminalSession); err != nil {
		slog.Error("create terminal session", "agent_id", agentID, "error", err)
	}
}

// cleanup 执行定期清理，包括过期数据删除、日志大小限制和工作区检查。
func (s *Scheduler) cleanup(ctx context.Context) {
	cutoff := time.Now().AddDate(0, 0, -s.cfg.Thresholds.LogRetentionDays)
	// 删除过期的资源快照
	if err := s.repos.ResourceSnapshots.DeleteOlderThan(cutoff); err != nil {
		slog.Error("cleanup resource snapshots", "error", err)
	}
	// 删除过期的事件
	if err := s.repos.Events.DeleteOlderThan(cutoff); err != nil {
		slog.Error("cleanup events", "error", err)
	}
	// 清理过期的日志文件
	if err := s.terminal.CleanupOldLogs(s.cfg.Thresholds.LogRetentionDays); err != nil {
		slog.Error("cleanup old log files", "error", err)
	}

	s.enforceLogSizeLimit()
	s.checkWorkspaceSizes(ctx)
	s.checkProcessTree(ctx)

	slog.Info("cleanup completed", "cutoff", cutoff)
}

// enforceLogSizeLimit 检查并强制执行日志总大小限制。
func (s *Scheduler) enforceLogSizeLimit() {
	limitMB := s.cfg.Thresholds.MaxTotalLogSizeMB
	if limitMB <= 0 {
		return
	}
	totalSize, err := s.terminal.TotalLogSize()
	if err != nil {
		slog.Error("check total log size", "error", err)
		return
	}
	totalMB := float64(totalSize) / 1024 / 1024
	if totalMB > float64(limitMB) {
		slog.Warn("total log size exceeds limit, cleaning oldest logs",
			"current_mb", fmt.Sprintf("%.1f", totalMB),
			"limit_mb", limitMB)
		// 清理一半保留期的日志以释放空间
		retention := s.cfg.Thresholds.LogRetentionDays
		if retention > 0 {
			halfRetention := retention / 2
			if halfRetention < 1 {
				halfRetention = 1
			}
			s.terminal.CleanupOldLogs(halfRetention)
		}
	}
}

// checkWorkspaceSizes 检查所有工作区的大小是否超过限制。
func (s *Scheduler) checkWorkspaceSizes(ctx context.Context) {
	maxMB := s.cfg.Thresholds.WorkspaceMaxSizeMB
	if maxMB <= 0 {
		return
	}

	workspaces, err := s.repos.Workspaces.ListActive()
	if err != nil {
		return
	}

	for _, ws := range workspaces {
		sizeMB, err := s.workspaceSize(ws.Path)
		if err != nil {
			continue
		}
		if sizeMB > float64(maxMB) {
			slog.Warn("workspace exceeds size limit",
				"workspace_id", ws.ID,
				"path", ws.Path,
				"size_mb", fmt.Sprintf("%.1f", sizeMB),
				"limit_mb", maxMB)
		}
	}
}

// workspaceSize 获取指定目录的总大小（MB）。
func (s *Scheduler) workspaceSize(path string) (float64, error) {
	out, err := exec.CommandContext(context.Background(), "du", "-sm", path).Output()
	if err != nil {
		return 0, err
	}
	var sizeMB float64
	fmt.Sscanf(string(out), "%f", &sizeMB)
	return sizeMB, nil
}

// checkProcessTree 检查活跃 Agent 的子进程数量是否异常。
func (s *Scheduler) checkProcessTree(ctx context.Context) {
	activeAgents, err := s.repos.AgentInstances.ListActiveWithTasks()
	if err != nil {
		return
	}

	for _, entry := range activeAgents {
		if entry.Agent.PID == nil || *entry.Agent.PID == 0 {
			continue
		}
		childCount := s.countChildProcesses(*entry.Agent.PID)
		if childCount > 50 {
			slog.Warn("agent has too many child processes",
				"agent_id", entry.Agent.ID,
				"pid", *entry.Agent.PID,
				"child_count", childCount)
		}
	}
}

// countChildProcesses 统计指定进程的子进程数量。
func (s *Scheduler) countChildProcesses(pid int) int {
	out, err := exec.CommandContext(context.Background(), "pgrep", "-P", fmt.Sprintf("%d", pid)).Output()
	if err != nil {
		return 0
	}
	lines := 0
	for _, c := range out {
		if c == '\n' {
			lines++
		}
	}
	return lines
}

// persistTerminalOutputs 将活跃 Agent 的终端输出持久化到日志文件。
func (s *Scheduler) persistTerminalOutputs(ctx context.Context, activeAgents []storage.ActiveAgentsResult) {
	for _, entry := range activeAgents {
		if entry.Agent.Status != domain.AgentStatusRunning {
			continue
		}
		if entry.Agent.TmuxSession == "" {
			continue
		}
		if err := s.terminal.CaptureAndPersist(ctx, entry.Agent.TmuxSession); err != nil {
			slog.Debug("persist terminal output", "agent_id", entry.Agent.ID, "error", err)
		}
	}
}

// collectMetrics 收集系统内存、CPU 和磁盘使用率。
func (s *Scheduler) collectMetrics() (float64, float64, float64) {
	var memPercent, cpuPercent, diskPercent float64

	if vmStat, err := mem.VirtualMemory(); err == nil {
		memPercent = vmStat.UsedPercent
	}

	if cpuPercents, err := cpu.Percent(0, false); err == nil && len(cpuPercents) > 0 {
		cpuPercent = cpuPercents[0]
	}

	if diskStat, err := disk.Usage("/"); err == nil {
		diskPercent = diskStat.UsedPercent
	}

	// macOS 下尝试获取 /Volumes 的磁盘使用率
	if runtime.GOOS == "darwin" {
		if diskStat, err := disk.Usage("/Volumes"); err == nil && diskPercent == 0 {
			diskPercent = diskStat.UsedPercent
		}
	}

	return memPercent, cpuPercent, diskPercent
}

// determinePressure 根据内存和磁盘使用率确定系统压力等级。
func (s *Scheduler) determinePressure(memPercent, diskPercent float64) PressureLevel {
	cfg := s.cfg.Thresholds

	if memPercent >= float64(cfg.CriticalMemoryPercent) || diskPercent >= float64(cfg.DiskHighPercent) {
		return PressureCritical
	}
	if memPercent >= float64(cfg.HighMemoryPercent) || diskPercent >= float64(cfg.DiskWarnPercent)+float64(cfg.DiskHighPercent-cfg.DiskWarnPercent)/2 {
		return PressureHigh
	}
	if memPercent >= float64(cfg.WarnMemoryPercent) || diskPercent >= float64(cfg.DiskWarnPercent) {
		return PressureWarn
	}
	return PressureNormal
}

// CanAdmit 判断是否可以接纳新任务，考虑并发限制和资源等级。
func (s *Scheduler) CanAdmit(activeCount int, resourceClass domain.ResourceClass) bool {
	cfg := s.cfg.Scheduler

	if activeCount >= cfg.MaxConcurrentAgents {
		return false
	}

	if resourceClass == domain.ResourceClassHeavy && activeCount >= cfg.MaxHeavyAgents {
		return false
	}

	return true
}

// ptrString 返回字符串的指针（用于可选字段）。
func ptrString(s string) *string {
	return &s
}

// buildPrompt 构建 LLM agent 的 prompt 内容。
// 优先走 PromptEngine（阶段模板 + 四层注入），失败时回退到 legacy 拼接逻辑。
func (s *Scheduler) buildPrompt(ctx context.Context, task *domain.Task, agentKind string) string {
	preparedTask := s.prepareTaskForPrompt(task)
	spec := s.getTaskSpec(preparedTask)

	builder := s.promptBuilder
	if builder == nil {
		builder = promptengine.NewEngine(promptTemplateDirFromConfig(s.cfg))
		s.promptBuilder = builder
	}

	if builder != nil && preparedTask != nil {
		runtimeState := s.collectPromptRuntimeState(ctx, preparedTask)
		learnings := s.collectPromptLearnings(preparedTask, spec)
		promptText, err := builder.BuildPrompt(promptengine.BuildOptions{
			Phase:     preparedTask.TaskType,
			Task:      preparedTask,
			AgentKind: agentKind,
			TaskSpec:  spec,
			Learnings: learnings,
			Runtime:   runtimeState,
		})
		if err == nil && strings.TrimSpace(promptText) != "" {
			return promptText
		}
		slog.Warn("build prompt with engine failed, fallback to legacy prompt", "task_id", preparedTask.ID, "task_type", preparedTask.TaskType, "error", err)
	}

	return s.buildPromptLegacy(preparedTask, spec)
}

func (s *Scheduler) prepareTaskForPrompt(task *domain.Task) *domain.Task {
	if task == nil {
		return nil
	}
	if !isBrowseTask(task) {
		return task
	}

	cloned := *task
	canary := promptengine.NewCanary()
	s.setTaskCanary(task.ID, canary)
	if strings.TrimSpace(cloned.InputData) != "" {
		cloned.InputData = promptengine.WrapUntrustedContent(cloned.InputData, canary)
	}
	rule := promptengine.QATrustBoundaryRule(canary)
	if strings.TrimSpace(cloned.Description) == "" {
		cloned.Description = rule
	} else {
		cloned.Description = strings.TrimSpace(cloned.Description) + "\n\n" + rule
	}
	return &cloned
}

func (s *Scheduler) collectPromptLearnings(task *domain.Task, spec *domain.TaskSpec) []string {
	if task == nil || s.learningSearcher == nil {
		return nil
	}

	queryParts := []string{
		task.TaskType,
		task.Title,
		task.Description,
		task.InputData,
	}
	if spec != nil {
		queryParts = append(queryParts, spec.RequiredInputs, spec.ExpectedOutputs)
	}
	projectSlug := s.resolveLearningProjectSlug(task)
	insights, err := s.learningSearcher.SearchInsights(learningengine.SearchOptions{
		ProjectSlug: projectSlug,
		Phase:       task.TaskType,
		QueryText:   strings.Join(queryParts, "\n"),
		Limit:       6,
	})
	if err != nil {
		slog.Warn("search learnings failed, skip layer-3 injection", "task_id", task.ID, "project_slug", projectSlug, "error", err)
		return nil
	}
	return insights
}

func (s *Scheduler) detectCanaryLeak(task *domain.Task, output string) (bool, string) {
	if task == nil || !isBrowseTask(task) {
		return false, ""
	}
	canary := s.getTaskCanary(task.ID)
	if canary == "" {
		return false, ""
	}
	if strings.Contains(output, canary) {
		return true, fmt.Sprintf("potential prompt injection: qa canary leaked (%s)", canary)
	}
	return false, ""
}

func (s *Scheduler) emitSecurityEvent(task *domain.Task, agentID, message string) {
	if task == nil || s.repos == nil || s.repos.Events == nil {
		return
	}
	event := &domain.Event{
		ID:        uuid.New().String(),
		RunID:     task.RunID,
		TaskID:    ptrString(task.ID),
		EventType: domain.EventType("security_alert"),
		Message:   strings.TrimSpace(message),
		CreatedAt: time.Now(),
	}
	if strings.TrimSpace(agentID) != "" {
		event.AgentID = ptrString(agentID)
	}
	if err := s.repos.Events.Create(event); err != nil {
		slog.Warn("create security event failed", "task_id", task.ID, "error", err)
	}
}

func (s *Scheduler) setTaskCanary(taskID, canary string) {
	taskID = strings.TrimSpace(taskID)
	canary = strings.TrimSpace(canary)
	if taskID == "" || canary == "" {
		return
	}
	s.qaCanaryMu.Lock()
	defer s.qaCanaryMu.Unlock()
	if s.qaCanaries == nil {
		s.qaCanaries = make(map[string]string)
	}
	s.qaCanaries[taskID] = canary
}

func (s *Scheduler) getTaskCanary(taskID string) string {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return ""
	}
	s.qaCanaryMu.RLock()
	defer s.qaCanaryMu.RUnlock()
	return s.qaCanaries[taskID]
}

func (s *Scheduler) clearTaskCanary(taskID string) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return
	}
	s.qaCanaryMu.Lock()
	defer s.qaCanaryMu.Unlock()
	if s.qaCanaries == nil {
		return
	}
	delete(s.qaCanaries, taskID)
}

func (s *Scheduler) appendFailureLearning(task *domain.Task, reason string) {
	if task == nil || s.learningStore == nil {
		return
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return
	}
	if len(reason) > 800 {
		reason = reason[:800] + "...(truncated)"
	}

	projectSlug := s.resolveLearningProjectSlug(task)
	branch, gitStatus := collectGitStateForPrompt(context.Background(), task.WorkspacePath)
	commit := strings.TrimSpace(runCommandForPrompt(context.Background(), "git", "-C", task.WorkspacePath, "rev-parse", "HEAD"))

	entry := learningengine.Entry{
		TS:         time.Now().UTC().Format(time.RFC3339),
		Skill:      strings.ToLower(strings.TrimSpace(task.TaskType)),
		Type:       "pitfall",
		Key:        "task-failure-" + strings.ToLower(strings.TrimSpace(task.TaskType)),
		Insight:    reason,
		Confidence: 6,
		Source:     "observed",
		Branch:     branch,
		Commit:     commit,
		Files:      extractFilesFromGitStatus(gitStatus),
	}
	if err := s.learningStore.Append(projectSlug, entry); err != nil {
		slog.Warn("append failure learning failed", "task_id", task.ID, "project_slug", projectSlug, "error", err)
	}
}

func (s *Scheduler) appendSuccessLearning(task *domain.Task) {
	if task == nil || s.learningStore == nil {
		return
	}

	projectSlug := s.resolveLearningProjectSlug(task)
	branch, gitStatus := collectGitStateForPrompt(context.Background(), task.WorkspacePath)
	commit := strings.TrimSpace(runCommandForPrompt(context.Background(), "git", "-C", task.WorkspacePath, "rev-parse", "HEAD"))

	insight := fmt.Sprintf("task completed successfully: %s", strings.TrimSpace(task.Title))
	if len(insight) > 800 {
		insight = insight[:800] + "...(truncated)"
	}

	entry := learningengine.Entry{
		TS:         time.Now().UTC().Format(time.RFC3339),
		Skill:      strings.ToLower(strings.TrimSpace(task.TaskType)),
		Type:       "pattern",
		Key:        "task-success-" + strings.ToLower(strings.TrimSpace(task.TaskType)),
		Insight:    insight,
		Confidence: 8,
		Source:     "observed",
		Branch:     branch,
		Commit:     commit,
		Files:      extractFilesFromGitStatus(gitStatus),
	}
	if err := s.learningStore.Append(projectSlug, entry); err != nil {
		slog.Warn("append success learning failed", "task_id", task.ID, "project_slug", projectSlug, "error", err)
	}
}

func (s *Scheduler) buildPromptLegacy(task *domain.Task, spec *domain.TaskSpec) string {
	if task == nil {
		return "执行任务"
	}

	var parts []string

	if task.Description != "" {
		parts = append(parts, task.Description)
	} else if task.Title != "" {
		parts = append(parts, fmt.Sprintf("任务: %s (类型: %s)", task.Title, task.TaskType))
	}

	if task.InputData != "" {
		parts = append(parts, "\n## 输入数据\n"+task.InputData)
	}

	if isBrowseTask(task) && s.cfg != nil && s.cfg.AgentRuntime.BrowseCLIPath != "" {
		parts = append(parts, "\n## Browser QA\n当前任务可以使用 `browse` 命令驱动浏览器。\n使用工作区级 daemon；状态文件在环境变量 `BROWSE_STATE_FILE` 指向的位置。\n典型命令：`browse tabs`、`browse goto <url>`、`browse snapshot -ic`、`browse click @e1`、`browse screenshot qa.png`。")
	}

	if spec != nil {
		if spec.RequiredInputs != "" {
			parts = append(parts, "\n## 需要的输入\n"+spec.RequiredInputs)
		}
		if spec.ExpectedOutputs != "" {
			parts = append(parts, "\n## 期望的输出\n"+spec.ExpectedOutputs)
		}
	}

	if len(parts) == 0 {
		if len(task.ID) >= 8 {
			return fmt.Sprintf("执行任务 %s", task.ID[:8])
		}
		return fmt.Sprintf("执行任务 %s", task.ID)
	}
	return strings.Join(parts, "\n")
}

func (s *Scheduler) getTaskSpec(task *domain.Task) *domain.TaskSpec {
	if task == nil || task.TaskSpecID == "" {
		return nil
	}
	if s.repos == nil || s.repos.TaskSpecs == nil {
		return nil
	}
	spec, err := s.repos.TaskSpecs.GetByID(task.TaskSpecID)
	if err != nil {
		return nil
	}
	return spec
}

func (s *Scheduler) collectPromptRuntimeState(ctx context.Context, task *domain.Task) promptengine.RuntimeState {
	state := promptengine.RuntimeState{}
	if task == nil {
		return state
	}

	state.WorkspacePath = task.WorkspacePath
	if isBrowseTask(task) && s.cfg != nil && strings.TrimSpace(s.cfg.AgentRuntime.BrowseCLIPath) != "" {
		state.BrowseEnabled = true
		if task.WorkspacePath != "" {
			state.BrowseStateFile = filepath.Join(task.WorkspacePath, ".gstack", "browse.json")
		}
	}

	state.GitBranch, state.GitStatus = collectGitStateForPrompt(ctx, task.WorkspacePath)
	return state
}

func collectGitStateForPrompt(ctx context.Context, workspacePath string) (string, string) {
	workspacePath = strings.TrimSpace(workspacePath)
	if workspacePath == "" {
		return "", ""
	}

	info, err := os.Stat(workspacePath)
	if err != nil || !info.IsDir() {
		return "", ""
	}

	branch := strings.TrimSpace(runCommandForPrompt(ctx, "git", "-C", workspacePath, "rev-parse", "--abbrev-ref", "HEAD"))
	status := strings.TrimSpace(runCommandForPrompt(ctx, "git", "-C", workspacePath, "status", "--short", "--branch"))
	if len(status) > 2000 {
		status = status[:2000] + "\n...(truncated)"
	}
	return branch, status
}

func runCommandForPrompt(parent context.Context, name string, args ...string) string {
	ctx := parent
	cancel := func() {}
	if parent == nil {
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	} else {
		ctx, cancel = context.WithTimeout(parent, 2*time.Second)
	}
	defer cancel()

	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func extractFilesFromGitStatus(status string) []string {
	lines := strings.Split(status, "\n")
	files := make([]string, 0, len(lines))
	seen := make(map[string]struct{})
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "## ") {
			continue
		}
		if len(line) >= 3 {
			line = strings.TrimSpace(line[3:])
		}
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		files = append(files, line)
	}
	return files
}

// buildEnv 构建任务的环境变量（会烘进 launcher 脚本，不走 adapter 注入）。
func (s *Scheduler) buildEnv(ctx context.Context, task *domain.Task) (map[string]string, error) {
	env := make(map[string]string)
	if !isBrowseTask(task) {
		return env, nil
	}
	if strings.TrimSpace(s.cfg.AgentRuntime.BrowseCLIPath) == "" {
		slog.Warn("browse cli path not configured, continuing without browser tooling", "task_id", task.ID, "task_type", task.TaskType)
		return env, nil
	}
	if strings.TrimSpace(task.WorkspacePath) == "" {
		return nil, fmt.Errorf("browse-enabled task requires workspace path")
	}

	manager := s.newBrowseManager(s.cfg.AgentRuntime.BrowseCLIPath, task.WorkspacePath)
	if err := manager.EnsureDaemon(ctx); err != nil {
		return nil, fmt.Errorf("ensure browse daemon: %w", err)
	}
	for k, v := range manager.BuildEnv() {
		env[k] = v
	}
	return env, nil
}

func isBrowseTask(task *domain.Task) bool {
	if task == nil {
		return false
	}
	switch task.TaskType {
	case "qa", "browser-qa":
		return true
	default:
		return false
	}
}

func (s *Scheduler) activeBrowseWorkspaces(activeAgents []storage.ActiveAgentsResult) map[string]struct{} {
	occupied := make(map[string]struct{})
	for _, entry := range activeAgents {
		if !isBrowseTask(entry.Task) || entry.Task == nil {
			continue
		}
		if entry.Task.WorkspacePath == "" {
			continue
		}
		occupied[entry.Task.WorkspacePath] = struct{}{}
	}
	return occupied
}

func (s *Scheduler) hasBrowseWorkspaceConflict(task *domain.Task, occupied map[string]struct{}) bool {
	if !isBrowseTask(task) || task == nil || task.WorkspacePath == "" {
		return false
	}
	_, exists := occupied[task.WorkspacePath]
	return exists
}

func promptTemplateDirFromConfig(cfg *config.Config) string {
	if cfg != nil && strings.TrimSpace(cfg.AgentRuntime.PromptTemplateDir) != "" {
		return cfg.AgentRuntime.PromptTemplateDir
	}
	return filepath.Join("configs", "prompts")
}

func learningsRootDirFromConfig(cfg *config.Config) string {
	if cfg != nil && strings.TrimSpace(cfg.AgentRuntime.LearningsRootDir) != "" {
		return cfg.AgentRuntime.LearningsRootDir
	}
	home, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, ".gstack", "projects")
	}
	return filepath.Join("data", "learnings", "projects")
}

func (s *Scheduler) resolveLearningProjectSlug(task *domain.Task) string {
	if task == nil {
		return "default"
	}

	if s.repos != nil && s.repos.Runs != nil && task.RunID != "" {
		run, err := s.repos.Runs.GetByID(task.RunID)
		if err == nil && run != nil {
			if s.repos.Projects != nil && run.ProjectID != "" {
				project, projErr := s.repos.Projects.GetByID(run.ProjectID)
				if projErr == nil && project != nil {
					if slug := learningengine.SanitizeSlug(project.Name); slug != "" {
						return slug
					}
					if slug := learningengine.SanitizeSlug(project.RepoURL); slug != "" {
						return slug
					}
				}
			}
			if slug := learningengine.SanitizeSlug(run.Title); slug != "" {
				return slug
			}
		}
	}

	if task.WorkspacePath != "" {
		if slug := learningengine.SanitizeSlug(filepath.Base(task.WorkspacePath)); slug != "" {
			return slug
		}
	}
	if task.RunID != "" {
		if slug := learningengine.SanitizeSlug(task.RunID); slug != "" {
			return slug
		}
	}
	return "default"
}
