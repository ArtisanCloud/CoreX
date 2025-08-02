package eino

// plan.go - ExecutionPlan 构造逻辑，自然语言意图解析、fallback 策略、优先级、验证、上下文融合

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ArtisanCloud/CoreX/pkg/agent/contract"
)

// EinoPlanner 实现 contract.Planner 接口
type EinoPlanner struct {
	config          *Config
	feedbackManager *FeedbackManager
}

// NewEinoPlanner 创建新的Eino计划生成器
func NewEinoPlanner(config *Config, feedbackManager *FeedbackManager) *EinoPlanner {
	return &EinoPlanner{
		config:          config,
		feedbackManager: feedbackManager,
	}
}

// Plan 根据意图与上下文生成新的执行计划
func (p *EinoPlanner) Plan(ctx context.Context, intent string, contextMetadata map[string]interface{}) (*contract.ExecutionPlan, error) {
	if intent == "" {
		return nil, contract.ErrEmptyIntent
	}

	// 使用配置中的意图提示模板
	intentPrompt := p.config.IntentPrompt
	if intentPrompt == "" {
		intentPrompt = defaultIntentPrompt
	}

	// 创建一个执行计划
	plan := &contract.ExecutionPlan{
		ID:        "plan-" + time.Now().Format("20060102150405"),
		Intent:    intent,
		CreatedAt: time.Now(),
		Steps:     []contract.PlanStep{},
		Metadata:  make(map[string]interface{}),
		Priority:  p.determinePriority(intent, contextMetadata),
	}

	// 合并上下文元数据
	if contextMetadata != nil {
		for k, v := range contextMetadata {
			plan.Metadata[k] = v
		}
	}

	// 添加配置中的上下文元数据
	if p.config.ContextMetadata != nil {
		for k, v := range p.config.ContextMetadata {
			plan.Metadata[k] = v
		}
	}

	// 解析意图，生成步骤
	steps, err := p.parseIntent(ctx, intent, plan.Metadata)
	if err != nil {
		return nil, err
	}
	plan.Steps = steps

	// 添加回退计划
	fallbackPlan, err := p.generateFallbackPlan(ctx, intent, plan.Metadata)
	if err == nil && fallbackPlan != nil {
		plan.FallbackPlan = fallbackPlan
	}

	// 添加重试策略
	if p.config.RetryPolicy != nil {
		plan.RetryPolicy = p.config.RetryPolicy
	} else {
		// 默认重试策略
		plan.RetryPolicy = &contract.RetryPolicy{
			MaxAttempts: 3,
			Interval:    time.Second,
			Backoff:     true,
			Jitter:      true,
			MaxInterval: 30 * time.Second,
		}
	}

	return plan, nil
}

// ValidatePlan 校验一个现有 plan 的合法性
func (p *EinoPlanner) ValidatePlan(ctx context.Context, plan *contract.ExecutionPlan) error {
	if plan == nil {
		return contract.ErrInvalidPlan
	}

	// 检查基本属性
	if plan.ID == "" {
		return errors.New("计划ID不能为空")
	}
	if plan.Intent == "" {
		return errors.New("计划意图不能为空")
	}
	if plan.CreatedAt.IsZero() {
		return errors.New("计划创建时间不能为零")
	}

	// 检查步骤
	if len(plan.Steps) == 0 {
		return errors.New("计划必须包含至少一个步骤")
	}

	// 检查每个步骤
	stepIDs := make(map[string]bool)
	for i, step := range plan.Steps {
		// 检查步骤ID
		if step.ID == "" {
			return fmt.Errorf("步骤 %d 的ID不能为空", i)
		}
		// 检查步骤ID是否重复
		if _, exists := stepIDs[step.ID]; exists {
			return fmt.Errorf("步骤ID '%s' 重复", step.ID)
		}
		stepIDs[step.ID] = true

		// 检查步骤类型
		if step.Type == "" {
			return fmt.Errorf("步骤 '%s' 的类型不能为空", step.ID)
		}

		// 检查步骤名称
		if step.Name == "" {
			return fmt.Errorf("步骤 '%s' 的名称不能为空", step.ID)
		}

		// 检查输入参数
		if step.Input == nil {
			return fmt.Errorf("步骤 '%s' 的输入参数不能为空", step.ID)
		}
	}

	// 检查依赖关系
	for _, step := range plan.Steps {
		if step.Metadata != nil {
			if dependsOn, ok := step.Metadata["depends_on"].(string); ok {
				if !stepIDs[dependsOn] {
					return fmt.Errorf("步骤 '%s' 依赖不存在的步骤 '%s'", step.ID, dependsOn)
				}
			}
		}
	}

	return nil
}

