package eino

// execution.go - driver 层执行器：短路(short-circuit)、flow 调度、并行、降级、事件发布（构建在 core flow/node 基础原语之上）

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ArtisanCloud/CoreX/pkg/agent/contract"
)

// EinoExecutor 实现 contract.Executor 接口
type EinoExecutor struct {
	feedbackManager *FeedbackManager

	// 存储执行中的计划
	activePlans map[string]*executionState
	mutex       sync.RWMutex
}

// executionState 存储执行状态
type executionState struct {
	plan      *contract.ExecutionPlan
	status    contract.ExecutionStatus
	results   map[string]*contract.ToolResult
	eventChan chan *contract.ExecutionEvent
}

// NewEinoExecutor 创建新的Eino执行器
func NewEinoExecutor(feedbackManager *FeedbackManager) *EinoExecutor {
	return &EinoExecutor{
		feedbackManager: feedbackManager,
		activePlans:     make(map[string]*executionState),
	}
}

// Execute 执行整个 plan，返回每个 step 的结果
func (e *EinoExecutor) Execute(ctx context.Context, plan *contract.ExecutionPlan) (map[string]*contract.ToolResult, error) {
	if plan == nil {
		return nil, contract.ErrInvalidPlan
	}

	// 创建执行状态
	state := &executionState{
		plan:      plan,
		status:    contract.StatusRunning,
		results:   make(map[string]*contract.ToolResult),
		eventChan: make(chan *contract.ExecutionEvent, 100),
	}

	// 存储执行状态
	e.mutex.Lock()
	if e.activePlans == nil {
		e.activePlans = make(map[string]*executionState)
	}
	e.activePlans[plan.ID] = state
	e.mutex.Unlock()

	// 发送开始执行事件
	e.sendEvent(state, "", "plan_started", map[string]interface{}{
		"plan_id": plan.ID,
		"intent":  plan.Intent,
	})

	// 检查是否有并行执行的步骤
	parallelSteps := e.identifyParallelSteps(plan)

	// 如果有并行步骤，使用并行执行
	if len(parallelSteps) > 0 {
		return e.executeParallel(ctx, plan, state, parallelSteps)
	}

	// 顺序执行每个步骤
	for _, step := range plan.Steps {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			e.sendEvent(state, step.ID, "step_cancelled", map[string]interface{}{
				"reason": "context cancelled",
			})
			state.status = contract.StatusCancelled
			return state.results, ctx.Err()
		default:
			// 继续执行
		}

		// 检查是否应该短路执行
		if e.shouldShortCircuit(state, step) {
			e.sendEvent(state, step.ID, "step_skipped", map[string]interface{}{
				"reason": "short circuit",
			})
			continue
		}

		// 发送步骤开始事件
		e.sendEvent(state, step.ID, "step_started", map[string]interface{}{
			"step_name": step.Name,
			"step_type": step.Type,
		})

		// 执行步骤
		startTime := time.Now()
		result, err := e.executeStep(ctx, step)
		duration := time.Since(startTime)

		// 创建结果
		toolResult := &contract.ToolResult{
			Output:   result,
			Success:  err == nil,
			Error:    err,
			Duration: duration,
		}

		// 存储结果
		state.results[step.ID] = toolResult

		// 发送步骤完成事件
		eventType := "step_completed"
		if err != nil {
			eventType = "step_failed"
		}

		e.sendEvent(state, step.ID, eventType, map[string]interface{}{
			"duration_ms": duration.Milliseconds(),
			"success":     err == nil,
			"error":       err,
		})

		// 如果步骤失败，尝试降级策略
		if err != nil {
			if fallbackResult := e.applyFallbackStrategy(ctx, step); fallbackResult != nil {
				// 降级成功，记录结果并继续
				state.results[step.ID] = fallbackResult
				e.sendEvent(state, step.ID, "step_fallback_applied", map[string]interface{}{
					"original_error":   err.Error(),
					"fallback_success": fallbackResult.Success,
				})

				// 如果降级也失败，停止执行
				if !fallbackResult.Success {
					state.status = contract.StatusFailed
					return state.results, fallbackResult.Error
				}
			} else {
				// 没有降级策略或降级失败，停止执行
				state.status = contract.StatusFailed
				return state.results, err
			}
		}
	}

	// 发送计划完成事件
	e.sendEvent(state, "", "plan_completed", map[string]interface{}{
		"plan_id": plan.ID,
		"success": true,
	})

	state.status = contract.StatusSucceeded
	return state.results, nil
}

