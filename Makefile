# Eino Agent API 主 Makefile
# 通过引入 make_files 目录下的不同场景文件来组织构建任务

# 默认目标
.DEFAULT_GOAL := help

# 引入配置文件和各个场景的 Makefile
include make_files/config.mk
include make_files/build.mk
include make_files/test.mk
include make_files/dev.mk
include make_files/clean.mk
include make_files/docker.mk

# 帮助信息
.PHONY: help
help:
	@echo "🤖 Eino Agent API 构建工具"
	@echo ""
	@echo "📦 构建相关:"
	@echo "  make build        - 构建项目"
	@echo "  make build-all    - 构建所有组件"
	@echo "  make build-agent  - 构建 Agent 服务"
	@echo "  make build-test   - 构建测试工具"
	@echo ""
	@echo "🧪 测试相关:"
	@echo "  make test-health  - 测试健康检查"
	@echo "  make test-chat    - 测试基本聊天"
	@echo "  make test-plan    - 测试计划生成"
	@echo "  make test-execute - 测试执行意图"
	@echo "  make test-config  - 测试配置验证"
	@echo "  make test-batch   - 测试批量请求"
	@echo "  make test-all     - 运行所有测试"
	@echo "  make test-quick   - 快速测试"
	@echo "  make unit-test    - 运行单元测试"
	@echo ""
	@echo "🚀 开发相关:"
	@echo "  make dev          - 启动开发服务器"
	@echo "  make dev-watch    - 监控文件变化并重启"
	@echo "  make lint         - 代码检查"
	@echo "  make format       - 代码格式化"
	@echo ""
	@echo "🧹 清理相关:"
	@echo "  make clean        - 清理构建文件"
	@echo "  make clean-all    - 深度清理"
	@echo "  make clean-temp   - 清理临时文件"
	@echo ""
	@echo "🔧 环境变量:"
	@echo "  API_BASE_URL     - API 基础URL (默认: http://localhost:8080/api/v1)"
	@echo "  BUILD_DIR        - 构建输出目录 (默认: bin)"
	@echo "  GO_FLAGS         - Go 编译参数"
	@echo ""
	@echo "📖 使用示例:"
	@echo "  make dev                    # 启动开发服务器"
	@echo "  make test-quick             # 快速测试"
	@echo "  API_BASE_URL=http://localhost:9000/api/v1 make test-all"

# 全局变量
API_BASE_URL ?= http://localhost:8080/api/v1
BUILD_DIR ?= bin
GO_FLAGS ?= -v

# 导出变量供子 Makefile 使用
export API_BASE_URL
export BUILD_DIR
export GO_FLAGS