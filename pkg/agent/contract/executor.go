package contract

import "context"

// ExecutionStatus 表示 plan 的执行状态。
type ExecutionStatus string

const (
	StatusRunning   ExecutionStatus = "running"
	StatusSucceeded ExecutionStatus = "succeeded"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
)

// ExecutionSnapshot 用于保存中间状态以支持恢复/回放。
type ExecutionSnapshot struct {
	PlanID     string
	Timestamp  int64                  // Unix 时间戳
	State      map[string]interface{} // 当前上下文/各 step 状态
	Progress   float64                // 完成度估计 0~1
	ToolResult map[string]*ToolResult // 已完成步骤结果
}

// ExecutionEvent 表示执行过程中流式事件（类似 Eino 中的 signal）。
type ExecutionEvent struct {
	PlanID    string
	StepID    string
	Type      string                 // e.g., "step_started", "step_failed", "retry", "fallback_triggered"
	Detail    map[string]interface{} // 附加信息（error/metrics/reason 等）
	Timestamp int64
}

// Executor 负责执行 ExecutionPlan，并提供控制 & 观测能力。
type Executor interface {
	// Execute 执行整个 plan，返回每个 step 的结果。
	Execute(ctx context.Context, plan *ExecutionPlan) (map[string]*ToolResult, error)

	// Cancel 试图取消正在执行的 plan（如果支持可中断）。
	Cancel(ctx context.Context, planID string) error

	// Status 查询 plan 当前状态。
	Status(ctx context.Context, planID string) (ExecutionStatus, error)

	// Snapshot 获取当前执行快照（用于恢复/回放）。
	Snapshot(ctx context.Context, planID string) (*ExecutionSnapshot, error)

	// Resume 从快照恢复执行。
	Resume(ctx context.Context, snapshot *ExecutionSnapshot) (map[string]*ToolResult, error)

	// SubscribeFeedback 订阅执行过程中的事件流（如失败、retry、fallback）。
	SubscribeFeedback(ctx context.Context, planID string) (<-chan *ExecutionEvent, error)
}
