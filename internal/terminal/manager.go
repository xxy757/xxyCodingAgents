package terminal

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

type Manager struct{}

func NewManager() *Manager {
	return &Manager{}
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
