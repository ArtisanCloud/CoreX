package main

import (
	"github.com/ArtisanCloud/CoreX/config"
	"github.com/ArtisanCloud/CoreXpkg/utils/logger"
	"go.uber.org/zap"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load("../config/example.yaml")
	if err != nil {
		panic(err)
	}

	// 2. 初始化全局Logger
	logger.InitGlobalLogger(&cfg.LogConfig)

	// 3. 使用全局Logger的各种方式

	// 方式1：使用全局便捷函数（推荐）
	logger.Info("这是一条Info日志")
	logger.Error("这是一条Error日志")
	logger.Debug("这是一条Debug日志")
	logger.Warn("这是一条Warn日志")

	// 方式2：使用格式化函数
	logger.InfoF("用户 %s 登录成功，IP: %s", "张三", "192.168.1.100")
	logger.ErrorF("数据库连接失败，重试次数: %d", 3)

	// 方式3：使用结构化字段（推荐用于生产环境）
	logger.Info("用户操作日志",
		zap.String("user_id", "user-123"),
		zap.String("action", "login"),
		zap.String("ip", "192.168.1.100"),
		zap.Int("status_code", 200),
	)

	logger.Error("API调用失败",
		zap.String("api", "/api/users"),
		zap.String("method", "POST"),
		zap.Int("status_code", 500),
		zap.String("error", "数据库连接超时"),
	)

	// 方式4：获取全局Logger实例进行更复杂操作
	globalLogger := logger.GetGlobalLogger()
	globalLogger.Info("通过Logger实例记录日志",
		zap.String("module", "example"),
		zap.String("function", "main"),
	)

	logger.Info("Logger使用示例完成！")
}
