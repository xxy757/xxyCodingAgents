package scheduler

import (
	"testing"

	"github.com/xxy757/xxyCodingAgents/internal/config"
	"github.com/xxy757/xxyCodingAgents/internal/domain"
)

func newTestScheduler() *Scheduler {
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			TickSeconds:         3,
			MaxConcurrentAgents: 2,
			MaxHeavyAgents:      1,
			MaxTestJobs:         1,
		},
		Thresholds: config.ThresholdsConfig{
			WarnMemoryPercent:     70,
			HighMemoryPercent:     80,
			CriticalMemoryPercent: 88,
			DiskWarnPercent:       80,
			DiskHighPercent:       90,
		},
		Timeouts: config.TimeoutsConfig{
			HeartbeatTimeoutSeconds:   30,
			OutputTimeoutSeconds:      900,
			StallTimeoutSeconds:       900,
			CheckpointIntervalSeconds: 30,
		},
	}
	return &Scheduler{cfg: cfg}
}

func TestCanAdmit_UnderLimit(t *testing.T) {
	s := newTestScheduler()
	if !s.CanAdmit(0, domain.ResourceClassLight) {
		t.Error("expected to admit with 0 active agents")
	}
	if !s.CanAdmit(1, domain.ResourceClassLight) {
		t.Error("expected to admit with 1 active agent")
	}
}

func TestCanAdmit_AtLimit(t *testing.T) {
	s := newTestScheduler()
	if s.CanAdmit(2, domain.ResourceClassLight) {
		t.Error("expected to reject at MaxConcurrentAgents=2")
	}
}

func TestCanAdmit_OverLimit(t *testing.T) {
	s := newTestScheduler()
	if s.CanAdmit(5, domain.ResourceClassLight) {
		t.Error("expected to reject over MaxConcurrentAgents")
	}
}

func TestCanAdmit_HeavyUnderHeavyLimit(t *testing.T) {
	s := newTestScheduler()
	if !s.CanAdmit(0, domain.ResourceClassHeavy) {
		t.Error("expected to admit heavy task with 0 active agents")
	}
}

func TestCanAdmit_HeavyAtHeavyLimit(t *testing.T) {
	s := newTestScheduler()
	if s.CanAdmit(1, domain.ResourceClassHeavy) {
		t.Error("expected to reject heavy task at MaxHeavyAgents=1")
	}
}

func TestCanAdmit_LightAtHeavyLimit(t *testing.T) {
	s := newTestScheduler()
	if !s.CanAdmit(1, domain.ResourceClassLight) {
		t.Error("expected light task to be admitted even at MaxHeavyAgents")
	}
}

func TestDeterminePressure_Normal(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(50, 50); level != PressureNormal {
		t.Errorf("expected normal, got %s", level)
	}
}

func TestDeterminePressure_WarnMemory(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(75, 50); level != PressureWarn {
		t.Errorf("expected warn, got %s", level)
	}
}

func TestDeterminePressure_WarnDisk(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(50, 80); level != PressureWarn {
		t.Errorf("expected warn at disk=80, got %s", level)
	}
}

func TestDeterminePressure_HighMemory(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(85, 50); level != PressureHigh {
		t.Errorf("expected high, got %s", level)
	}
}

func TestDeterminePressure_HighDisk(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(50, 85); level != PressureHigh {
		t.Errorf("expected high at disk=85, got %s", level)
	}
}

func TestDeterminePressure_CriticalMemory(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(90, 50); level != PressureCritical {
		t.Errorf("expected critical, got %s", level)
	}
}

func TestDeterminePressure_CriticalDisk(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(50, 95); level != PressureCritical {
		t.Errorf("expected critical, got %s", level)
	}
}

func TestDeterminePressure_BoundaryWarn(t *testing.T) {
	s := newTestScheduler()
	cfg := s.cfg.Thresholds
	if level := s.determinePressure(float64(cfg.WarnMemoryPercent-1), float64(cfg.DiskWarnPercent-1)); level != PressureNormal {
		t.Errorf("expected normal below boundary, got %s", level)
	}
	if level := s.determinePressure(float64(cfg.WarnMemoryPercent), float64(cfg.DiskWarnPercent-1)); level != PressureWarn {
		t.Errorf("expected warn at exact memory boundary, got %s", level)
	}
	if level := s.determinePressure(float64(cfg.WarnMemoryPercent-1), float64(cfg.DiskWarnPercent)); level != PressureWarn {
		t.Errorf("expected warn at exact disk boundary, got %s", level)
	}
}

func TestDeterminePressure_ZeroValues(t *testing.T) {
	s := newTestScheduler()
	if level := s.determinePressure(0, 0); level != PressureNormal {
		t.Errorf("expected normal at 0%%, got %s", level)
	}
}

func TestHandleLoadShedding_NormalDoesNothing(t *testing.T) {
	s := newTestScheduler()
	s.handleLoadShedding(nil, PressureNormal, nil)
}

func TestHandleLoadShedding_WarnDoesNothing(t *testing.T) {
	s := newTestScheduler()
	s.handleLoadShedding(nil, PressureWarn, nil)
}

func TestPressureLevelConstants(t *testing.T) {
	if PressureNormal != "normal" {
		t.Errorf("expected 'normal', got %s", PressureNormal)
	}
	if PressureWarn != "warn" {
		t.Errorf("expected 'warn', got %s", PressureWarn)
	}
	if PressureHigh != "high" {
		t.Errorf("expected 'high', got %s", PressureHigh)
	}
	if PressureCritical != "critical" {
		t.Errorf("expected 'critical', got %s", PressureCritical)
	}
}
