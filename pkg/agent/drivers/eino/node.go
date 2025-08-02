package eino

// node.go - driver 封装的 node：包装 core node、附加 intent/context 处理、适配输入输出与 side-effect

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ArtisanCloud/CoreX/pkg/agent/contract"
)

// NodeExecutor 负责执行单个节点
type NodeExecutor struct {
	// 存储已注册的节点处理器
	nodeHandlers map[string]NodeHandler
}

// NodeHandler 定义节点处理器接口
type NodeHandler interface {
	// Execute 执行节点
	Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
	// GetMetadata 获取节点元数据
	GetMetadata() map[string]interface{}
}

// NodeContext 节点执行上下文
type NodeContext struct {
	// 节点ID
	NodeID string
	// 节点类型
	NodeType string
	// 节点名称
	NodeName string
	// 节点配置
	Config map[string]interface{}
	// 节点元数据
	Metadata map[string]interface{}
	// 输入参数
	Input map[string]interface{}
	// 期望输出
	ExpectedOutput map[string]interface{}
	// 实际输出
	Output map[string]interface{}
	// 执行状态
	Status string
	// 错误信息
	Error error
	// 开始时间
	StartTime time.Time
	// 结束时间
	EndTime time.Time
	// 执行时长
	Duration time.Duration
}

// NewNodeExecutor 创建新的节点执行器
func NewNodeExecutor() *NodeExecutor {
	return &NodeExecutor{
		nodeHandlers: make(map[string]NodeHandler),
	}
}

// RegisterNodeHandler 注册节点处理器
func (e *NodeExecutor) RegisterNodeHandler(nodeType string, handler NodeHandler) {
	e.nodeHandlers[nodeType] = handler
}

// ExecuteNode 执行节点
func (e *NodeExecutor) ExecuteNode(ctx context.Context, step contract.PlanStep) (*contract.ToolResult, error) {
	// 查找节点处理器
	handler, exists := e.nodeHandlers[step.Type]
	if !exists {
		return nil, fmt.Errorf("找不到节点类型 '%s' 的处理器", step.Type)
	}

	// 创建节点上下文
	nodeCtx := &NodeContext{
		NodeID:         step.ID,
		NodeType:       step.Type,
		NodeName:       step.Name,
		Config:         make(map[string]interface{}),
		Metadata:       step.Metadata,
		Input:          step.Input,
		ExpectedOutput: step.ExpectedOut,
		Output:         make(map[string]interface{}),
		Status:         "pending",
		StartTime:      time.Now(),
	}

	// 执行节点
	nodeCtx.Status = "running"
	output, err := handler.Execute(ctx, step.Input)
	nodeCtx.EndTime = time.Now()
	nodeCtx.Duration = nodeCtx.EndTime.Sub(nodeCtx.StartTime)

	if err != nil {
		nodeCtx.Status = "failed"
		nodeCtx.Error = err
		return &contract.ToolResult{
			Output:   nil,
			Success:  false,
			Error:    err,
			Duration: nodeCtx.Duration,
		}, err
	}

	nodeCtx.Status = "completed"
	nodeCtx.Output = output

	// 验证输出是否符合期望
	if err := e.validateOutput(nodeCtx); err != nil {
		nodeCtx.Status = "invalid_output"
		nodeCtx.Error = err
		return &contract.ToolResult{
			Output:   output,
			Success:  false,
			Error:    err,
			Duration: nodeCtx.Duration,
		}, err
	}

	return &contract.ToolResult{
		Output:   output,
		Success:  true,
		Error:    nil,
		Duration: nodeCtx.Duration,
	}, nil
}

// validateOutput 验证输出是否符合期望
func (e *NodeExecutor) validateOutput(nodeCtx *NodeContext) error {
	// 如果没有期望输出，则不需要验证
	if nodeCtx.ExpectedOutput == nil || len(nodeCtx.ExpectedOutput) == 0 {
		return nil
	}

	// 检查输出是否包含所有期望的字段
	for key := range nodeCtx.ExpectedOutput {
		if _, exists := nodeCtx.Output[key]; !exists {
			return fmt.Errorf("输出缺少期望的字段: %s", key)
		}
	}

	return nil
}

// DefaultNodeHandler 默认节点处理器
type DefaultNodeHandler struct {
	// 处理函数
	handler func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
	// 元数据
	metadata map[string]interface{}
}

// NewDefaultNodeHandler 创建新的默认节点处理器
func NewDefaultNodeHandler(handler func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error), metadata map[string]interface{}) *DefaultNodeHandler {
	return &DefaultNodeHandler{
		handler:  handler,
		metadata: metadata,
	}
}

// Execute 执行节点
func (h *DefaultNodeHandler) Execute(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	if h.handler == nil {
		return nil, errors.New("处理函数不能为空")
	}
	return h.handler(ctx, input)
}

// GetMetadata 获取节点元数据
func (h *DefaultNodeHandler) GetMetadata() map[string]interface{} {
	return h.metadata
}

