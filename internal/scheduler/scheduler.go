package scheduler

import (
	"context"
	"log/slog"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/xxy757/xxyCodingAgents/internal/config"
	"github.com/xxy757/xxyCodingAgents/internal/domain"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
)

type PressureLevel string

const (
	PressureNormal   PressureLevel = "normal"
	PressureWarn     PressureLevel = "warn"
	PressureHigh     PressureLevel = "high"
	PressureCritical PressureLevel = "critical"
)

type Scheduler struct {
	cfg       *config.Config
	repos     *storage.Repos
	stop      chan struct{}
	tickCount int64
}

func NewScheduler(cfg *config.Config, repos *storage.Repos) *Scheduler {
	return &Scheduler{
		cfg:   cfg,
		repos: repos,
		stop:  make(chan struct{}),
	}
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.Scheduler.TickDuration())
	defer ticker.Stop()

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
		}
	}
}

func (s *Scheduler) Stop() {
	close(s.stop)
}

func (s *Scheduler) tick(ctx context.Context) {
	memPercent, cpuPercent, diskPercent := s.collectMetrics()
	activeAgents, _ := s.repos.AgentInstances.ListActiveWithTasks()
	activeCount := len(activeAgents)

	level := s.determinePressure(memPercent, diskPercent)

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

	s.scheduleTasks(ctx, activeCount)

	s.tickCount++
	if s.tickCount%100 == 0 {
		s.cleanup(ctx)
	}

	slog.Debug("scheduler tick", "memory", memPercent, "cpu", cpuPercent, "disk", diskPercent, "agents", activeCount, "pressure", level)
}

func (s *Scheduler) scheduleTasks(ctx context.Context, activeAgentCount int) {
	queuedTasks, err := s.repos.Tasks.ListByStatus(domain.TaskStatusQueued)
	if err != nil || len(queuedTasks) == 0 {
		return
	}

	for _, task := range queuedTasks {
		if !s.CanAdmit(activeAgentCount, task.ResourceClass) {
			break
		}
		if err := s.repos.Tasks.UpdateStatus(task.ID, domain.TaskStatusAdmitted); err != nil {
			slog.Error("admit task", "task_id", task.ID, "error", err)
			continue
		}
		if err := s.repos.Tasks.UpdateQueueStatus(task.ID, "admitted"); err != nil {
			slog.Error("update queue status", "task_id", task.ID, "error", err)
		}
		activeAgentCount++
		slog.Info("task admitted", "task_id", task.ID, "resource_class", task.ResourceClass)
	}
}

func (s *Scheduler) cleanup(ctx context.Context) {
	cutoff := time.Now().AddDate(0, 0, -s.cfg.Thresholds.LogRetentionDays)
	if err := s.repos.ResourceSnapshots.DeleteOlderThan(cutoff); err != nil {
		slog.Error("cleanup resource snapshots", "error", err)
	}
	if err := s.repos.Events.DeleteOlderThan(cutoff); err != nil {
		slog.Error("cleanup events", "error", err)
	}
	slog.Info("cleanup completed", "cutoff", cutoff)
}

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

	if runtime.GOOS == "darwin" {
		if diskStat, err := disk.Usage("/Volumes"); err == nil && diskPercent == 0 {
			diskPercent = diskStat.UsedPercent
		}
	}

	return memPercent, cpuPercent, diskPercent
}

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