// executeParallel 并行执行步骤
func (e *EinoExecutor) executeParallel(ctx context.Context, plan *contract.ExecutionPlan, state *executionState, parallelGroups map[string][]contract.PlanStep) (map[string]*contract.ToolResult, error) {
	// 创建一个等待组
	var wg sync.WaitGroup

	// 创建一个互斥锁，用于保护结果映射
	var resultsMutex sync.Mutex

	// 创建一个错误通道
	errChan := make(chan error, len(plan.Steps))

	// 顺序执行每个步骤组
	for groupID, steps := range parallelGroups {
		// 如果是并行组，并行执行所有步骤
		if groupID != "" {
			e.sendEvent(state, "", "parallel_group_started", map[string]interface{}{
				"group_id":    groupID,
				"steps_count": len(steps),
			})

			// 为每个步骤启动一个goroutine
			for _, step := range steps {
				wg.Add(1)
				go func(s contract.PlanStep) {
					defer wg.Done()

					// 执行步骤
					startTime := time.Now()
					result, err := e.executeStep(ctx, s)
					duration := time.Since(startTime)

					// 创建结果
					toolResult := &contract.ToolResult{
						Output:   result,
						Success:  err == nil,
						Error:    err,
						Duration: duration,
					}

					// 存储结果
					resultsMutex.Lock()
					state.results[s.ID] = toolResult
					resultsMutex.Unlock()

					// 发送步骤完成事件
					eventType := "step_completed"
					if err != nil {
						eventType = "step_failed"
						errChan <- err
					}

					e.sendEvent(state, s.ID, eventType, map[string]interface{}{
						"duration_ms": duration.Milliseconds(),
						"success":     err == nil,
						"error":       err,
						"group_id":    groupID,
					})
				}(step)
			}

			// 等待所有步骤完成
			wg.Wait()

			e.sendEvent(state, "", "parallel_group_completed", map[string]interface{}{
				"group_id": groupID,
			})

			// 检查是否有错误
			select {
			case err := <-errChan:
				state.status = contract.StatusFailed
				return state.results, err
			default:
				// 没有错误，继续执行
			}
		} else {
			// 顺序执行单个步骤
			step := steps[0]

			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				e.sendEvent(state, step.ID, "step_cancelled", map[string]interface{}{
					"reason": "context cancelled",
				})
				state.status = contract.StatusCancelled
				return state.results, ctx.Err()
			default:
				// 继续执行
			}

			// 发送步骤开始事件
			e.sendEvent(state, step.ID, "step_started", map[string]interface{}{
				"step_name": step.Name,
				"step_type": step.Type,
			})

			// 执行步骤
			startTime := time.Now()
			result, err := e.executeStep(ctx, step)
			duration := time.Since(startTime)

			// 创建结果
			toolResult := &contract.ToolResult{
				Output:   result,
				Success:  err == nil,
				Error:    err,
				Duration: duration,
			}

			// 存储结果
			state.results[step.ID] = toolResult

			// 发送步骤完成事件
			eventType := "step_completed"
			if err != nil {
				eventType = "step_failed"
			}

			e.sendEvent(state, step.ID, eventType, map[string]interface{}{
				"duration_ms": duration.Milliseconds(),
				"success":     err == nil,
				"error":       err,
			})

			// 如果步骤失败，停止执行
			if err != nil {
				state.status = contract.StatusFailed
				return state.results, err
			}
		}
	}

	// 发送计划完成事件
	e.sendEvent(state, "", "plan_completed", map[string]interface{}{
		"plan_id": plan.ID,
		"success": true,
	})

	state.status = contract.StatusSucceeded
	return state.results, nil
}

// identifyParallelSteps 识别可以并行执行的步骤
func (e *EinoExecutor) identifyParallelSteps(plan *contract.ExecutionPlan) map[string][]contract.PlanStep {
	// 创建一个映射，用于存储并行组
	parallelGroups := make(map[string][]contract.PlanStep)

	// 遍历所有步骤
	for _, step := range plan.Steps {
		// 检查步骤元数据中是否有并行组标识
		groupID := ""
		if step.Metadata != nil {
			if val, ok := step.Metadata["parallel_group"].(string); ok {
				groupID = val
			}
		}

		// 如果没有并行组标识，创建一个只包含这个步骤的组
		if groupID == "" {
			parallelGroups[""] = append(parallelGroups[""], step)
		} else {
			// 如果有并行组标识，将步骤添加到对应的组
			parallelGroups[groupID] = append(parallelGroups[groupID], step)
		}
	}

	return parallelGroups
}

// shouldShortCircuit 判断是否应该短路执行
func (e *EinoExecutor) shouldShortCircuit(state *executionState, step contract.PlanStep) bool {
	// 检查步骤元数据中是否有短路条件
	if step.Metadata == nil {
		return false
	}

	// 检查是否有依赖步骤
	if dependsOn, ok := step.Metadata["depends_on"].(string); ok {
		// 检查依赖步骤是否成功
		if result, exists := state.results[dependsOn]; exists && !result.Success {
			// 依赖步骤失败，短路当前步骤
			return true
		}
	}

	// 检查是否有条件表达式
	if condition, ok := step.Metadata["condition"].(string); ok {
		// 这里可以实现一个简单的条件表达式求值器
		// 例如，可以检查前面步骤的结果是否满足某些条件

		// 简单示例：如果条件是"skip"，则短路
		if condition == "skip" {
			return true
		}
	}

	return false
}

