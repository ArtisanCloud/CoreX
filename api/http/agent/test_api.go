package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TestClient API 测试客户端
type TestClient struct {
	BaseURL string
	Client  *http.Client
}

// NewTestClient 创建测试客户端
func NewTestClient(baseURL string) *TestClient {
	return &TestClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// TestHealthCheck 测试健康检查
func (tc *TestClient) TestHealthCheck() error {
	fmt.Println("=== 测试健康检查 ===")
	
	resp, err := tc.Client.Get(tc.BaseURL + "/agents/health")
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	fmt.Printf("状态码: %d\n", resp.StatusCode)
	fmt.Printf("响应: %s\n\n", string(body))
	return nil
}

// TestChat 测试基本聊天
func (tc *TestClient) TestChat() error {
	fmt.Println("=== 测试基本聊天 ===")
	
	request := ChatRequest{
		Message: "你好，请介绍一下自己",
		Config: &ChatConfig{
			ModelName:    "gpt-3.5-turbo",
			Temperature:  0.7,
			MaxTokens:    2048,
			SystemPrompt: "你是一个基于 eino 框架的AI助手",
		},
		Context: map[string]interface{}{
			"user_id":    "test_user_001",
			"session_id": "test_session_001",
		},
	}

	return tc.makeRequest("POST", "/agents/chat", request)
}

// TestPlan 测试计划生成
func (tc *TestClient) TestPlan() error {
	fmt.Println("=== 测试计划生成 ===")
	
	request := PlanRequest{
		Intent: "请帮我搜索最新的AI技术新闻",
		Config: &ChatConfig{
			ModelName:   "gpt-3.5-turbo",
			Temperature: 0.3,
		},
		Context: map[string]interface{}{
			"language":    "zh-CN",
			"max_results": 10,
		},
	}

	return tc.makeRequest("POST", "/agents/plan", request)
}

// TestExecute 测试执行意图
func (tc *TestClient) TestExecute() error {
	fmt.Println("=== 测试执行意图 ===")
	
	request := ExecuteRequest{
		Intent: "请解释什么是机器学习",
		Config: &ChatConfig{
			ModelName:    "gpt-3.5-turbo",
			Temperature:  0.7,
			SystemPrompt: "你是一个专业的技术讲师",
		},
		Context: map[string]interface{}{
			"detail_level": "beginner",
			"language":     "zh-CN",
		},
	}

	return tc.makeRequest("POST", "/agents/execute", request)
}

// TestConfigValidation 测试配置验证
func (tc *TestClient) TestConfigValidation() error {
	fmt.Println("=== 测试配置验证 ===")
	
	config := ChatConfig{
		ModelName:    "gpt-3.5-turbo",
		Provider:     "openai",
		Temperature:  0.7,
		MaxTokens:    2048,
		SystemPrompt: "你是一个有用的AI助手",
	}

	return tc.makeRequest("POST", "/agents/config/test", config)
}

// TestInvalidConfig 测试无效配置
func (tc *TestClient) TestInvalidConfig() error {
	fmt.Println("=== 测试无效配置 ===")
	
	config := ChatConfig{
		ModelName:   "", // 空的模型名称
		Temperature: 3.0, // 无效的温度值
		MaxTokens:   -1,  // 无效的令牌数
	}

	return tc.makeRequest("POST", "/agents/config/test", config)
}

// TestChatWithCustomEndpoint 测试自定义端点
func (tc *TestClient) TestChatWithCustomEndpoint() error {
	fmt.Println("=== 测试自定义端点 ===")
	
	request := ChatRequest{
		Message: "测试自定义端点配置",
		Config: &ChatConfig{
			ModelName:   "gpt-3.5-turbo",
			Provider:    "openai",
			Endpoint:    "https://api.custom.com/v1", // 自定义端点
			APIKey:      "test-api-key",
			Temperature: 0.7,
		},
	}

	return tc.makeRequest("POST", "/agents/chat", request)
}

// TestBatchRequests 测试批量请求
func (tc *TestClient) TestBatchRequests() error {
	fmt.Println("=== 测试批量请求 ===")
	
	messages := []string{
		"今天天气怎么样？",
		"推荐一本好书",
		"如何学习编程？",
		"什么是人工智能？",
		"解释一下区块链技术",
	}

	for i, message := range messages {
		fmt.Printf("--- 批量请求 %d/%d ---\n", i+1, len(messages))
		
		request := ChatRequest{
			Message: message,
			Config: &ChatConfig{
				ModelName:   "gpt-3.5-turbo",
				Temperature: 0.7,
			},
		}

		if err := tc.makeRequest("POST", "/agents/chat", request); err != nil {
			fmt.Printf("批量请求 %d 失败: %v\n", i+1, err)
		}
		
		// 添加延迟避免请求过于频繁
		time.Sleep(1 * time.Second)
	}

	return nil
}

// TestErrorHandling 测试错误处理
func (tc *TestClient) TestErrorHandling() error {
	fmt.Println("=== 测试错误处理 ===")
	
	// 测试空消息
	fmt.Println("--- 测试空消息 ---")
	request := ChatRequest{
		Message: "", // 空消息
		Config: &ChatConfig{
			ModelName: "gpt-3.5-turbo",
		},
	}
	tc.makeRequest("POST", "/agents/chat", request)

	// 测试无效的 JSON
	fmt.Println("--- 测试无效 JSON ---")
	invalidJSON := `{"message": "test", "config": {invalid json}}`
	tc.makeRawRequest("POST", "/agents/chat", []byte(invalidJSON))

	return nil
}

// makeRequest 发送请求
func (tc *TestClient) makeRequest(method, endpoint string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %w", err)
	}

	return tc.makeRawRequest(method, endpoint, jsonData)
}

