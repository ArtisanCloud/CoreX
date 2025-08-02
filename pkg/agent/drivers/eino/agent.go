package eino

// agent.go - 实现 contract.Agent 与 Planner 的封装（整合 plan→execute→feedback 流程，提供统一调用接口）

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ArtisanCloud/CoreX/pkg/agent/contract"
)

// EinoAgent 实现 contract.Agent 接口
// 整合计划生成、执行和反馈的完整流程
type EinoAgent struct {
	planner         contract.Planner
	executor        contract.Executor
	feedbackManager *FeedbackManager

	// 用于存储执行中的计划状态
	activePlans map[string]*planState
	plansMutex  sync.RWMutex

	// 用于存储指标数据
	metrics      map[string]map[string]interface{}
	metricsMutex sync.RWMutex
}

// planState 存储计划执行状态
type planState struct {
	plan       *contract.ExecutionPlan
	status     contract.ExecutionStatus
	startTime  time.Time
	endTime    time.Time
	results    map[string]*contract.ToolResult
	eventChan  chan *contract.ExecutionEvent
	cancelFunc context.CancelFunc
}

// NewAgentFromConfig 根据配置创建新的Eino智能体实例
func NewAgentFromConfig(ctx context.Context, config *Config) (*EinoAgent, error) {
	// 创建反馈管理器
	feedbackManager := NewFeedbackManager()

	// 创建计划生成器
	planner := NewEinoPlanner(config, feedbackManager)

	// 创建执行器
	executor := NewEinoExecutor(feedbackManager)

	return &EinoAgent{
		planner:         planner,
		executor:        executor,
		feedbackManager: feedbackManager,
		activePlans:     make(map[string]*planState),
		metrics:         make(map[string]map[string]interface{}),
	}, nil
}

// GetPlan 获取 plan（等价于 Planner.Plan）
func (a *EinoAgent) GetPlan(ctx context.Context, intent string, contextMetadata map[string]interface{}) (*contract.ExecutionPlan, error) {
	if intent == "" {
		return nil, contract.ErrEmptyIntent
	}

	// 直接调用 Planner 的 Plan 方法
	return a.planner.Plan(ctx, intent, contextMetadata)
}

// Run 完整从意图到执行（Plan -> Execute）
func (a *EinoAgent) Run(ctx context.Context, intent string, contextMetadata map[string]interface{}) (map[string]*contract.ToolResult, error) {
	// 1. 生成执行计划
	plan, err := a.GetPlan(ctx, intent, contextMetadata)
	if err != nil {
		return nil, err
	}

	// 2. 执行计划
	results, err := a.Execute(ctx, plan)
	if err != nil {
		return nil, err
	}

	// 3. 记录指标
	a.recordMetrics(plan.ID, results)

	return results, nil
}

// Feedback 在已有 plan 和执行结果基础上注入反馈，调整或再生成 plan
func (a *EinoAgent) Feedback(ctx context.Context, plan *contract.ExecutionPlan, results map[string]*contract.ToolResult, feedbackData map[string]interface{}) (*contract.ExecutionPlan, error) {
	// 验证输入
	if plan == nil {
		return nil, errors.New("计划不能为空")
	}

	// 使用反馈管理器生成改进的计划
	return a.feedbackManager.GenerateImprovedPlan(plan, results, feedbackData)
}

// GetMetrics 获取 plan 或 agent 级别的指标（例如成功率、耗时等）
func (a *EinoAgent) GetMetrics(ctx context.Context, planID string) (map[string]interface{}, error) {
	a.metricsMutex.RLock()
	defer a.metricsMutex.RUnlock()

	if planID == "" {
		// 返回全局指标
		globalMetrics := make(map[string]interface{})

		// 计算全局成功率
		totalPlans := len(a.metrics)
		successfulPlans := 0

		for _, metrics := range a.metrics {
			if success, ok := metrics["success"].(bool); ok && success {
				successfulPlans++
			}
		}

		if totalPlans > 0 {
			globalMetrics["success_rate"] = float64(successfulPlans) / float64(totalPlans)
		}

		globalMetrics["total_plans"] = totalPlans
		globalMetrics["successful_plans"] = successfulPlans

		return globalMetrics, nil
	}

	// 返回特定计划的指标
	metrics, exists := a.metrics[planID]
	if !exists {
		return nil, errors.New("找不到指定计划的指标")
	}

	return metrics, nil
}

