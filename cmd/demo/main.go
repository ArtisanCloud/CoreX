package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ArtisanCloud/CoreX/config"
	"github.com/ArtisanCloud/CoreX/pkg/auth"
	"github.com/ArtisanCloud/CoreX/pkg/event_bus"
	"github.com/ArtisanCloud/CoreX/pkg/logger"
	"github.com/ArtisanCloud/CoreX/pkg/low_code"
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

// printRouteInfo 打印路由信息
func printRouteInfo(r *gin.Engine, cfg *config.Config) {
	fmt.Println()
	fmt.Println("📋 已注册的 API 路由（摘要）:")
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Printf("│ 服务地址: http://localhost:%-29d│\n", cfg.Server.Port)
	fmt.Println("├─────────────────────────────────────────────────────────────┤")
	routes := r.Routes()
	if len(routes) == 0 {
		fmt.Println("│ (未发现已注册路由，请确认路由是否在 bootstrap 阶段同步注册)        │")
	}
	for _, route := range routes {
		authStatus := "公开"
		if strings.HasPrefix(route.Path, "/api/") {
			authStatus = "🔒 需要 JWT"
		}
		fmt.Printf("│ %-6s %-25s - %-10s │\n", route.Method, route.Path, authStatus)
	}
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	fmt.Println()
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
	r := initRouter(authMiddleware, cfg)

	// 6. 打印路由信息
	printRouteInfo(r, cfg)

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
	logger.Info("🚀 全局Logger初始化成功")
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

// initRouter 构造 gin 引擎并挂载路由
func initRouter(authMiddleware gin.HandlerFunc, cfg *config.Config) *gin.Engine {
	r := gin.Default()

	// 公开健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 公开的JWT令牌生成端点（仅用于开发测试）
	r.POST("/auth/generate_token", func(c *gin.Context) {
		var req struct {
			TenantID string `json:"tenant_id" binding:"required"`
			Subject  string `json:"subject" binding:"required"`
			Platform string `json:"platform"`
			Audience string `json:"audience"`
			Scope    string `json:"scope"`
			TTL      int    `json:"ttl_hours"` // 小时数
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误", "detail": err.Error()})
			return
		}

		// 设置默认值
		if req.Platform == "" {
			req.Platform = "web"
		}
		if req.Audience == "" {
			req.Audience = "admin"
		}
		if req.Scope == "" {
			req.Scope = "flow:execute"
		}
		if req.TTL == 0 {
			req.TTL = 24 // 默认24小时
		}

		// 生成JWT令牌
		token, err := auth.GenerateJWT(
			req.TenantID,
			req.Subject,
			req.Platform,
			req.Audience,
			req.Scope,
			time.Duration(req.TTL)*time.Hour,
			[]byte(cfg.Auth.JWTSecret),
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败", "detail": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":      token,
			"tenant_id":  req.TenantID,
			"subject":    req.Subject,
			"platform":   req.Platform,
			"audience":   req.Audience,
			"scope":      req.Scope,
			"expires_in": req.TTL * 3600, // 秒数
			"usage":      fmt.Sprintf("curl -X POST http://localhost:%d/api/start_flow -H \"Authorization: Bearer %s\" -H \"Content-Type: application/json\"", cfg.Server.Port, token),
		})
	})

	// 受保护组
	protected := r.Group("/api")
	protected.Use(authMiddleware)

	// 启动 flow 示例
	protected.POST("/start_flow", func(c *gin.Context) {
		ctx := c.Request.Context()
		tenant := auth.GetTenantID(ctx)
		subject := auth.GetSubject(ctx)
		traceID := auth.GetTraceID(ctx)

		flow := low_code.Flow{
			Name: "example_flow",
			Steps: []low_code.Step{
				{
					ToolName: "apply_tag",
					Input: map[string]interface{}{
						"customer_id": "c-123",
						"tag":         "vip",
					},
				},
			},
		}
		if err := low_code.ExecuteFlow(ctx, flow); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		event_bus.Publish(event_bus.Event{
			Name: "flow_completed",
			Payload: map[string]interface{}{
				"tenant_id": tenant,
				"flow_name": flow.Name,
				"subject":   subject,
				"trace_id":  traceID,
			},
			Ctx: ctx,
			ID:  traceID,
		})

		c.JSON(http.StatusOK, gin.H{
			"status":    "flow executed",
			"tenant":    tenant,
			"subject":   subject,
			"trace_id":  traceID,
			"flow_name": flow.Name,
		})
	})

	return r
}
