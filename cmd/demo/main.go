package main

import (
	"context"
	"fmt"
	"os"

	apiHttp "github.com/ArtisanCloud/CoreX/api/http"
	"github.com/ArtisanCloud/CoreX/config"
	httpRouter "github.com/ArtisanCloud/CoreX/internal/http"
	"github.com/ArtisanCloud/CoreX/pkg/auth"
	"github.com/ArtisanCloud/CoreX/pkg/event_bus"
	"github.com/ArtisanCloud/CoreX/pkg/utils/logger"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// loadEnvFile 加载 .env 文件
func loadEnvFile() {
	// 尝试加载 .env 文件，如果不存在也不报错
	if err := godotenv.Load(); err != nil {
		fmt.Println("提示：未找到 .env 文件，将使用环境变量或默认值")
	} else {
		fmt.Println("✅ 成功加载 .env 配置文件")
	}
}

// loadConfig 加载统一配置
func loadConfig() (*config.Config, error) {
	// 先加载 .env 文件
	loadEnvFile()

	// 加载配置
	cfg, err := config.Load("config/example.yaml")
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func main() {
	// 1. 加载统一配置
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("配置加载失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化全局Logger
	if err := initLogger(cfg); err != nil {
		fmt.Printf("Logger初始化失败: %v\n", err)
		os.Exit(1)
	}

	// 3. 初始化 auth（JWT secret、middleware）
	authMiddleware := initAuth(cfg)

	// 3. 初始化工具（agent_tools）
	initTools()

	// 4. 初始化事件总线 / 插件订阅
	initEventBus()

	// 5. 构建 router 并挂载路由
	r := httpRouter.SetupRouter(authMiddleware, func(r *gin.Engine) {
		apiHttp.RegisterAPIRoutes(r, authMiddleware, cfg)
	})

	// 6. 打印路由信息
	httpRouter.PrintRouteInfo(r, cfg)

	// 7. 启动服务
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("🚀 CoreX 服务启动成功！监听地址: http://localhost%s\n", addr)
	r.Run(addr)
}

// initLogger 初始化全局Logger
func initLogger(cfg *config.Config) error {
	// 使用配置中的日志配置初始化全局Logger
	logger.InitGlobalLogger(&cfg.LogConfig)

	// 测试全局Logger是否工作正常
	logger.Info(context.Background(), "🚀 全局Logger初始化成功")
	return nil
}

// initAuth 设置全局 JWT secret 并返回 gin middleware 实例
func initAuth(cfg *config.Config) gin.HandlerFunc {
	// 赋值给 auth 包
	auth.SetJWTSecret([]byte(cfg.Auth.JWTSecret))

	// 使用配置中的认证参数
	expectedAudience := cfg.Auth.ExpectedAudience
	requiredScopes := cfg.Auth.RequiredScopes

	// 传入 SampleCallback 做扩展判断&事件广播
	return auth.JwtMiddleware(expectedAudience, requiredScopes, auth.SampleCallback)
}

// initTools 注册 agent_tools（apply_tag 已在其 init 里注册，示例保底）
func initTools() {
	// 如果有动态注册逻辑可以写在这里
	// 例如：agent_tools.Register(NewCustomTool(...))
}

// initEventBus 订阅全局事件（示例 plugin 行为）
func initEventBus() {
	// 初始化默认事件总线
	event_bus.InitDefaultEventBus(&event_bus.Config{Type: "local"})

	// 订阅认证成功事件
	event_bus.Subscribe("auth_succeeded", func(e event_bus.Event) error {
		if payload, ok := e.Payload.(map[string]interface{}); ok {
			fmt.Printf("[plugin] auth_succeeded: tenant=%v subject=%v platform=%v trace_id=%v scope=%v\n",
				payload["tenant_id"], payload["subject"], payload["platform"], payload["trace_id"], payload["scope"])
		}
		return nil
	})

	// 订阅流程完成事件
	event_bus.Subscribe("flow_completed", func(e event_bus.Event) error {
		if payload, ok := e.Payload.(map[string]interface{}); ok {
			fmt.Printf("[plugin] flow_completed: tenant=%v flow=%v subject=%v trace_id=%v\n",
				payload["tenant_id"], payload["flow_name"], payload["subject"], payload["trace_id"])
		}
		return nil
	})
}
