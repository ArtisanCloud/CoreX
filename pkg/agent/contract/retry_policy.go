package contract

import (
	"math"
	"math/rand"
	"time"
)

// RetryPolicy 定义失败后的重试策略。
type RetryPolicy struct {
	MaxAttempts int           // 包含第一次尝试在内
	Interval    time.Duration // 基础等待时间
	Backoff     bool          // 是否启用指数退避
	Jitter      bool          // 是否加抖动（防止雪崩）
	MaxInterval time.Duration // 最大退避上限（0 表示不限制）
}

// ShouldRetry 判断当前 attempt（从 1 开始）之后是否还可以重试。
func (r *RetryPolicy) ShouldRetry(attempt int) bool {
	if r == nil {
		return false
	}
	return attempt < r.MaxAttempts
}

// NextInterval 计算下一次重试前应等待的时长（基于当前 attempt）。
func (r *RetryPolicy) NextInterval(attempt int) time.Duration {
	if r == nil {
		return 0
	}
	base := float64(r.Interval)
	var interval float64
	if r.Backoff {
		interval = base * math.Pow(2, float64(attempt-1))
	} else {
		interval = base
	}
	if r.MaxInterval > 0 && interval > float64(r.MaxInterval) {
		interval = float64(r.MaxInterval)
	}
	if r.Jitter {
		min := interval / 2
		interval = min + rand.Float64()*(interval-min)
	}
	return time.Duration(interval)
}
