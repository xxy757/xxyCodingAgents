// Package agentlauncher 负责将任务参数（prompt 或 shell 命令 + 环境变量）
// 烘进一个 launcher shell 脚本，让 tmux 只需执行一条命令。
// 不同 agent kind 走不同的脚本模板，但产物统一是"一个脚本路径"。
package agentlauncher

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config 是生成 launcher 脚本所需的全部参数。
type Config struct {
	TaskID        string            // 任务 ID，用于文件命名
	WorkspacePath string            // 工作目录
	Env           map[string]string // 环境变量（烘进脚本，不由 adapter 注入）
	AgentKind     string            // agent 类型
	PromptContent string            // prompt 内容（仅 LLM CLI agent 使用）
	ShellCommand  string            // shell 命令（仅 generic-shell 使用）
	BaseDir       string            // prompt 和 launcher 的根目录
}

// Build 为给定配置生成 prompt 文件 + launcher 脚本。
// 返回 launcher 脚本的绝对路径。
func Build(cfg Config) (string, error) {
	promptDir := filepath.Join(cfg.BaseDir, "prompts", cfg.TaskID)
	launcherDir := filepath.Join(cfg.BaseDir, "launchers", cfg.TaskID)

	if err := os.MkdirAll(promptDir, 0755); err != nil {
		return "", fmt.Errorf("create prompt dir: %w", err)
	}
	if err := os.MkdirAll(launcherDir, 0755); err != nil {
		return "", fmt.Errorf("create launcher dir: %w", err)
	}

	switch cfg.AgentKind {
	case "claude-code":
		return buildClaudeLauncher(cfg, promptDir, launcherDir)
	default:
		return buildShellLauncher(cfg, launcherDir)
	}
}

// Cleanup 删除任务对应的 prompt 和 launcher 文件。
func Cleanup(baseDir, taskID string) {
	os.RemoveAll(filepath.Join(baseDir, "prompts", taskID))
	os.RemoveAll(filepath.Join(baseDir, "launchers", taskID))
}

func buildClaudeLauncher(cfg Config, promptDir, launcherDir string) (string, error) {
	if cfg.PromptContent == "" {
		return "", fmt.Errorf("claude-code agent requires prompt content")
	}
	if cfg.WorkspacePath == "" {
		return "", fmt.Errorf("claude-code agent requires workspace path")
	}

	// 1. 写 prompt 文件（权限 0600，prompt 可能含代码上下文和 token）
	promptFile := filepath.Join(promptDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte(cfg.PromptContent), 0600); err != nil {
		return "", fmt.Errorf("write prompt file: %w", err)
	}

	// 2. 生成 launcher 脚本
	script := "#!/bin/bash\n"
	script += "set -euo pipefail\n"

	// 按 exit code 输出完成/失败标记（scheduler 靠这两个标记检测任务状态）
	script += "trap 'code=$?; if [ $code -eq 0 ]; then echo \"[TASK_COMPLETED]\"; else echo \"[TASK_FAILED] exit_code=$code\"; fi; exit $code' EXIT\n"

	if cfg.WorkspacePath != "" {
		script += fmt.Sprintf("cd %q\n", cfg.WorkspacePath)
	}

	for k, v := range cfg.Env {
		script += fmt.Sprintf("export %s=%q\n", k, v)
	}

	script += fmt.Sprintf("claude -p < %q\n", promptFile)

	launcherFile := filepath.Join(launcherDir, "run.sh")
	if err := os.WriteFile(launcherFile, []byte(script), 0700); err != nil {
		return "", fmt.Errorf("write launcher script: %w", err)
	}

	abs, err := filepath.Abs(launcherFile)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}
	return abs, nil
}

func buildShellLauncher(cfg Config, launcherDir string) (string, error) {
	if cfg.ShellCommand == "" {
		return "", fmt.Errorf("generic-shell agent requires shell command")
	}

	script := "#!/bin/bash\n"
	script += "set -euo pipefail\n"
	script += "trap 'code=$?; if [ $code -eq 0 ]; then echo \"[TASK_COMPLETED]\"; else echo \"[TASK_FAILED] exit_code=$code\"; fi; exit $code' EXIT\n"

	if cfg.WorkspacePath != "" {
		script += fmt.Sprintf("cd %q\n", cfg.WorkspacePath)
	}

	for k, v := range cfg.Env {
		script += fmt.Sprintf("export %s=%q\n", k, v)
	}

	script += cfg.ShellCommand + "\n"

	launcherFile := filepath.Join(launcherDir, "run.sh")
	if err := os.WriteFile(launcherFile, []byte(script), 0700); err != nil {
		return "", fmt.Errorf("write launcher script: %w", err)
	}

	abs, err := filepath.Abs(launcherFile)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}
	return abs, nil
}
