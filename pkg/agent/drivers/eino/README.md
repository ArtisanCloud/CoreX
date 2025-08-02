# Eino Agent 实现

这是一个基于 [cloudwego/eino](https://github.com/cloudwego/eino) 框架的智能代理实现。

## 概述

本实现遵循 cloudwego/eino 的设计理念和 API 规范，提供了一个完整的智能代理解决方案，支持：

- 基本对话功能
- 流式对话
- 工具集成
- 上下文管理
- 批量处理
- 重试机制
- 配置管理

## 文件结构

```
pkg/agent/drivers/eino/
├── agent.go        # 主要的 Agent 实现
├── config.go       # 配置管理
├── plan.go         # 执行计划生成器
├── example.go      # 使用示例
└── README.md       # 说明文档
```

## 核心组件

### Agent

`Agent` 是核心组件，提供了与 AI 模型交互的主要接口：

```go
type Agent struct {
    config *Config
}
```

### Config

`Config` 管理 Agent 的配置信息：

```go
type Config struct {
    IntentPrompt    string                     // 意图解析提示模板
    ContextMetadata map[string]interface{}     // 上下文元数据
    RetryPolicy     *contract.RetryPolicy      // 重试策略
    Options         map[string]interface{}     // 可扩展选项
}
```

## 使用方法

### 基本使用

```go
// 创建配置
config := NewConfig()
config.WithOption("model_name", "gpt-3.5-turbo")
config.WithOption("temperature", 0.7)
config.WithOption("system_prompt", "你是一个有用的AI助手")

// 创建 Agent
agent, err := NewAgent(config)
if err != nil {
    log.Fatal(err)
}
defer agent.Close()

// 进行对话
ctx := context.Background()
response, err := agent.Chat(ctx, "你好")
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Content)
```

### 流式对话

```go
// 启用流式模式
config.WithOption("stream_mode", true)

// 创建 Agent
agent, err := NewAgent(config)
if err != nil {
    log.Fatal(err)
}

// 流式对话
streamChan, err := agent.StreamChat(ctx, "请写一首诗")
if err != nil {
    log.Fatal(err)
}

// 处理流式响应
for chunk := range streamChan {
    fmt.Print(chunk.Content)
}
```

### 添加工具

```go
// 配置工具
config.WithOption("tools", []string{"search", "calculator"})

// 动态添加工具
err := agent.AddTool("web_scraper")
if err != nil {
    log.Fatal(err)
}
```

### 带上下文的对话

```go
contextMetadata := map[string]interface{}{
    "user_id":    "12345",
    "session_id": "session_001",
    "language":   "zh-CN",
}

response, err := agent.ChatWithContext(ctx, "请帮我查询天气", contextMetadata)
if err != nil {
    log.Fatal(err)
}
```

## 配置选项

### 模型配置

- `model_name`: 模型名称（如 "gpt-3.5-turbo", "gpt-4"）
- `temperature`: 温度参数（0.0-2.0）
- `max_tokens`: 最大令牌数
- `system_prompt`: 系统提示词

### 功能配置

- `tools`: 工具列表
- `stream_mode`: 是否启用流式输出
- `memory`: 是否启用记忆功能

### 上下文配置

- `context_*`: 上下文相关的配置项

## API 参考

### Agent 方法

#### Chat
```go
func (a *Agent) Chat(ctx context.Context, message string) (*schema.Message, error)
```
进行基本对话。

#### StreamChat
```go
func (a *Agent) StreamChat(ctx context.Context, message string) (<-chan *schema.Message, error)
```
进行流式对话。

#### ChatWithContext
```go
func (a *Agent) ChatWithContext(ctx context.Context, message string, contextMetadata map[string]interface{}) (*schema.Message, error)
```
带上下文的对话。

#### BatchChat
```go
func (a *Agent) BatchChat(ctx context.Context, messages []string) ([]*schema.Message, error)
```
批量处理多个消息。

#### AddTool
```go
func (a *Agent) AddTool(toolName string) error
```
动态添加工具。

#### SetSystemPrompt
```go
func (a *Agent) SetSystemPrompt(prompt string) error
```
设置系统提示词。

#### UpdateConfig
```go
func (a *Agent) UpdateConfig(key string, value interface{}) error
```
更新配置。

#### ValidateConfig
```go
func (a *Agent) ValidateConfig() error
```
验证配置的有效性。

### Config 方法

#### WithOption
```go
func (c *Config) WithOption(key string, value interface{}) *Config
```
设置配置选项。

#### GetOption
```go
func (c *Config) GetOption(key string) (interface{}, bool)
```
获取配置选项。

#### GetOptionWithDefault
```go
func (c *Config) GetOptionWithDefault(key string, defaultValue interface{}) interface{}
```
获取配置选项，如果不存在则返回默认值。

## 支持的工具

- `search`: 搜索工具
- `calculator`: 计算器工具
- `code_executor`: 代码执行工具
- `web_scraper`: 网页抓取工具
- `file_reader`: 文件读取工具

## 错误处理

所有方法都返回适当的错误信息，建议在使用时进行错误检查：

```go
response, err := agent.Chat(ctx, message)
if err != nil {
    // 处理错误
    log.Printf("对话失败: %v", err)
    return
}
```

## 最佳实践

1. **配置验证**: 在使用 Agent 之前，调用 `ValidateConfig()` 验证配置。
2. **资源清理**: 使用完毕后调用 `Close()` 方法清理资源。
3. **上下文管理**: 合理使用 context 进行超时和取消控制。
4. **错误处理**: 始终检查和处理返回的错误。
5. **流式处理**: 对于长文本生成，使用流式模式提升用户体验。

## 示例代码

完整的使用示例请参考 `example.go` 文件，其中包含了各种使用场景的演示代码。

## 依赖

- `github.com/cloudwego/eino`: eino 框架核心库
- `github.com/ArtisanCloud/CoreX/pkg/agent/contract`: 内部合约接口

## 注意事项

1. 本实现目前提供了基础框架，实际的模型调用需要根据具体的模型提供商进行实现。
2. 工具集成需要根据实际需求实现具体的工具逻辑。
3. 流式处理的具体实现依赖于底层模型的流式支持。

## 贡献

欢迎提交 Issue 和 Pull Request 来改进这个实现。