// makeRawRequest 发送原始请求
func (tc *TestClient) makeRawRequest(method, endpoint string, data []byte) error {
	url := tc.BaseURL + endpoint
	
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	fmt.Printf("请求: %s %s\n", method, endpoint)
	fmt.Printf("请求体: %s\n", string(data))

	resp, err := tc.Client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	fmt.Printf("状态码: %d\n", resp.StatusCode)
	fmt.Printf("响应: %s\n\n", string(body))

	return nil
}

// RunAllTests 运行所有测试
func (tc *TestClient) RunAllTests() {
	fmt.Println("开始运行 Eino Agent API 测试...")
	fmt.Println("基础URL:", tc.BaseURL)
	fmt.Println()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"健康检查", tc.TestHealthCheck},
		{"基本聊天", tc.TestChat},
		{"计划生成", tc.TestPlan},
		{"执行意图", tc.TestExecute},
		{"配置验证", tc.TestConfigValidation},
		{"无效配置", tc.TestInvalidConfig},
		{"自定义端点", tc.TestChatWithCustomEndpoint},
		{"批量请求", tc.TestBatchRequests},
		{"错误处理", tc.TestErrorHandling},
	}

	successCount := 0
	for _, test := range tests {
		fmt.Printf("🧪 运行测试: %s\n", test.name)
		if err := test.fn(); err != nil {
			fmt.Printf("❌ 测试失败: %v\n\n", err)
		} else {
			fmt.Printf("✅ 测试通过\n\n")
			successCount++
		}
	}

	fmt.Printf("测试完成! 通过: %d/%d\n", successCount, len(tests))
}

// ExampleUsage 使用示例
func ExampleUsage() {
	// 创建测试客户端
	client := NewTestClient("http://localhost:8080/api/v1")
	
	// 运行所有测试
	client.RunAllTests()
}

// QuickTest 快速测试
func QuickTest() {
	client := NewTestClient("http://localhost:8080/api/v1")
	
	fmt.Println("🚀 快速测试 Eino Agent API")
	
	// 健康检查
	if err := client.TestHealthCheck(); err != nil {
		fmt.Printf("❌ 健康检查失败: %v\n", err)
		return
	}
	
	// 基本聊天测试
	if err := client.TestChat(); err != nil {
		fmt.Printf("❌ 聊天测试失败: %v\n", err)
		return
	}
	
	fmt.Println("✅ 快速测试通过!")
}