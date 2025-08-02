package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/CoreX/pkg/agent/factory"
	logCfg "github.com/ArtisanCloud/CoreX/pkg/utils/logger/config"
)

// CoreX 全局配置
type Config struct {
	Server      ServerConfig        `yaml:"server"`         // HTTP/gRPC 监听与行为
	LogConfig   logCfg.LogConfig    `yaml:"logging_config"` // 输出配置
	Auth        AuthConfig          `yaml:"auth"`           // JWT / 认证相关
	EventBus    EventBusConfig      `yaml:"event_bus"`      // 事件总线（local/redis）
	LowCode     LowCodeConfig       `yaml:"dynamic_form"`   // flow 执行相关
	AgentTools  factory.AgentConfig `yaml:"agent_tools"`    // 智能体工具注册/限流等
	FeatureGate FeatureGateConfig   `yaml:"feature_gate"`   // 细粒度开关、license
}

// HTTP服务器配置
type ServerConfig struct {
	Port                int    `yaml:"port"`                  // HTTP 端口
	ReadTimeoutSeconds  int    `yaml:"read_timeout_seconds"`  // 读取超时
	WriteTimeoutSeconds int    `yaml:"write_timeout_seconds"` // 写入超时
	Mode                string `yaml:"mode"`                  // gin 模式: debug/release
	APIPrefix           string `yaml:"api_prefix"`            // API 前缀
}

// JWT认证配置
type AuthConfig struct {
	JWTSecret        string   `yaml:"jwt_secret"`        // HMAC secret 或私钥路径
	ExpectedAudience string   `yaml:"expected_audience"` // 期望 audience，例如 admin/openapi/miniapp
	RequiredScopes   []string `yaml:"required_scopes"`   // 必需 scope
	TokenTTLHours    int      `yaml:"token_ttl_hours"`   // 默认 token 过期小时
}

// 事件总线配置
type EventBusConfig struct {
	Type          string `yaml:"type"`           // local / redis
	RedisAddr     string `yaml:"redis_addr"`     // redis 地址
	RedisPassword string `yaml:"redis_password"` // redis 密码
	DedupeTTLSec  int    `yaml:"dedupe_ttl_sec"` // 幂等缓存过期
}

// 低代码引擎配置
type LowCodeConfig struct {
	MaxConcurrentFlows int `yaml:"max_concurrent_flows"` // 并发 flow 限制
	DefaultTimeoutSec  int `yaml:"default_timeout_sec"`  // 每个 flow 默认超时
}

// 功能开关配置
type FeatureGateConfig struct {
	LicenseKey string `yaml:"license_key"` // license 或灰度控制 token
}

// Load 加载配置文件并合并环境变量
func Load(configPath string) (*Config, error) {
	// 1. 加载默认配置
	cfg := GetDefaults()

	// 2. 从YAML文件加载（如果存在）
	if configPath != "" {
		if err := loadFromYAML(cfg, configPath); err != nil {
			return nil, fmt.Errorf("加载YAML配置失败: %w", err)
		}
	}

	// 3. 从环境变量覆盖
	loadFromEnv(cfg)

	// 4. 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return cfg, nil
}

// loadFromEnv 从环境变量加载配置
func loadFromEnv(cfg *Config) {
	// Server配置
	if port := os.Getenv("CORE_X_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}
	if mode := os.Getenv("CORE_X_SERVER_MODE"); mode != "" {
		cfg.Server.Mode = mode
	}
	if timeout := os.Getenv("CORE_X_SERVER_READ_TIMEOUT"); timeout != "" {
		if t, err := strconv.Atoi(timeout); err == nil {
			cfg.Server.ReadTimeoutSeconds = t
		}
	}
	if timeout := os.Getenv("CORE_X_SERVER_WRITE_TIMEOUT"); timeout != "" {
		if t, err := strconv.Atoi(timeout); err == nil {
			cfg.Server.WriteTimeoutSeconds = t
		}
	}

	// Auth配置
	if secret := os.Getenv("CORE_X_AUTH_JWT_SECRET"); secret != "" {
		cfg.Auth.JWTSecret = secret
	}
	if audience := os.Getenv("CORE_X_AUTH_EXPECTED_AUDIENCE"); audience != "" {
		cfg.Auth.ExpectedAudience = audience
	}
	if scopes := os.Getenv("CORE_X_AUTH_REQUIRED_SCOPES"); scopes != "" {
		cfg.Auth.RequiredScopes = strings.Split(scopes, ",")
	}
	if ttl := os.Getenv("CORE_X_AUTH_TOKEN_TTL_HOURS"); ttl != "" {
		if t, err := strconv.Atoi(ttl); err == nil {
			cfg.Auth.TokenTTLHours = t
		}
	}

	// EventBus配置
	if busType := os.Getenv("CORE_X_EVENT_BUS_TYPE"); busType != "" {
		cfg.EventBus.Type = busType
	}
	if redisAddr := os.Getenv("CORE_X_EVENT_BUS_REDIS_ADDR"); redisAddr != "" {
		cfg.EventBus.RedisAddr = redisAddr
	}
	if redisPassword := os.Getenv("CORE_X_EVENT_BUS_REDIS_PASSWORD"); redisPassword != "" {
		cfg.EventBus.RedisPassword = redisPassword
	}
	if ttl := os.Getenv("CORE_X_EVENT_BUS_DEDUPE_TTL_SEC"); ttl != "" {
		if t, err := strconv.Atoi(ttl); err == nil {
			cfg.EventBus.DedupeTTLSec = t
		}
	}

	// LowCode配置
	if maxFlows := os.Getenv("CORE_X_LOW_CODE_MAX_CONCURRENT_FLOWS"); maxFlows != "" {
		if m, err := strconv.Atoi(maxFlows); err == nil {
			cfg.LowCode.MaxConcurrentFlows = m
		}
	}
	if timeout := os.Getenv("CORE_X_LOW_CODE_DEFAULT_TIMEOUT_SEC"); timeout != "" {
		if t, err := strconv.Atoi(timeout); err == nil {
			cfg.LowCode.DefaultTimeoutSec = t
		}
	}

	// AgentTools配置
	if audit := os.Getenv("CORE_X_AGENT_TOOLS_ENABLE_AUDIT"); audit != "" {
		// cfg.AgentTools.EnableAudit = strings.ToLower(audit) == "true"
	}

	// LogConfig配置 - 使用外部logger配置
	// 这里可以根据需要添加对LogConfig字段的环境变量支持
	// 例如：cfg.LogConfig.Level, cfg.LogConfig.Format 等

	// FeatureGate配置
	if license := os.Getenv("CORE_X_FEATURE_GATE_LICENSE_KEY"); license != "" {
		cfg.FeatureGate.LicenseKey = license
	}

	// 兼容旧的环境变量
	if secret := os.Getenv("CORE_X_JWT_SECRET"); secret != "" && cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = secret
	}
	if port := os.Getenv("CORE_X_PORT"); port != "" && cfg.Server.Port == 8080 {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}
	if mode := os.Getenv("GIN_MODE"); mode != "" && cfg.Server.Mode == "debug" {
		cfg.Server.Mode = mode
	}
	if busType := os.Getenv("EVENT_BUS_TYPE"); busType != "" && cfg.EventBus.Type == "local" {
		cfg.EventBus.Type = busType
	}
}
