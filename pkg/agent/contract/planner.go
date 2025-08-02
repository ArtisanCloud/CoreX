package contract

import "context"

// Planner 负责从意图/上下文生成和调整 ExecutionPlan。
type Planner interface {
	// Plan 根据意图与上下文生成新的执行计划。
	Plan(ctx context.Context, intent string, contextMetadata map[string]interface{}) (*ExecutionPlan, error)

	// ValidatePlan 校验一个现有 plan 的合法性（结构、依赖、资源、冲突等）。
	ValidatePlan(ctx context.Context, plan *ExecutionPlan) error

	// ExplainPlan 返回 plan 的人类可读解释（用于审阅/调试）。
	ExplainPlan(ctx context.Context, plan *ExecutionPlan) (string, error)

	// RefreshPlan 在已有 plan 基础上根据新上下文做轻量调整（不完整 feedback）。
	RefreshPlan(ctx context.Context, plan *ExecutionPlan, additionalContext map[string]interface{}) (*ExecutionPlan, error)
}
