package eino

// config.go - Eino智能体的配置结构体

import (
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

	// 可扩展的 driver-specific 选项
	Options map[string]interface{}
}

// NewConfig 创建新的Eino配置
func NewConfig() *Config {
	return &Config{
		IntentPrompt:    "",
		ContextMetadata: make(map[string]interface{}),
		RetryPolicy:     nil,
		Options:         make(map[string]interface{}),
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

// WithOption 设置特定选项
func (c *Config) WithOption(key string, value interface{}) *Config {
	c.Options[key] = value
	return c
}

// GetOption 获取特定选项
func (c *Config) GetOption(key string) (interface{}, bool) {
	value, exists := c.Options[key]
	return value, exists
}

// GetOptionWithDefault 获取特定选项，如果不存在则返回默认值
func (c *Config) GetOptionWithDefault(key string, defaultValue interface{}) interface{} {
	if value, exists := c.Options[key]; exists {
		return value
	}
	return defaultValue
}
