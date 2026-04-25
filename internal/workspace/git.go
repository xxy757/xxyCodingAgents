// Package workspace 提供 Git 工作区管理功能。
// 支持创建工作区、克隆仓库、切换分支、提交变更和查询状态等操作。
package workspace

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitManager 管理 Git 工作区的生命周期。
type GitManager struct {
	workspaceRoot string // 工作区根目录
}

// NewGitManager 创建 Git 管理器实例。
func NewGitManager(workspaceRoot string) *GitManager {
	return &GitManager{workspaceRoot: workspaceRoot}
}

// CreateWorkspace 为指定任务创建一个工作目录。
func (g *GitManager) CreateWorkspace(ctx context.Context, taskID string) (string, error) {
	dir := filepath.Join(g.workspaceRoot, taskID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create workspace dir: %w", err)
	}
	slog.Info("workspace created", "path", dir)
	return dir, nil
}

// Clone 克隆远程仓库到指定目录。
func (g *GitManager) Clone(ctx context.Context, repoURL, destDir string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", repoURL, destDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %s: %w, output: %s", repoURL, err, string(out))
	}
	return nil
}

// CheckoutBranch 在指定目录创建并切换到新分支。
func (g *GitManager) CheckoutBranch(ctx context.Context, dir, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "checkout", "-b", branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout -b %s: %w, output: %s", branch, err, string(out))
	}
	return nil
}

// Status 获取指定目录的 Git 工作区状态（简洁格式）。
func (g *GitManager) Status(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git status: %w", err)
	}
	return string(out), nil
}

// Diff 获取指定目录的未暂存变更内容。
func (g *GitManager) Diff(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "diff")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return string(out), nil
}

// AddAndCommit 暂存所有变更并提交。
func (g *GitManager) AddAndCommit(ctx context.Context, dir, message string) error {
	// 暂存所有变更
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "add", "-A")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add: %w, output: %s", err, string(out))
	}
	// 提交变更
	cmd = exec.CommandContext(ctx, "git", "-C", dir, "commit", "-m", message)
	out, err = cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("git commit: %w, output: %s", err, string(out))
	}
	return nil
}

// GetCommitSHA 获取指定目录的当前 HEAD 提交 SHA。
func (g *GitManager) GetCommitSHA(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// DirSize 计算指定目录的总文件大小。
func (g *GitManager) DirSize(dir string) (int64, error) {
	var size int64
	filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, nil
}

// RemoveWorkspace 递归删除工作区目录。
func (g *GitManager) RemoveWorkspace(dir string) error {
	return os.RemoveAll(dir)
}