// Plan 根据意图与上下文生成新的执行计划
func (a *EinoAgent) Plan(ctx context.Context, intent string, contextMetadata map[string]interface{}) (*contract.ExecutionPlan, error) {
	return a.planner.Plan(ctx, intent, contextMetadata)
}

// ValidatePlan 校验一个现有 plan 的合法性（结构、依赖、资源、冲突等）
func (a *EinoAgent) ValidatePlan(ctx context.Context, plan *contract.ExecutionPlan) error {
	return a.planner.ValidatePlan(ctx, plan)
}

// ExplainPlan 返回 plan 的人类可读解释（用于审阅/调试）
func (a *EinoAgent) ExplainPlan(ctx context.Context, plan *contract.ExecutionPlan) (string, error) {
	return a.planner.ExplainPlan(ctx, plan)
}

// RefreshPlan 在已有 plan 基础上根据新上下文做轻量调整（不完整 feedback）
func (a *EinoAgent) RefreshPlan(ctx context.Context, plan *contract.ExecutionPlan, additionalContext map[string]interface{}) (*contract.ExecutionPlan, error) {
	return a.planner.RefreshPlan(ctx, plan, additionalContext)
}

// Execute 执行整个 plan，返回每个 step 的结果
func (a *EinoAgent) Execute(ctx context.Context, plan *contract.ExecutionPlan) (map[string]*contract.ToolResult, error) {
	// 创建可取消的上下文
	execCtx, cancel := context.WithCancel(ctx)

	// 创建并存储计划状态
	state := &planState{
		plan:       plan,
		status:     contract.StatusRunning,
		startTime:  time.Now(),
		results:    make(map[string]*contract.ToolResult),
		eventChan:  make(chan *contract.ExecutionEvent, 100), // 缓冲通道
		cancelFunc: cancel,
	}

	a.plansMutex.Lock()
	a.activePlans[plan.ID] = state
	a.plansMutex.Unlock()

	// 执行计划
	results, err := a.executor.Execute(execCtx, plan)

	// 更新计划状态
	a.plansMutex.Lock()
	state.endTime = time.Now()
	state.results = results

	if err != nil {
		state.status = contract.StatusFailed
	} else {
		state.status = contract.StatusSucceeded
	}

	// 如果通道未关闭，则关闭它
	close(state.eventChan)
	a.plansMutex.Unlock()

	return results, err
}

// Cancel 试图取消正在执行的 plan（如果支持可中断）
func (a *EinoAgent) Cancel(ctx context.Context, planID string) error {
	a.plansMutex.RLock()
	state, exists := a.activePlans[planID]
	a.plansMutex.RUnlock()

	if !exists {
		return errors.New("找不到指定的计划")
	}

	// 调用取消函数
	state.cancelFunc()

	// 更新状态
	a.plansMutex.Lock()
	state.status = contract.StatusCancelled
	state.endTime = time.Now()
	a.plansMutex.Unlock()

	return nil
}

// Status 查询 plan 当前状态
func (a *EinoAgent) Status(ctx context.Context, planID string) (contract.ExecutionStatus, error) {
	a.plansMutex.RLock()
	defer a.plansMutex.RUnlock()

	state, exists := a.activePlans[planID]
	if !exists {
		return "", errors.New("找不到指定的计划")
	}

	return state.status, nil
}

// Snapshot 获取当前执行快照（用于恢复/回放）
func (a *EinoAgent) Snapshot(ctx context.Context, planID string) (*contract.ExecutionSnapshot, error) {
	a.plansMutex.RLock()
	defer a.plansMutex.RUnlock()

	state, exists := a.activePlans[planID]
	if !exists {
		return nil, errors.New("找不到指定的计划")
	}

	// 创建快照
	snapshot := &contract.ExecutionSnapshot{
		PlanID:     planID,
		Timestamp:  time.Now().Unix(),
		State:      make(map[string]interface{}),
		Progress:   a.calculateProgress(state),
		ToolResult: state.results,
	}

	// 添加状态信息
	snapshot.State["status"] = string(state.status)
	snapshot.State["start_time"] = state.startTime.Unix()
	if !state.endTime.IsZero() {
		snapshot.State["end_time"] = state.endTime.Unix()
	}

	return snapshot, nil
}

