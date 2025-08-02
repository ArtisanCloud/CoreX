package factory

import (
	"github.com/ArtisanCloud/CoreX/pkg/agent/contract"
)

// AgentConfig 由调用方提供，用来决定用哪个驱动和基础策略。
type AgentConfig struct {
	// 选用的驱动名称，空则默认 "eino"
	Driver string

	// Intent 解析 / prompt 转 plan 所需的上下文（按各驱动解释）
	IntentPrompt string

	// 额外上下文 metadata（例如 tenant_id, user_id 等）
	ContextMetadata map[string]interface{}

	// 失败重试策略，可以为空，驱动内部会合并
	RetryPolicy *contract.RetryPolicy

	// 可扩展的 driver-specific 选项
	Options map[string]interface{}
}
