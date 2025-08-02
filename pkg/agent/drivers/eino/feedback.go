package eino

// feedback.go - 多轮反馈机制：订阅 core 事件、分析执行结果、调整或再生成 plan

import (
	"github.com/ArtisanCloud/CoreX/pkg/agent/contract"
)

// FeedbackManager 负责处理执行结果的反馈，用于优化后续执行
type FeedbackManager struct {
	feedbackHistory []*FeedbackEntry
}

// FeedbackEntry 表示一条反馈记录
type FeedbackEntry struct {
	Result map[string]*contract.ToolResult
	Plan   *contract.ExecutionPlan
}

// NewFeedbackManager 创建新的反馈管理器
func NewFeedbackManager() *FeedbackManager {
	return &FeedbackManager{
		feedbackHistory: make([]*FeedbackEntry, 0),
	}
}

// ProcessFeedback 处理执行结果的反馈
func (m *FeedbackManager) ProcessFeedback(results map[string]*contract.ToolResult, plan *contract.ExecutionPlan) {
	entry := &FeedbackEntry{
		Result: results,
		Plan:   plan,
	}
	m.feedbackHistory = append(m.feedbackHistory, entry)
}

// GetSuggestions 根据历史反馈获取改进建议
func (m *FeedbackManager) GetSuggestions(context map[string]interface{}) []string {
	suggestions := []string{}

	// 基于历史反馈生成建议
	if len(m.feedbackHistory) > 0 {
		// 示例逻辑：如果有失败记录，提供相关建议
		for _, entry := range m.feedbackHistory {
			for _, result := range entry.Result {
				if !result.Success {
					suggestions = append(suggestions,
						"考虑添加更多上下文信息",
						"尝试使用替代工具或方法",
					)
					break
				}
			}
		}
	}

	return suggestions
}

// AnalyzeFeedback 分析执行结果，提供改进建议
func (m *FeedbackManager) AnalyzeFeedback(results map[string]*contract.ToolResult, plan *contract.ExecutionPlan) map[string]interface{} {
	// 处理反馈
	m.ProcessFeedback(results, plan)

	// 分析结果
	analysis := make(map[string]interface{})

	// 计算成功率
	totalSteps := len(results)
	successfulSteps := 0

	for _, result := range results {
		if result.Success {
			successfulSteps++
		}
	}

	analysis["success_rate"] = float64(0)
	if totalSteps > 0 {
		analysis["success_rate"] = float64(successfulSteps) / float64(totalSteps)
	}

	analysis["total_steps"] = totalSteps
	analysis["successful_steps"] = successfulSteps
	analysis["overall_success"] = (totalSteps > 0) && (successfulSteps == totalSteps)

	// 识别失败的步骤
	failedSteps := []string{}
	for stepID, result := range results {
		if !result.Success {
			failedSteps = append(failedSteps, stepID)
		}
	}
	analysis["failed_steps"] = failedSteps

	// 获取改进建议
	analysis["suggestions"] = m.GetSuggestions(plan.Metadata)

	return analysis
}

// GenerateImprovedPlan 根据反馈生成改进的计划
func (m *FeedbackManager) GenerateImprovedPlan(plan *contract.ExecutionPlan, results map[string]*contract.ToolResult, additionalContext map[string]interface{}) (*contract.ExecutionPlan, error) {
	// 分析反馈
	analysis := m.AnalyzeFeedback(results, plan)

	// 创建计划的副本
	newPlan := *plan

	// 更新元数据
	if newPlan.Metadata == nil {
		newPlan.Metadata = make(map[string]interface{})
	}

	// 添加分析结果到元数据
	newPlan.Metadata["feedback_analysis"] = analysis

	// 添加额外上下文
	for k, v := range additionalContext {
		newPlan.Metadata[k] = v
	}

	// 如果有失败的步骤，尝试调整或替换这些步骤
	failedSteps, ok := analysis["failed_steps"].([]string)
	if ok && len(failedSteps) > 0 {
		// 这里可以实现更复杂的逻辑来调整失败的步骤
		// 例如，可以尝试使用不同的工具、参数或方法

		// 简单示例：为失败的步骤添加重试标记
		for i, step := range newPlan.Steps {
			for _, failedStepID := range failedSteps {
				if step.ID == failedStepID {
					if newPlan.Steps[i].Metadata == nil {
						newPlan.Steps[i].Metadata = make(map[string]interface{})
					}
					newPlan.Steps[i].Metadata["retry_attempt"] = true
					newPlan.Steps[i].Metadata["previous_failure"] = true

					// 可以在这里修改步骤的输入参数、期望输出等
				}
			}
		}
	}

	return &newPlan, nil
}
