// Package agent_tools 提供智能体工具注册和管理功能
package agent_tools

import (
	"fmt"
)

// Registry 工具注册中心
type Registry struct {
	tools map[string]interface{}
}

// NewRegistry 创建新的工具注册中心
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]interface{}),
	}
}

// Register 注册工具
func (r *Registry) Register(name string, tool interface{}) error {
	if r.tools == nil {
		r.tools = make(map[string]interface{})
	}

	r.tools[name] = tool
	fmt.Printf("工具 %s 注册完成\n", name)
	return nil
}

// Get 获取工具
func (r *Registry) Get(name string) (interface{}, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// List 列出所有已注册的工具
func (r *Registry) List() []string {
	var names []string
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}
