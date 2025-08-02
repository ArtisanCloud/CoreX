package eino

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ArtisanCloud/CoreX/pkg/agent/contract"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// EinoExecutor 基于 cloudwego/eino 的执行器实现
type EinoExecutor struct {
	config          *Config
	feedbackManager *FeedbackManager

	// eino 核心组件
	chatModel model.ChatModel

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

// NewEinoExecutor 创建基于 eino 的执行器
func NewEinoExecutor(config *Config, feedbackManager *FeedbackManager) (*EinoExecutor, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	executor := &EinoExecutor{
		config:          config,
		feedbackManager: feedbackManager,
		activePlans:     make(map[string]*executionState),
	}

	return executor, nil
}

// Execute 使用 eino 执行整个 plan
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
	e.activePlans[plan.ID] = state
	e.mutex.Unlock()

	// 发送开始执行事件
	e.sendEvent(state, "", "plan_started", map[string]interface{}{
		"plan_id": plan.ID,
		"intent":  plan.Intent,
	})

	// 使用 eino 执行计划
	results, err := e.executeWithEino(ctx, plan, state)
	if err != nil {
		state.status = contract.StatusFailed
		e.sendEvent(state, "", "plan_failed", map[string]interface{}{
			"plan_id": plan.ID,
			"error":   err.Error(),
		})
		return results, err
	}

	// 发送计划完成事件
	e.sendEvent(state, "", "plan_completed", map[string]interface{}{
		"plan_id": plan.ID,
		"success": true,
	})

	state.status = contract.StatusSucceeded
	return results, nil
}

// executeWithEino 使用 eino 框架执行计划
func (e *EinoExecutor) executeWithEino(ctx context.Context, plan *contract.ExecutionPlan, state *executionState) (map[string]*contract.ToolResult, error) {
	// 为每个步骤创建 eino 执行任务
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

		// 发送步骤开始事件
		e.sendEvent(state, step.ID, "step_started", map[string]interface{}{
			"step_name": step.Name,
			"step_type": step.Type,
		})

		// 使用 eino 执行单个步骤
		startTime := time.Now()
		result, err := e.executeStepWithEino(ctx, step, state)
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

		// 如果步骤失败，尝试使用 eino 的错误处理机制
		if err != nil {
			if fallbackResult := e.handleStepFailureWithEino(ctx, step, state, err); fallbackResult != nil {
				state.results[step.ID] = fallbackResult
				e.sendEvent(state, step.ID, "step_fallback_applied", map[string]interface{}{
					"original_error":   err.Error(),
					"fallback_success": fallbackResult.Success,
				})

				if !fallbackResult.Success {
					state.status = contract.StatusFailed
					return state.results, fallbackResult.Error
				}
			} else {
				state.status = contract.StatusFailed
				return state.results, err
			}
		}
	}

	return state.results, nil
}

// executeStepWithEino 使用 eino 执行单个步骤
func (e *EinoExecutor) executeStepWithEino(ctx context.Context, step contract.PlanStep, state *executionState) (map[string]interface{}, error) {
	// 根据步骤类型选择不同的 eino 执行方式
	switch step.Type {
	case "chat", "llm":
		return e.executeChatStepWithEino(ctx, step, state)
	case "tool":
		return e.executeToolStepWithEino(ctx, step, state)
	case "flow":
		return e.executeFlowStepWithEino(ctx, step, state)
	default:
		return nil, fmt.Errorf("不支持的步骤类型: %s", step.Type)
	}
}

// executeChatStepWithEino 使用 eino 执行聊天步骤
func (e *EinoExecutor) executeChatStepWithEino(ctx context.Context, step contract.PlanStep, state *executionState) (map[string]interface{}, error) {
	// 构建聊天消息
	var messages []*schema.Message

	// 添加系统消息（如果有）
	if systemPrompt, exists := step.Input["system_prompt"].(string); exists {
		messages = append(messages, &schema.Message{
			Role:    schema.System,
			Content: systemPrompt,
		})
	}

	// 添加用户消息
	if userMessage, exists := step.Input["message"].(string); exists {
		messages = append(messages, &schema.Message{
			Role:    schema.User,
			Content: userMessage,
		})
	} else {
		return nil, fmt.Errorf("聊天步骤缺少用户消息")
	}

	// 如果有聊天模型，使用 eino 的聊天模型执行
	if e.chatModel != nil {
		// 使用 eino 聊天模型
		response, err := e.chatModel.Generate(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("eino 聊天模型执行失败: %w", err)
		}

		// 提取响应内容
		result := map[string]interface{}{
			"response": response,
			"type":     "chat_response",
			"content":  response.Content,
			"role":     string(response.Role),
		}

		return result, nil
	}

	// 如果没有聊天模型，使用模拟实现
	result := map[string]interface{}{
		"content": fmt.Sprintf("基于 eino 框架处理消息: %s", messages[len(messages)-1].Content),
		"type":    "simulated_chat_response",
		"role":    string(schema.Assistant),
	}

	return result, nil
}

