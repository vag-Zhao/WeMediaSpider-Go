package utils

import (
	"math/rand"
	"sync"
	"time"
)

// RateLimiter 智能请求频率限制器
type RateLimiter struct {
	mu              sync.Mutex
	lastRequest     time.Time
	minInterval     time.Duration
	maxInterval     time.Duration
	requestCount    int
	windowStart     time.Time
	maxRequests     int // 时间窗口内最大请求数
	failureCount    int // 连续失败次数
	lastFailureTime time.Time
	adaptiveMode    bool // 自适应模式
}

// NewRateLimiter 创建智能频率限制器
// minInterval: 最小请求间隔
// maxInterval: 最大请求间隔（用于随机抖动）
// maxRequests: 每分钟最大请求数
func NewRateLimiter(minInterval, maxInterval time.Duration, maxRequests int) *RateLimiter {
	return &RateLimiter{
		minInterval:  minInterval,
		maxInterval:  maxInterval,
		lastRequest:  time.Now().Add(-maxInterval),
		windowStart:  time.Now(),
		maxRequests:  maxRequests,
		failureCount: 0,
		adaptiveMode: true,
	}
}

// Wait 等待直到可以发送下一个请求（智能延迟）
func (rl *RateLimiter) Wait() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// 检查时间窗口（1分钟）
	if now.Sub(rl.windowStart) >= time.Minute {
		rl.windowStart = now
		rl.requestCount = 0
	}

	// 如果超过窗口限制，等待到下一个窗口
	if rl.requestCount >= rl.maxRequests {
		waitTime := time.Minute - now.Sub(rl.windowStart)
		if waitTime > 0 {
			time.Sleep(waitTime)
			rl.windowStart = time.Now()
			rl.requestCount = 0
		}
	}

	// 自适应延迟：根据失败次数动态调整
	var delay time.Duration
	if rl.adaptiveMode && rl.failureCount > 0 {
		// 有失败记录，增加延迟
		multiplier := 1.0 + float64(rl.failureCount)*0.5 // 每次失败增加50%延迟
		if multiplier > 3.0 {
			multiplier = 3.0 // 最多3倍延迟
		}
		delay = time.Duration(float64(rl.minInterval) * multiplier)

		// 如果最近失败过（5分钟内），保持较高延迟
		if now.Sub(rl.lastFailureTime) < 5*time.Minute {
			delay = time.Duration(float64(delay) * 1.5)
		}
	} else {
		// 正常模式：轻微随机抖动（±20%）
		jitter := float64(rl.minInterval) * 0.2 * (rand.Float64() - 0.5)
		delay = rl.minInterval + time.Duration(jitter)
	}

	// 确保不超过最大间隔
	if delay > rl.maxInterval {
		delay = rl.maxInterval
	}

	// 计算距离上次请求的时间
	elapsed := now.Sub(rl.lastRequest)
	if elapsed < delay {
		time.Sleep(delay - elapsed)
	}

	rl.lastRequest = time.Now()
	rl.requestCount++
}

// RecordSuccess 记录成功请求（降低失败计数）
func (rl *RateLimiter) RecordSuccess() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.failureCount > 0 {
		rl.failureCount--
	}
}

// RecordFailure 记录失败请求（增加失败计数）
func (rl *RateLimiter) RecordFailure() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.failureCount++
	rl.lastFailureTime = time.Now()
	if rl.failureCount > 10 {
		rl.failureCount = 10 // 最多记录10次
	}
}

// GetCurrentDelay 获取当前延迟时间（用于日志）
func (rl *RateLimiter) GetCurrentDelay() time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.failureCount > 0 {
		multiplier := 1.0 + float64(rl.failureCount)*0.5
		if multiplier > 3.0 {
			multiplier = 3.0
		}
		return time.Duration(float64(rl.minInterval) * multiplier)
	}
	return rl.minInterval
}

// GetRandomDelay 获取随机延迟时间
func GetRandomDelay(min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}
	return min + time.Duration(rand.Int63n(int64(max-min)))
}

// GetExponentialBackoff 获取指数退避延迟
func GetExponentialBackoff(attempt int, baseDelay time.Duration) time.Duration {
	if attempt <= 0 {
		return baseDelay
	}
	// 2^attempt * baseDelay，最大不超过2分钟（降低最大延迟）
	delay := baseDelay * time.Duration(1<<uint(attempt))
	maxDelay := 2 * time.Minute
	if delay > maxDelay {
		delay = maxDelay
	}
	// 添加随机抖动（±20%）
	jitter := time.Duration(rand.Int63n(int64(delay) / 5))
	if rand.Intn(2) == 0 {
		delay += jitter
	} else {
		delay -= jitter
	}
	return delay
}
