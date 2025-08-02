package http

import (
	"github.com/ArtisanCloud/CoreX/config"
	"github.com/gin-gonic/gin"
)

// RegisterAPIRoutes 负责挂载所有业务路由
func RegisterAPIRoutes(r *gin.Engine, authMiddleware gin.HandlerFunc, cfg *config.Config) {
	prefix := cfg.Server.APIPrefix
	if prefix == "" {
		prefix = "/api"
	}
	publicGroup := r.Group(prefix)
	// 公开健康检查
	publicGroup.GET("/health", HealthHandler)

	// 公开的JWT令牌生成端点（仅用于开发测试）
	publicGroup.POST("/auth/generate_token", GenerateTokenHandler(cfg))

	// 受保护的API组
	protected := r.Group(prefix)
	protected.Use(authMiddleware)

	gFlow := protected.Group("/flows")
	// 启动流程端点
	gFlow.POST("/start_flow", StartFlowHandler)
}
