package eino

// config.go - Eino智能体的配置结构体

import (
	"fmt"
	"time"

	"github.com/ArtisanCloud/CoreX/pkg/agent/contract"
)

// Config 定义Eino智能体的配置
type Config struct {
	// Intent 解析 / prompt 转 plan 所需的上下文
	IntentPrompt string

	// 额外上下文 metadata（例如 tenant_id, user_id 等）
	ContextMetadata map[string]interface{}

	// 失败重试策略，可以为空，驱动内部会合并
	RetryPolicy *contract.RetryPolicy

	// 模型配置 - 强类型定义
	Model *ModelConfig

	// 工具配置
	Tools *ToolsConfig

	// 执行配置
	Execution *ExecutionConfig

	// 回调配置
	Callbacks *CallbacksConfig

	// 其他扩展选项（保留少量灵活性）
	Extensions map[string]interface{}
}

// ModelConfig 模型相关配置
type ModelConfig struct {
	// 模型名称
	Name string

	// 模型提供商 (openai, anthropic, qwen, etc.)
	Provider string

	// API 端点
	Endpoint string

	// API 密钥
	APIKey string

	// 模型参数
	Temperature      float64
	MaxTokens        int
	TopP             float64
	TopK             int
	FrequencyPenalty float64
	PresencePenalty  float64

	// 超时设置
	Timeout time.Duration

	// 代理设置
	ProxyURL string

	// 自定义头部
	Headers map[string]string

	// 模型特定选项
	ModelOptions map[string]interface{}
}

// ToolsConfig 工具相关配置
type ToolsConfig struct {
	// 启用的工具列表
	EnabledTools []string

	// 工具配置映射
	ToolConfigs map[string]*ToolConfig

	// 工具超时
	DefaultTimeout time.Duration

	// 并行执行工具的最大数量
	MaxParallelTools int
}

// ToolConfig 单个工具的配置
type ToolConfig struct {
	// 工具名称
	Name string

	// 工具类型
	Type string

	// 工具端点（如果是API工具）
	Endpoint string

	// 认证信息
	Auth *AuthConfig

	// 工具特定参数
	Parameters map[string]interface{}

	// 超时设置
	Timeout time.Duration
}

// AuthConfig 认证配置
type AuthConfig struct {
	Type     string // bearer, basic, api_key, etc.
	Token    string
	Username string
	Password string
	APIKey   string
	Headers  map[string]string
}

// ExecutionConfig 执行相关配置
type ExecutionConfig struct {
	// 最大执行步骤数
	MaxSteps int

	// 执行超时
	Timeout time.Duration

	// 是否启用并行执行
	EnableParallel bool

	// 最大并行度
	MaxParallel int

	// 是否启用流式执行
	EnableStreaming bool

	// 检查点存储配置
	CheckpointStore *CheckpointConfig
}

// CheckpointConfig 检查点存储配置
type CheckpointConfig struct {
	// 存储类型 (memory, redis, file, etc.)
	Type string

	// 存储配置
	Config map[string]interface{}
}

// CallbacksConfig 回调配置
type CallbacksConfig struct {
	// 启用的回调类型
	EnabledCallbacks []string

	// 日志级别
	LogLevel string

	// 指标收集
	EnableMetrics bool

	// 事件处理器配置
	EventHandlers map[string]interface{}
}

// NewConfig 创建新的Eino配置
func NewConfig() *Config {
	return &Config{
		IntentPrompt:    "",
		ContextMetadata: make(map[string]interface{}),
		RetryPolicy:     nil,
		Model:           NewModelConfig(),
		Tools:           NewToolsConfig(),
		Execution:       NewExecutionConfig(),
		Callbacks:       NewCallbacksConfig(),
		Extensions:      make(map[string]interface{}),
	}
}

