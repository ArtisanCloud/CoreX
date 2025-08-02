package eino

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
)

func TestNewAgent(t *testing.T) {
	// 测试创建 Agent
	config := NewConfig()
	config.WithOption("model_name", "gpt-3.5-turbo")
	config.WithOption("temperature", 0.7)

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent 不应该为 nil")
	}

	if agent.config != config {
		t.Error("Agent 配置不匹配")
	}

	// 测试 nil 配置
	_, err = NewAgent(nil)
	if err == nil {
		t.Error("使用 nil 配置应该返回错误")
	}
}

func TestAgentChat(t *testing.T) {
	config := NewConfig()
	config.WithOption("model_name", "gpt-3.5-turbo")

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	ctx := context.Background()

	// 测试正常对话
	response, err := agent.Chat(ctx, "你好")
	if err != nil {
		t.Fatalf("对话失败: %v", err)
	}

	if response == nil {
		t.Fatal("响应不应该为 nil")
	}

	if response.Role != schema.Assistant {
		t.Error("响应角色应该是 Assistant")
	}

	if response.Content == "" {
		t.Error("响应内容不应该为空")
	}

	// 测试空消息
	_, err = agent.Chat(ctx, "")
	if err == nil {
		t.Error("空消息应该返回错误")
	}
}

func TestAgentStreamChat(t *testing.T) {
	config := NewConfig()
	config.WithOption("stream_mode", true)

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	ctx := context.Background()

	// 测试流式对话
	streamChan, err := agent.StreamChat(ctx, "写一首诗")
	if err != nil {
		t.Fatalf("流式对话失败: %v", err)
	}

	if streamChan == nil {
		t.Fatal("流式通道不应该为 nil")
	}

	// 收集流式响应
	var chunks []*schema.Message
	timeout := time.After(5 * time.Second)

	for {
		select {
		case chunk, ok := <-streamChan:
			if !ok {
				// 通道已关闭
				goto done
			}
			chunks = append(chunks, chunk)
		case <-timeout:
			t.Fatal("流式响应超时")
		}
	}

done:
	if len(chunks) == 0 {
		t.Error("应该收到至少一个流式响应")
	}

	// 测试未启用流式模式
	config2 := NewConfig()
	config2.WithOption("stream_mode", false)

	agent2, err := NewAgent(config2)
	if err != nil {
		t.Fatalf("创建 Agent2 失败: %v", err)
	}
	defer agent2.Close()

	_, err = agent2.StreamChat(ctx, "测试")
	if err == nil {
		t.Error("未启用流式模式应该返回错误")
	}
}

func TestAgentChatWithContext(t *testing.T) {
	config := NewConfig()
	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	ctx := context.Background()
	contextMetadata := map[string]interface{}{
		"user_id":    "12345",
		"session_id": "session_001",
	}

	response, err := agent.ChatWithContext(ctx, "测试消息", contextMetadata)
	if err != nil {
		t.Fatalf("带上下文对话失败: %v", err)
	}

	if response == nil {
		t.Fatal("响应不应该为 nil")
	}

	// 验证上下文是否被添加到配置中
	if value, exists := agent.config.GetOption("context_user_id"); !exists || value != "12345" {
		t.Error("上下文元数据应该被添加到配置中")
	}
}

func TestAgentBatchChat(t *testing.T) {
	config := NewConfig()
	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	ctx := context.Background()
	messages := []string{"消息1", "消息2", "消息3"}

	responses, err := agent.BatchChat(ctx, messages)
	if err != nil {
		t.Fatalf("批量对话失败: %v", err)
	}

	if len(responses) != len(messages) {
		t.Errorf("响应数量不匹配，期望 %d，实际 %d", len(messages), len(responses))
	}

	for i, response := range responses {
		if response == nil {
			t.Errorf("响应 %d 不应该为 nil", i)
		}
		if response.Role != schema.Assistant {
			t.Errorf("响应 %d 角色应该是 Assistant", i)
		}
	}

	// 测试空消息列表
	_, err = agent.BatchChat(ctx, []string{})
	if err == nil {
		t.Error("空消息列表应该返回错误")
	}
}

func TestAgentAddTool(t *testing.T) {
	config := NewConfig()
	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	// 添加工具
	err = agent.AddTool("search")
	if err != nil {
		t.Fatalf("添加工具失败: %v", err)
	}

	// 验证工具是否被添加
	tools, exists := agent.config.GetOption("tools")
	if !exists {
		t.Fatal("工具配置应该存在")
	}

	toolList, ok := tools.([]string)
	if !ok {
		t.Fatal("工具配置应该是字符串切片")
	}

	if len(toolList) != 1 || toolList[0] != "search" {
		t.Error("工具应该被正确添加")
	}
}

func TestAgentSetSystemPrompt(t *testing.T) {
	config := NewConfig()
	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	prompt := "你是一个专业的技术顾问"
	err = agent.SetSystemPrompt(prompt)
	if err != nil {
		t.Fatalf("设置系统提示词失败: %v", err)
	}

	// 验证系统提示词是否被设置
	storedPrompt, exists := agent.config.GetOption("system_prompt")
	if !exists {
		t.Fatal("系统提示词配置应该存在")
	}

	if storedPrompt != prompt {
		t.Error("系统提示词应该被正确设置")
	}
}