// ExplainPlan 返回 plan 的人类可读解释
func (p *EinoPlanner) ExplainPlan(ctx context.Context, plan *contract.ExecutionPlan) (string, error) {
	if plan == nil {
		return "", contract.ErrInvalidPlan
	}

	explanation := "执行计划 " + plan.ID + ":\n"
	explanation += "意图: " + plan.Intent + "\n"
	explanation += "创建时间: " + plan.CreatedAt.Format("2006-01-02 15:04:05") + "\n"
	explanation += "优先级: " + fmt.Sprintf("%d", plan.Priority) + "\n"
	explanation += "步骤数量: " + fmt.Sprintf("%d", len(plan.Steps)) + "\n\n"

	for i, step := range plan.Steps {
		explanation += "步骤 " + fmt.Sprintf("%d", i+1) + ": " + step.Name + " (ID: " + step.ID + ")\n"
		explanation += "  类型: " + step.Type + "\n"
		explanation += "  输入参数: " + formatMap(step.Input) + "\n"
		explanation += "  期望输出: " + formatMap(step.ExpectedOut) + "\n"

		// 添加依赖关系
		if step.Metadata != nil {
			if dependsOn, ok := step.Metadata["depends_on"].(string); ok {
				explanation += "  依赖步骤: " + dependsOn + "\n"
			}
			if parallelGroup, ok := step.Metadata["parallel_group"].(string); ok {
				explanation += "  并行组: " + parallelGroup + "\n"
			}
		}

		explanation += "\n"
	}

	// 添加回退计划
	if plan.FallbackPlan != nil {
		explanation += "回退计划:\n"
		fallbackExplanation, err := p.ExplainPlan(ctx, plan.FallbackPlan)
		if err != nil {
			explanation += "  无法解释回退计划: " + err.Error() + "\n"
		} else {
			// 缩进回退计划的解释
			fallbackLines := strings.Split(fallbackExplanation, "\n")
			for _, line := range fallbackLines {
				explanation += "  " + line + "\n"
			}
		}
	}

	// 添加重试策略
	if plan.RetryPolicy != nil {
		explanation += "重试策略:\n"
		explanation += "  最大尝试次数: " + fmt.Sprintf("%d", plan.RetryPolicy.MaxAttempts) + "\n"
		explanation += "  基础等待时间: " + plan.RetryPolicy.Interval.String() + "\n"
		explanation += "  最大等待时间: " + plan.RetryPolicy.MaxInterval.String() + "\n"
		explanation += "  启用指数退避: " + fmt.Sprintf("%t", plan.RetryPolicy.Backoff) + "\n"
		explanation += "  启用抖动: " + fmt.Sprintf("%t", plan.RetryPolicy.Jitter) + "\n"
	}

	return explanation, nil
}

