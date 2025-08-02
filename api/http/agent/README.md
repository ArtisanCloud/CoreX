# Eino Agent API 测试文档

基于 cloudwego/eino 框架的智能代理 HTTP API 接口。

## API 端点

### 1. 健康检查
```
GET /agents/health
```

**响应示例:**
```json
{
  "success": true,
  "message": "Eino Agent API 运行正常",
  "framework": "eino",
  "timestamp": 1703123456,
  "endpoints": [
    "POST /agents/chat - 基本聊天",
    "POST /agents/plan - 生成计划",
    "POST /agents/execute - 执行意图",
    "POST /agents/stream - 流式聊天",
    "POST /agents/config/test - 配置测试"
  ]
}
```

### 2. 基本聊天
```
POST /agents/chat
```

**请求体:**
```json
{
  "message": "你好，请介绍一下自己",
  "config": {
    "model_name": "gpt-3.5-turbo",
    "provider": "openai",
    "endpoint": "https://api.openai.com/v1",
    "api_key": "your-api-key",
    "temperature": 0.7,
    "max_tokens": 2048,
    "system_prompt": "你是一个有用的AI助手"
  },
  "context": {
    "user_id": "12345",
    "session_id": "session_001"
  }
}
```

**响应示例:**
```json
{
  "success": true,
  "message": "你好，请介绍一下自己",
  "content": "你好！我是基于 eino 框架的AI助手...",
  "metadata": {
    "role": "assistant",
    "framework": "eino"
  },
  "timestamp": 1703123456
}
```

### 3. 生成执行计划
```
POST /agents/plan
```

**请求体:**
```json
{
  "intent": "请帮我搜索最新的AI技术新闻",
  "context": {
    "language": "zh-CN",
    "max_results": 10
  },
  "config": {
    "model_name": "gpt-4",
    "temperature": 0.3
  }
}
```

**响应示例:**
```json
{
  "success": true,
  "plan_id": "eino_plan_1703123456789",
  "intent": "请帮我搜索最新的AI技术新闻",
  "steps": [
    {
      "id": "tool_step_1703123456790",
      "type": "tool",
      "name": "search",
      "input": {
        "query": "请帮我搜索最新的AI技术新闻"
      },
      "expected_out": {
        "result": "object"
      },
      "metadata": {
        "framework": "eino",
        "step_type": "tool"
      }
    }
  ],
  "explanation": "基于 eino 框架的执行计划:\n计划ID: eino_plan_1703123456789\n...",
  "metadata": {
    "framework": "eino",
    "steps_count": 1,
    "priority": 1
  },
  "timestamp": 1703123456
}
```

### 4. 执行意图
```
POST /agents/execute
```

**请求体:**
```json
{
  "intent": "请解释什么是机器学习",
  "context": {
    "detail_level": "beginner",
    "language": "zh-CN"
  },
  "config": {
    "model_name": "gpt-3.5-turbo",
    "temperature": 0.7,
    "system_prompt": "你是一个专业的技术讲师"
  }
}
```

**响应示例:**
```json
{
  "success": true,
  "results": {
    "chat_step_1703123456789": {
      "output": {
        "content": "机器学习是人工智能的一个分支...",
        "type": "chat_response",
        "role": "assistant"
      },
      "success": true,
      "duration": 1500
    }
  },
  "metadata": {
    "framework": "eino",
    "results_count": 1
  },
  "timestamp": 1703123456
}
```

### 5. 流式聊天
```
POST /agents/stream
```

**请求体:**
```json
{
  "message": "请写一首关于春天的诗",
  "config": {
    "model_name": "gpt-3.5-turbo",
    "temperature": 0.8,
    "enable_stream": true
  }
}
```

**响应 (Server-Sent Events):**
```
data: {"content": "春", "role": "assistant"}

data: {"content": "风", "role": "assistant"}

data: {"content": "轻", "role": "assistant"}

...

event: end
data: {"success": true, "message": "流式聊天完成"}
```

### 6. 配置测试
```
POST /agents/config/test
```

**请求体:**
```json
{
  "model_name": "gpt-3.5-turbo",
  "provider": "openai",
  "endpoint": "https://api.openai.com/v1",
  "api_key": "your-api-key",
  "temperature": 0.7,
  "max_tokens": 2048,
  "system_prompt": "你是一个有用的AI助手"
}
```

