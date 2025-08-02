package contract

import "context"

// Agent 组合 Planner 与 Executor，并提供高阶入口与观测接口。
type Agent interface {
	Planner
	Executor

	// GetPlan 获取 plan（等价于 Planner.Plan）。
	GetPlan(ctx context.Context, intent string, contextMetadata map[string]interface{}) (*ExecutionPlan, error)

	// Run 完整从意图到执行（Plan -> Execute）。
	Run(ctx context.Context, intent string, contextMetadata map[string]interface{}) (map[string]*ToolResult, error)

	// Feedback 在已有 plan 和执行结果基础上注入反馈，调整或再生成 plan。
	Feedback(ctx context.Context, plan *ExecutionPlan, results map[string]*ToolResult, feedbackData map[string]interface{}) (*ExecutionPlan, error)

	// GetMetrics 获取 plan 或 agent 级别的指标（例如成功率、耗时等）。
	GetMetrics(ctx context.Context, planID string) (map[string]interface{}, error)
}
