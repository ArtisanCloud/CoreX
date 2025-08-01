# CoreX 全局Logger使用指南

## 🎯 概述

CoreX提供了强大的全局Logger系统，支持多种输出方式（控制台、文件、Loki），并提供便捷的全局函数供项目中任何地方使用。

## 🚀 快速开始

### 1. 初始化全局Logger

在应用启动时初始化全局Logger（通常在main.go中）：

```go
import (
    "github.com/ArtisanCloud/CoreX/config"
    "github.com/ArtisanCloud/CoreXpkg/utils/logger"
)

func main() {
    // 加载配置
    cfg, err := config.Load("config/example.yaml")
    if err != nil {
        panic(err)
    }
    
    // 初始化全局Logger
    logger.InitGlobalLogger(&cfg.LogConfig)
    
    // 现在可以在项目任何地方使用全局Logger了
    logger.Info("应用启动成功")
}
```

### 2. 在项目中使用全局Logger

初始化后，您可以在项目的任何地方直接使用全局Logger：

```go
package mypackage

import "github.com/ArtisanCloud/CoreXpkg/utils/logger"

func SomeFunction() {
    // 直接使用全局Logger函数
    logger.Info("函数执行开始")
    logger.Debug("调试信息")
    logger.Warn("警告信息")
    logger.Error("错误信息")
}
```

## 📝 使用方式

### 方式1：简单文本日志（推荐日常使用）

```go
logger.Info("用户登录成功")
logger.Error("数据库连接失败")
logger.Debug("调试信息")
logger.Warn("内存使用率过高")
```

### 方式2：格式化日志

```go
logger.InfoF("用户 %s 登录成功，IP: %s", "张三", "192.168.1.100")
logger.ErrorF("API调用失败，状态码: %d", 500)
logger.DebugF("处理了 %d 条记录", 1000)
```

### 方式3：结构化日志（推荐生产环境）

```go
import "go.uber.org/zap"

logger.Info("用户操作",
    zap.String("user_id", "user-123"),
    zap.String("action", "login"),
    zap.String("ip", "192.168.1.100"),
    zap.Int("status_code", 200),
)

logger.Error("API错误",
    zap.String("api", "/api/users"),
    zap.String("method", "POST"),
    zap.Int("status_code", 500),
    zap.String("error", "数据库连接超时"),
    zap.Duration("response_time", time.Millisecond*500),
)
```

### 方式4：获取Logger实例

```go
// 获取全局Logger实例进行更复杂操作
globalLogger := logger.GetGlobalLogger()
globalLogger.Info("通过实例记录日志")
```

## ⚙️ 配置说明

在 `config/example.yaml` 中配置Logger：

```yaml
logging_config:
  level: "debug"              # 日志级别: debug/info/warn/error
  console: true               # 是否输出到控制台
  useJsonFormat: false        # 是否使用JSON格式
  file:                       # 文件日志配置
    enable: true              # 是否启用文件日志
    infoFilePath: "logs/info.log"     # Info日志文件路径
    errorFilePath: "logs/error.log"   # Error日志文件路径
    maxSize: 100              # 单个文件最大大小(MB)
    maxBackups: 5             # 保留的备份文件数量
    maxAge: 30                # 文件保留天数
    compress: true            # 是否压缩备份文件
  loki:                       # Loki日志配置
    enable: false             # 是否启用Loki
    url: ""                   # Loki服务器地址
    jobName: "corex"          # 任务名称
    batchWait: 1              # 批量等待时间(秒)
    batchSize: 100            # 批量大小
  httpDebug: false            # HTTP调试模式
  debug: true                 # 调试模式
```

## 🌟 最佳实践

### 1. 日志级别使用建议

- **Debug**: 详细的调试信息，仅在开发环境使用
- **Info**: 一般信息，记录重要的业务流程
- **Warn**: 警告信息，需要注意但不影响正常运行
- **Error**: 错误信息，需要立即处理的问题

### 2. 结构化日志建议

生产环境推荐使用结构化日志，便于日志分析和监控：

```go
// ✅ 推荐：结构化日志
logger.Info("用户登录",
    zap.String("user_id", userID),
    zap.String("ip", clientIP),
    zap.Duration("response_time", duration),
)

// ❌ 不推荐：纯文本日志
logger.InfoF("用户 %s 从 %s 登录，耗时 %v", userID, clientIP, duration)
```

### 3. 错误处理建议

```go
func ProcessUser(userID string) error {
    logger.Info("开始处理用户", zap.String("user_id", userID))
    
    user, err := getUserFromDB(userID)
    if err != nil {
        logger.Error("获取用户失败",
            zap.String("user_id", userID),
            zap.Error(err),
        )
        return err
    }
    
    logger.Info("用户处理完成", zap.String("user_id", userID))
    return nil
}
```

### 4. 性能考虑

- 在高频调用的函数中，避免使用Debug级别日志
- 使用结构化字段而不是字符串拼接
- 避免在日志中记录大量数据

## 🔧 环境变量配置

您也可以通过环境变量覆盖日志配置：

```bash
export COREX_LOGGING_LEVEL=info
export COREX_LOGGING_CONSOLE=true
export COREX_LOGGING_FILE_ENABLE=true
```

## 📁 示例代码

查看 `examples/logger_usage.go` 获取完整的使用示例。

## 🎯 总结

CoreX的全局Logger系统提供了：

- ✅ **简单易用**：一行代码即可记录日志
- ✅ **全局可用**：项目任何地方都可以使用
- ✅ **多种输出**：支持控制台、文件、Loki
- ✅ **结构化日志**：便于分析和监控
- ✅ **配置灵活**：支持YAML和环境变量配置
- ✅ **性能优化**：基于高性能的zap库

现在您可以在CoreX项目的任何地方轻松使用Logger了！🎉