// executeToolStepWithEino 使用 eino 执行工具步骤
func (e *EinoExecutor) executeToolStepWithEino(ctx context.Context, step contract.PlanStep, state *executionState) (map[string]interface{}, error) {
	// 这里需要根据 eino 的工具执行机制来实现
	// 由于 eino 的工具系统可能需要特定的工具注册和调用方式
	// 这里提供一个基础实现框架

	toolName := step.Name
	toolInput := step.Input

	// 基于 eino 框架的工具执行（模拟实现）
	result := map[string]interface{}{
		"tool_name":   toolName,
		"tool_input":  toolInput,
		"tool_output": fmt.Sprintf("使用 eino 框架执行工具 %s 成功", toolName),
		"type":        "tool_result",
		"framework":   "eino",
	}

	return result, nil
}

// executeFlowStepWithEino 使用 eino 执行流程步骤
func (e *EinoExecutor) executeFlowStepWithEino(ctx context.Context, step contract.PlanStep, state *executionState) (map[string]interface{}, error) {
	// 这里需要根据 eino 的流程执行机制来实现
	// 可能涉及到子链的创建和执行

	flowName := step.Name
	flowInput := step.Input

	// 基于 eino 框架的流程执行（模拟实现）
	result := map[string]interface{}{
		"flow_name":   flowName,
		"flow_input":  flowInput,
		"flow_output": fmt.Sprintf("使用 eino 框架执行流程 %s 成功", flowName),
		"type":        "flow_result",
		"framework":   "eino",
	}

	return result, nil
}

// handleStepFailureWithEino 使用 eino 处理步骤失败
func (e *EinoExecutor) handleStepFailureWithEino(ctx context.Context, step contract.PlanStep, state *executionState, originalError error) *contract.ToolResult {
	// 检查是否有降级策略
	if step.Metadata == nil {
		return nil
	}

	fallbackStep, hasFallback := step.Metadata["fallback"].(map[string]interface{})
	if !hasFallback {
		return nil
	}

	// 使用 eino 执行降级策略
	fallbackPlanStep := contract.PlanStep{
		ID:          step.ID + "_fallback",
		Type:        "fallback",
		Name:        "降级: " + step.Name,
		Input:       step.Input,
		ExpectedOut: step.ExpectedOut,
		Metadata:    fallbackStep,
	}

	// 执行降级步骤
	startTime := time.Now()
	result, err := e.executeStepWithEino(ctx, fallbackPlanStep, state)
	duration := time.Since(startTime)

	return &contract.ToolResult{
		Output:   result,
		Success:  err == nil,
		Error:    err,
		Duration: duration,
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

// Cancel 取消正在执行的计划
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

// Status 查询计划当前状态
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

	// 添加 eino 相关状态信息
	snapshot.State["status"] = string(state.status)
	snapshot.State["framework"] = "eino"

	return snapshot, nil
}

// calculateProgress 计算执行进度
func (e *EinoExecutor) calculateProgress(state *executionState) float64 {
	if state.status == contract.StatusSucceeded {
		return 1.0
	}

	if state.status == contract.StatusFailed || state.status == contract.StatusCancelled {
		if len(state.plan.Steps) == 0 {
			return 0.0
		}
		return float64(len(state.results)) / float64(len(state.plan.Steps))
	}

	// 正在执行中
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

	// 这里需要实现从快照恢复 eino 执行状态的逻辑
	// 包括恢复执行上下文等

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

// Close 关闭执行器，清理资源
func (e *EinoExecutor) Close() error {
	// 清理活跃计划
	e.mutex.Lock()
	for planID, state := range e.activePlans {
		close(state.eventChan)
		delete(e.activePlans, planID)
	}
	e.mutex.Unlock()

	return nil
}

// SetChatModel 设置聊天模型
func (e *EinoExecutor) SetChatModel(chatModel model.ChatModel) {
	e.chatModel = chatModel
}

// GetChatModel 获取聊天模型
func (e *EinoExecutor) GetChatModel() model.ChatModel {
	return e.chatModel
}
