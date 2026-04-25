// Package terminal 提供 tmux 终端会话管理功能。
// 支持创建/销毁会话、捕获终端输出、发送按键、日志持久化和清理。
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

// Manager 管理 tmux 终端会话，提供会话生命周期管理和日志持久化。
type Manager struct {
	logRoot string     // 日志文件存储根目录
	mu      sync.Mutex // 保护并发写入日志文件
}

// NewManager 创建终端管理器实例，使用默认日志目录。
func NewManager() *Manager {
	return &Manager{logRoot: "data/logs"}
}

// NewManagerWithLogRoot 创建终端管理器实例，指定日志目录。
func NewManagerWithLogRoot(logRoot string) *Manager {
	return &Manager{logRoot: logRoot}
}

// CreateSession 创建一个新的 tmux 会话（分离模式）。
func (m *Manager) CreateSession(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "tmux", "new-session", "-d", "-s", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create tmux session %s: %w, output: %s", name, err, string(out))
	}
	slog.Info("tmux session created", "name", name)
	return nil
}

// KillSession 终止指定名称的 tmux 会话。
func (m *Manager) KillSession(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "tmux", "kill-session", "-t", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kill tmux session %s: %w, output: %s", name, err, string(out))
	}
	slog.Info("tmux session killed", "name", name)
	return nil
}

// ListSessions 列出当前所有 tmux 会话名称。
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

// SessionExists 检查指定名称的 tmux 会话是否存在。
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

// SendKeys 向指定 tmux 会话发送按键序列（附加 Enter）。
func (m *Manager) SendKeys(ctx context.Context, session string, keys string) error {
	cmd := exec.CommandContext(ctx, "tmux", "send-keys", "-t", session, keys, "Enter")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("send keys to session %s: %w, output: %s", session, err, string(out))
	}
	return nil
}

// CapturePane 捕获指定 tmux 会话当前终端面板的全部内容。
func (m *Manager) CapturePane(ctx context.Context, session string) (string, error) {
	cmd := exec.CommandContext(ctx, "tmux", "capture-pane", "-t", session, "-p")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("capture pane for session %s: %w", session, err)
	}
	return string(out), nil
}

// GetPanePID 获取指定 tmux 会话中主面板的进程 ID。
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

// CaptureAndPersist 捕获终端输出并追加到日志文件。
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

// appendToLog 将终端输出追加到对应的日志文件中。
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

// ReadLog 读取指定会话的日志文件内容，可限制最大行数。
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

// CleanupOldLogs 清理超过保留天数的日志文件。
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

// TotalLogSize 计算日志目录下所有日志文件的总大小。
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
