# Agent Contract 接口设计文档

## 概述

Agent Contract 定义了智能体系统的核心接口和数据结构，为不同的智能体实现提供统一的契约。这套接口设计支持从意图理解、计划生成到执行和反馈的完整智能体工作流程。

## 核心接口

### Agent 接口

Agent 是最高级别的接口，组合了 Planner 和 Executor 的能力，提供完整的智能体功能。

```go
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
```

### Planner 接口

Planner 负责从意图和上下文生成执行计划，是智能体的"思考"部分。

```go
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
```

### Executor 接口

Executor 负责执行计划，是智能体的"行动"部分。

```go
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
```

## 核心数据结构

### ExecutionPlan

ExecutionPlan 描述智能体要执行的计划，包含多个步骤。

```go
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
```

### PlanStep

PlanStep 表示计划中的一个高阶步骤。

```go
// PlanStep 表示计划中的一个高阶步骤（tool 调用 / 嵌套 flow）。
type PlanStep struct {
	ID          string                 // 本 step 唯一 ID
	Type        string                 // 类型，例如 "tool"、"flow"
	Name        string                 // tool 名称或 flow 名称
	Input       map[string]interface{} // 输入参数
	ExpectedOut map[string]interface{} // 期望输出（用于验证/断言）
	Metadata    map[string]interface{} // step 级上下文
}
```

### ToolResult

ToolResult 封装单个工具或步骤的执行结果。

```go
// ToolResult 封装单个 tool 或 step 的执行结果。
type ToolResult struct {
	Output   map[string]interface{} // 原始输出数据
	Success  bool                   // 是否成功
	Error    error                  // 执行过程中的错误（如果有）
	Duration time.Duration          // 执行耗时
}
```

### ExecutionStatus

ExecutionStatus 表示计划的执行状态。

```go
// ExecutionStatus 表示 plan 的执行状态。
type ExecutionStatus string

const (
	StatusRunning   ExecutionStatus = "running"
	StatusSucceeded ExecutionStatus = "succeeded"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
)
```

### ExecutionSnapshot

ExecutionSnapshot 用于保存中间状态以支持恢复/回放。

```go
// ExecutionSnapshot 用于保存中间状态以支持恢复/回放。
type ExecutionSnapshot struct {
	PlanID     string
	Timestamp  int64                  // Unix 时间戳
	State      map[string]interface{} // 当前上下文/各 step 状态
	Progress   float64                // 完成度估计 0~1
	ToolResult map[string]*ToolResult // 已完成步骤结果
}
```

### ExecutionEvent

ExecutionEvent 表示执行过程中流式事件。

```go
// ExecutionEvent 表示执行过程中流式事件（类似 Eino 中的 signal）。
type ExecutionEvent struct {
	PlanID    string
	StepID    string
	Type      string                 // e.g., "step_started", "step_failed", "retry", "fallback_triggered"
	Detail    map[string]interface{} // 附加信息（error/metrics/reason 等）
	Timestamp int64
}
```

### RetryPolicy

RetryPolicy 定义失败后的重试策略。

```go
// RetryPolicy 定义失败后的重试策略。
type RetryPolicy struct {
	MaxAttempts int           // 包含第一次尝试在内
	Interval    time.Duration // 基础等待时间
	Backoff     bool          // 是否启用指数退避
	Jitter      bool          // 是否加抖动（防止雪崩）
	MaxInterval time.Duration // 最大退避上限（0 表示不限制）
}
```

## 错误处理

定义了常见的错误类型，便于统一错误处理。

```go
var (
	ErrEmptyIntent      = errors.New("intent is empty")
	ErrPlanGeneration   = errors.New("failed to generate execution plan")
	ErrExecutionFailure = errors.New("execution failed")
	ErrInvalidPlan      = errors.New("execution plan invalid")
	ErrFeedbackConflict = errors.New("feedback causes conflict in plan")
)
```

## 设计特点

1. **上下文感知**
   - 所有主要方法都接受context.Context参数
   - 支持超时控制、取消和上下文传递

2. **错误处理**
   - 定义了常见错误类型
   - 提供了详细的错误信息

3. **可观测性**
   - 提供状态查询和指标收集
   - 支持事件订阅和反馈机制

4. **灵活性和可扩展性**
   - 使用map[string]interface{}支持灵活的参数和结果传递
   - 元数据字段允许扩展上下文信息

5. **容错机制**
   - 支持回退计划
   - 提供可配置的重试策略

## 实现示例

以Eino智能体为例，实现了完整的Agent接口：

```go
// EinoAgent 实现 contract.Agent 接口
// 整合计划生成、执行和反馈的完整流程
type EinoAgent struct {
	planner  contract.Planner
	executor contract.Executor
	feedback *feedback.FeedbackManager
}

// NewEinoAgent 创建新的Eino智能体实例
func NewEinoAgent() *EinoAgent {
	feedbackManager := feedback.NewFeedbackManager()
	planner := plan.NewEinoPlanner()
	executor := execution.NewEinoExecutor(feedbackManager)
	
	return &EinoAgent{
		planner:  planner,
		executor: executor,
		feedback: feedbackManager,
	}
}

// 实现Agent接口的方法...
```

## 使用示例

```go
// 创建智能体实例
agent := eino.NewEinoAgent()

// 执行意图
intent := "查找最近三个月的销售数据并生成报表"
contextMetadata := map[string]interface{}{
	"user_id": "user123",
	"tenant_id": "tenant456",
}

// 执行完整流程
results, err := agent.Run(ctx, intent, contextMetadata)
if err != nil {
	log.Fatalf("执行失败: %v", err)
}

// 处理结果
for stepID, result := range results {
	if result.Success {
		fmt.Printf("步骤 %s 执行成功: %v\n", stepID, result.Output)
	} else {
		fmt.Printf("步骤 %s 执行失败: %v\n", stepID, result.Error)
	}
}
```

## 最佳实践

1. **合理使用上下文**
   - 在contextMetadata中传递必要的上下文信息
   - 利用context.Context进行超时控制和取消

2. **错误处理**
   - 检查特定错误类型并采取相应措施
   - 利用RetryPolicy处理临时性故障

3. **反馈机制**
   - 使用Feedback方法优化执行计划
   - 订阅执行事件以获取实时进度

4. **扩展性**
   - 实现自定义Agent以满足特定需求
   - 利用元数据字段传递额外信息