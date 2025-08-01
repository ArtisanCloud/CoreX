package logger

import (
	"context"
	"fmt"
	"os"
	"sync"

	lumberjack "github.com/ArtisanCloud/CoreX/pkg/logger/lib"
	"github.com/ArtisanCloud/CoreX/pkg/logger/utils"

	"github.com/ArtisanCloud/CoreX/pkg/logger/config"
	"github.com/ArtisanCloud/CoreX/pkg/logger/writer"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	driver *zap.Logger
	config config.LogConfig
}

// 全局变量用来存储 Logger 实例
var (
	once           sync.Once
	instance       *Logger
	globalInstance *Logger
	globalOnce     sync.Once
)

func getFileWriteSyncer(fileConfig *config.FileConfig, fileName string) zapcore.WriteSyncer {

	return zapcore.AddSync(&lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    fileConfig.MaxSize, // megabytes
		MaxBackups: fileConfig.MaxBackups,
		MaxAge:     fileConfig.MaxAge,   // days
		Compress:   fileConfig.Compress, // disabled by default
	})
}

func NewLogger(config *config.LogConfig) *Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	var cores []zapcore.Core

	// Console 日志
	if config.Console {
		consoleCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), config.ParseLogLevel())
		cores = append(cores, consoleCore)
	}

	// 文件日志
	if config.File.Enable {
		if config.File.InfoFilePath == "" {
			config.File.InfoFilePath = "logs/info.log"
		}
		utils.EnsureFileExists(config.File.InfoFilePath)
		if config.File.ErrorFilePath == "" {
			config.File.ErrorFilePath = "logs/error.log"
		}
		utils.EnsureFileExists(config.File.ErrorFilePath)

		// 只记录 Info 及以上（但不包括 Error）级别的日志
		infoLevelEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.InfoLevel && lvl < zapcore.ErrorLevel
		})
		infoFileCore := zapcore.NewCore(encoder, getFileWriteSyncer(&config.File, config.File.InfoFilePath), infoLevelEnabler)
		cores = append(cores, infoFileCore)

		// 只记录 Error 及以上级别的日志
		errorLevelEnabler := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel
		})
		errorFileCore := zapcore.NewCore(encoder, getFileWriteSyncer(&config.File, config.File.ErrorFilePath), errorLevelEnabler)
		cores = append(cores, errorFileCore)
	}

	// Loki 日志
	if config.Loki.Enable {
		lokiCore := zapcore.NewCore(encoder, writer.NewLokiWriter(&config.Loki), config.ParseLogLevel())
		cores = append(cores, lokiCore)
	}

	// 组合多个日志 Core
	core := zapcore.NewTee(cores...)

	// 创建 Logger
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &Logger{
		driver: logger,
		config: *config,
	}
}

func GetLogger(c *config.LogConfig) *Logger {
	if c == nil {
		c = &config.LogConfig{
			Level:         "debug",
			Console:       true,
			UseJsonFormat: false,
			File: config.FileConfig{
				Enable: false,
			},
			Loki: config.LokiConfig{
				Enable: false,
			},
			HttpDebug: true,
			Debug:     true,
		}
	}

	once.Do(func() {
		// 仅在第一次调用时初始化
		instance = NewLogger(c)
	})
	return instance
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	return l
}

// Info 输出 Info 日志
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.driver.Info(msg, fields...)
}

// Error 输出 Error 日志
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.driver.Error(msg, fields...)
}

// Debug 输出 Debug 日志
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.driver.Debug(msg, fields...)
}

// Warn 输出 Warn 日志
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.driver.Warn(msg, fields...)
}

// InfoF 格式化后输出 Info 日志
func (l *Logger) InfoF(format string, args ...interface{}) {
	l.driver.Info(fmt.Sprintf(format, args...))
}

// ErrorF 格式化后输出 Error 日志
func (l *Logger) ErrorF(format string, args ...interface{}) {
	l.driver.Error(fmt.Sprintf(format, args...))
}

// DebugF 格式化后输出 Debug 日志
func (l *Logger) DebugF(format string, args ...interface{}) {
	l.driver.Debug(fmt.Sprintf(format, args...))
}

// WarnF 格式化后输出 Warn 日志
func (l *Logger) WarnF(format string, args ...interface{}) {
	l.driver.Warn(fmt.Sprintf(format, args...))
}

// InitGlobalLogger 初始化全局Logger实例
func InitGlobalLogger(config *config.LogConfig) {
	globalOnce.Do(func() {
		globalInstance = NewLogger(config)
	})
}

// GetGlobalLogger 获取全局Logger实例
func GetGlobalLogger() *Logger {
	if globalInstance == nil {
		// 如果没有初始化，使用默认配置
		InitGlobalLogger(&config.LogConfig{
			Level:         "debug",
			Console:       true,
			UseJsonFormat: false,
			File: config.FileConfig{
				Enable: false,
			},
			Loki: config.LokiConfig{
				Enable: false,
			},
			HttpDebug: true,
			Debug:     true,
		})
	}
	return globalInstance
}

// 全局便捷函数

// Info 全局Info日志
func Info(msg string, fields ...zap.Field) {
	GetGlobalLogger().Info(msg, fields...)
}

// Error 全局Error日志
func Error(msg string, fields ...zap.Field) {
	GetGlobalLogger().Error(msg, fields...)
}

// Debug 全局Debug日志
func Debug(msg string, fields ...zap.Field) {
	GetGlobalLogger().Debug(msg, fields...)
}

// Warn 全局Warn日志
func Warn(msg string, fields ...zap.Field) {
	GetGlobalLogger().Warn(msg, fields...)
}

// InfoF 全局格式化Info日志
func InfoF(format string, args ...interface{}) {
	GetGlobalLogger().InfoF(format, args...)
}

// ErrorF 全局格式化Error日志
func ErrorF(format string, args ...interface{}) {
	GetGlobalLogger().ErrorF(format, args...)
}

// DebugF 全局格式化Debug日志
func DebugF(format string, args ...interface{}) {
	GetGlobalLogger().DebugF(format, args...)
}

// WarnF 全局格式化Warn日志
func WarnF(format string, args ...interface{}) {
	GetGlobalLogger().WarnF(format, args...)
}
