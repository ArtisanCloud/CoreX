package eino

import (
	"context"
	"testing"
)

// TestAgentBasic 基本的 Agent 测试
func TestAgentBasic(t *testing.T) {
	// 创建配置
	config := NewConfig()
	config.WithOption("model_name", "gpt-3.5-turbo")
	config.WithOption("temperature", 0.7)

	// 创建 Agent
	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	// 验证 Agent 不为空
	if agent == nil {
		t.Fatal("Agent 不应该为 nil")
	}

	// 验证配置
	if err := agent.ValidateConfig(); err != nil {
		t.Fatalf("配置验证失败: %v", err)
	}

	ctx := context.Background()

	// 测试 GetPlan 方法
	plan, err := agent.GetPlan(ctx, "测试意图", nil)
	if err != nil {
		t.Fatalf("获取计划失败: %v", err)
	}

	if plan == nil {
		t.Fatal("计划不应该为 nil")
	}

	if plan.Intent != "测试意图" {
		t.Errorf("计划意图不匹配，期望: %s, 实际: %s", "测试意图", plan.Intent)
	}

	// 测试 Run 方法
	results, err := agent.Run(ctx, "测试运行", nil)
	if err != nil {
		t.Fatalf("运行失败: %v", err)
	}

	if results == nil {
		t.Fatal("结果不应该为 nil")
	}

	t.Logf("测试通过，结果数量: %d", len(results))
}

// TestAgentConfig 测试配置功能
func TestAgentConfig(t *testing.T) {
	config := NewConfig()
	config.WithOption("model_name", "gpt-4")
	config.WithOption("temperature", 0.5)

	agent, err := NewAgent(config)
	if err != nil {
		t.Fatalf("创建 Agent 失败: %v", err)
	}
	defer agent.Close()

	// 测试获取配置
	agentConfig := agent.GetConfig()
	if agentConfig == nil {
		t.Fatal("配置不应该为 nil")
	}

	// 测试配置选项
	if modelName, exists := agentConfig.GetOption("model_name"); !exists || modelName != "gpt-4" {
		t.Error("模型名称配置不正确")
	}

	if temperature, exists := agentConfig.GetOption("temperature"); !exists || temperature != 0.5 {
		t.Error("温度配置不正确")
	}

	// 测试更新配置
	err = agent.UpdateConfig("temperature", 0.8)
	if err != nil {
		t.Fatalf("更新配置失败: %v", err)
	}

	// 验证配置已更新
	if temperature, exists := agentConfig.GetOption("temperature"); !exists || temperature != 0.8 {
		t.Error("配置更新失败")
	}

	t.Log("配置测试通过")
}
