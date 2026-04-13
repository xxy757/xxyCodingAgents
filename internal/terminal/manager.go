package terminal

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	logRoot string
	mu      sync.Mutex
}

func NewManager() *Manager {
	return &Manager{logRoot: "data/logs"}
}

func NewManagerWithLogRoot(logRoot string) *Manager {
	return &Manager{logRoot: logRoot}
}

func (m *Manager) CreateSession(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "tmux", "new-session", "-d", "-s", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create tmux session %s: %w, output: %s", name, err, string(out))
	}
	slog.Info("tmux session created", "name", name)
	return nil
}

func (m *Manager) KillSession(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "tmux", "kill-session", "-t", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kill tmux session %s: %w, output: %s", name, err, string(out))
	}
	slog.Info("tmux session killed", "name", name)
	return nil
}

func (m *Manager) ListSessions(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "tmux", "list-sessions", "-F", "#{session_name}")
	out, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "no sessions") {
			return nil, nil
		}
		return nil, fmt.Errorf("list tmux sessions: %w", err)
	}
	sessions := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(sessions) == 1 && sessions[0] == "" {
		return nil, nil
	}
	return sessions, nil
}

func (m *Manager) SessionExists(ctx context.Context, name string) bool {
	sessions, err := m.ListSessions(ctx)
	if err != nil {
		return false
	}
	for _, s := range sessions {
		if s == name {
			return true
		}
	}
	return false
}

func (m *Manager) SendKeys(ctx context.Context, session string, keys string) error {
	cmd := exec.CommandContext(ctx, "tmux", "send-keys", "-t", session, keys, "Enter")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("send keys to session %s: %w, output: %s", session, err, string(out))
	}
	return nil
}

func (m *Manager) CapturePane(ctx context.Context, session string) (string, error) {
	cmd := exec.CommandContext(ctx, "tmux", "capture-pane", "-t", session, "-p")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("capture pane for session %s: %w", session, err)
	}
	return string(out), nil
}

func (m *Manager) GetPanePID(ctx context.Context, session string) (int, error) {
	cmd := exec.CommandContext(ctx, "tmux", "list-panes", "-t", session, "-F", "#{pane_pid}")
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("get pane pid for session %s: %w", session, err)
	}
	var pid int
	fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &pid)
	return pid, nil
}

func (m *Manager) CaptureAndPersist(ctx context.Context, session string) error {
	output, err := m.CapturePane(ctx, session)
	if err != nil {
		return err
	}
	if strings.TrimSpace(output) == "" {
		return nil
	}
	return m.appendToLog(session, output)
}

func (m *Manager) appendToLog(session string, output string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(m.logRoot, 0755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	logPath := filepath.Join(m.logRoot, session+".log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", logPath, err)
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	_, err = fmt.Fprintf(f, "[%s] %s", timestamp, output)
	return err
}

func (m *Manager) ReadLog(session string, maxLines int) (string, error) {
	logPath := filepath.Join(m.logRoot, session+".log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n"), nil
}

func (m *Manager) CleanupOldLogs(retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}

	entries, err := os.ReadDir(m.logRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			logPath := filepath.Join(m.logRoot, entry.Name())
			os.Remove(logPath)
			slog.Info("cleaned up old log file", "file", entry.Name(), "modified", info.ModTime())
		}
	}
	return nil
}

func (m *Manager) TotalLogSize() (int64, error) {
	var total int64
	entries, err := os.ReadDir(m.logRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		total += info.Size()
	}
	return total, nil
}
