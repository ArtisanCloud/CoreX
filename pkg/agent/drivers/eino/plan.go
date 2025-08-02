package eino

import (
	"context"
	"fmt"
	"time"

	"github.com/ArtisanCloud/CoreX/pkg/agent/contract"
	"github.com/cloudwego/eino/schema"
)

// EinoPlanner 基于 cloudwego/eino 的计划生成器实现
type EinoPlanner struct {
	config          *Config
	feedbackManager *FeedbackManager
}

// NewEinoPlanner 创建基于 eino 的计划生成器
func NewEinoPlanner(config *Config, feedbackManager *FeedbackManager) *EinoPlanner {
	return &EinoPlanner{
		config:          config,
		feedbackManager: feedbackManager,
	}
}

// Plan 根据意图与上下文生成新的执行计划
func (p *EinoPlanner) Plan(ctx context.Context, intent string, contextMetadata map[string]interface{}) (*contract.ExecutionPlan, error) {
	if intent == "" {
		return nil, fmt.Errorf("意图不能为空")
	}

	// 使用 eino 框架分析意图并生成计划
	plan, err := p.generatePlanWithEino(ctx, intent, contextMetadata)
	if err != nil {
		return nil, fmt.Errorf("使用 eino 生成计划失败: %w", err)
	}

	return plan, nil
}

// generatePlanWithEino 使用 eino 框架生成执行计划
func (p *EinoPlanner) generatePlanWithEino(ctx context.Context, intent string, contextMetadata map[string]interface{}) (*contract.ExecutionPlan, error) {
	// 生成唯一的计划ID
	planID := fmt.Sprintf("eino_plan_%d", time.Now().UnixNano())

	// 创建基础计划
	plan := &contract.ExecutionPlan{
		ID:        planID,
		Intent:    intent,
		CreatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
		Priority:  1,
	}

	// 添加上下文元数据
	if contextMetadata != nil {
		for k, v := range contextMetadata {
			plan.Metadata[k] = v
		}
	}

	// 添加 eino 框架标识
	plan.Metadata["framework"] = "eino"
	plan.Metadata["planner"] = "EinoPlanner"

	// 使用 eino 分析意图并生成步骤
	steps, err := p.analyzeIntentWithEino(ctx, intent, contextMetadata)
	if err != nil {
		return nil, fmt.Errorf("使用 eino 分析意图失败: %w", err)
	}

	plan.Steps = steps

	// 设置重试策略
	plan.RetryPolicy = &contract.RetryPolicy{
		MaxAttempts: 3,
		Interval:    time.Second * 2,
		Backoff:     true,
		Jitter:      true,
		MaxInterval: time.Second * 30,
	}

	return plan, nil
}

// analyzeIntentWithEino 使用 eino 分析意图并生成步骤
func (p *EinoPlanner) analyzeIntentWithEino(ctx context.Context, intent string, contextMetadata map[string]interface{}) ([]contract.PlanStep, error) {
	var steps []contract.PlanStep

	// 基于 eino 的意图分析
	// 这里我们根据意图的内容和类型来决定生成什么样的步骤

	// 检查是否是聊天类型的意图
	if p.isChatIntent(intent) {
		step := contract.PlanStep{
			ID:   fmt.Sprintf("chat_step_%d", time.Now().UnixNano()),
			Type: "chat",
			Name: "eino_chat",
			Input: map[string]interface{}{
				"message": intent,
			},
			ExpectedOut: map[string]interface{}{
				"response": "string",
			},
			Metadata: map[string]interface{}{
				"framework": "eino",
				"step_type": "chat",
			},
		}

		// 添加系统提示词（如果配置中有）
		if systemPrompt, exists := p.config.GetOption("system_prompt"); exists {
			step.Input["system_prompt"] = systemPrompt
		}

		steps = append(steps, step)
	}

	// 检查是否需要工具调用
	if p.needsToolCall(intent) {
		toolStep := contract.PlanStep{
			ID:   fmt.Sprintf("tool_step_%d", time.Now().UnixNano()),
			Type: "tool",
			Name: p.determineToolName(intent),
			Input: map[string]interface{}{
				"query": intent,
			},
			ExpectedOut: map[string]interface{}{
				"result": "object",
			},
			Metadata: map[string]interface{}{
				"framework": "eino",
				"step_type": "tool",
			},
		}

		steps = append(steps, toolStep)
	}

	// 检查是否需要流程执行
	if p.needsFlowExecution(intent) {
		flowStep := contract.PlanStep{
			ID:   fmt.Sprintf("flow_step_%d", time.Now().UnixNano()),
			Type: "flow",
			Name: "eino_flow",
			Input: map[string]interface{}{
				"intent":  intent,
				"context": contextMetadata,
			},
			ExpectedOut: map[string]interface{}{
				"result": "object",
			},
			Metadata: map[string]interface{}{
				"framework": "eino",
				"step_type": "flow",
			},
		}

		steps = append(steps, flowStep)
	}

	// 如果没有生成任何步骤，创建一个默认的聊天步骤
	if len(steps) == 0 {
		defaultStep := contract.PlanStep{
			ID:   fmt.Sprintf("default_step_%d", time.Now().UnixNano()),
			Type: "chat",
			Name: "eino_default_chat",
			Input: map[string]interface{}{
				"message": intent,
			},
			ExpectedOut: map[string]interface{}{
				"response": "string",
			},
			Metadata: map[string]interface{}{
				"framework": "eino",
				"step_type": "default_chat",
			},
		}

		steps = append(steps, defaultStep)
	}

	return steps, nil
}

