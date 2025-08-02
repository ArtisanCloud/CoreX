package eino

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/schema"
)

// ExampleUsage 展示如何使用 Eino Agent
func ExampleUsage() {
	// 创建配置
	config := NewConfig().
		WithModelName("gpt-3.5-turbo").
		WithModelTemperature(0.7).
		WithModelMaxTokens(2048).
		WithEnabledTools([]string{"search", "calculator"}).
		WithExtension("system_prompt", "你是一个有用的AI助手").
		WithExtension("stream_mode", true)

	// 创建 Agent
	agent, err := NewAgent(config)
	if err != nil {
		log.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	// 验证配置
	if err := agent.ValidateConfig(); err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}

	ctx := context.Background()

	// 基本对话
	fmt.Println("=== 基本对话 ===")
	response, err := agent.Chat(ctx, "你好，请介绍一下自己")
	if err != nil {
		log.Printf("对话失败: %v", err)
	} else {
		fmt.Printf("用户: 你好，请介绍一下自己\n")
		fmt.Printf("助手: %s\n\n", response.Content)
	}

	// 带上下文的对话（使用 Run 方法）
	fmt.Println("=== 带上下文的对话 ===")
	contextMetadata := map[string]interface{}{
		"user_id":    "12345",
		"session_id": "session_001",
		"language":   "zh-CN",
	}
	results, err := agent.Run(ctx, "请帮我计算 2+2", contextMetadata)
	if err != nil {
		log.Printf("带上下文对话失败: %v", err)
	} else {
		fmt.Printf("用户: 请帮我计算 2+2\n")
		fmt.Printf("助手: 执行完成，结果数量: %d\n\n", len(results))
	}

	// 批量对话（使用 Run 方法逐个处理）
	fmt.Println("=== 批量对话 ===")
	messages := []string{
		"今天天气怎么样？",
		"推荐一本好书",
		"如何学习编程？",
	}
	for _, msg := range messages {
		results, err := agent.Run(ctx, msg, nil)
		if err != nil {
			log.Printf("批量对话失败: %v", err)
		} else {
			fmt.Printf("用户: %s\n", msg)
			fmt.Printf("助手: 执行完成，结果数量: %d\n\n", len(results))
		}
	}

	// 流式对话
	fmt.Println("=== 流式对话 ===")
	streamChan, err := agent.StreamChat(ctx, "请写一首关于春天的诗")
	if err != nil {
		log.Printf("流式对话失败: %v", err)
	} else {
		fmt.Printf("用户: 请写一首关于春天的诗\n")
		fmt.Printf("助手: ")
		for chunk := range streamChan {
			fmt.Print(chunk.Content)
		}
		fmt.Println()
	}

	// 带重试机制的执行（使用 Run 方法）
	fmt.Println("=== 带重试机制的执行 ===")
	retryResults, err := agent.Run(ctx, "这是一个测试消息", nil)
	if err != nil {
		log.Printf("重试执行失败: %v", err)
	} else {
		fmt.Println("✓ 重试执行成功")
		fmt.Printf("  结果数量: %d\n", len(retryResults))
	}

	// 动态添加工具（使用配置更新）
	fmt.Println("=== 动态添加工具 ===")
	err = agent.UpdateConfig("tools", []string{"search", "calculator", "web_scraper"})
	if err != nil {
		log.Printf("添加工具失败: %v", err)
	} else {
		fmt.Println("成功添加 web_scraper 工具")
	}

	// 更新系统提示词
	fmt.Println("=== 更新系统提示词 ===")
	err = agent.UpdateConfig("system_prompt", "你是一个专业的技术顾问，专门帮助用户解决编程问题")
	if err != nil {
		log.Printf("更新系统提示词失败: %v", err)
	} else {
		fmt.Println("成功更新系统提示词")
	}

	// 获取配置信息
	fmt.Println("=== 配置信息 ===")
	agentConfig := agent.GetConfig()
	if modelName, exists := agentConfig.GetOption("model_name"); exists {
		fmt.Printf("model_name: %v\n", modelName)
	}
	if temperature, exists := agentConfig.GetOption("temperature"); exists {
		fmt.Printf("temperature: %v\n", temperature)
	}
	if tools, exists := agentConfig.GetOption("tools"); exists {
		fmt.Printf("tools: %v\n", tools)
	}

	// 获取支持的工具
	fmt.Println("\n=== 支持的工具 ===")
	supportedTools := agent.GetSupportedTools()
	for _, tool := range supportedTools {
		fmt.Printf("- %s\n", tool)
	}

	// 使用 eino 框架处理
	fmt.Println("\n=== 使用 eino 框架处理 ===")
	inputMsg := &schema.Message{
		Role:    schema.User,
		Content: "使用 eino 框架处理这条消息",
	}
	response, err = agent.ProcessWithEino(ctx, inputMsg)
	if err != nil {
		log.Printf("eino 框架处理失败: %v", err)
	} else {
		fmt.Printf("输入: %s\n", inputMsg.Content)
		fmt.Printf("输出: %s\n", response.Content)
	}
}

// ExampleAdvancedUsage 展示高级用法
func ExampleAdvancedUsage() {
	fmt.Println("\n=== 高级用法示例 ===")

	// 创建自定义配置
	// 创建自定义配置
	config := NewConfig().
		WithIntentPrompt("根据用户意图生成执行计划").
		WithContextMetadata(map[string]interface{}{
			"tenant_id": "tenant_001",
			"app_id":    "app_001",
		}).
		WithModelName("gpt-4").
		WithModelTemperature(0.3).
		WithModelMaxTokens(4096)
	config.WithOption("model_name", "gpt-4")
	config.WithOption("temperature", 0.3)
	config.WithOption("max_tokens", 4096)

	// 创建 Agent
	agent, err := NewAgent(config)
	if err != nil {
		log.Fatalf("创建高级 Agent 失败: %v", err)
	}
	defer agent.Close()

	ctx := context.Background()

	// 测试计划生成
	fmt.Println("--- 测试计划生成 ---")
	plan, err := agent.GetPlan(ctx, "请解释什么是机器学习", nil)
	if err != nil {
		log.Printf("生成计划失败: %v", err)
	} else {
		fmt.Printf("成功生成计划，ID: %s\n", plan.ID)
	}

	// 动态更新配置
	err = agent.UpdateConfig("temperature", 0.8)
	if err != nil {
		log.Printf("更新配置失败: %v", err)
	} else {
		fmt.Println("成功更新温度参数为 0.8")
	}

	// 测试对话
	response, err := agent.Chat(ctx, "请解释什么是机器学习")
	if err != nil {
		log.Printf("高级对话失败: %v", err)
	} else {
		fmt.Printf("用户: 请解释什么是机器学习\n")
		fmt.Printf("助手: %s\n", response.Content)
	}
}
