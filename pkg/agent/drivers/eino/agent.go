package eino

import (
	"context"
	"fmt"
	"time"

	"github.com/ArtisanCloud/CoreX/pkg/agent/contract"
	"github.com/cloudwego/eino/schema"
)

// Agent 基于 cloudwego/eino 的智能代理实现，实现 contract.Agent 接口
type Agent struct {
	config          *Config
	planner         *EinoPlanner
	executor        *EinoExecutor
	feedbackManager *FeedbackManager
}

// NewAgent 创建新的 Agent 实例
func NewAgent(cfg *Config) (*Agent, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	// 创建反馈管理器
	feedbackManager := NewFeedbackManager()

	// 创建计划生成器
	planner := NewEinoPlanner(cfg, feedbackManager)

	// 创建执行器
	executor, err := NewEinoExecutor(cfg, feedbackManager)
	if err != nil {
		return nil, fmt.Errorf("创建执行器失败: %w", err)
	}

	agent := &Agent{
		config:          cfg,
		planner:         planner,
		executor:        executor,
		feedbackManager: feedbackManager,
	}

	return agent, nil
}

// ========== 实现 contract.Planner 接口 ==========

// Plan 根据意图与上下文生成新的执行计划
func (a *Agent) Plan(ctx context.Context, intent string, contextMetadata map[string]interface{}) (*contract.ExecutionPlan, error) {
	return a.planner.Plan(ctx, intent, contextMetadata)
}

// ValidatePlan 校验一个现有 plan 的合法性
func (a *Agent) ValidatePlan(ctx context.Context, plan *contract.ExecutionPlan) error {
	return a.planner.ValidatePlan(ctx, plan)
}

// ExplainPlan 返回 plan 的人类可读解释
func (a *Agent) ExplainPlan(ctx context.Context, plan *contract.ExecutionPlan) (string, error) {
	return a.planner.ExplainPlan(ctx, plan)
}

// RefreshPlan 在已有 plan 基础上根据新上下文做轻量调整
func (a *Agent) RefreshPlan(ctx context.Context, plan *contract.ExecutionPlan, additionalContext map[string]interface{}) (*contract.ExecutionPlan, error) {
	return a.planner.RefreshPlan(ctx, plan, additionalContext)
}

// ========== 实现 contract.Executor 接口 ==========

// Execute 执行整个 plan，返回每个 step 的结果
func (a *Agent) Execute(ctx context.Context, plan *contract.ExecutionPlan) (map[string]*contract.ToolResult, error) {
	return a.executor.Execute(ctx, plan)
}

// Cancel 试图取消正在执行的 plan
func (a *Agent) Cancel(ctx context.Context, planID string) error {
	return a.executor.Cancel(ctx, planID)
}

// Status 查询 plan 当前状态
func (a *Agent) Status(ctx context.Context, planID string) (contract.ExecutionStatus, error) {
	return a.executor.Status(ctx, planID)
}

// Snapshot 获取当前执行快照
func (a *Agent) Snapshot(ctx context.Context, planID string) (*contract.ExecutionSnapshot, error) {
	return a.executor.Snapshot(ctx, planID)
}

// Resume 从快照恢复执行
func (a *Agent) Resume(ctx context.Context, snapshot *contract.ExecutionSnapshot) (map[string]*contract.ToolResult, error) {
	return a.executor.Resume(ctx, snapshot)
}

// SubscribeFeedback 订阅执行过程中的事件流
func (a *Agent) SubscribeFeedback(ctx context.Context, planID string) (<-chan *contract.ExecutionEvent, error) {
	return a.executor.SubscribeFeedback(ctx, planID)
}

// ========== 实现 contract.Agent 接口的高阶方法 ==========

// GetPlan 获取 plan（等价于 Planner.Plan）
func (a *Agent) GetPlan(ctx context.Context, intent string, contextMetadata map[string]interface{}) (*contract.ExecutionPlan, error) {
	return a.Plan(ctx, intent, contextMetadata)
}

// Run 完整从意图到执行（Plan -> Execute）
func (a *Agent) Run(ctx context.Context, intent string, contextMetadata map[string]interface{}) (map[string]*contract.ToolResult, error) {
	// 第一步：生成执行计划
	plan, err := a.Plan(ctx, intent, contextMetadata)
	if err != nil {
		return nil, fmt.Errorf("生成执行计划失败: %w", err)
	}

	// 第二步：验证计划
	if err := a.ValidatePlan(ctx, plan); err != nil {
		return nil, fmt.Errorf("计划验证失败: %w", err)
	}

	// 第三步：执行计划
	results, err := a.Execute(ctx, plan)
	if err != nil {
		return nil, fmt.Errorf("执行计划失败: %w", err)
	}

	return results, nil
}

