// Package runtime 的 adapter_registry 文件提供 Agent 适配器的注册和路由功能。
// 根据 agent_kind 返回对应的 AgentRuntime 实现实例。
package runtime

import (
	"fmt"
	"sync"
)

// AdapterRegistry 管理 Agent 类型到运行时适配器的映射，支持线程安全注册。
type AdapterRegistry struct {
	mu       sync.RWMutex
	adapters map[string]AgentRuntime
}

// NewAdapterRegistry 创建注册表并注册内置适配器（Claude Code、通用 Shell）。
func NewAdapterRegistry() *AdapterRegistry {
	r := &AdapterRegistry{
		adapters: make(map[string]AgentRuntime),
	}
	r.Register("claude-code", NewClaudeCodeAdapter())
	r.Register("generic-shell", NewGenericShellAdapter())
	return r
}

// Register 注册一个适配器，若已存在则覆盖。
func (r *AdapterRegistry) Register(kind string, adapter AgentRuntime) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[kind] = adapter
}

// Get 根据 agent_kind 获取对应的运行时适配器，未找到返回错误。
func (r *AdapterRegistry) Get(kind string) (AgentRuntime, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapter, ok := r.adapters[kind]
	if !ok {
		return nil, fmt.Errorf("no adapter registered for agent kind: %s", kind)
	}
	return adapter, nil
}

// GetOrDefault 根据 agent_kind 获取适配器，未找到时返回通用 Shell 适配器。
func (r *AdapterRegistry) GetOrDefault(kind string) AgentRuntime {
	adapter, err := r.Get(kind)
	if err != nil {
		return r.adapters["generic-shell"]
	}
	return adapter
}
