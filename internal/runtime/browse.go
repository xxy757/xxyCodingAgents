// Package runtime 提供 gstack browse CLI 的最小集成层。
// 负责管理 workspace 级 browse daemon 的启动、健康检查和环境变量构建。
package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultBrowseStartTimeout = 8 * time.Second
	defaultBrowsePollInterval = 200 * time.Millisecond
)

// BrowseState 对应 gstack 的 browse.json 状态文件。
type BrowseState struct {
	PID           int    `json:"pid"`
	Port          int    `json:"port"`
	Token         string `json:"token"`
	StartedAt     string `json:"startedAt"`
	ServerPath    string `json:"serverPath,omitempty"`
	BinaryVersion string `json:"binaryVersion,omitempty"`
	Mode          string `json:"mode,omitempty"`
}

// BrowseManager 管理一个 workspace 级的 browse daemon。
type BrowseManager struct {
	cliPath      string
	workspace    string
	stateFile    string
	httpClient   *http.Client
	runCommand   func(ctx context.Context, name string, args []string, env []string, cwd string) error
	sleep        func(time.Duration)
	startTimeout time.Duration
	pollInterval time.Duration
}

// NewBrowseManager 创建 browse manager。
func NewBrowseManager(cliPath, workspacePath string) *BrowseManager {
	return &BrowseManager{
		cliPath:      cliPath,
		workspace:    workspacePath,
		stateFile:    filepath.Join(workspacePath, ".gstack", "browse.json"),
		httpClient:   &http.Client{Timeout: 2 * time.Second},
		runCommand:   defaultBrowseRunCommand,
		sleep:        time.Sleep,
		startTimeout: defaultBrowseStartTimeout,
		pollInterval: defaultBrowsePollInterval,
	}
}

// EnsureDaemon 确保 workspace 级 browse daemon 已经启动。
func (m *BrowseManager) EnsureDaemon(ctx context.Context) error {
	if strings.TrimSpace(m.cliPath) == "" {
		return fmt.Errorf("browse cli path is empty")
	}
	if strings.TrimSpace(m.workspace) == "" {
		return fmt.Errorf("workspace path is required for browse")
	}
	if err := os.MkdirAll(filepath.Dir(m.stateFile), 0755); err != nil {
		return fmt.Errorf("create browse state dir: %w", err)
	}

	if state, err := m.ReadState(); err == nil && state != nil {
		healthy, err := m.IsHealthy(ctx, state)
		if err == nil && healthy {
			return nil
		}
	}

	env := append(os.Environ(), fmt.Sprintf("BROWSE_STATE_FILE=%s", m.stateFile))
	startErr := m.runCommand(ctx, m.cliPath, []string{"tabs"}, env, m.workspace)

	deadline := time.Now().Add(m.startTimeout)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		state, err := m.ReadState()
		if err == nil && state != nil {
			healthy, healthErr := m.IsHealthy(ctx, state)
			if healthErr == nil && healthy {
				return nil
			}
		}
		m.sleep(m.pollInterval)
	}

	if startErr != nil {
		return fmt.Errorf("browse daemon failed to start: %w", startErr)
	}
	return fmt.Errorf("browse daemon failed to become healthy within %s", m.startTimeout)
}

// ReadState 读取 browse 状态文件。
func (m *BrowseManager) ReadState() (*BrowseState, error) {
	data, err := os.ReadFile(m.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read browse state: %w", err)
	}

	var state BrowseState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse browse state: %w", err)
	}
	if state.Port == 0 {
		return nil, fmt.Errorf("browse state missing port")
	}
	return &state, nil
}

// IsHealthy 通过 /health 检查 daemon 是否健康。
func (m *BrowseManager) IsHealthy(ctx context.Context, state *BrowseState) (bool, error) {
	if state == nil || state.Port == 0 {
		return false, fmt.Errorf("invalid browse state")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/health", state.Port), nil)
	if err != nil {
		return false, err
	}
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("health check returned %d", resp.StatusCode)
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, fmt.Errorf("decode health response: %w", err)
	}
	return body.Status == "healthy", nil
}

// BuildEnv 返回 QA 任务需要注入到 launcher 的环境变量。
func (m *BrowseManager) BuildEnv() map[string]string {
	env := map[string]string{
		"BROWSE_STATE_FILE": m.stateFile,
		"BROWSE_CLI_PATH":   m.cliPath,
	}
	cliDir := filepath.Dir(m.cliPath)
	if cliDir != "" && cliDir != "." {
		env["PATH"] = cliDir + string(os.PathListSeparator) + "$PATH"
	}
	return env
}

func defaultBrowseRunCommand(ctx context.Context, name string, args []string, env []string, cwd string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = env
	if strings.TrimSpace(cwd) != "" {
		cmd.Dir = cwd
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}