// RegisterCommonNodeHandlers 注册常用节点处理器
func (e *NodeExecutor) RegisterCommonNodeHandlers() {
	// 注册Echo节点处理器
	e.RegisterNodeHandler("echo", NewDefaultNodeHandler(
		func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
			// 简单的回显节点
			return input, nil
		},
		map[string]interface{}{
			"description": "回显输入参数",
		},
	))

	// 注册Transform节点处理器
	e.RegisterNodeHandler("transform", NewDefaultNodeHandler(
		func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
			// 数据转换节点
			output := make(map[string]interface{})

			// 获取转换规则
			rules, ok := input["transform_rules"].(map[string]string)
			if !ok {
				return nil, errors.New("转换规则无效")
			}

			// 应用转换规则
			for targetKey, sourceKey := range rules {
				if value, exists := input[sourceKey]; exists {
					output[targetKey] = value
				}
			}

			return output, nil
		},
		map[string]interface{}{
			"description": "转换输入数据",
		},
	))

	// 注册Condition节点处理器
	e.RegisterNodeHandler("condition", NewDefaultNodeHandler(
		func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
			// 条件判断节点
			condition, ok := input["condition"].(string)
			if !ok {
				return nil, errors.New("条件表达式无效")
			}

			// 这里可以实现一个简单的条件表达式求值器
			// 例如，可以检查输入是否满足某些条件

			// 简单示例：如果条件是"true"，则返回true，否则返回false
			result := condition == "true"

			return map[string]interface{}{
				"result": result,
			}, nil
		},
		map[string]interface{}{
			"description": "条件判断",
		},
	))

	// 注册Delay节点处理器
	e.RegisterNodeHandler("delay", NewDefaultNodeHandler(
		func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
			// 延迟节点
			duration, ok := input["duration"].(int)
			if !ok {
				return nil, errors.New("延迟时间无效")
			}

			// 延迟指定时间
			select {
			case <-time.After(time.Duration(duration) * time.Millisecond):
				// 延迟结束
			case <-ctx.Done():
				// 上下文取消
				return nil, ctx.Err()
			}

			return map[string]interface{}{
				"delayed":     true,
				"duration_ms": duration,
			}, nil
		},
		map[string]interface{}{
			"description": "延迟执行",
		},
	))
}

// ExecutePlanStep 执行计划步骤
func (e *NodeExecutor) ExecutePlanStep(ctx context.Context, step contract.PlanStep, previousResults map[string]*contract.ToolResult) (*contract.ToolResult, error) {
	// 处理输入参数中的变量引用
	processedInput, err := e.processInputVariables(step.Input, previousResults)
	if err != nil {
		return nil, err
	}

	// 创建新的步骤，使用处理后的输入参数
	processedStep := contract.PlanStep{
		ID:          step.ID,
		Type:        step.Type,
		Name:        step.Name,
		Input:       processedInput,
		ExpectedOut: step.ExpectedOut,
		Metadata:    step.Metadata,
	}

	// 执行节点
	return e.ExecuteNode(ctx, processedStep)
}

// processInputVariables 处理输入参数中的变量引用
func (e *NodeExecutor) processInputVariables(input map[string]interface{}, previousResults map[string]*contract.ToolResult) (map[string]interface{}, error) {
	// 创建新的输入参数映射
	processedInput := make(map[string]interface{})

	// 复制输入参数
	for key, value := range input {
		// 检查值是否是字符串
		if strValue, ok := value.(string); ok {
			// 检查是否是变量引用（格式：{{stepID.key}}）
			if len(strValue) > 4 && strValue[0:2] == "{{" && strValue[len(strValue)-2:] == "}}" {
				// 提取变量引用
				varRef := strValue[2 : len(strValue)-2]

				// 解析变量引用
				stepID, outputKey, found := parseVariableReference(varRef)
				if !found {
					return nil, fmt.Errorf("无效的变量引用: %s", strValue)
				}

				// 查找前一步骤的结果
				stepResult, exists := previousResults[stepID]
				if !exists {
					return nil, fmt.Errorf("找不到步骤 '%s' 的结果", stepID)
				}

				// 获取输出值
				if outputValue, exists := stepResult.Output[outputKey]; exists {
					processedInput[key] = outputValue
				} else {
					return nil, fmt.Errorf("步骤 '%s' 的结果中找不到键 '%s'", stepID, outputKey)
				}
			} else {
				// 不是变量引用，直接复制
				processedInput[key] = value
			}
		} else {
			// 不是字符串，直接复制
			processedInput[key] = value
		}
	}

	return processedInput, nil
}

// parseVariableReference 解析变量引用
func parseVariableReference(varRef string) (string, string, bool) {
	// 查找分隔符
	for i := 0; i < len(varRef); i++ {
		if varRef[i] == '.' {
			return varRef[:i], varRef[i+1:], true
		}
	}

	return "", "", false
}
