package http

import (
	"net/http"

	"github.com/ArtisanCloud/CoreX/pkg/auth"
	"github.com/ArtisanCloud/CoreX/pkg/dynamic_form"
	"github.com/ArtisanCloud/CoreX/pkg/event_bus"
	"github.com/gin-gonic/gin"
)

// StartFlowHandler 启动流程处理器
func StartFlowHandler(c *gin.Context) {
	ctx := c.Request.Context()
	tenant := auth.GetTenantID(ctx)
	subject := auth.GetSubject(ctx)
	traceID := auth.GetTraceID(ctx)

	// 构建示例流程
	flow := dynamic_form.Flow{
		Name: "example_flow",
		Steps: []dynamic_form.Step{
			{
				ToolName: "apply_tag",
				Input: map[string]interface{}{
					"customer_id": "c-123",
					"tag":         "vip",
				},
			},
		},
	}

	// 执行流程
	if err := dynamic_form.ExecuteFlow(ctx, flow); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 发布流程完成事件
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

	// 返回执行结果
	c.JSON(http.StatusOK, gin.H{
		"status":    "flow executed",
		"tenant":    tenant,
		"subject":   subject,
		"trace_id":  traceID,
		"flow_name": flow.Name,
	})
}
