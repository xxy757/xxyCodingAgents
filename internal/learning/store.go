// Package learning 提供与 gstack learnings.jsonl 兼容的读写能力。
package learning

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Entry 对应一条 learnings.jsonl 记录，字段设计与 gstack 兼容。
type Entry struct {
	TS         string   `json:"ts"`         // ISO8601 时间
	Skill      string   `json:"skill"`      // 阶段或 skill 名
	Type       string   `json:"type"`       // pattern/pitfall/preference/architecture/tool/operational
	Key        string   `json:"key"`        // 简短键
	Insight    string   `json:"insight"`    // 核心经验
	Confidence int      `json:"confidence"` // 1-10
	Source     string   `json:"source"`     // observed/user-stated/inferred/cross-model
	Branch     string   `json:"branch"`     // 观察时分支
	Commit     string   `json:"commit"`     // 观察时提交
	Files      []string `json:"files"`      // 相关文件
}

// Store 管理 learnings.jsonl 的落盘与加载。
type Store struct {
	rootDir string
	mu      sync.Mutex // 保护 Append 的并发写入
}

// NewStore 创建 JSONL 存储器。
func NewStore(rootDir string) *Store {
	return &Store{rootDir: strings.TrimSpace(rootDir)}
}

// RootDir 返回存储根目录。
func (s *Store) RootDir() string {
	return s.rootDir
}

// FilePath 返回指定项目的 learnings.jsonl 绝对路径。
func (s *Store) FilePath(projectSlug string) string {
	slug := SanitizeSlug(projectSlug)
	if slug == "" {
		slug = "default"
	}
	return filepath.Join(s.rootDir, slug, "learnings.jsonl")
}

// Append 追加写入一条学习记录（append-only）。
func (s *Store) Append(projectSlug string, entry Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(s.rootDir) == "" {
		return fmt.Errorf("learning store root dir is empty")
	}
	entry = normalizeEntry(entry)
	if strings.TrimSpace(entry.Insight) == "" {
		return fmt.Errorf("learning insight is empty")
	}

	path := s.FilePath(projectSlug)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create learning dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open learning file: %w", err)
	}
	defer f.Close()

	b, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal learning entry: %w", err)
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return fmt.Errorf("append learning entry: %w", err)
	}
	return nil
}

// ReadAll 加载指定项目的所有学习记录。
func (s *Store) ReadAll(projectSlug string) ([]Entry, error) {
	if strings.TrimSpace(s.rootDir) == "" {
		return nil, fmt.Errorf("learning store root dir is empty")
	}

	path := s.FilePath(projectSlug)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open learning file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	entries := make([]Entry, 0, 32)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			// 容错：跳过坏行，避免单条脏数据阻断检索。
			continue
		}
		e = normalizeEntry(e)
		if strings.TrimSpace(e.Insight) == "" {
			continue
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan learning file: %w", err)
	}
	return entries, nil
}

// SanitizeSlug 规范化项目 slug，确保可作为目录名。
func SanitizeSlug(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	raw = strings.ReplaceAll(raw, "\\", "/")
	raw = strings.Trim(raw, "/")
	raw = strings.ReplaceAll(raw, "/", "-")

	re := regexp.MustCompile(`[^a-z0-9._-]+`)
	raw = re.ReplaceAllString(raw, "-")
	raw = strings.Trim(raw, "-._")
	reDash := regexp.MustCompile(`-+`)
	raw = reDash.ReplaceAllString(raw, "-")
	return raw
}

func normalizeEntry(e Entry) Entry {
	e.TS = strings.TrimSpace(e.TS)
	if e.TS == "" {
		e.TS = time.Now().UTC().Format(time.RFC3339)
	}
	e.Skill = strings.ToLower(strings.TrimSpace(e.Skill))
	e.Type = strings.ToLower(strings.TrimSpace(e.Type))
	e.Key = strings.TrimSpace(e.Key)
	e.Insight = strings.TrimSpace(e.Insight)
	e.Source = strings.ToLower(strings.TrimSpace(e.Source))
	e.Branch = strings.TrimSpace(e.Branch)
	e.Commit = strings.TrimSpace(e.Commit)
	if e.Confidence < 0 {
		e.Confidence = 0
	}
	if e.Confidence > 10 {
		e.Confidence = 10
	}
	cleanFiles := make([]string, 0, len(e.Files))
	for _, f := range e.Files {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		cleanFiles = append(cleanFiles, f)
	}
	e.Files = cleanFiles
	return e
}
