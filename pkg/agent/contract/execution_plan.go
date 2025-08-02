package contract

import "time"

// ExecutionPlan 描述智能体要执行的计划，包含多个步骤。
type ExecutionPlan struct {
	ID           string                 // 唯一标识
	Intent       string                 // 原始意图（自然语言或结构化意图）
	CreatedAt    time.Time              // 生成时间
	Steps        []PlanStep             // 细化步骤
	Metadata     map[string]interface{} // 扩展上下文
	Priority     int                    // 优先级（越大越高）
	FallbackPlan *ExecutionPlan         // 回退计划（可为空）
	RetryPolicy  *RetryPolicy           // 重试策略（可为空）
}

// PlanStep 表示计划中的一个高阶步骤（tool 调用 / 嵌套 flow）。
type PlanStep struct {
	ID          string                 // 本 step 唯一 ID
	Type        string                 // 类型，例如 "tool"、"flow"
	Name        string                 // tool 名称或 flow 名称
	Input       map[string]interface{} // 输入参数
	ExpectedOut map[string]interface{} // 期望输出（用于验证/断言）
	Metadata    map[string]interface{} // step 级上下文
}
