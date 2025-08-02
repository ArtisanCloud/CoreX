package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ArtisanCloud/CoreX/api/http/agent"
)

func main() {
	// 检查命令行参数
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]
	baseURL := "http://localhost:8080/api/v1"

	// 如果提供了自定义 URL
	if len(os.Args) > 2 {
		baseURL = os.Args[2]
	}

	// 创建测试客户端
	client := agent.NewTestClient(baseURL)

	switch command {
	case "health":
		fmt.Println("🏥 测试健康检查...")
		if err := client.TestHealthCheck(); err != nil {
			log.Printf("健康检查失败: %v", err)
		}

	case "chat":
		fmt.Println("💬 测试基本聊天...")
		if err := client.TestChat(); err != nil {
			log.Printf("聊天测试失败: %v", err)
		}

	case "plan":
		fmt.Println("📋 测试计划生成...")
		if err := client.TestPlan(); err != nil {
			log.Printf("计划生成测试失败: %v", err)
		}

	case "execute":
		fmt.Println("🚀 测试执行意图...")
		if err := client.TestExecute(); err != nil {
			log.Printf("执行测试失败: %v", err)
		}

	case "config":
		fmt.Println("⚙️ 测试配置验证...")
		if err := client.TestConfigValidation(); err != nil {
			log.Printf("配置测试失败: %v", err)
		}

	case "batch":
		fmt.Println("🔄 测试批量请求...")
		if err := client.TestBatchRequests(); err != nil {
			log.Printf("批量测试失败: %v", err)
		}

	case "error":
		fmt.Println("❌ 测试错误处理...")
		if err := client.TestErrorHandling(); err != nil {
			log.Printf("错误处理测试失败: %v", err)
		}

	case "all":
		fmt.Println("🧪 运行所有测试...")
		client.RunAllTests()

	case "quick":
		fmt.Println("⚡ 快速测试...")
		agent.QuickTest()

	default:
		fmt.Printf("未知命令: %s\n", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Eino Agent API 测试工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  go run cmd/test_agent_api/main.go <command> [base_url]")
	fmt.Println()
	fmt.Println("命令:")
	fmt.Println("  health   - 测试健康检查")
	fmt.Println("  chat     - 测试基本聊天")
	fmt.Println("  plan     - 测试计划生成")
	fmt.Println("  execute  - 测试执行意图")
	fmt.Println("  config   - 测试配置验证")
	fmt.Println("  batch    - 测试批量请求")
	fmt.Println("  error    - 测试错误处理")
	fmt.Println("  all      - 运行所有测试")
	fmt.Println("  quick    - 快速测试")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  go run cmd/test_agent_api/main.go health")
	fmt.Println("  go run cmd/test_agent_api/main.go chat http://localhost:8080/api/v1")
	fmt.Println("  go run cmd/test_agent_api/main.go all")
}