// Feedback 在已有 plan 和执行结果基础上注入反馈，调整或再生成 plan
func (a *Agent) Feedback(ctx context.Context, plan *contract.ExecutionPlan, results map[string]*contract.ToolResult, feedbackData map[string]interface{}) (*contract.ExecutionPlan, error) {
	if plan == nil {
		return nil, fmt.Errorf("计划不能为空")
	}

	if results == nil {
		return nil, fmt.Errorf("执行结果不能为空")
	}

	// 使用反馈管理器生成改进的计划
	improvedPlan, err := a.feedbackManager.GenerateImprovedPlan(plan, results, feedbackData)
	if err != nil {
		return nil, fmt.Errorf("生成改进计划失败: %w", err)
	}

	// 验证改进后的计划
	if err := a.ValidatePlan(ctx, improvedPlan); err != nil {
		return nil, fmt.Errorf("改进计划验证失败: %w", err)
	}

	return improvedPlan, nil
}

// GetMetrics 获取 plan 或 agent 级别的指标
func (a *Agent) GetMetrics(ctx context.Context, planID string) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// 获取执行状态
	status, err := a.Status(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("获取执行状态失败: %w", err)
	}

	metrics["status"] = string(status)

	// 获取执行快照
	snapshot, err := a.Snapshot(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("获取执行快照失败: %w", err)
	}

	if snapshot != nil {
		metrics["progress"] = snapshot.Progress
		metrics["timestamp"] = snapshot.Timestamp
		metrics["total_steps"] = len(snapshot.ToolResult)

		// 计算成功率
		successCount := 0
		totalDuration := time.Duration(0)
		for _, result := range snapshot.ToolResult {
			if result.Success {
				successCount++
			}
			totalDuration += result.Duration
		}

		if len(snapshot.ToolResult) > 0 {
			metrics["success_rate"] = float64(successCount) / float64(len(snapshot.ToolResult))
			metrics["average_duration_ms"] = totalDuration.Milliseconds() / int64(len(snapshot.ToolResult))
		} else {
			metrics["success_rate"] = 0.0
			metrics["average_duration_ms"] = 0
		}

		metrics["successful_steps"] = successCount
		metrics["total_duration_ms"] = totalDuration.Milliseconds()
	}

	return metrics, nil
}

// ========== 基于 cloudwego/eino 的扩展方法 ==========

// Chat 使用 eino 进行对话（简化接口）
func (a *Agent) Chat(ctx context.Context, message string) (*schema.Message, error) {
	if message == "" {
		return nil, fmt.Errorf("消息不能为空")
	}

	// 将对话转换为意图执行
	contextMetadata := map[string]interface{}{
		"message_type": "chat",
		"timestamp":    time.Now().Unix(),
	}

	// 运行完整的意图到执行流程
	results, err := a.Run(ctx, message, contextMetadata)
	if err != nil {
		return nil, fmt.Errorf("对话执行失败: %w", err)
	}

	// 从结果中提取响应
	response := &schema.Message{
		Role:    schema.Assistant,
		Content: a.formatChatResponse(results),
	}

	return response, nil
}

// StreamChat 流式对话
func (a *Agent) StreamChat(ctx context.Context, message string) (<-chan *schema.Message, error) {
	// 检查是否启用流式模式
	streamMode, _ := a.config.GetOptionWithDefault("stream_mode", false).(bool)
	if !streamMode {
		return nil, fmt.Errorf("流式模式未启用")
	}

	if message == "" {
		return nil, fmt.Errorf("消息不能为空")
	}

	// 创建结果通道
	resultChan := make(chan *schema.Message, 10)

	// 启动异步处理
	go func() {
		defer close(resultChan)

		// 生成执行计划
		contextMetadata := map[string]interface{}{
			"message_type": "stream_chat",
			"timestamp":    time.Now().Unix(),
		}

		plan, err := a.Plan(ctx, message, contextMetadata)
		if err != nil {
			resultChan <- &schema.Message{
				Role:    schema.Assistant,
				Content: fmt.Sprintf("计划生成失败: %v", err),
			}
			return
		}

		// 订阅执行事件
		eventChan, err := a.SubscribeFeedback(ctx, plan.ID)
		if err != nil {
			resultChan <- &schema.Message{
				Role:    schema.Assistant,
				Content: fmt.Sprintf("订阅执行事件失败: %v", err),
			}
			return
		}

		// 启动执行
		go func() {
			_, execErr := a.Execute(ctx, plan)
			if execErr != nil {
				resultChan <- &schema.Message{
					Role:    schema.Assistant,
					Content: fmt.Sprintf("执行失败: %v", execErr),
				}
			}
		}()

		// 处理执行事件并转换为流式消息
		for event := range eventChan {
			if event != nil {
				content := a.formatStreamEvent(event)
				if content != "" {
					resultChan <- &schema.Message{
						Role:    schema.Assistant,
						Content: content,
					}
				}
			}
		}
	}()

	return resultChan, nil
}

