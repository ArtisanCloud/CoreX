package agent

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/ArtisanCloud/CoreX/pkg/agent/drivers/eino"
	"github.com/gin-gonic/gin"
)

// ChatRequest 聊天请求结构
type ChatRequest struct {
	Message string                 `json:"message" binding:"required"`
	Config  *ChatConfig            `json:"config,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// ChatConfig API 聊天配置
type ChatConfig struct {
	ModelName    string  `json:"model_name,omitempty"`
	Provider     string  `json:"provider,omitempty"`
	Endpoint     string  `json:"endpoint,omitempty"`
	APIKey       string  `json:"api_key,omitempty"`
	Temperature  float64 `json:"temperature,omitempty"`
	MaxTokens    int     `json:"max_tokens,omitempty"`
	SystemPrompt string  `json:"system_prompt,omitempty"`
	EnableStream bool    `json:"enable_stream,omitempty"`
}

// ChatResponse 聊天响应结构
type ChatResponse struct {
	Success   bool                   `json:"success"`
	Message   string                 `json:"message,omitempty"`
	Content   string                 `json:"content,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// PlanRequest 计划生成请求
type PlanRequest struct {
	Intent  string                 `json:"intent" binding:"required"`
	Context map[string]interface{} `json:"context,omitempty"`
	Config  *ChatConfig            `json:"config,omitempty"`
}

// PlanResponse 计划生成响应
type PlanResponse struct {
	Success     bool                   `json:"success"`
	PlanID      string                 `json:"plan_id,omitempty"`
	Intent      string                 `json:"intent,omitempty"`
	Steps       []PlanStep             `json:"steps,omitempty"`
	Explanation string                 `json:"explanation,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   int64                  `json:"timestamp"`
}

// PlanStep 计划步骤
type PlanStep struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Input       map[string]interface{} `json:"input,omitempty"`
	ExpectedOut map[string]interface{} `json:"expected_out,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ExecuteRequest 执行请求
type ExecuteRequest struct {
	Intent  string                 `json:"intent" binding:"required"`
	Context map[string]interface{} `json:"context,omitempty"`
	Config  *ChatConfig            `json:"config,omitempty"`
}

// ExecuteResponse 执行响应
type ExecuteResponse struct {
	Success   bool                              `json:"success"`
	PlanID    string                            `json:"plan_id,omitempty"`
	Results   map[string]map[string]interface{} `json:"results,omitempty"`
	Error     string                            `json:"error,omitempty"`
	Metadata  map[string]interface{}            `json:"metadata,omitempty"`
	Timestamp int64                             `json:"timestamp"`
}

// StreamChatRequest 流式聊天请求
type StreamChatRequest struct {
	Message string                 `json:"message" binding:"required"`
	Config  *ChatConfig            `json:"config,omitempty"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// ChatHandler 基本聊天接口
func ChatHandler(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ChatResponse{
			Success:   false,
			Error:     "请求参数错误: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}

	// 创建 agent 配置
	config := createAgentConfig(req.Config)

	// 创建 agent
	agent, err := eino.NewAgent(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ChatResponse{
			Success:   false,
			Error:     "创建 Agent 失败: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}
	defer agent.Close()

	// 执行聊天
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := agent.Chat(ctx, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ChatResponse{
			Success:   false,
			Error:     "聊天执行失败: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, ChatResponse{
		Success: true,
		Message: req.Message,
		Content: response.Content,
		Metadata: map[string]interface{}{
			"role":      string(response.Role),
			"framework": "eino",
		},
		Timestamp: time.Now().Unix(),
	})
}

// PlanHandler 计划生成接口
func PlanHandler(c *gin.Context) {
	var req PlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, PlanResponse{
			Success:   false,
			Error:     "请求参数错误: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}

	// 创建 agent 配置
	config := createAgentConfig(req.Config)

	// 创建 agent
	agent, err := eino.NewAgent(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, PlanResponse{
			Success:   false,
			Error:     "创建 Agent 失败: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}
	defer agent.Close()

	// 生成计划
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	plan, err := agent.GetPlan(ctx, req.Intent, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, PlanResponse{
			Success:   false,
			Error:     "生成计划失败: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}

	// 生成计划解释
	explanation := fmt.Sprintf("执行计划包含 %d 个步骤，基于 eino 框架生成", len(plan.Steps))

	// 转换步骤格式
	var steps []PlanStep
	for _, step := range plan.Steps {
		steps = append(steps, PlanStep{
			ID:          step.ID,
			Type:        step.Type,
			Name:        step.Name,
			Input:       step.Input,
			ExpectedOut: step.ExpectedOut,
			Metadata:    step.Metadata,
		})
	}

	// 返回成功响应
	c.JSON(http.StatusOK, PlanResponse{
		Success:     true,
		PlanID:      plan.ID,
		Intent:      plan.Intent,
		Steps:       steps,
		Explanation: explanation,
		Metadata: map[string]interface{}{
			"framework":   "eino",
			"steps_count": len(steps),
			"priority":    plan.Priority,
		},
		Timestamp: time.Now().Unix(),
	})
}

// ExecuteHandler 执行接口
func ExecuteHandler(c *gin.Context) {
	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ExecuteResponse{
			Success:   false,
			Error:     "请求参数错误: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}

	// 创建 agent 配置
	config := createAgentConfig(req.Config)

	// 创建 agent
	agent, err := eino.NewAgent(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ExecuteResponse{
			Success:   false,
			Error:     "创建 Agent 失败: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}
	defer agent.Close()

	// 执行意图
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	results, err := agent.Run(ctx, req.Intent, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ExecuteResponse{
			Success:   false,
			Error:     "执行失败: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}

	// 转换结果格式
	resultMap := make(map[string]map[string]interface{})
	for stepID, result := range results {
		resultMap[stepID] = map[string]interface{}{
			"output":   result.Output,
			"success":  result.Success,
			"duration": result.Duration.Milliseconds(),
		}
		if result.Error != nil {
			resultMap[stepID]["error"] = result.Error.Error()
		}
	}

	// 返回成功响应
	c.JSON(http.StatusOK, ExecuteResponse{
		Success: true,
		Results: resultMap,
		Metadata: map[string]interface{}{
			"framework":     "eino",
			"results_count": len(results),
		},
		Timestamp: time.Now().Unix(),
	})
}

// StreamChatHandler 流式聊天接口
func StreamChatHandler(c *gin.Context) {
	var req StreamChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ChatResponse{
			Success:   false,
			Error:     "请求参数错误: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}

	// 创建 agent 配置
	config := createAgentConfig(req.Config)

	// 创建 agent
	agent, err := eino.NewAgent(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ChatResponse{
			Success:   false,
			Error:     "创建 Agent 失败: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}
	defer agent.Close()

	// 设置流式响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 执行流式聊天
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	streamChan, err := agent.StreamChat(ctx, req.Message)
	if err != nil {
		c.SSEvent("error", map[string]interface{}{
			"success": false,
			"error":   "流式聊天执行失败: " + err.Error(),
		})
		return
	}

	// 发送流式数据
	for chunk := range streamChan {
		c.SSEvent("data", map[string]interface{}{
			"content": chunk.Content,
			"role":    string(chunk.Role),
		})
		c.Writer.Flush()
	}

	// 发送结束标记
	c.SSEvent("end", map[string]interface{}{
		"success": true,
		"message": "流式聊天完成",
	})
}

// ConfigTestHandler 配置测试接口
func ConfigTestHandler(c *gin.Context) {
	var config ChatConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "配置参数错误: " + err.Error(),
		})
		return
	}

	// 创建 agent 配置
	agentConfig := createAgentConfig(&config)

	// 验证配置
	if err := agentConfig.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "配置验证失败: " + err.Error(),
		})
		return
	}

	// 尝试创建 agent
	agent, err := eino.NewAgent(agentConfig)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "创建 Agent 失败: " + err.Error(),
		})
		return
	}
	defer agent.Close()

	// 返回配置信息
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "配置测试成功",
		"config": gin.H{
			"model_name":    agentConfig.Model.Name,
			"provider":      agentConfig.Model.Provider,
			"endpoint":      agentConfig.Model.Endpoint,
			"temperature":   agentConfig.Model.Temperature,
			"max_tokens":    agentConfig.Model.MaxTokens,
			"system_prompt": agentConfig.GetExtensionWithDefault("system_prompt", ""),
			"framework":     "eino",
		},
	})
}

// HealthHandler 健康检查接口
func HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Eino Agent API 运行正常",
		"framework": "eino",
		"timestamp": time.Now().Unix(),
		"endpoints": []string{
			"POST /agents/chat - 基本聊天",
			"POST /agents/plan - 生成计划",
			"POST /agents/execute - 执行意图",
			"POST /agents/stream - 流式聊天",
			"POST /agents/config/test - 配置测试",
			"GET /agents/health - 健康检查",
		},
	})
}

// createAgentConfig 创建 agent 配置
func createAgentConfig(apiConfig *ChatConfig) *eino.Config {
	config := eino.NewConfig()

	if apiConfig != nil {
		// 设置模型配置
		if apiConfig.ModelName != "" {
			config.WithModelName(apiConfig.ModelName)
		}
		if apiConfig.Provider != "" {
			config.WithModelProvider(apiConfig.Provider)
		}
		if apiConfig.Endpoint != "" {
			config.WithModelEndpoint(apiConfig.Endpoint)
		}
		if apiConfig.APIKey != "" {
			config.WithModelAPIKey(apiConfig.APIKey)
		}
		if apiConfig.Temperature > 0 {
			config.WithModelTemperature(apiConfig.Temperature)
		}
		if apiConfig.MaxTokens > 0 {
			config.WithModelMaxTokens(apiConfig.MaxTokens)
		}
		if apiConfig.SystemPrompt != "" {
			config.WithExtension("system_prompt", apiConfig.SystemPrompt)
		}
		if apiConfig.EnableStream {
			config.WithExtension("enable_stream", true)
		}
	}

	return config
}

// parseIntParam 解析整数参数
func parseIntParam(c *gin.Context, key string, defaultValue int) int {
	if str := c.Query(key); str != "" {
		if val, err := strconv.Atoi(str); err == nil {
			return val
		}
	}
	return defaultValue
}

// parseFloatParam 解析浮点数参数
func parseFloatParam(c *gin.Context, key string, defaultValue float64) float64 {
	if str := c.Query(key); str != "" {
		if val, err := strconv.ParseFloat(str, 64); err == nil {
			return val
		}
	}
	return defaultValue
}
