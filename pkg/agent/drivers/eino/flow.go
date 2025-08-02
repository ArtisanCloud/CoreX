package eino

// flow.go - 复合 flow 定义与编排（条件分支、嵌套 flow、子任务组合）

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ArtisanCloud/CoreX/pkg/agent/contract"
)

// FlowManager 负责管理和执行复合流程
type FlowManager struct {
	// 存储已注册的流程定义
	flowDefinitions map[string]*FlowDefinition
	mutex           sync.RWMutex
}

// FlowDefinition 定义一个复合流程
type FlowDefinition struct {
	ID          string
	Name        string
	Description string
	Nodes       []FlowNode
	Edges       []FlowEdge
}

// FlowNode 表示流程中的一个节点
type FlowNode struct {
	ID       string
	Type     string // "start", "end", "task", "decision", "subflow"
	Name     string
	Config   map[string]interface{}
	Metadata map[string]interface{}
}

// FlowEdge 表示流程中的一条边（节点之间的连接）
type FlowEdge struct {
	From      string
	To        string
	Condition string // 可选的条件表达式
}

// NewFlowManager 创建新的流程管理器
func NewFlowManager() *FlowManager {
	return &FlowManager{
		flowDefinitions: make(map[string]*FlowDefinition),
	}
}

// RegisterFlow 注册一个流程定义
func (m *FlowManager) RegisterFlow(flow *FlowDefinition) error {
	if flow == nil {
		return errors.New("流程定义不能为空")
	}
	if flow.ID == "" {
		return errors.New("流程ID不能为空")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 检查是否已存在同ID的流程
	if _, exists := m.flowDefinitions[flow.ID]; exists {
		return fmt.Errorf("流程ID '%s' 已存在", flow.ID)
	}

	// 验证流程定义
	if err := m.validateFlow(flow); err != nil {
		return err
	}

	// 存储流程定义
	m.flowDefinitions[flow.ID] = flow
	return nil
}

// GetFlow 获取流程定义
func (m *FlowManager) GetFlow(flowID string) (*FlowDefinition, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	flow, exists := m.flowDefinitions[flowID]
	if !exists {
		return nil, fmt.Errorf("找不到流程ID '%s'", flowID)
	}

	return flow, nil
}

// ExecuteFlow 执行一个流程
func (m *FlowManager) ExecuteFlow(ctx context.Context, flowID string, input map[string]interface{}) (map[string]interface{}, error) {
	// 获取流程定义
	flow, err := m.GetFlow(flowID)
	if err != nil {
		return nil, err
	}

	// 创建执行上下文
	flowCtx := &FlowExecutionContext{
		FlowID:      flowID,
		Input:       input,
		Output:      make(map[string]interface{}),
		NodeResults: make(map[string]map[string]interface{}),
	}

	// 查找起始节点
	startNode := m.findStartNode(flow)
	if startNode == nil {
		return nil, errors.New("找不到起始节点")
	}

	// 从起始节点开始执行
	err = m.executeNode(ctx, flow, startNode, flowCtx)
	if err != nil {
		return nil, err
	}

	return flowCtx.Output, nil
}

// FlowExecutionContext 存储流程执行的上下文
type FlowExecutionContext struct {
	FlowID      string
	Input       map[string]interface{}
	Output      map[string]interface{}
	NodeResults map[string]map[string]interface{}
	CurrentPath []string // 当前执行路径，用于检测循环
}

// validateFlow 验证流程定义
func (m *FlowManager) validateFlow(flow *FlowDefinition) error {
	// 检查是否有节点
	if len(flow.Nodes) == 0 {
		return errors.New("流程必须包含至少一个节点")
	}

	// 检查是否有起始节点和结束节点
	hasStart := false
	hasEnd := false
	nodeMap := make(map[string]bool)

	for _, node := range flow.Nodes {
		// 检查节点ID是否为空
		if node.ID == "" {
			return errors.New("节点ID不能为空")
		}

		// 检查节点ID是否重复
		if nodeMap[node.ID] {
			return fmt.Errorf("节点ID '%s' 重复", node.ID)
		}
		nodeMap[node.ID] = true

		// 检查节点类型
		if node.Type == "start" {
			hasStart = true
		} else if node.Type == "end" {
			hasEnd = true
		}
	}

	if !hasStart {
		return errors.New("流程必须包含一个起始节点")
	}
	if !hasEnd {
		return errors.New("流程必须包含一个结束节点")
	}

	// 检查边的有效性
	for _, edge := range flow.Edges {
		// 检查源节点是否存在
		if !nodeMap[edge.From] {
			return fmt.Errorf("边的源节点 '%s' 不存在", edge.From)
		}

		// 检查目标节点是否存在
		if !nodeMap[edge.To] {
			return fmt.Errorf("边的目标节点 '%s' 不存在", edge.To)
		}
	}

	return nil
}

// findStartNode 查找起始节点
func (m *FlowManager) findStartNode(flow *FlowDefinition) *FlowNode {
	for i, node := range flow.Nodes {
		if node.Type == "start" {
			return &flow.Nodes[i]
		}
	}
	return nil
}

// findNextNodes 查找下一个节点
func (m *FlowManager) findNextNodes(flow *FlowDefinition, currentNodeID string, flowCtx *FlowExecutionContext) ([]*FlowNode, error) {
	nextNodes := []*FlowNode{}

	// 查找所有从当前节点出发的边
	for _, edge := range flow.Edges {
		if edge.From == currentNodeID {
			// 如果有条件，评估条件
			if edge.Condition != "" {
				// 这里可以实现一个简单的条件表达式求值器
				// 例如，可以检查节点结果是否满足某些条件

				// 简单示例：如果条件是"true"，则通过
				if edge.Condition != "true" {
					// 条件不满足，跳过这条边
					continue
				}
			}

			// 查找目标节点
			for i, node := range flow.Nodes {
				if node.ID == edge.To {
					nextNodes = append(nextNodes, &flow.Nodes[i])
					break
				}
			}
		}
	}

	return nextNodes, nil
}

// executeNode 执行一个节点
func (m *FlowManager) executeNode(ctx context.Context, flow *FlowDefinition, node *FlowNode, flowCtx *FlowExecutionContext) error {
	// 检查是否有循环
	for _, nodeID := range flowCtx.CurrentPath {
		if nodeID == node.ID {
			return fmt.Errorf("检测到循环: %s", node.ID)
		}
	}

	// 更新当前路径
	flowCtx.CurrentPath = append(flowCtx.CurrentPath, node.ID)

	// 根据节点类型执行不同的逻辑
	var result map[string]interface{}
	var err error

	switch node.Type {
	case "start":
		// 起始节点，不执行任何操作
		result = map[string]interface{}{"status": "started"}
	case "end":
		// 结束节点，将结果存储到输出
		result = map[string]interface{}{"status": "completed"}
		// 将所有节点的结果合并到输出
		for nodeID, nodeResult := range flowCtx.NodeResults {
			for k, v := range nodeResult {
				flowCtx.Output[nodeID+"."+k] = v
			}
		}
	case "task":
		// 执行任务
		result, err = m.executeTask(ctx, node, flowCtx)
	case "decision":
		// 决策节点，不执行任何操作
		result = map[string]interface{}{"status": "decision"}
	case "subflow":
		// 执行子流程
		result, err = m.executeSubflow(ctx, node, flowCtx)
	default:
		return fmt.Errorf("不支持的节点类型: %s", node.Type)
	}

	if err != nil {
		return err
	}

	// 存储节点执行结果
	flowCtx.NodeResults[node.ID] = result

	// 如果是结束节点，直接返回
	if node.Type == "end" {
		return nil
	}

	// 查找下一个节点
	nextNodes, err := m.findNextNodes(flow, node.ID, flowCtx)
	if err != nil {
		return err
	}

	// 执行下一个节点
	for _, nextNode := range nextNodes {
		// 创建新的路径副本
		newPath := make([]string, len(flowCtx.CurrentPath))
		copy(newPath, flowCtx.CurrentPath)

		// 创建新的上下文
		newFlowCtx := &FlowExecutionContext{
			FlowID:      flowCtx.FlowID,
			Input:       flowCtx.Input,
			Output:      flowCtx.Output,
			NodeResults: flowCtx.NodeResults,
			CurrentPath: newPath,
		}

		if err := m.executeNode(ctx, flow, nextNode, newFlowCtx); err != nil {
			return err
		}
	}

	return nil
}

// executeTask 执行任务节点
func (m *FlowManager) executeTask(ctx context.Context, node *FlowNode, flowCtx *FlowExecutionContext) (map[string]interface{}, error) {
	// 这里是一个简单的模拟实现
	// 实际实现中，应该根据节点配置执行相应的任务

	// 获取任务类型
	taskType, _ := node.Config["task_type"].(string)
	if taskType == "" {
		return nil, errors.New("任务类型不能为空")
	}

	// 根据任务类型执行不同的逻辑
	switch taskType {
	case "echo":
		// 简单的回显任务
		message, _ := node.Config["message"].(string)
		return map[string]interface{}{
			"message": message,
			"status":  "success",
		}, nil
	case "transform":
		// 数据转换任务
		inputKey, _ := node.Config["input_key"].(string)
		outputKey, _ := node.Config["output_key"].(string)

		// 获取输入数据
		var inputValue interface{}
		if inputKey != "" {
			// 从上下文中获取输入数据
			for _, nodeResult := range flowCtx.NodeResults {
				if value, exists := nodeResult[inputKey]; exists {
					inputValue = value
					break
				}
			}
			// 如果在节点结果中找不到，尝试从输入中获取
			if inputValue == nil {
				inputValue = flowCtx.Input[inputKey]
			}
		}

		// 执行转换
		// 这里是一个简单的示例，实际实现中可能需要更复杂的转换逻辑
		var outputValue interface{} = inputValue

		return map[string]interface{}{
			outputKey: outputValue,
			"status":  "success",
		}, nil
	default:
		return nil, fmt.Errorf("不支持的任务类型: %s", taskType)
	}
}

// executeSubflow 执行子流程节点
func (m *FlowManager) executeSubflow(ctx context.Context, node *FlowNode, flowCtx *FlowExecutionContext) (map[string]interface{}, error) {
	// 获取子流程ID
	subflowID, _ := node.Config["subflow_id"].(string)
	if subflowID == "" {
		return nil, errors.New("子流程ID不能为空")
	}

	// 获取子流程输入
	subflowInput := make(map[string]interface{})

	// 从节点配置中获取输入映射
	inputMapping, _ := node.Config["input_mapping"].(map[string]string)
	if inputMapping != nil {
		for targetKey, sourceKey := range inputMapping {
			// 从上下文中获取输入数据
			var sourceValue interface{}

			// 首先尝试从节点结果中获取
			for _, nodeResult := range flowCtx.NodeResults {
				if value, exists := nodeResult[sourceKey]; exists {
					sourceValue = value
					break
				}
			}

			// 如果在节点结果中找不到，尝试从输入中获取
			if sourceValue == nil {
				sourceValue = flowCtx.Input[sourceKey]
			}

			// 设置子流程输入
			subflowInput[targetKey] = sourceValue
		}
	}

	// 执行子流程
	subflowOutput, err := m.ExecuteFlow(ctx, subflowID, subflowInput)
	if err != nil {
		return nil, err
	}

	// 处理子流程输出
	result := make(map[string]interface{})

	// 从节点配置中获取输出映射
	outputMapping, _ := node.Config["output_mapping"].(map[string]string)
	if outputMapping != nil {
		for targetKey, sourceKey := range outputMapping {
			if value, exists := subflowOutput[sourceKey]; exists {
				result[targetKey] = value
			}
		}
	} else {
		// 如果没有输出映射，直接返回子流程的输出
		result = subflowOutput
	}

	result["status"] = "success"
	return result, nil
}

// ConvertPlanToFlow 将执行计划转换为流程定义
func (m *FlowManager) ConvertPlanToFlow(plan *contract.ExecutionPlan) (*FlowDefinition, error) {
	if plan == nil {
		return nil, errors.New("执行计划不能为空")
	}

	// 创建流程定义
	flow := &FlowDefinition{
		ID:          "flow-" + plan.ID,
		Name:        "Flow for " + plan.Intent,
		Description: "Generated from execution plan",
		Nodes:       []FlowNode{},
		Edges:       []FlowEdge{},
	}

	// 添加起始节点
	startNode := FlowNode{
		ID:       "start",
		Type:     "start",
		Name:     "开始",
		Config:   make(map[string]interface{}),
		Metadata: make(map[string]interface{}),
	}
	flow.Nodes = append(flow.Nodes, startNode)

	// 添加结束节点
	endNode := FlowNode{
		ID:       "end",
		Type:     "end",
		Name:     "结束",
		Config:   make(map[string]interface{}),
		Metadata: make(map[string]interface{}),
	}
	flow.Nodes = append(flow.Nodes, endNode)

	// 添加任务节点
	lastNodeID := "start"
	for i, step := range plan.Steps {
		// 创建任务节点
		taskNode := FlowNode{
			ID:   step.ID,
			Type: "task",
			Name: step.Name,
			Config: map[string]interface{}{
				"task_type": step.Type,
				"input":     step.Input,
				"output":    step.ExpectedOut,
			},
			Metadata: step.Metadata,
		}
		flow.Nodes = append(flow.Nodes, taskNode)

		// 添加边
		edge := FlowEdge{
			From: lastNodeID,
			To:   step.ID,
		}
		flow.Edges = append(flow.Edges, edge)

		// 更新最后一个节点ID
		lastNodeID = step.ID

		// 如果是最后一个步骤，添加到结束节点的边
		if i == len(plan.Steps)-1 {
			edge := FlowEdge{
				From: step.ID,
				To:   "end",
			}
			flow.Edges = append(flow.Edges, edge)
		}
	}

	return flow, nil
}