// NewModelConfig 创建默认模型配置
func NewModelConfig() *ModelConfig {
	return &ModelConfig{
		Name:             "gpt-3.5-turbo",
		Provider:         "openai",
		Endpoint:         "",
		APIKey:           "",
		Temperature:      0.7,
		MaxTokens:        2048,
		TopP:             1.0,
		TopK:             0,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
		Timeout:          30 * time.Second,
		ProxyURL:         "",
		Headers:          make(map[string]string),
		ModelOptions:     make(map[string]interface{}),
	}
}

// NewToolsConfig 创建默认工具配置
func NewToolsConfig() *ToolsConfig {
	return &ToolsConfig{
		EnabledTools:     []string{},
		ToolConfigs:      make(map[string]*ToolConfig),
		DefaultTimeout:   10 * time.Second,
		MaxParallelTools: 3,
	}
}

// NewExecutionConfig 创建默认执行配置
func NewExecutionConfig() *ExecutionConfig {
	return &ExecutionConfig{
		MaxSteps:        50,
		Timeout:         5 * time.Minute,
		EnableParallel:  false,
		MaxParallel:     3,
		EnableStreaming: false,
		CheckpointStore: nil,
	}
}

// NewCallbacksConfig 创建默认回调配置
func NewCallbacksConfig() *CallbacksConfig {
	return &CallbacksConfig{
		EnabledCallbacks: []string{"logging"},
		LogLevel:         "info",
		EnableMetrics:    false,
		EventHandlers:    make(map[string]interface{}),
	}
}

// WithIntentPrompt 设置意图提示模板
func (c *Config) WithIntentPrompt(prompt string) *Config {
	c.IntentPrompt = prompt
	return c
}

// WithContextMetadata 设置上下文元数据
func (c *Config) WithContextMetadata(metadata map[string]interface{}) *Config {
	c.ContextMetadata = metadata
	return c
}

// WithRetryPolicy 设置重试策略
func (c *Config) WithRetryPolicy(policy *contract.RetryPolicy) *Config {
	c.RetryPolicy = policy
	return c
}

// WithModel 设置模型配置
func (c *Config) WithModel(model *ModelConfig) *Config {
	c.Model = model
	return c
}

// WithModelName 设置模型名称
func (c *Config) WithModelName(name string) *Config {
	if c.Model == nil {
		c.Model = NewModelConfig()
	}
	c.Model.Name = name
	return c
}

// WithModelProvider 设置模型提供商
func (c *Config) WithModelProvider(provider string) *Config {
	if c.Model == nil {
		c.Model = NewModelConfig()
	}
	c.Model.Provider = provider
	return c
}

// WithModelEndpoint 设置模型端点
func (c *Config) WithModelEndpoint(endpoint string) *Config {
	if c.Model == nil {
		c.Model = NewModelConfig()
	}
	c.Model.Endpoint = endpoint
	return c
}

// WithModelAPIKey 设置模型API密钥
func (c *Config) WithModelAPIKey(apiKey string) *Config {
	if c.Model == nil {
		c.Model = NewModelConfig()
	}
	c.Model.APIKey = apiKey
	return c
}

// WithModelTemperature 设置模型温度
func (c *Config) WithModelTemperature(temperature float64) *Config {
	if c.Model == nil {
		c.Model = NewModelConfig()
	}
	c.Model.Temperature = temperature
	return c
}

// WithModelMaxTokens 设置最大令牌数
func (c *Config) WithModelMaxTokens(maxTokens int) *Config {
	if c.Model == nil {
		c.Model = NewModelConfig()
	}
	c.Model.MaxTokens = maxTokens
	return c
}

// WithTools 设置工具配置
func (c *Config) WithTools(tools *ToolsConfig) *Config {
	c.Tools = tools
	return c
}

// WithEnabledTools 设置启用的工具列表
func (c *Config) WithEnabledTools(tools []string) *Config {
	if c.Tools == nil {
		c.Tools = NewToolsConfig()
	}
	c.Tools.EnabledTools = tools
	return c
}

