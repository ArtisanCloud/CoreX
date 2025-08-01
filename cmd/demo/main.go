package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ArtisanCloud/CoreX/pkg/agent_tools"
	"github.com/ArtisanCloud/CoreX/pkg/auth"
	"github.com/ArtisanCloud/CoreX/pkg/event_bus"
	"github.com/ArtisanCloud/CoreX/pkg/license"
	"github.com/ArtisanCloud/CoreX/pkg/plugin"
)

func main() {
	ctx := context.Background()
	
	// 初始化日志
	log.Println("正在启动 CoreX 演示程序...")
	
	// 初始化许可证系统
	log.Println("初始化许可证系统...")
	if err := license.Init(); err != nil {
		log.Fatalf("许可证系统初始化失败: %v", err)
	}
	
	// 初始化认证系统
	log.Println("初始化认证系统...")
	if err := auth.Init(); err != nil {
		log.Fatalf("认证系统初始化失败: %v", err)
	}
	
	// 初始化插件系统
	log.Println("初始化插件系统...")
	if err := plugin.Init(); err != nil {
		log.Fatalf("插件系统初始化失败: %v", err)
	}
	
	// 初始化事件总线
	log.Println("初始化事件总线...")
	eventBus := event_bus.NewEventBus()
	if eventBus == nil {
		log.Fatal("事件总线初始化失败")
	}
	
	// 初始化智能体工具注册中心
	log.Println("初始化智能体工具注册中心...")
	toolRegistry := agent_tools.NewRegistry()
	if toolRegistry == nil {
		log.Fatal("工具注册中心初始化失败")
	}
	
	// 注册基础工具
	log.Println("注册基础工具...")
	registerBasicTools(toolRegistry)
	
	// 注册基础插件
	log.Println("注册基础插件...")
	registerBasicPlugins()
	
	// 启动HTTP服务器
	log.Println("启动HTTP服务器...")
	// TODO: 实现HTTP服务器启动逻辑
	
	// 启动gRPC服务器
	log.Println("启动gRPC服务器...")
	// TODO: 实现gRPC服务器启动逻辑
	
	// 启动WebSocket服务
	log.Println("启动WebSocket服务...")
	// TODO: 实现WebSocket服务启动逻辑
	
	log.Println("CoreX 系统启动完成，等待信号...")
	
	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	<-sigChan
	log.Println("收到关闭信号，正在优雅关闭...")
	
	// 执行清理工作
	cleanup(ctx)
	
	log.Println("CoreX 系统已关闭")
}

// registerBasicTools 注册基础工具
func registerBasicTools(registry *agent_tools.Registry) {
	// TODO: 注册基础工具
	// 例如：
	// registry.Register("apply_tag", &tools.ApplyTagTool{})
	// registry.Register("get_customer_profile", &tools.GetCustomerProfileTool{})
	// registry.Register("start_flow", &tools.StartFlowTool{})
	// registry.Register("send_message", &tools.SendMessageTool{})
	
	log.Println("基础工具注册完成")
}

// registerBasicPlugins 注册基础插件
func registerBasicPlugins() {
	// TODO: 注册基础插件
	// 例如：
	// plugin.Register("customer_tag_sync", &plugins.CustomerTagSyncPlugin{})
	// plugin.Register("flow_completion_handler", &plugins.FlowCompletionPlugin{})
	
	log.Println("基础插件注册完成")
}

// cleanup 执行清理工作
func cleanup(ctx context.Context) {
	log.Println("执行系统清理...")
	
	// 清理插件系统
	plugin.Cleanup()
	
	// 清理事件总线
	// TODO: 实现事件总线清理逻辑
	
	// 清理其他资源
	// TODO: 实现其他资源清理逻辑
	
	log.Println("系统清理完成")
}