// RefreshPlan 在已有 plan 基础上根据新上下文做轻量调整
func (p *EinoPlanner) RefreshPlan(ctx context.Context, plan *contract.ExecutionPlan, additionalContext map[string]interface{}) (*contract.ExecutionPlan, error) {
	if plan == nil {
		return nil, contract.ErrInvalidPlan
	}

	// 创建计划的副本
	newPlan := *plan

	// 更新元数据
	if newPlan.Metadata == nil {
		newPlan.Metadata = additionalContext
	} else {
		for k, v := range additionalContext {
			newPlan.Metadata[k] = v
		}
	}

	// 根据新上下文调整步骤
	for i, step := range newPlan.Steps {
		// 检查步骤是否需要调整
		if step.Metadata != nil && step.Metadata["context_sensitive"] == true {
			// 这里可以根据新上下文调整步骤的输入参数
			// 例如，可以根据上下文中的特定字段更新步骤的输入参数

			// 简单示例：如果上下文中有与步骤输入参数同名的字段，则更新步骤的输入参数
			for k, v := range additionalContext {
				if _, exists := step.Input[k]; exists {
					newPlan.Steps[i].Input[k] = v
				}
			}
		}
	}

	return &newPlan, nil
}

// parseIntent 解析意图，生成步骤
func (p *EinoPlanner) parseIntent(ctx context.Context, intent string, metadata map[string]interface{}) ([]contract.PlanStep, error) {
	// 这里是一个简单的模拟实现
	// 实际实现中，应该使用NLP或LLM来解析意图，生成步骤

	// 简单示例：根据意图关键词生成步骤
	steps := []contract.PlanStep{}

	// 解析意图，生成步骤
	if strings.Contains(strings.ToLower(intent), "查询") || strings.Contains(strings.ToLower(intent), "搜索") {
		// 添加查询步骤
		step := contract.PlanStep{
			ID:          "step-query",
			Type:        "tool",
			Name:        "执行查询",
			Input:       map[string]interface{}{"query": intent},
			ExpectedOut: map[string]interface{}{"result": "查询结果"},
			Metadata:    map[string]interface{}{},
		}
		steps = append(steps, step)
	} else if strings.Contains(strings.ToLower(intent), "创建") || strings.Contains(strings.ToLower(intent), "新建") {
		// 添加创建步骤
		step := contract.PlanStep{
			ID:          "step-create",
			Type:        "tool",
			Name:        "创建资源",
			Input:       map[string]interface{}{"resource": intent},
			ExpectedOut: map[string]interface{}{"result": "创建结果"},
			Metadata:    map[string]interface{}{},
		}
		steps = append(steps, step)
	} else if strings.Contains(strings.ToLower(intent), "更新") || strings.Contains(strings.ToLower(intent), "修改") {
		// 添加更新步骤
		step1 := contract.PlanStep{
			ID:          "step-query",
			Type:        "tool",
			Name:        "查询资源",
			Input:       map[string]interface{}{"query": intent},
			ExpectedOut: map[string]interface{}{"result": "查询结果"},
			Metadata:    map[string]interface{}{},
		}
		step2 := contract.PlanStep{
			ID:          "step-update",
			Type:        "tool",
			Name:        "更新资源",
			Input:       map[string]interface{}{"resource": "{{step-query.result}}"},
			ExpectedOut: map[string]interface{}{"result": "更新结果"},
			Metadata: map[string]interface{}{
				"depends_on": "step-query",
			},
		}
		steps = append(steps, step1, step2)
	} else if strings.Contains(strings.ToLower(intent), "删除") {
		// 添加删除步骤
		step1 := contract.PlanStep{
			ID:          "step-query",
			Type:        "tool",
			Name:        "查询资源",
			Input:       map[string]interface{}{"query": intent},
			ExpectedOut: map[string]interface{}{"result": "查询结果"},
			Metadata:    map[string]interface{}{},
		}
		step2 := contract.PlanStep{
			ID:          "step-confirm",
			Type:        "tool",
			Name:        "确认删除",
			Input:       map[string]interface{}{"resource": "{{step-query.result}}"},
			ExpectedOut: map[string]interface{}{"result": "确认结果"},
			Metadata: map[string]interface{}{
				"depends_on": "step-query",
			},
		}
		step3 := contract.PlanStep{
			ID:          "step-delete",
			Type:        "tool",
			Name:        "删除资源",
			Input:       map[string]interface{}{"resource": "{{step-query.result}}"},
			ExpectedOut: map[string]interface{}{"result": "删除结果"},
			Metadata: map[string]interface{}{
				"depends_on": "step-confirm",
			},
		}
		steps = append(steps, step1, step2, step3)
	} else {
		// 默认步骤
		step := contract.PlanStep{
			ID:          "step-default",
			Type:        "tool",
			Name:        "处理意图",
			Input:       map[string]interface{}{"intent": intent},
			ExpectedOut: map[string]interface{}{"result": "处理结果"},
			Metadata:    map[string]interface{}{},
		}
		steps = append(steps, step)
	}

	return steps, nil
}

