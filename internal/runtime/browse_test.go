package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBrowseManagerBuildEnv(t *testing.T) {
	mgr := NewBrowseManager("/opt/gstack/dist/browse", "/tmp/workspace")
	env := mgr.BuildEnv()

	if env["BROWSE_STATE_FILE"] != "/tmp/workspace/.gstack/browse.json" {
		t.Fatalf("unexpected state file env: %#v", env)
	}
	if env["BROWSE_CLI_PATH"] != "/opt/gstack/dist/browse" {
		t.Fatalf("unexpected cli path env: %#v", env)
	}
	if env["PATH"] != "/opt/gstack/dist:$PATH" {
		t.Fatalf("unexpected PATH env: %#v", env)
	}
}

func TestBrowseManagerEnsureDaemon(t *testing.T) {
	health := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}))
	defer health.Close()

	dir := t.TempDir()
	stateFile := filepath.Join(dir, ".gstack", "browse.json")

	mgr := NewBrowseManager("/fake/browse", dir)
	mgr.stateFile = stateFile
	mgr.startTimeout = 2 * defaultBrowsePollInterval
	mgr.sleep = func(time.Duration) {}
	mgr.runCommand = func(_ context.Context, _ string, _ []string, _ []string, cwd string) error {
		if cwd != dir {
			t.Fatalf("unexpected cwd passed to runCommand: got %q want %q", cwd, dir)
		}
		if err := os.MkdirAll(filepath.Dir(stateFile), 0755); err != nil {
			return err
		}
		return os.WriteFile(stateFile, []byte(`{"pid":123,"port":`+healthURLPort(health.URL)+`,"token":"root","startedAt":"now"}`), 0644)
	}

	if err := mgr.EnsureDaemon(context.Background()); err != nil {
		t.Fatalf("EnsureDaemon failed: %v", err)
	}
}

func TestBrowseManagerReadStateMissing(t *testing.T) {
	mgr := NewBrowseManager("/fake/browse", t.TempDir())
	state, err := mgr.ReadState()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if state != nil {
		t.Fatalf("expected nil state, got %#v", state)
	}
}

func healthURLPort(raw string) string {
	trimmed := raw
	if idx := len("http://"); len(trimmed) > idx && trimmed[:idx] == "http://" {
		trimmed = trimmed[idx:]
	}
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] == ':' {
			return trimmed[i+1:]
		}
	}
	return trimmed
}
