// Package api 提供 HTTP API 服务器，包括路由注册、中间件和辅助函数。
// 它将 HTTP 请求转发给对应的处理器（handlers.go）并管理 WebSocket 连接。
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
	agentruntime "github.com/xxy757/xxyCodingAgents/internal/runtime"
	"github.com/xxy757/xxyCodingAgents/internal/storage"
	"github.com/xxy757/xxyCodingAgents/internal/terminal"
)

// Server 是 API 服务器，聚合了配置、数据库、路由和各类管理器。
type Server struct {
	cfg             *config.Config                    // 应用配置
	db              *storage.DB                       // 数据库连接
	repos           *storage.Repos                    // 数据仓库集合
	mux             *http.ServeMux                    // HTTP 路由多路复用器
	hub             *WebSocketHub                     // WebSocket 广播中心
	orch            *orchestrator.Orchestrator        // 编排器
	termMgr         *terminal.Manager                 // 终端会话管理器
	runtimeRegistry *agentruntime.AdapterRegistry     // Agent 运行时注册表
}

// NewServer 创建并初始化 API 服务器，注册所有路由。
func NewServer(cfg *config.Config, db *storage.DB, repos *storage.Repos, orch *orchestrator.Orchestrator, termMgr *terminal.Manager, registry *agentruntime.AdapterRegistry) *Server {
	s := &Server{
		cfg:             cfg,
		db:              db,
		repos:           repos,
		mux:             http.NewServeMux(),
		hub:             NewWebSocketHub(),
		orch:            orch,
		termMgr:         termMgr,
		runtimeRegistry: registry,
	}
	s.setupRoutes()
	return s
}

// setupRoutes 注册所有 API 路由，包括健康检查、项目、运行、任务、Agent、终端和系统管理端点。
func (s *Server) setupRoutes() {
	// 健康检查端点
	s.mux.HandleFunc("/healthz", s.handleHealthz)
	s.mux.HandleFunc("/readyz", s.handleReadyz)

	// 项目管理 API
	s.mux.HandleFunc("POST /api/projects", s.handleCreateProject)
	s.mux.HandleFunc("GET /api/projects", s.handleListProjects)
	s.mux.HandleFunc("GET /api/projects/{id}", s.handleGetProject)

	// 运行管理 API
	s.mux.HandleFunc("POST /api/runs", s.handleCreateRun)
	s.mux.HandleFunc("GET /api/runs", s.handleListAllRuns)
	s.mux.HandleFunc("GET /api/runs/{id}", s.handleGetRun)
	s.mux.HandleFunc("GET /api/runs/{id}/timeline", s.handleGetRunTimeline)
	s.mux.HandleFunc("GET /api/projects/{id}/runs", s.handleListRuns)

	// 任务管理 API
	s.mux.HandleFunc("GET /api/runs/{id}/tasks", s.handleListTasks)
	s.mux.HandleFunc("GET /api/runs/{id}/workflow", s.handleRunWorkflow)
	s.mux.HandleFunc("POST /api/tasks/{id}/retry", s.handleRetryTask)
	s.mux.HandleFunc("POST /api/tasks/{id}/cancel", s.handleCancelTask)

	// Agent 管理 API
	s.mux.HandleFunc("GET /api/agents", s.handleListAgents)
	s.mux.HandleFunc("GET /api/agents/{id}", s.handleGetAgent)
	s.mux.HandleFunc("POST /api/agents/{id}/pause", s.handlePauseAgent)
	s.mux.HandleFunc("POST /api/agents/{id}/resume", s.handleResumeAgent)
	s.mux.HandleFunc("POST /api/agents/{id}/stop", s.handleStopAgent)

	// 终端管理 API
	s.mux.HandleFunc("GET /api/terminals", s.handleListTerminals)
	s.mux.HandleFunc("POST /api/terminals", s.handleCreateTerminal)
	s.mux.HandleFunc("GET /api/terminals/{id}", s.handleGetTerminal)
	s.mux.HandleFunc("GET /api/terminals/{id}/ws", s.handleTerminalWS)

	// 系统监控 API
	s.mux.HandleFunc("GET /api/system/metrics", s.handleSystemMetrics)
	s.mux.HandleFunc("GET /api/system/diagnostics", s.handleDiagnostics)

	// 规格和模板 API
	s.mux.HandleFunc("GET /api/task-specs", s.handleListTaskSpecs)
	s.mux.HandleFunc("GET /api/agent-specs", s.handleListAgentSpecs)
	s.mux.HandleFunc("GET /api/workflow-templates", s.handleListWorkflowTemplates)
	s.mux.HandleFunc("POST /api/workflow-templates", s.handleCreateWorkflowTemplate)

	// 提示词草稿 API
	s.mux.HandleFunc("GET /api/tech-stacks", s.handleListTechStacks)
	s.mux.HandleFunc("POST /api/prompt-drafts/generate", s.handleGeneratePromptDraft)
	s.mux.HandleFunc("GET /api/prompt-drafts", s.handleListPromptDrafts)
	s.mux.HandleFunc("PUT /api/prompt-drafts/{id}", s.handleUpdatePromptDraft)
	s.mux.HandleFunc("POST /api/prompt-drafts/{id}/send", s.handleSendPromptDraft)

	// 质量门禁 API
	s.mux.HandleFunc("POST /api/gates/{id}/approve", s.handleApproveGate)
	s.mux.HandleFunc("GET /api/gates", s.handleListGates)
	s.mux.HandleFunc("GET /api/gates/{id}", s.handleGetGate)
}

// Handler 返回包装了 CORS 和日志中间件的 HTTP Handler。
func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.cfg, loggingMiddleware(s.mux))
}

// Start 启动 HTTP 服务器，并在上下文取消时优雅关闭。
func (s *Server) Start(ctx context.Context) error {
	go s.hub.Run()
	srv := &http.Server{
		Addr:    s.cfg.Server.HTTPAddr,
		Handler: s.Handler(),
	}

	// 监听上下文取消信号，触发优雅关闭
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	slog.Info("API server starting", "addr", s.cfg.Server.HTTPAddr)
	return srv.ListenAndServe()
}

// handleHealthz 处理存活探针请求，始终返回 OK。
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReadyz 处理就绪探针请求，检查数据库连接是否正常。
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

// corsMiddleware 处理跨域请求，根据配置设置 Access-Control-Allow-Origin 响应头。
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
		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware 记录每个请求的方法、路径和耗时。
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start))
	})
}

// writeJSON 将数据序列化为 JSON 并写入响应。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError 写入错误响应。
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// readJSON 从请求体解码 JSON 数据。
func readJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("request body is required")
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
