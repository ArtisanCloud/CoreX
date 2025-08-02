package contract

import "time"

// ToolResult 封装单个 tool 或 step 的执行结果。
type ToolResult struct {
	Output   map[string]interface{} // 原始输出数据
	Success  bool                   // 是否成功
	Error    error                  // 执行过程中的错误（如果有）
	Duration time.Duration          // 执行耗时
}