// applyFallbackStrategy 应用降级策略
func (e *EinoExecutor) applyFallbackStrategy(ctx context.Context, step contract.PlanStep) *contract.ToolResult {
	// 检查步骤元数据中是否有降级策略
	if step.Metadata == nil {
		return nil
	}

	// 检查是否有降级步骤
	fallbackStep, hasFallback := step.Metadata["fallback"].(map[string]interface{})
	if !hasFallback {
		return nil
	}

	// 创建一个降级步骤
	fallbackPlanStep := contract.PlanStep{
		ID:          step.ID + "_fallback",
		Type:        "fallback",
		Name:        "降级: " + step.Name,
		Input:       step.Input,
		ExpectedOut: step.ExpectedOut,
		Metadata:    make(map[string]interface{}),
	}

	// 复制降级步骤的属性
	if fallbackType, ok := fallbackStep["type"].(string); ok {
		fallbackPlanStep.Type = fallbackType
	}
	if fallbackName, ok := fallbackStep["name"].(string); ok {
		fallbackPlanStep.Name = fallbackName
	}
	if fallbackInput, ok := fallbackStep["input"].(map[string]interface{}); ok {
		fallbackPlanStep.Input = fallbackInput
	}

	// 执行降级步骤
	startTime := time.Now()
	result, err := e.executeStep(ctx, fallbackPlanStep)
	duration := time.Since(startTime)

	// 创建结果
	return &contract.ToolResult{
		Output:   result,
		Success:  err == nil,
		Error:    err,
		Duration: duration,
	}
}

// executeStep 执行单个步骤
func (e *EinoExecutor) executeStep(ctx context.Context, step contract.PlanStep) (map[string]interface{}, error) {
	// 这里是一个简单的模拟实现
	// 实际实现中，应该根据步骤类型调用不同的处理逻辑

	switch step.Type {
	case "tool":
		// 模拟工具执行
		return map[string]interface{}{
			"result": "执行工具 " + step.Name + " 成功",
		}, nil
	case "flow":
		// 模拟流程执行
		return map[string]interface{}{
			"result": "执行流程 " + step.Name + " 成功",
		}, nil
	case "fallback":
		// 模拟降级执行
		return map[string]interface{}{
			"result": "执行降级 " + step.Name + " 成功",
		}, nil
	default:
		return nil, errors.New("不支持的步骤类型: " + step.Type)
	}
}

// sendEvent 发送执行事件
func (e *EinoExecutor) sendEvent(state *executionState, stepID string, eventType string, detail map[string]interface{}) {
	event := &contract.ExecutionEvent{
		PlanID:    state.plan.ID,
		StepID:    stepID,
		Type:      eventType,
		Detail:    detail,
		Timestamp: time.Now().Unix(),
	}

	// 尝试发送事件，如果通道已满则丢弃
	select {
	case state.eventChan <- event:
		// 事件已发送
	default:
		// 通道已满，丢弃事件
	}
}

// Cancel 试图取消正在执行的 plan
func (e *EinoExecutor) Cancel(ctx context.Context, planID string) error {
	e.mutex.RLock()
	state, exists := e.activePlans[planID]
	e.mutex.RUnlock()

	if !exists {
		return errors.New("找不到指定的计划")
	}

	// 更新状态
	e.mutex.Lock()
	state.status = contract.StatusCancelled
	e.mutex.Unlock()

	// 发送取消事件
	e.sendEvent(state, "", "plan_cancelled", map[string]interface{}{
		"reason": "user cancelled",
	})

	return nil
}

// Status 查询 plan 当前状态
func (e *EinoExecutor) Status(ctx context.Context, planID string) (contract.ExecutionStatus, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	state, exists := e.activePlans[planID]
	if !exists {
		return "", errors.New("找不到指定的计划")
	}

	return state.status, nil
}

// Snapshot 获取当前执行快照
func (e *EinoExecutor) Snapshot(ctx context.Context, planID string) (*contract.ExecutionSnapshot, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	state, exists := e.activePlans[planID]
	if !exists {
		return nil, errors.New("找不到指定的计划")
	}

	// 创建快照
	snapshot := &contract.ExecutionSnapshot{
		PlanID:     planID,
		Timestamp:  time.Now().Unix(),
		State:      make(map[string]interface{}),
		Progress:   e.calculateProgress(state),
		ToolResult: state.results,
	}

	// 添加状态信息
	snapshot.State["status"] = string(state.status)

	return snapshot, nil
}

// calculateProgress 计算执行进度
func (e *EinoExecutor) calculateProgress(state *executionState) float64 {
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

// Resume 从快照恢复执行
func (e *EinoExecutor) Resume(ctx context.Context, snapshot *contract.ExecutionSnapshot) (map[string]*contract.ToolResult, error) {
	if snapshot == nil {
		return nil, errors.New("快照不能为空")
	}

	// 这里是一个简单的模拟实现
	// 实际实现中，应该从快照中恢复执行状态，并继续执行未完成的步骤

	return snapshot.ToolResult, nil
}

// SubscribeFeedback 订阅执行过程中的事件流
func (e *EinoExecutor) SubscribeFeedback(ctx context.Context, planID string) (<-chan *contract.ExecutionEvent, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	state, exists := e.activePlans[planID]
	if !exists {
		return nil, errors.New("找不到指定的计划")
	}

	return state.eventChan, nil
}