// isChatIntent 判断是否是聊天类型的意图
func (p *EinoPlanner) isChatIntent(intent string) bool {
	// 简单的启发式规则来判断是否是聊天意图
	chatKeywords := []string{
		"你好", "介绍", "解释", "什么是", "如何", "为什么",
		"请问", "告诉我", "帮我", "说说", "聊聊",
	}

	for _, keyword := range chatKeywords {
		if contains(intent, keyword) {
			return true
		}
	}

	return true // 默认认为是聊天意图
}

// needsToolCall 判断是否需要工具调用
func (p *EinoPlanner) needsToolCall(intent string) bool {
	// 检查意图中是否包含需要工具的关键词
	toolKeywords := []string{
		"搜索", "查找", "计算", "翻译", "天气",
		"股价", "新闻", "数据", "分析", "查询",
	}

	for _, keyword := range toolKeywords {
		if contains(intent, keyword) {
			return true
		}
	}

	return false
}

// needsFlowExecution 判断是否需要流程执行
func (p *EinoPlanner) needsFlowExecution(intent string) bool {
	// 检查意图中是否包含需要复杂流程的关键词
	flowKeywords := []string{
		"流程", "步骤", "处理", "执行", "任务",
		"工作流", "批量", "自动化", "管道",
	}

	for _, keyword := range flowKeywords {
		if contains(intent, keyword) {
			return true
		}
	}

	return false
}

// determineToolName 根据意图确定工具名称
func (p *EinoPlanner) determineToolName(intent string) string {
	if contains(intent, "搜索") || contains(intent, "查找") {
		return "search"
	}
	if contains(intent, "计算") {
		return "calculator"
	}
	if contains(intent, "翻译") {
		return "translator"
	}
	if contains(intent, "天气") {
		return "weather"
	}
	if contains(intent, "股价") {
		return "stock"
	}
	if contains(intent, "新闻") {
		return "news"
	}

	return "general_tool"
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsInMiddle(s, substr))))
}