// generateFallbackPlan 生成回退计划
func (p *EinoPlanner) generateFallbackPlan(ctx context.Context, intent string, metadata map[string]interface{}) (*contract.ExecutionPlan, error) {
	// 这里是一个简单的模拟实现
	// 实际实现中，应该根据意图和上下文生成一个更简单或更可靠的回退计划

	// 创建一个简单的回退计划
	fallbackPlan := &contract.ExecutionPlan{
		ID:        "fallback-" + time.Now().Format("20060102150405"),
		Intent:    "回退: " + intent,
		CreatedAt: time.Now(),
		Steps:     []contract.PlanStep{},
		Metadata:  metadata,
		Priority:  1, // 回退计划优先级较低
	}

	// 添加一个简单的回退步骤
	step := contract.PlanStep{
		ID:          "step-fallback",
		Type:        "fallback",
		Name:        "执行回退操作",
		Input:       map[string]interface{}{"intent": intent},
		ExpectedOut: map[string]interface{}{"result": "回退结果"},
		Metadata:    map[string]interface{}{},
	}
	fallbackPlan.Steps = append(fallbackPlan.Steps, step)

	return fallbackPlan, nil
}

// determinePriority 根据意图和上下文确定优先级
func (p *EinoPlanner) determinePriority(intent string, contextMetadata map[string]interface{}) int {
	// 这里是一个简单的模拟实现
	// 实际实现中，应该根据意图和上下文的重要性确定优先级

	// 默认优先级
	priority := 1

	// 检查上下文中是否有优先级设置
	if contextMetadata != nil {
		if p, ok := contextMetadata["priority"].(int); ok {
			priority = p
		}
	}

	// 根据意图关键词调整优先级
	if strings.Contains(strings.ToLower(intent), "紧急") || strings.Contains(strings.ToLower(intent), "urgent") {
		priority += 2
	}
	if strings.Contains(strings.ToLower(intent), "重要") || strings.Contains(strings.ToLower(intent), "important") {
		priority += 1
	}

	// 确保优先级在有效范围内
	if priority < 1 {
		priority = 1
	}
	if priority > 5 {
		priority = 5
	}

	return priority
}

// formatMap 格式化map为字符串
func formatMap(m map[string]interface{}) string {
	result := "{"
	first := true
	for k, v := range m {
		if !first {
			result += ", "
		}
		result += k + ": " + formatValue(v)
		first = false
	}
	result += "}"
	return result
}

// formatValue 格式化值为字符串
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return "\"" + val + "\""
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%.2f", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case map[string]interface{}:
		return formatMap(val)
	case []interface{}:
		return formatArray(val)
	default:
		return "..."
	}
}

// formatArray 格式化数组为字符串
func formatArray(arr []interface{}) string {
	result := "["
	for i, v := range arr {
		if i > 0 {
			result += ", "
		}
		result += formatValue(v)
	}
	result += "]"
	return result
}

// 默认意图提示模板
const defaultIntentPrompt = `
你是一个智能助手，负责将用户意图转换为执行计划。

用户意图: {{intent}}

请生成一个执行计划，包含以下步骤:
1. 分析用户意图
2. 确定需要执行的操作
3. 执行操作并返回结果

上下文信息:
{{context}}
`
