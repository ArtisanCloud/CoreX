package agent

import "github.com/gin-gonic/gin"

func RegisterAPIRoutes(publicGroup *gin.RouterGroup, protectedGroup *gin.RouterGroup) {
	agentGroup := protectedGroup.Group("/agents")
	{
		agentGroup.GET("/health", HealthHandler)
		agentGroup.POST("/chat", ChatHandler)
		agentGroup.POST("/plan", PlanHandler)
		agentGroup.POST("/execute", ExecuteHandler)
		agentGroup.POST("/stream", StreamChatHandler)
		agentGroup.POST("/config/test", ConfigTestHandler)
	}
}