**响应示例:**
```json
{
  "success": true,
  "message": "配置测试成功",
  "config": {
    "model_name": "gpt-3.5-turbo",
    "provider": "openai",
    "endpoint": "https://api.openai.com/v1",
    "temperature": 0.7,
    "max_tokens": 2048,
    "system_prompt": "你是一个有用的AI助手",
    "framework": "eino"
  }
}
```

## 配置参数说明

### ChatConfig 配置项

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model_name | string | 否 | 模型名称，如 "gpt-3.5-turbo", "gpt-4" |
| provider | string | 否 | 模型提供商，如 "openai", "anthropic", "qwen" |
| endpoint | string | 否 | 自定义 API 端点 |
| api_key | string | 否 | API 密钥 |
| temperature | float64 | 否 | 温度参数 (0.0-2.0) |
| max_tokens | int | 否 | 最大令牌数 |
| system_prompt | string | 否 | 系统提示词 |
| enable_stream | bool | 否 | 是否启用流式响应 |

## 测试示例

### 使用 curl 测试

1. **健康检查:**
```bash
curl -X GET http://localhost:8080/api/v1/agents/health
```

2. **基本聊天:**
```bash
curl -X POST http://localhost:8080/api/v1/agents/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "你好，请介绍一下自己",
    "config": {
      "model_name": "gpt-3.5-turbo",
      "temperature": 0.7,
      "system_prompt": "你是一个有用的AI助手"
    }
  }'
```

3. **生成计划:**
```bash
curl -X POST http://localhost:8080/api/v1/agents/plan \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "请帮我搜索最新的AI技术新闻",
    "config": {
      "model_name": "gpt-4",
      "temperature": 0.3
    }
  }'
```

4. **执行意图:**
```bash
curl -X POST http://localhost:8080/api/v1/agents/execute \
  -H "Content-Type: application/json" \
  -d '{
    "intent": "请解释什么是机器学习",
    "config": {
      "model_name": "gpt-3.5-turbo",
      "temperature": 0.7
    }
  }'
```

5. **配置测试:**
```bash
curl -X POST http://localhost:8080/api/v1/agents/config/test \
  -H "Content-Type: application/json" \
  -d '{
    "model_name": "gpt-3.5-turbo",
    "provider": "openai",
    "temperature": 0.7,
    "max_tokens": 2048
  }'
```

### 使用 JavaScript 测试

```javascript
// 基本聊天
async function testChat() {
  const response = await fetch('/api/v1/agents/chat', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      message: '你好，请介绍一下自己',
      config: {
        model_name: 'gpt-3.5-turbo',
        temperature: 0.7,
        system_prompt: '你是一个有用的AI助手'
      }
    })
  });
  
  const data = await response.json();
  console.log('聊天响应:', data);
}

// 流式聊天
async function testStreamChat() {
  const response = await fetch('/api/v1/agents/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      message: '请写一首关于春天的诗',
      config: {
        model_name: 'gpt-3.5-turbo',
        temperature: 0.8,
        enable_stream: true
      }
    })
  });

  const reader = response.body.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    
    const chunk = decoder.decode(value);
    console.log('流式数据:', chunk);
  }
}
```

## 错误处理

所有 API 都会返回统一的错误格式：

```json
{
  "success": false,
  "error": "错误描述信息",
  "timestamp": 1703123456
}
```

常见错误码：
- `400 Bad Request`: 请求参数错误
- `500 Internal Server Error`: 服务器内部错误

## 注意事项

1. **API 密钥安全**: 请妥善保管您的 API 密钥，不要在客户端代码中暴露
2. **请求频率**: 请合理控制请求频率，避免超出 API 限制
3. **超时设置**: 聊天请求超时时间为 30 秒，执行请求超时时间为 60 秒
4. **流式响应**: 流式聊天使用 Server-Sent Events，需要支持 SSE 的客户端
5. **配置验证**: 建议在正式使用前先调用配置测试接口验证配置的正确性

## 框架特性

- **基于 eino**: 所有功能都基于 cloudwego/eino 框架实现
- **强类型配置**: 支持详细的模型和执行配置
- **自定义端点**: 支持配置自定义的 API 端点
- **流式响应**: 支持实时流式聊天
- **计划生成**: 支持智能意图分析和执行计划生成
- **错误处理**: 完善的错误处理和降级机制