// containsInMiddle 检查字符串中间是否包含子字符串
func containsInMiddle(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ValidatePlan 校验一个现有 plan 的合法性
func (p *EinoPlanner) ValidatePlan(ctx context.Context, plan *contract.ExecutionPlan) error {
	if plan == nil {
		return fmt.Errorf("计划不能为空")
	}

	if plan.ID == "" {
		return fmt.Errorf("计划ID不能为空")
	}

	if plan.Intent == "" {
		return fmt.Errorf("计划意图不能为空")
	}

	if len(plan.Steps) == 0 {
		return fmt.Errorf("计划必须包含至少一个步骤")
	}

	// 验证每个步骤
	for i, step := range plan.Steps {
		if err := p.validateStep(step); err != nil {
			return fmt.Errorf("步骤 %d 验证失败: %w", i, err)
		}
	}

	// 验证步骤之间的依赖关系
	if err := p.validateStepDependencies(plan.Steps); err != nil {
		return fmt.Errorf("步骤依赖关系验证失败: %w", err)
	}

	return nil
}

// validateStep 验证单个步骤
func (p *EinoPlanner) validateStep(step contract.PlanStep) error {
	if step.ID == "" {
		return fmt.Errorf("步骤ID不能为空")
	}

	if step.Type == "" {
		return fmt.Errorf("步骤类型不能为空")
	}

	if step.Name == "" {
		return fmt.Errorf("步骤名称不能为空")
	}

	// 验证支持的步骤类型
	supportedTypes := []string{"chat", "tool", "flow", "fallback"}
	typeSupported := false
	for _, supportedType := range supportedTypes {
		if step.Type == supportedType {
			typeSupported = true
			break
		}
	}

	if !typeSupported {
		return fmt.Errorf("不支持的步骤类型: %s", step.Type)
	}

	return nil
}

// validateStepDependencies 验证步骤依赖关系
func (p *EinoPlanner) validateStepDependencies(steps []contract.PlanStep) error {
	stepIDs := make(map[string]bool)

	// 收集所有步骤ID
	for _, step := range steps {
		stepIDs[step.ID] = true
	}

	// 检查依赖关系
	for _, step := range steps {
		if step.Metadata != nil {
			if dependsOn, exists := step.Metadata["depends_on"].(string); exists {
				if !stepIDs[dependsOn] {
					return fmt.Errorf("步骤 %s 依赖的步骤 %s 不存在", step.ID, dependsOn)
				}
			}
		}
	}

	return nil
}

// ExplainPlan 返回 plan 的人类可读解释
func (p *EinoPlanner) ExplainPlan(ctx context.Context, plan *contract.ExecutionPlan) (string, error) {
	if plan == nil {
		return "", fmt.Errorf("计划不能为空")
	}

	explanation := fmt.Sprintf("基于 eino 框架的执行计划:\n")
	explanation += fmt.Sprintf("计划ID: %s\n", plan.ID)
	explanation += fmt.Sprintf("意图: %s\n", plan.Intent)
	explanation += fmt.Sprintf("创建时间: %s\n", plan.CreatedAt.Format("2006-01-02 15:04:05"))
	explanation += fmt.Sprintf("优先级: %d\n", plan.Priority)
	explanation += fmt.Sprintf("步骤数量: %d\n\n", len(plan.Steps))

	for i, step := range plan.Steps {
		explanation += fmt.Sprintf("步骤 %d:\n", i+1)
		explanation += fmt.Sprintf("  ID: %s\n", step.ID)
		explanation += fmt.Sprintf("  类型: %s\n", step.Type)
		explanation += fmt.Sprintf("  名称: %s\n", step.Name)

		if len(step.Input) > 0 {
			explanation += "  输入参数:\n"
			for k, v := range step.Input {
				explanation += fmt.Sprintf("    %s: %v\n", k, v)
			}
		}

		if step.Metadata != nil && step.Metadata["framework"] == "eino" {
			explanation += "  框架: eino\n"
		}

		explanation += "\n"
	}

	if plan.RetryPolicy != nil {
		explanation += fmt.Sprintf("重试策略:\n")
		explanation += fmt.Sprintf("  最大尝试次数: %d\n", plan.RetryPolicy.MaxAttempts)
		explanation += fmt.Sprintf("  基础间隔: %s\n", plan.RetryPolicy.Interval)
		explanation += fmt.Sprintf("  指数退避: %t\n", plan.RetryPolicy.Backoff)
		explanation += fmt.Sprintf("  抖动: %t\n", plan.RetryPolicy.Jitter)
	}

	return explanation, nil
}

// RefreshPlan 在已有 plan 基础上根据新上下文做轻量调整
func (p *EinoPlanner) RefreshPlan(ctx context.Context, plan *contract.ExecutionPlan, additionalContext map[string]interface{}) (*contract.ExecutionPlan, error) {
	if plan == nil {
		return nil, fmt.Errorf("计划不能为空")
	}

	// 创建计划的副本
	refreshedPlan := *plan

	// 更新元数据
	if refreshedPlan.Metadata == nil {
		refreshedPlan.Metadata = make(map[string]interface{})
	}

	// 添加刷新标记
	refreshedPlan.Metadata["refreshed"] = true
	refreshedPlan.Metadata["refresh_time"] = time.Now().Unix()

	// 合并额外上下文
	if additionalContext != nil {
		for k, v := range additionalContext {
			refreshedPlan.Metadata[k] = v
		}
	}

	// 使用 eino 框架调整步骤
	adjustedSteps, err := p.adjustStepsWithEino(ctx, plan.Steps, additionalContext)
	if err != nil {
		return nil, fmt.Errorf("使用 eino 调整步骤失败: %w", err)
	}

	refreshedPlan.Steps = adjustedSteps

	return &refreshedPlan, nil
}

// adjustStepsWithEino 使用 eino 框架调整步骤
func (p *EinoPlanner) adjustStepsWithEino(ctx context.Context, originalSteps []contract.PlanStep, additionalContext map[string]interface{}) ([]contract.PlanStep, error) {
	var adjustedSteps []contract.PlanStep

	for _, step := range originalSteps {
		// 创建步骤的副本
		adjustedStep := step

		// 根据额外上下文调整步骤
		if additionalContext != nil {
			// 更新步骤的输入参数
			if adjustedStep.Input == nil {
				adjustedStep.Input = make(map[string]interface{})
			}

			// 添加上下文信息到步骤输入
			for k, v := range additionalContext {
				if k == "temperature" || k == "max_tokens" || k == "model_name" {
					adjustedStep.Input[k] = v
				}
			}

			// 更新步骤元数据
			if adjustedStep.Metadata == nil {
				adjustedStep.Metadata = make(map[string]interface{})
			}
			adjustedStep.Metadata["adjusted_by_eino"] = true
		}

		adjustedSteps = append(adjustedSteps, adjustedStep)
	}

	return adjustedSteps, nil
}

// GetConfig 获取配置
func (p *EinoPlanner) GetConfig() *Config {
	return p.config
}

// SetConfig 设置配置
func (p *EinoPlanner) SetConfig(config *Config) {
	p.config = config
}

// GetFeedbackManager 获取反馈管理器
func (p *EinoPlanner) GetFeedbackManager() *FeedbackManager {
	return p.feedbackManager
}

// CreateMessageFromIntent 从意图创建 eino 消息
func (p *EinoPlanner) CreateMessageFromIntent(intent string) *schema.Message {
	return &schema.Message{
		Role:    schema.User,
		Content: intent,
	}
}

// CreateSystemMessage 创建系统消息
func (p *EinoPlanner) CreateSystemMessage(systemPrompt string) *schema.Message {
	return &schema.Message{
		Role:    schema.System,
		Content: systemPrompt,
	}
}
