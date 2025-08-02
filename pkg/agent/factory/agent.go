package factory

import (
	"context"
	"fmt"

	"github.com/ArtisanCloud/CoreX/pkg/agent/contract"
	"github.com/ArtisanCloud/CoreX/pkg/agent/drivers/eino"
)

const defaultDriver = "eino"

// NewAgent 根据配置创建一个 contract.Agent。
func NewAgent(ctx context.Context, cfg *AgentConfig) (contract.Agent, error) {
	driverName := cfg.Driver
	if driverName == "" {
		driverName = defaultDriver
	}

	switch driverName {
	case "eino":
		// 由 eino 包提供构造函数（你在 drivers/eino 里实现）
		// 约定 eino 包有一个 NewAgentFromConfig 接口接收 AgentConfig 或拆解后的依赖
		// 创建 eino 配置
		einoConfig := eino.NewConfig().
			WithIntentPrompt(cfg.IntentPrompt).
			WithContextMetadata(cfg.ContextMetadata).
			WithRetryPolicy(cfg.RetryPolicy)

		// 如果有 Options，转换为扩展配置
		if cfg.Options != nil {
			for key, value := range cfg.Options {
				einoConfig.WithExtension(key, value)
			}
		}

		agent, err := eino.NewAgent(einoConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build eino agent: %w", err)
		}
		return agent, nil
	default:
		return nil, fmt.Errorf("unknown agent driver: %s", driverName)
	}
}