// WithExecution 设置执行配置
func (c *Config) WithExecution(execution *ExecutionConfig) *Config {
	c.Execution = execution
	return c
}

// WithMaxSteps 设置最大执行步骤数
func (c *Config) WithMaxSteps(maxSteps int) *Config {
	if c.Execution == nil {
		c.Execution = NewExecutionConfig()
	}
	c.Execution.MaxSteps = maxSteps
	return c
}

// WithCallbacks 设置回调配置
func (c *Config) WithCallbacks(callbacks *CallbacksConfig) *Config {
	c.Callbacks = callbacks
	return c
}

// WithExtension 设置扩展选项
func (c *Config) WithExtension(key string, value interface{}) *Config {
	if c.Extensions == nil {
		c.Extensions = make(map[string]interface{})
	}
	c.Extensions[key] = value
	return c
}

// GetExtension 获取扩展选项
func (c *Config) GetExtension(key string) (interface{}, bool) {
	if c.Extensions == nil {
		return nil, false
	}
	value, exists := c.Extensions[key]
	return value, exists
}

// GetExtensionWithDefault 获取扩展选项，如果不存在则返回默认值
func (c *Config) GetExtensionWithDefault(key string, defaultValue interface{}) interface{} {
	if value, exists := c.GetExtension(key); exists {
		return value
	}
	return defaultValue
}

// GetOptionWithDefault 获取特定选项，如果不存在则返回默认值（向后兼容）
func (c *Config) GetOptionWithDefault(key string, defaultValue interface{}) interface{} {
	if value, exists := c.GetOption(key); exists {
		return value
	}
	return defaultValue
}

// WithOption 设置特定选项（向后兼容）
func (c *Config) WithOption(key string, value interface{}) *Config {
	// 尝试映射到强类型字段
	if c.Model != nil {
		switch key {
		case "model_name":
			if str, ok := value.(string); ok {
				c.Model.Name = str
				return c
			}
		case "temperature":
			if f, ok := value.(float64); ok {
				c.Model.Temperature = f
				return c
			}
		case "max_tokens":
			if i, ok := value.(int); ok {
				c.Model.MaxTokens = i
				return c
			}
		case "endpoint":
			if str, ok := value.(string); ok {
				c.Model.Endpoint = str
				return c
			}
		case "api_key":
			if str, ok := value.(string); ok {
				c.Model.APIKey = str
				return c
			}
		}
	}

	// 如果无法映射到强类型字段，存储到扩展选项中
	return c.WithExtension(key, value)
}

// GetOption 获取特定选项（向后兼容）
func (c *Config) GetOption(key string) (interface{}, bool) {
	// 首先检查模型配置
	if c.Model != nil {
		switch key {
		case "model_name":
			return c.Model.Name, true
		case "temperature":
			return c.Model.Temperature, true
		case "max_tokens":
			return c.Model.MaxTokens, true
		case "endpoint":
			return c.Model.Endpoint, true
		case "api_key":
			return c.Model.APIKey, true
		}
	}

	// 然后检查扩展选项
	return c.GetExtension(key)
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	if c.Model != nil {
		if c.Model.Name == "" {
			return fmt.Errorf("模型名称不能为空")
		}
		if c.Model.Provider == "" {
			return fmt.Errorf("模型提供商不能为空")
		}
		if c.Model.Temperature < 0 || c.Model.Temperature > 2 {
			return fmt.Errorf("温度值必须在 0-2 之间")
		}
		if c.Model.MaxTokens <= 0 {
			return fmt.Errorf("最大令牌数必须大于 0")
		}
	}

	if c.Execution != nil {
		if c.Execution.MaxSteps <= 0 {
			return fmt.Errorf("最大步骤数必须大于 0")
		}
		if c.Execution.Timeout <= 0 {
			return fmt.Errorf("执行超时时间必须大于 0")
		}
	}

	return nil
}
