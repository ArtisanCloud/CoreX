package http

import (
	"fmt"
	"strings"

	"github.com/ArtisanCloud/CoreX/config"
	"github.com/gin-gonic/gin"
)

// SetupRouter 构造带基础中间件的 Gin 引擎，外部传入 auth middleware 和自定义 route 注册函数
func SetupRouter(authMiddleware gin.HandlerFunc, registerFunc func(r *gin.Engine)) *gin.Engine {
	r := gin.New()

	// 全局模式/日志/恢复/trace
	r.Use(RecoveryMiddleware())
	r.Use(RequestLoggingMiddleware())
	r.Use(TraceInjectionMiddleware())

	// 可以挂在全局 feature injection、版本头等
	r.Use(FeatureInjectionMiddleware())

	// 注册用户传入的路由
	registerFunc(r)

	return r
}

// PrintRouteInfo 打印路由信息
func PrintRouteInfo(r *gin.Engine, cfg *config.Config) {
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