// formatChatResponse 格式化对话响应
func (a *Agent) formatChatResponse(results map[string]*contract.ToolResult) string {
	if len(results) == 0 {
		return "没有执行结果"
	}

	// 简单的响应格式化
	response := "执行完成："
	for stepID, result := range results {
		if result.Success {
			if output, ok := result.Output["result"].(string); ok {
				response += fmt.Sprintf("\n- %s: %s", stepID, output)
			} else {
				response += fmt.Sprintf("\n- %s: 执行成功", stepID)
			}
		} else {
			response += fmt.Sprintf("\n- %s: 执行失败 - %v", stepID, result.Error)
		}
	}

	return response
}

// formatStreamEvent 格式化流式事件
func (a *Agent) formatStreamEvent(event *contract.ExecutionEvent) string {
	switch event.Type {
	case "step_started":
		if stepName, ok := event.Detail["step_name"].(string); ok {
			return fmt.Sprintf("开始执行: %s", stepName)
		}
	case "step_completed":
		if stepName, ok := event.Detail["step_name"].(string); ok {
			return fmt.Sprintf("完成执行: %s", stepName)
		}
	case "step_failed":
		if stepName, ok := event.Detail["step_name"].(string); ok {
			return fmt.Sprintf("执行失败: %s", stepName)
		}
	case "plan_completed":
		return "所有步骤执行完成"
	}
	return ""
}

// GetConfig 获取配置
func (a *Agent) GetConfig() *Config {
	return a.config
}

// Close 关闭 Agent，释放资源
func (a *Agent) Close() error {
	// 清理资源
	return nil
}

// UpdateConfig 更新配置
func (a *Agent) UpdateConfig(key string, value interface{}) error {
	a.config.WithOption(key, value)
	return nil
}

// ValidateConfig 验证配置
func (a *Agent) ValidateConfig() error {
	if a.config == nil {
		return fmt.Errorf("配置不能为空")
	}

	// 验证必要的配置项
	if modelName, exists := a.config.GetOption("model_name"); exists {
		if name, ok := modelName.(string); !ok || name == "" {
			return fmt.Errorf("模型名称必须是非空字符串")
		}
	}

	if temperature, exists := a.config.GetOption("temperature"); exists {
		if temp, ok := temperature.(float64); !ok || temp < 0 || temp > 2 {
			return fmt.Errorf("温度参数必须是 0-2 之间的数值")
		}
	}

	if maxTokens, exists := a.config.GetOption("max_tokens"); exists {
		if tokens, ok := maxTokens.(int); !ok || tokens <= 0 {
			return fmt.Errorf("最大令牌数必须是正整数")
		}
	}

	return nil
}

// GetSupportedTools 获取支持的工具列表
func (a *Agent) GetSupportedTools() []string {
	return []string{
		"search",        // 搜索工具
		"calculator",    // 计算器工具
		"code_executor", // 代码执行工具
		"web_scraper",   // 网页抓取工具
		"file_reader",   // 文件读取工具
		"database",      // 数据库查询工具
		"api_caller",    // API 调用工具
	}
}

// ProcessWithEino 使用 eino 框架处理消息
func (a *Agent) ProcessWithEino(ctx context.Context, input *schema.Message) (*schema.Message, error) {
	if input == nil {
		return nil, fmt.Errorf("输入消息不能为空")
	}

	// 将 schema.Message 转换为意图执行
	contextMetadata := map[string]interface{}{
		"message_role":    string(input.Role),
		"processing_type": "eino_framework",
		"timestamp":       time.Now().Unix(),
	}

	// 执行完整的意图到执行流程
	results, err := a.Run(ctx, input.Content, contextMetadata)
	if err != nil {
		return nil, fmt.Errorf("eino 框架处理失败: %w", err)
	}

	// 创建响应消息
	response := &schema.Message{
		Role:    schema.Assistant,
		Content: fmt.Sprintf("使用 eino 框架处理完成: %s", a.formatChatResponse(results)),
	}

	return response, nil
}
