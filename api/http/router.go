package http

import (
	"github.com/ArtisanCloud/CoreX/config"
	"github.com/gin-gonic/gin"
)

// RegisterAPIRoutes 负责挂载所有业务路由
func RegisterAPIRoutes(r *gin.Engine, authMiddleware gin.HandlerFunc, cfg *config.Config) {
	// 公开健康检查
	r.GET("/health", HealthHandler)

	// 公开的JWT令牌生成端点（仅用于开发测试）
	r.POST("/auth/generate_token", GenerateTokenHandler(cfg))

	// 受保护的API组
	protected := r.Group("/api")
	protected.Use(authMiddleware)

	// 启动流程端点
	protected.POST("/start_flow", StartFlowHandler)
}
