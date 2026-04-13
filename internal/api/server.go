package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/xxy757/xxyCodingAgents/internal/config"
	"github.com/xxy757/xxyCodingAgents/internal/orchestrator"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
	"github.com/xxy757/xxyCodingAgents/internal/terminal"
)

type Server struct {
	cfg          *config.Config
	db           *storage.DB
	repos        *storage.Repos
	mux          *http.ServeMux
	hub          *WebSocketHub
	orch         *orchestrator.Orchestrator
	termMgr      *terminal.Manager
}

func NewServer(cfg *config.Config, db *storage.DB, repos *storage.Repos, orch *orchestrator.Orchestrator, termMgr *terminal.Manager) *Server {
	s := &Server{
		cfg:     cfg,
		db:      db,
		repos:   repos,
		mux:     http.NewServeMux(),
		hub:     NewWebSocketHub(),
		orch:    orch,
		termMgr: termMgr,
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/healthz", s.handleHealthz)
	s.mux.HandleFunc("/readyz", s.handleReadyz)

	s.mux.HandleFunc("POST /api/projects", s.handleCreateProject)
	s.mux.HandleFunc("GET /api/projects", s.handleListProjects)
	s.mux.HandleFunc("GET /api/projects/{id}", s.handleGetProject)

	s.mux.HandleFunc("POST /api/runs", s.handleCreateRun)
	s.mux.HandleFunc("GET /api/runs", s.handleListAllRuns)
	s.mux.HandleFunc("GET /api/runs/{id}", s.handleGetRun)
	s.mux.HandleFunc("GET /api/runs/{id}/timeline", s.handleGetRunTimeline)
	s.mux.HandleFunc("GET /api/projects/{id}/runs", s.handleListRuns)

	s.mux.HandleFunc("GET /api/runs/{id}/tasks", s.handleListTasks)
	s.mux.HandleFunc("GET /api/runs/{id}/workflow", s.handleRunWorkflow)
	s.mux.HandleFunc("POST /api/tasks/{id}/retry", s.handleRetryTask)
	s.mux.HandleFunc("POST /api/tasks/{id}/cancel", s.handleCancelTask)

	s.mux.HandleFunc("GET /api/agents", s.handleListAgents)
	s.mux.HandleFunc("GET /api/agents/{id}", s.handleGetAgent)
	s.mux.HandleFunc("POST /api/agents/{id}/pause", s.handlePauseAgent)
	s.mux.HandleFunc("POST /api/agents/{id}/resume", s.handleResumeAgent)
	s.mux.HandleFunc("POST /api/agents/{id}/stop", s.handleStopAgent)

	s.mux.HandleFunc("GET /api/terminals", s.handleListTerminals)
	s.mux.HandleFunc("POST /api/terminals", s.handleCreateTerminal)
	s.mux.HandleFunc("GET /api/terminals/{id}", s.handleGetTerminal)
	s.mux.HandleFunc("GET /api/terminals/{id}/ws", s.handleTerminalWS)

	s.mux.HandleFunc("GET /api/system/metrics", s.handleSystemMetrics)
	s.mux.HandleFunc("GET /api/system/diagnostics", s.handleDiagnostics)

	s.mux.HandleFunc("GET /api/task-specs", s.handleListTaskSpecs)
	s.mux.HandleFunc("GET /api/agent-specs", s.handleListAgentSpecs)
	s.mux.HandleFunc("GET /api/workflow-templates", s.handleListWorkflowTemplates)
	s.mux.HandleFunc("POST /api/workflow-templates", s.handleCreateWorkflowTemplate)
}

func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.cfg, loggingMiddleware(s.mux))
}

func (s *Server) Start(ctx context.Context) error {
	go s.hub.Run()
	srv := &http.Server{
		Addr:    s.cfg.Server.HTTPAddr,
		Handler: s.Handler(),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	slog.Info("API server starting", "addr", s.cfg.Server.HTTPAddr)
	return srv.ListenAndServe()
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func corsMiddleware(cfg *config.Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := false
		for _, o := range cfg.Server.AllowedOrigins {
			if o == origin || o == "*" {
				allowed = true
				w.Header().Set("Access-Control-Allow-Origin", o)
				break
			}
		}
		if !allowed && len(cfg.Server.AllowedOrigins) > 0 {
			w.Header().Set("Access-Control-Allow-Origin", cfg.Server.AllowedOrigins[0])
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("request body is required")
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
