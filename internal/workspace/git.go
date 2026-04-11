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

type GitManager struct {
	workspaceRoot string
}

func NewGitManager(workspaceRoot string) *GitManager {
	return &GitManager{workspaceRoot: workspaceRoot}
}

func (g *GitManager) CreateWorkspace(ctx context.Context, taskID string) (string, error) {
	dir := filepath.Join(g.workspaceRoot, taskID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create workspace dir: %w", err)
	}
	slog.Info("workspace created", "path", dir)
	return dir, nil
}

func (g *GitManager) Clone(ctx context.Context, repoURL, destDir string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", repoURL, destDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %s: %w, output: %s", repoURL, err, string(out))
	}
	return nil
}

func (g *GitManager) CheckoutBranch(ctx context.Context, dir, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "checkout", "-b", branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout -b %s: %w, output: %s", branch, err, string(out))
	}
	return nil
}

func (g *GitManager) Status(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git status: %w", err)
	}
	return string(out), nil
}

func (g *GitManager) Diff(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "diff")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return string(out), nil
}

func (g *GitManager) AddAndCommit(ctx context.Context, dir, message string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "add", "-A")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add: %w, output: %s", err, string(out))
	}
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

func (g *GitManager) GetCommitSHA(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

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

func (g *GitManager) RemoveWorkspace(dir string) error {
	return os.RemoveAll(dir)
}