// Resume 从快照恢复执行
func (a *EinoAgent) Resume(ctx context.Context, snapshot *contract.ExecutionSnapshot) (map[string]*contract.ToolResult, error) {
	if snapshot == nil {
		return nil, errors.New("快照不能为空")
	}

	// 检查计划是否存在
	a.plansMutex.RLock()
	_, exists := a.activePlans[snapshot.PlanID]
	a.plansMutex.RUnlock()

	if exists {
		return nil, errors.New("计划已在执行中，无法从快照恢复")
	}

	// 从存储中获取原始计划
	// 注意：这里假设有一个方法可以获取原始计划
	// 实际实现中，可能需要从数据库或其他存储中获取
	plan, err := a.getPlanFromStorage(snapshot.PlanID)
	if err != nil {
		return nil, err
	}

	// 创建新的执行上下文
	execCtx, cancel := context.WithCancel(ctx)

	// 创建新的计划状态
	state := &planState{
		plan:       plan,
		status:     contract.StatusRunning,
		startTime:  time.Now(),
		results:    snapshot.ToolResult, // 使用快照中的结果
		eventChan:  make(chan *contract.ExecutionEvent, 100),
		cancelFunc: cancel,
	}

	// 存储计划状态
	a.plansMutex.Lock()
	a.activePlans[plan.ID] = state
	a.plansMutex.Unlock()

	// 从快照恢复执行
	results, err := a.executor.Resume(execCtx, snapshot)

	// 更新计划状态
	a.plansMutex.Lock()
	state.endTime = time.Now()
	state.results = results

	if err != nil {
		state.status = contract.StatusFailed
	} else {
		state.status = contract.StatusSucceeded
	}

	close(state.eventChan)
	a.plansMutex.Unlock()

	return results, err
}

// SubscribeFeedback 订阅执行过程中的事件流（如失败、retry、fallback）
func (a *EinoAgent) SubscribeFeedback(ctx context.Context, planID string) (<-chan *contract.ExecutionEvent, error) {
	a.plansMutex.RLock()
	defer a.plansMutex.RUnlock()

	state, exists := a.activePlans[planID]
	if !exists {
		return nil, errors.New("找不到指定的计划")
	}

	return state.eventChan, nil
}

// 计算执行进度
func (a *EinoAgent) calculateProgress(state *planState) float64 {
	if state.status == contract.StatusSucceeded {
		return 1.0
	}

	if state.status == contract.StatusFailed || state.status == contract.StatusCancelled {
		// 根据已完成的步骤估算进度
		if len(state.plan.Steps) == 0 {
			return 0.0
		}
		return float64(len(state.results)) / float64(len(state.plan.Steps))
	}

	// 正在执行中，根据已完成的步骤估算进度
	if len(state.plan.Steps) == 0 {
		return 0.0
	}
	return float64(len(state.results)) / float64(len(state.plan.Steps))
}

// 记录指标
func (a *EinoAgent) recordMetrics(planID string, results map[string]*contract.ToolResult) {
	a.metricsMutex.Lock()
	defer a.metricsMutex.Unlock()

	metrics := make(map[string]interface{})

	// 计算成功率
	totalSteps := len(results)
	successfulSteps := 0

	for _, result := range results {
		if result.Success {
			successfulSteps++
		}
	}

	metrics["total_steps"] = totalSteps
	metrics["successful_steps"] = successfulSteps

	if totalSteps > 0 {
		metrics["step_success_rate"] = float64(successfulSteps) / float64(totalSteps)
	}

	// 计算总体成功状态
	metrics["success"] = (totalSteps > 0) && (successfulSteps == totalSteps)

	// 计算总执行时间
	totalDuration := time.Duration(0)
	for _, result := range results {
		totalDuration += result.Duration
	}
	metrics["total_duration_ms"] = totalDuration.Milliseconds()

	// 存储指标
	a.metrics[planID] = metrics
}

// 从存储中获取计划（示例实现）
func (a *EinoAgent) getPlanFromStorage(planID string) (*contract.ExecutionPlan, error) {
	// 实际实现中，这里应该从数据库或其他存储中获取计划
	// 这里仅作为示例，返回一个错误
	return nil, errors.New("未实现的方法：从存储中获取计划")
}

// GetCapabilities 获取智能体能力描述
func (a *EinoAgent) GetCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"name":        "Eino智能体",
		"description": "基于意图解析和执行计划的智能体实现",
		"version":     "1.0.0",
		"features": []string{
			"意图解析",
			"执行计划生成",
			"多步骤执行",
			"错误处理与回退",
			"反馈优化",
			"执行状态监控",
			"指标收集",
		},
	}
}
