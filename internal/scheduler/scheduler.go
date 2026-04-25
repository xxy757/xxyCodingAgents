// Package scheduler 实现核心调度器，负责定时执行任务调度、资源监控、
// 负载保护（驱逐/暂停 Agent）、定期检查点、日志清理和工作区大小检查。
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
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
	cfg       *config.Config
	repos     *storage.Repos
	runtime   agentruntime.AgentRuntime
	terminal  *terminal.Manager
	stop      chan struct{}
	tickCount int64 // 调度周期计数器
}

// NewScheduler 创建调度器实例。
func NewScheduler(cfg *config.Config, repos *storage.Repos, rt agentruntime.AgentRuntime, tm *terminal.Manager) *Scheduler {
	return &Scheduler{
		cfg:      cfg,
		repos:    repos,
		runtime:  rt,
		terminal: tm,
		stop:     make(chan struct{}),
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

// tick 执行一次调度周期，包含资源监控、负载保护、任务调度和定期清理。
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

	// 执行负载保护（暂停或驱逐 Agent）
	s.handleLoadShedding(ctx, level, activeAgents)

	// 根据压力等级调度任务
	if level == PressureNormal {
		s.scheduleTasks(ctx, activeCount)
	} else if level == PressureWarn {
		s.scheduleTasksLightOnly(ctx, activeCount)
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
		if err := s.runtime.Pause(ctx, entry.Agent.TmuxSession); err != nil {
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

// evictAgents 在临界压力时先创建检查点再驱逐所有可抢占的 Agent。
func (s *Scheduler) evictAgents(ctx context.Context, activeAgents []storage.ActiveAgentsResult) {
	for _, entry := range activeAgents {
		if entry.Agent.Status != domain.AgentStatusRunning && entry.Agent.Status != domain.AgentStatusPaused {
			continue
		}
		if !entry.Task.Preemptible {
			continue
		}

		// 驱逐前尝试创建检查点
		cp, err := s.runtime.Checkpoint(ctx, entry.Agent.ID)
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
		if err := s.runtime.Stop(ctx, entry.Agent.TmuxSession); err != nil {
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
func (s *Scheduler) scheduleTasksLightOnly(ctx context.Context, activeAgentCount int) {
	queuedTasks, err := s.repos.Tasks.ListByStatus(domain.TaskStatusQueued)
	if err != nil || len(queuedTasks) == 0 {
		return
	}

	for _, task := range queuedTasks {
		// 跳过重型任务
		if task.ResourceClass == domain.ResourceClassHeavy {
			slog.Info("skipping heavy task under WARN pressure", "task_id", task.ID)
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
	}
}

// runCheckpoints 为所有运行中的 Agent 创建定期检查点。
func (s *Scheduler) runCheckpoints(ctx context.Context) {
	agents, err := s.repos.AgentInstances.ListByStatus(domain.AgentStatusRunning)
	if err != nil || len(agents) == 0 {
		return
	}

	for _, agent := range agents {
		cp, err := s.runtime.Checkpoint(ctx, agent.TmuxSession)
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

	agentID := uuid.New().String()
	tmuxSession := fmt.Sprintf("agent-%s", agentID[:8])

	// 创建恢复用的 Agent 实例
	agent := &domain.AgentInstance{
		ID:            agentID,
		RunID:         task.RunID,
		TaskID:        task.ID,
		AgentKind:     "generic-shell",
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

	// 启动 Agent 执行
	startReq := agentruntime.StartRequest{
		AgentID:     agentID,
		TaskID:      task.ID,
		RunID:       task.RunID,
		AgentKind:   agent.AgentKind,
		Command:     s.resolveCommand(task),
		TmuxSession: tmuxSession,
	}

	result, err := s.runtime.Start(ctx, startReq)
	if err != nil {
		s.terminal.KillSession(ctx, tmuxSession)
		s.repos.AgentInstances.UpdateStatus(agentID, domain.AgentStatusFailed)
		return fmt.Errorf("start recovery agent: %w", err)
	}

	if result.PID > 0 {
		s.repos.AgentInstances.UpdatePID(agentID, result.PID)
	}

	s.repos.AgentInstances.UpdateStatus(agentID, domain.AgentStatusRunning)
	s.repos.Tasks.UpdateStatus(task.ID, domain.TaskStatusRunning)

	// 记录恢复事件
	now := time.Now()
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

// scheduleTasks 在正常压力下调度所有类型的排队任务。
func (s *Scheduler) scheduleTasks(ctx context.Context, activeAgentCount int) {
	queuedTasks, err := s.repos.Tasks.ListByStatus(domain.TaskStatusQueued)
	if err != nil || len(queuedTasks) == 0 {
		return
	}

	for _, task := range queuedTasks {
		if !s.CanAdmit(activeAgentCount, task.ResourceClass) {
			break
		}
		if err := s.launchAgent(ctx, task); err != nil {
			slog.Error("launch agent for task", "task_id", task.ID, "error", err)
			continue
		}
		activeAgentCount++
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

	agentID := uuid.New().String()
	tmuxSession := fmt.Sprintf("agent-%s", agentID[:8])

	// 创建 Agent 实例记录
	agent := &domain.AgentInstance{
		ID:            agentID,
		RunID:         task.RunID,
		TaskID:        task.ID,
		AgentKind:     "generic-shell",
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
		return fmt.Errorf("create tmux session: %w", err)
	}

	// 解析并执行命令
	startReq := agentruntime.StartRequest{
		AgentID:     agentID,
		TaskID:      task.ID,
		RunID:       task.RunID,
		AgentKind:   agent.AgentKind,
		Command:     s.resolveCommand(task),
		TmuxSession: tmuxSession,
	}

	result, err := s.runtime.Start(ctx, startReq)
	if err != nil {
		s.terminal.KillSession(ctx, tmuxSession)
		s.repos.AgentInstances.UpdateStatus(agentID, domain.AgentStatusFailed)
		s.repos.Tasks.UpdateStatus(task.ID, domain.TaskStatusFailed)
		return fmt.Errorf("start agent runtime: %w", err)
	}

	if result.PID > 0 {
		s.repos.AgentInstances.UpdatePID(agentID, result.PID)
	}

	// 更新任务状态为运行中
	now := time.Now()
	if err := s.repos.Tasks.UpdateStatus(task.ID, domain.TaskStatusRunning); err != nil {
		slog.Error("update task to running", "task_id", task.ID, "error", err)
	}

	// 创建关联的终端会话记录
	terminalSession := &domain.TerminalSession{
		ID:          uuid.New().String(),
		TaskID:      task.ID,
		AgentID:     ptrString(agentID),
		TmuxSession: tmuxSession,
		TmuxPane:    result.TmuxPane,
		Status:      domain.TerminalStatusActive,
		LogFilePath: fmt.Sprintf("data/logs/%s.log", tmuxSession),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repos.TerminalSessions.Create(terminalSession); err != nil {
		slog.Error("create terminal session", "agent_id", agentID, "error", err)
	}

	s.repos.AgentInstances.UpdateStatus(agentID, domain.AgentStatusRunning)

	// 记录 Agent 启动事件
	s.repos.Events.Create(&domain.Event{
		ID:        uuid.New().String(),
		RunID:     task.RunID,
		TaskID:    ptrString(task.ID),
		AgentID:   ptrString(agentID),
		EventType: "agent_started",
		Message:   fmt.Sprintf("Agent %s started for task %s", agentID[:8], task.Title),
		Metadata:  fmt.Sprintf(`{"tmux_session":"%s","pid":%d}`, tmuxSession, result.PID),
		CreatedAt: now,
	})

	return nil
}

// resolveCommand 解析任务要执行的命令，优先使用 TaskSpec 中的命令模板。
func (s *Scheduler) resolveCommand(task *domain.Task) string {
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
	return "echo 'task " + task.ID[:8] + " started'"
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