func TestAgentUpdateConfig(t *testing.T) {
	config := NewConfig()
	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	// 更新配置
	err = agent.UpdateConfig("temperature", 0.8)
	if err != nil {
		t.Fatalf("更新配置失败: %v", err)
	}

	// 验证配置是否被更新
	temperature, exists := agent.config.GetOption("temperature")
	if !exists {
		t.Fatal("温度配置应该存在")
	}

	if temperature != 0.8 {
		t.Error("温度配置应该被正确更新")
	}
}

func TestAgentValidateConfig(t *testing.T) {
	// 测试有效配置
	config := NewConfig()
	config.WithOption("model_name", "gpt-3.5-turbo")
	config.WithOption("temperature", 0.7)
	config.WithOption("max_tokens", 2048)

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	err = agent.ValidateConfig()
	if err != nil {
		t.Fatalf("有效配置验证失败: %v", err)
	}

	// 测试无效的模型名称
	config.WithOption("model_name", "")
	err = agent.ValidateConfig()
	if err == nil {
		t.Error("空模型名称应该验证失败")
	}

	// 测试无效的温度
	config.WithOption("model_name", "gpt-3.5-turbo")
	config.WithOption("temperature", 3.0)
	err = agent.ValidateConfig()
	if err == nil {
		t.Error("超出范围的温度应该验证失败")
	}

	// 测试无效的最大令牌数
	config.WithOption("temperature", 0.7)
	config.WithOption("max_tokens", -1)
	err = agent.ValidateConfig()
	if err == nil {
		t.Error("负数的最大令牌数应该验证失败")
	}
}

func TestAgentExecuteWithRetry(t *testing.T) {
	config := NewConfig()
	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	ctx := context.Background()

	// 测试重试执行
	response, err := agent.ExecuteWithRetry(ctx, "测试消息", 3)
	if err != nil {
		t.Fatalf("重试执行失败: %v", err)
	}

	if response == nil {
		t.Fatal("响应不应该为 nil")
	}

	// 测试上下文取消
	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // 立即取消

	_, err = agent.ExecuteWithRetry(cancelCtx, "测试消息", 3)
	if err == nil {
		t.Error("取消的上下文应该返回错误")
	}
}

func TestAgentGetSupportedTools(t *testing.T) {
	config := NewConfig()
	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	tools := agent.GetSupportedTools()
	if len(tools) == 0 {
		t.Error("应该返回支持的工具列表")
	}

	expectedTools := []string{"search", "calculator", "code_executor", "web_scraper", "file_reader"}
	if len(tools) != len(expectedTools) {
		t.Errorf("工具数量不匹配，期望 %d，实际 %d", len(expectedTools), len(tools))
	}

	for i, tool := range tools {
		if tool != expectedTools[i] {
			t.Errorf("工具 %d 不匹配，期望 %s，实际 %s", i, expectedTools[i], tool)
		}
	}
}

func TestAgentGetModelInfo(t *testing.T) {
	config := NewConfig()
	config.WithOption("model_name", "gpt-4")
	config.WithOption("temperature", 0.5)
	config.WithOption("max_tokens", 4096)
	config.WithOption("tools", []string{"search", "calculator"})

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	info := agent.GetModelInfo()
	if len(info) == 0 {
		t.Error("应该返回模型信息")
	}

	if info["model_name"] != "gpt-4" {
		t.Error("模型名称不匹配")
	}

	if info["temperature"] != 0.5 {
		t.Error("温度参数不匹配")
	}

	if info["max_tokens"] != 4096 {
		t.Error("最大令牌数不匹配")
	}

	tools, ok := info["tools"].([]string)
	if !ok || len(tools) != 2 {
		t.Error("工具信息不匹配")
	}
}

func TestAgentProcessWithEino(t *testing.T) {
	config := NewConfig()
	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	ctx := context.Background()
	inputMsg := &schema.Message{
		Role:    schema.User,
		Content: "测试 eino 框架处理",
	}

	response, err := agent.ProcessWithEino(ctx, inputMsg)
	if err != nil {
		t.Fatalf("eino 框架处理失败: %v", err)
	}

	if response == nil {
		t.Fatal("响应不应该为 nil")
	}

	if response.Role != schema.Assistant {
		t.Error("响应角色应该是 Assistant")
	}

	if response.Content == "" {
		t.Error("响应内容不应该为空")
	}

	// 测试 nil 输入
	_, err = agent.ProcessWithEino(ctx, nil)
	if err == nil {
		t.Error("nil 输入应该返回错误")
	}
}

func TestAgentCreateChain(t *testing.T) {
	config := NewConfig()
	config.WithOption("system_prompt", "你是一个AI助手")
	config.WithOption("tools", []string{"search", "calculator"})

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	err = agent.CreateChain()
	if err != nil {
		t.Fatalf("创建处理链失败: %v", err)
	}
}

// 基准测试
func BenchmarkAgentChat(b *testing.B) {
	config := NewConfig()
	agent, err := NewAgent(config)
	if err != nil {
		b.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := agent.Chat(ctx, "基准测试消息")
		if err != nil {
			b.Fatalf("对话失败: %v", err)
		}
	}
}

func BenchmarkAgentBatchChat(b *testing.B) {
	config := NewConfig()
	agent, err := NewAgent(config)
	if err != nil {
		b.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	ctx := context.Background()
	messages := []string{"消息1", "消息2", "消息3", "消息4", "消息5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := agent.BatchChat(ctx, messages)
		if err != nil {
			b.Fatalf("批量对话失败: %v", err)
		}
	}
}
