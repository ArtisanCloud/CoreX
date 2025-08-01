package config

import (
	logCfg "github.com/ArtisanCloud/CoreX/pkg/logger/config"
)

// GetDefaults 返回默认配置
func GetDefaults() *Config {
	return &Config{
		Server: ServerConfig{
			Port:                8080,
			ReadTimeoutSeconds:  5,
			WriteTimeoutSeconds: 10,
			Mode:                "debug",
		},
		LogConfig: logCfg.LogConfig{
			Level:         "debug",
			Console:       true,
			UseJsonFormat: false,
			File: logCfg.FileConfig{
				Enable:        false,
				InfoFilePath:  "logs/info.log",
				ErrorFilePath: "logs/error.log",
				MaxSize:       100,
				MaxBackups:    5,
				MaxAge:        30,
				Compress:      true,
			},
			Loki: logCfg.LokiConfig{
				Enable:    false,
				URL:       "",
				JobName:   "corex",
				BatchWait: 1,
				BatchSize: 100,
			},
			HttpDebug: false,
			Debug:     true,
		},
		Auth: AuthConfig{
			JWTSecret:        "K8mN2pQ7rS9tU4vW6xY1zA3bC5dE8fG0",
			ExpectedAudience: "admin",
			RequiredScopes:   []string{"flow:execute"},
			TokenTTLHours:    24,
		},
		EventBus: EventBusConfig{
			Type:          "local",
			RedisAddr:     "localhost:6379",
			RedisPassword: "",
			DedupeTTLSec:  30,
		},
		LowCode: LowCodeConfig{
			MaxConcurrentFlows: 10,
			DefaultTimeoutSec:  60,
		},
		AgentTools: AgentToolsConfig{
			EnableAudit: true,
		},
		FeatureGate: FeatureGateConfig{
			LicenseKey: "demo-license-xyz",
		},
	}
}
